// SPDX-License-Identifier: Apache-2.0
package layout

import (
	"testing"
	"github.com/filmil/hodos/pkg/ast"
)

func TestAutoLayout(t *testing.T) {
	diag := &ast.Diagram{
		Blocks: []ast.Block{
			{Name: "B1"},
			{Name: "B2"},
			{Name: "B3"},
		},
		Wires: []ast.Wire{
			{Source: ast.PinRef{Block: "B1"}, Destination: ast.PinRef{Block: "B2"}},
			{Source: ast.PinRef{Block: "B2"}, Destination: ast.PinRef{Block: "B3"}},
		},
	}

	AutoLayout(diag)

	if len(diag.Layout) != 3 {
		t.Errorf("expected 3 layout entries, got %d", len(diag.Layout))
	}

	layoutMap := make(map[string]ast.LayoutEntry)
	for _, l := range diag.Layout {
		layoutMap[l.Block] = l
	}

	if layoutMap["B1"].X >= layoutMap["B2"].X {
		t.Errorf("expected B1 to be to the left of B2")
	}

	if layoutMap["B2"].X >= layoutMap["B3"].X {
		t.Errorf("expected B2 to be to the left of B3")
	}
}
