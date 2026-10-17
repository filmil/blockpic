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
// and returns refined SVG bytes with groups and thick buses.
func Process(diag *ast.Diagram, svgBytes []byte) ([]byte, error) {
	// Let's use string manipulation or a lightweight XML library.
	// Since netlistsvg output is predictable, we can use regexes to find bounds.
	svgStr := string(svgBytes)

	// 1. Add dashed bounding boxes for groups
	for _, group := range diag.Groups {
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

			if w > 0 && h > 0 {
				found = true
				if tx < minX { minX = tx }
				if ty < minY { minY = ty }
				if tx+w > maxX { maxX = tx + w }
				if ty+h > maxY { maxY = ty + h }
			}
		}

		if found {
			pad := 20.0
			minX -= pad
			minY -= pad
			maxX += pad
			maxY += pad
			w := maxX - minX
			h := maxY - minY

			rectStr := fmt.Sprintf(`<rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" fill="none" stroke="#000" stroke-dasharray="5,5" stroke-width="1"/>`, minX, minY, w, h)
			textStr := fmt.Sprintf(`<text x="%.1f" y="%.1f" font-family="monospace" font-size="12px" font-weight="bold">%s</text>`, minX+5, minY+15, group.Name)
			svgStr = strings.Replace(svgStr, "</svg>", fmt.Sprintf("  %s\n  %s\n</svg>", rectStr, textStr), 1)
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

	svgStr = drawNames(diag, svgStr)
	svgStr = fitPortLabels(svgStr)

	return []byte(svgStr), nil
}

// cellTitle finds the title of a skinned cell: the text netlistsvg fills
// with the cell's label. A skin may give it any class besides the cell's.
var cellTitle = `(<text[^>]*class="[^"]*\bcell_%s"[^>]*s:attribute="ref"[^>]*>)[^<]*(</text>)`

// genericCell finds a generic cell's group and its body, whose height
// netlistsvg sets by the number of ports.
var genericCell = `<g s:type="generic" transform="translate\(([-\d.]+),\s*([-\d.]+)\)"[^>]*id="cell_%s">\s*<text[^>]*>[^<]*</text>\s*<rect width="([-\d.]+)" height="([-\d.]+)"`

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

// portLabel finds a port cell and the text that names it.
var portLabel = regexp.MustCompile(`<g s:type="(?:inputExt|outputExt)" transform="translate\(([-\d.]+),\s*([-\d.]+)\)"[^>]*>\s*<text x="([-\d.]+)"[^>]*class="([^"]*)"[^>]*>([^<]*)</text>`)

// svgSize finds the size netlistsvg gives the drawing.
var svgSize = regexp.MustCompile(`<svg([^>]*) width="([\d.]+)" height="([\d.]+)"`)

// charWidth is the width, in pixels, of an average character of the
// skins' 9 to 10 pixel faces, a little wide to be safe.
const charWidth = 5.5

// fitPortLabels widens the drawing so that the names of its ports fit.
// netlistsvg sizes the drawing to its cells, so a name set beside a port
// at the edge falls outside it. The drawing gets a viewBox that takes in
// every port name, estimated from its length.
func fitPortLabels(svgStr string) string {
	size := svgSize.FindStringSubmatch(svgStr)
	if size == nil {
		return svgStr
	}
	width, _ := strconv.ParseFloat(size[2], 64)
	height, _ := strconv.ParseFloat(size[3], 64)
	minX, maxX := 0.0, width
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
	if minX == 0 && maxX == width {
		return svgStr
	}
	w := maxX - minX
	return strings.Replace(svgStr, size[0], fmt.Sprintf(`<svg%s width="%g" height="%g" viewBox="%g 0 %g %g"`,
		size[1], w, height, minX, w, height), 1)
}
