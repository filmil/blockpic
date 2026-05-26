// SPDX-License-Identifier: Apache-2.0
package layout

import (
	"github.com/filmil/hodos/pkg/ast"
)

func AutoLayout(diag *ast.Diagram) {
	// If layout is already populated, don't overwrite? 
	// Or only add missing ones.
	layoutMap := make(map[string]bool)
	for _, l := range diag.Layout {
		layoutMap[l.Block] = true
	}

	curX, curY := 20.0, 40.0
	spacingX := 120.0

	for _, b := range diag.Blocks {
		if _, ok := layoutMap[b.Name]; !ok {
			diag.Layout = append(diag.Layout, ast.LayoutEntry{
				Block: b.Name,
				X:     curX,
				Y:     curY,
			})
			curX += spacingX
			// Simple wrapping if too many blocks? (Maybe later)
		}
	}
}
