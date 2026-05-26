// SPDX-License-Identifier: Apache-2.0
package renderer

import (
	"testing"
	"github.com/filmil/hodos/pkg/ast"
)

func TestRenderer(t *testing.T) {
	diag := &ast.Diagram{
		Name: "TestDiag",
		Blocks: []ast.Block{
			{Name: "B1", Kind: "LUT"},
			{Name: "B2", Kind: "DFF"},
		},
		Wires: []ast.Wire{
			{
				Source:      ast.PinRef{Block: "B1", Pin: "O"},
				Destination: ast.PinRef{Block: "B2", Pin: "D"},
			},
		},
		Layout: []ast.LayoutEntry{
			{Block: "B1", X: 10, Y: 10},
			{Block: "B2", X: 100, Y: 10},
		},
	}

	r := New(200, 100)
	img, err := r.Render(diag)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if img == nil {
		t.Fatalf("Rendered image is nil")
	}
}
