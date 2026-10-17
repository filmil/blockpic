// SPDX-License-Identifier: Apache-2.0
package refine

import (
	"fmt"
	"html"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/filmil/blockpic/pkg/ast"
)

// Process takes the parsed diagram and raw SVG bytes,
// and returns refined SVG bytes with groups and thick buses. A group's box
// is drawn around its blocks.
func Process(diag *ast.Diagram, svgBytes []byte) ([]byte, error) {
	return ProcessLaidOut(diag, svgBytes, nil)
}

// Box is where the layout put a group, in the drawing's coordinates.
type Box struct {
	X, Y, Width, Height float64
}

// ProcessLaidOut is Process for a drawing whose layout gave its groups
// boxes of their own, keyed by the group's index in the diagram. Such a
// box is drawn where the layout put it: the layout kept every other block
// out of it. A group without one gets a box around its blocks.
func ProcessLaidOut(diag *ast.Diagram, svgBytes []byte, boxes map[int]Box) ([]byte, error) {
	// Let's use string manipulation or a lightweight XML library.
	// Since netlistsvg output is predictable, we can use regexes to find bounds.
	svgStr := string(svgBytes)

	// 1. Add dashed bounding boxes for groups
	for gi, group := range diag.Groups {
		if b, ok := boxes[gi]; ok {
			svgStr = strings.Replace(svgStr, "</svg>", groupBoxSVG(group.Name, b.X, b.Y, b.Width, b.Height), 1)
			continue
		}
		minX, minY := math.MaxFloat64, math.MaxFloat64
		maxX, maxY := -math.MaxFloat64, -math.MaxFloat64
		found := false

		for _, blockName := range group.Blocks {
			// Look for <g id="cell_BLOCKNAME" transform="translate(X,Y)" ... s:width="W" s:height="H">
			pattern := fmt.Sprintf(`<g[^>]*id="cell_%s"[^>]*>`, regexp.QuoteMeta(blockName))
			re := regexp.MustCompile(pattern)
			match := re.FindString(svgStr)
			if match == "" {
				continue
			}

			// Extract translate(X,Y) or translate(X, Y)
			transRe := regexp.MustCompile(`translate\s*\(\s*([-\d.]+)\s*,\s*([-\d.]+)\s*\)`)
			transMatch := transRe.FindStringSubmatch(match)
			var tx, ty float64
			if len(transMatch) == 3 {
				tx, _ = strconv.ParseFloat(transMatch[1], 64)
				ty, _ = strconv.ParseFloat(transMatch[2], 64)
			}

			// Extract s:width and s:height
			wRe := regexp.MustCompile(`s:width="([-\d.]+)"`)
			wMatch := wRe.FindStringSubmatch(match)
			var w float64
			if len(wMatch) == 2 {
				w, _ = strconv.ParseFloat(wMatch[1], 64)
			}

			hRe := regexp.MustCompile(`s:height="([-\d.]+)"`)
			hMatch := hRe.FindStringSubmatch(match)
			var h float64
			if len(hMatch) == 2 {
				h, _ = strconv.ParseFloat(hMatch[1], 64)
			}
			// A generic cell grows with its ports, past the s:height of
			// its template; its body says how tall it is.
			if strings.Contains(match, `s:type="generic"`) {
				start := strings.Index(svgStr, match) + len(match)
				end := strings.Index(svgStr[start:], "</g>")
				if end >= 0 {
					if b := bodyHeight.FindStringSubmatch(svgStr[start : start+end]); b != nil {
						h, _ = strconv.ParseFloat(b[1], 64)
					}
				}
			}

			if w > 0 && h > 0 {
				found = true
				if tx < minX { minX = tx }
				if ty < minY { minY = ty }
				if tx+w > maxX { maxX = tx + w }
				if ty+h > maxY { maxY = ty + h }
			}
		}

		if found {
			// The label has a row of its own at the top of the box,
			// above the titles that blocks carry over their bodies.
			pad, labelRow := 20.0, 10.0
			minX -= pad
			minY -= pad + labelRow
			maxX += pad
			maxY += pad
			w := maxX - minX
			h := maxY - minY

			svgStr = strings.Replace(svgStr, "</svg>", groupBoxSVG(group.Name, minX, minY, w, h), 1)
		}
	}

	// 2. Thick Hollow Bus Arrows
	// Reconstruct the bit numbering from yosys.go
	bitCounter := 2
	sourceBits := make(map[string]int)
	for _, w := range diag.Wires {
		srcKey := fmt.Sprintf("%s.%s", w.Source.Block, w.Source.Pin)
		if _, exists := sourceBits[srcKey]; !exists {
			sourceBits[srcKey] = bitCounter
			bitCounter++
		}
	}

	// Every wire from one pin is one net, so a net is thickened once
	// however many thick wires leave the pin.
	thickened := make(map[int]bool)
	for _, w := range diag.Wires {
		if w.Thickness > 1 {
			srcKey := fmt.Sprintf("%s.%s", w.Source.Block, w.Source.Pin)
			bit := sourceBits[srcKey]
			if thickened[bit] {
				continue
			}
			thickened[bit] = true
			netClass := fmt.Sprintf("net_%d", bit)

			// Find all <line ... class="netClass" /> and modify their stroke
			lineRe := regexp.MustCompile(fmt.Sprintf(`(<line[^>]*class="%s"[^>]*>)`, regexp.QuoteMeta(netClass)))
			
			svgStr = lineRe.ReplaceAllStringFunc(svgStr, func(match string) string {
				// We want a solid thick black line.
				// By replacing the closing '/>', we add the stroke and stroke-width.
				return strings.Replace(match, "/>", ` stroke="#000" stroke-width="3" />`, 1)
			})
		}
	}

	svgStr = drawBusWidths(diag, svgStr, sourceBits)
	svgStr = drawNames(diag, svgStr)
	svgStr = fitCanvas(svgStr)
	svgStr = strokeLines(svgStr)

	return []byte(svgStr), nil
}

// groupBoxSVG is the dashed box of a group, with its name in the top left
// corner, and the end of the drawing after them.
func groupBoxSVG(name string, x, y, w, h float64) string {
	rectStr := fmt.Sprintf(`<rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" class="groupBox" fill="none" stroke="#000" stroke-dasharray="5,5" stroke-width="1"/>`, x, y, w, h)
	textStr := fmt.Sprintf(`<text x="%.1f" y="%.1f" class="groupLabel" text-anchor="start" style="font-weight:bold">%s</text>`, x+5, y+11, escapeText(name))
	return fmt.Sprintf("  %s\n  %s\n</svg>", rectStr, textStr)
}

// cellTitle finds the title of a skinned cell: the text netlistsvg fills
// with the cell's label. A skin may give it any class besides the cell's.
var cellTitle = `(<text[^>]*class="[^"]*\bcell_%s"[^>]*s:attribute="ref"[^>]*>)[^<]*(</text>)`

// genericCell finds a generic cell's group and its body, whose height
// netlistsvg sets by the number of ports.
var genericCell = `<g s:type="generic" transform="translate\(([-\d.]+),\s*([-\d.]+)\)"[^>]*id="cell_%s">\s*<text[^>]*>[^<]*</text>\s*<rect\b[^>]*?\swidth="([-\d.]+)" height="([-\d.]+)"`

// drawNames shows each block's name, when it has one other than its kind.
// netlistsvg titles a skinned cell with its label, so the title is
// replaced by the name. It titles a generic cell with its kind, which
// says what the cell is, so the name goes under the cell instead.
func drawNames(diag *ast.Diagram, svgStr string) string {
	for _, b := range diag.Blocks {
		if b.Name == "" || b.Name == b.Kind {
			continue
		}
		label := regexp.QuoteMeta(b.Label)
		name := escapeText(b.Name)
		if m := regexp.MustCompile(fmt.Sprintf(genericCell, label)).FindStringSubmatch(svgStr); m != nil {
			tx, _ := strconv.ParseFloat(m[1], 64)
			ty, _ := strconv.ParseFloat(m[2], 64)
			w, _ := strconv.ParseFloat(m[3], 64)
			h, _ := strconv.ParseFloat(m[4], 64)
			text := fmt.Sprintf(`<text x="%.1f" y="%.1f" class="nodelabel cell_%s" text-anchor="middle">%s</text>`,
				tx+w/2, ty+h+11, b.Label, name)
			svgStr = strings.Replace(svgStr, "</svg>", "  "+text+"\n</svg>", 1)
			continue
		}
		re := regexp.MustCompile(fmt.Sprintf(cellTitle, label))
		svgStr = re.ReplaceAllString(svgStr, "${1}"+strings.ReplaceAll(name, "$", "$$")+"${2}")
	}
	return svgStr
}

// escapeText makes s safe as the text of an SVG element.
func escapeText(s string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;").Replace(s)
}

// portLabel finds a port cell, or a tie-off, and the text that names it.
// A tie-off's value is set to the left of its stub.
var portLabel = regexp.MustCompile(`<g s:type="(?:inputExt|outputExt|GND|VCC|CONST)" transform="translate\(([-\d.]+),\s*([-\d.]+)\)"[^>]*>\s*<text x="([-\d.]+)"[^>]*class="([^"]*)"[^>]*>([^<]*)</text>`)

// bodyHeight finds the height of a generic cell's body.
var bodyHeight = regexp.MustCompile(`<rect\b[^>]*?\sheight="([-\d.]+)"[^>]*s:generic="body"`)

// groupBox finds the dashed box of a group.
var groupBox = regexp.MustCompile(`<rect x="([-\d.]+)" y="([-\d.]+)" width="([-\d.]+)" height="([-\d.]+)" class="groupBox"`)

// svgSize finds the size netlistsvg gives the drawing.
var svgSize = regexp.MustCompile(`<svg([^>]*) width="([\d.]+)" height="([\d.]+)"`)

// charWidth is the width, in pixels, of an average character of the
// skins' 9 to 10 pixel faces, a little wide to be safe.
const charWidth = 5.5

// fitCanvas grows the drawing so that the names of its ports and
// tie-offs, and the boxes of its groups, fit. netlistsvg sizes the drawing to its cells, so a
// name set beside a port at the edge, or a group box around a block at the
// edge, falls outside it. The drawing gets a viewBox that takes them in;
// a port name's width is estimated from its length.
func fitCanvas(svgStr string) string {
	size := svgSize.FindStringSubmatch(svgStr)
	if size == nil {
		return svgStr
	}
	width, _ := strconv.ParseFloat(size[2], 64)
	height, _ := strconv.ParseFloat(size[3], 64)
	minX, maxX := 0.0, width
	minY, maxY := 0.0, height
	for _, m := range groupBox.FindAllStringSubmatch(svgStr, -1) {
		var c [4]float64
		for i := range c {
			c[i], _ = strconv.ParseFloat(m[i+1], 64)
		}
		minX, minY = math.Min(minX, c[0]-1), math.Min(minY, c[1]-1)
		maxX, maxY = math.Max(maxX, c[0]+c[2]+1), math.Max(maxY, c[1]+c[3]+1)
	}
	for _, m := range portLabel.FindAllStringSubmatch(svgStr, -1) {
		tx, _ := strconv.ParseFloat(m[1], 64)
		x, _ := strconv.ParseFloat(m[3], 64)
		w := float64(len([]rune(html.UnescapeString(m[5])))) * charWidth
		left := tx + x
		switch {
		case strings.Contains(m[4], "inputPortLabel"), strings.Contains(m[4], "pinOut"):
			left -= w
		case strings.Contains(m[4], "pinIn"):
		default:
			left -= w / 2
		}
		minX = math.Min(minX, left-2)
		maxX = math.Max(maxX, left+w+2)
	}
	if minX == 0 && maxX == width && minY == 0 && maxY == height {
		return svgStr
	}
	w, h := maxX-minX, maxY-minY
	return strings.Replace(svgStr, size[0], fmt.Sprintf(`<svg%s width="%g" height="%g" viewBox="%g %g %g %g"`,
		size[1], w, h, minX, minY, w, h), 1)
}

// netLine finds a straight segment of a net.
var netLine = `<line x1="([-\d.]+)" x2="([-\d.]+)" y1="([-\d.]+)" y2="([-\d.]+)" class="%s"`

// busLabel is what the bus-width mark of a wire says: the wire's type
// when it has one, else its crossline, else its thickness. It is empty
// for a wire of one bit with no type.
func busLabel(w ast.Wire) string {
	switch {
	case w.Type != "":
		return w.Type
	case w.Crossline > 0:
		return strconv.Itoa(w.Crossline)
	case w.Thickness > 1:
		return strconv.Itoa(w.Thickness)
	}
	return ""
}

// drawBusWidths marks every bus with a short slash across it and its width
// or type beside the slash, as hardware figures do. The mark goes at the
// middle of the net's longest horizontal segment, or of its longest
// vertical one when it has none. A net is marked once, however many wires
// leave its pin; sourceBits numbers the nets as yosys.Convert does.
func drawBusWidths(diag *ast.Diagram, svgStr string, sourceBits map[string]int) string {
	labels := make(map[int]string)
	var order []int
	for _, w := range diag.Wires {
		label := busLabel(w)
		if label == "" {
			continue
		}
		bit := sourceBits[fmt.Sprintf("%s.%s", w.Source.Block, w.Source.Pin)]
		if _, ok := labels[bit]; !ok {
			order = append(order, bit)
			labels[bit] = label
		}
	}

	var marks strings.Builder
	for _, bit := range order {
		re := regexp.MustCompile(fmt.Sprintf(netLine, fmt.Sprintf("net_%d", bit)))
		var best [4]float64
		bestLen, bestHorizontal := -1.0, false
		for _, m := range re.FindAllStringSubmatch(svgStr, -1) {
			var c [4]float64
			for i := range c {
				c[i], _ = strconv.ParseFloat(m[i+1], 64)
			}
			horizontal := c[2] == c[3]
			length := math.Abs(c[1]-c[0]) + math.Abs(c[3]-c[2])
			if (horizontal && !bestHorizontal) || (horizontal == bestHorizontal && length > bestLen) {
				best, bestLen, bestHorizontal = c, length, horizontal
			}
		}
		if bestLen < 0 {
			continue
		}
		x, y := (best[0]+best[1])/2, (best[2]+best[3])/2
		label := escapeText(labels[bit])
		if bestHorizontal {
			fmt.Fprintf(&marks, `  <line x1="%.1f" x2="%.1f" y1="%.1f" y2="%.1f" class="busWidth"/>`+"\n", x-3, x+3, y+5, y-5)
			fmt.Fprintf(&marks, `  <text x="%.1f" y="%.1f" class="busWidth" style="font-size:7px">%s</text>`+"\n", x+4, y-4, label)
		} else {
			fmt.Fprintf(&marks, `  <line x1="%.1f" x2="%.1f" y1="%.1f" y2="%.1f" class="busWidth"/>`+"\n", x-5, x+5, y+3, y-3)
			fmt.Fprintf(&marks, `  <text x="%.1f" y="%.1f" class="busWidth" style="font-size:7px">%s</text>`+"\n", x+6, y+3, label)
		}
	}
	if marks.Len() == 0 {
		return svgStr
	}
	return strings.Replace(svgStr, "</svg>", marks.String()+"</svg>", 1)
}

// bareLine finds a line, with its attributes.
var bareLine = regexp.MustCompile(`<line\b([^>]*?)\s*/>`)

// strokeLines gives every line without a stroke attribute a black one.
// netlistsvg draws the nets as lines that its stylesheet strokes, and a
// line has no stroke by default, so a renderer that ignores the stylesheet
// would draw no nets at all. The stylesheet outranks the attribute
// wherever it is read.
func strokeLines(svgStr string) string {
	return bareLine.ReplaceAllStringFunc(svgStr, func(line string) string {
		if strings.Contains(line, " stroke=") {
			return line
		}
		m := bareLine.FindStringSubmatch(line)
		return `<line` + m[1] + ` stroke="#000"/>`
	})
}
