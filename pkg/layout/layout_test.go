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
		},
	}

	AutoLayout(diag)

	if len(diag.Layout) != 2 {
		t.Errorf("expected 2 layout entries, got %d", len(diag.Layout))
	}

	if diag.Layout[0].Block != "B1" || diag.Layout[0].X != 20 {
		t.Errorf("unexpected layout for B1: %+v", diag.Layout[0])
	}

	if diag.Layout[1].Block != "B2" || diag.Layout[1].X != 140 {
		t.Errorf("unexpected layout for B2: %+v", diag.Layout[1])
	}
}
