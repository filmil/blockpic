// SPDX-License-Identifier: Apache-2.0
package refine

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/filmil/hodos/pkg/ast"
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

	for _, w := range diag.Wires {
		if w.Thickness > 1 {
			srcKey := fmt.Sprintf("%s.%s", w.Source.Block, w.Source.Pin)
			bit := sourceBits[srcKey]
			netClass := fmt.Sprintf("net_%d", bit)

			// Find all <line ... class="netClass" /> and modify their stroke
			lineRe := regexp.MustCompile(fmt.Sprintf(`(<line[^>]*class="%s"[^>]*>)`, regexp.QuoteMeta(netClass)))
			
			svgStr = lineRe.ReplaceAllStringFunc(svgStr, func(match string) string {
				// We want a solid thick black line.
				// By replacing the closing '/>', we add the stroke and stroke-width.
				return strings.Replace(match, "/>", ` stroke="#000" stroke-width="4" />`, 1)
			})
		}
	}

	return []byte(svgStr), nil
}
