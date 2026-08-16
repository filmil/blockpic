// SPDX-License-Identifier: Apache-2.0
package parser

import (
	"testing"
	"github.com/filmil/blockpic/pkg/scanner"
)

func TestParseDiagram(t *testing.T) {
	input := `
diagram MyLogic {
    block LUT6_A {
        kind: "LUT"
        params: { init: "0x8000" }
        inputs: I0, I1 [left]
        outputs: O [right]
        size: (40, 60)
    }

    block DFF_1 {
        kind: "FLIPFLOP"
        name: "DFF Display"
        hint: "show-ports"
        inputs: D, CLK [bottom]
        outputs: Q
    }

    wire LUT6_A.O -> DFF_1.D {
        hint: "Z-shape"
    }

    wire BUS_IN.O -> LUT6_A.I0 {
        thickness: 2
        crossline: 8
    }

    layout {
        LUT6_A at (10, 10)
        DFF_1 at (100, 10)
    }
}
`
	s := scanner.New(input)
	p := New(s)
	diag := p.ParseDiagram()

	if len(p.Errors()) > 0 {
		for _, err := range p.Errors() {
			t.Errorf("parser error: %s", err)
		}
		t.FailNow()
	}

	if diag == nil {
		t.Fatalf("ParseDiagram() returned nil")
	}

	if diag.Name != "MyLogic" {
		t.Errorf("expected diagram name 'MyLogic', got '%s'", diag.Name)
	}

	if len(diag.Blocks) != 2 {
		t.Errorf("expected 2 blocks, got %d", len(diag.Blocks))
	}

	if diag.Blocks[0].Label != "LUT6_A" {
		t.Errorf("expected first block label 'LUT6_A', got '%s'", diag.Blocks[0].Label)
	}

	if diag.Blocks[0].Kind != "LUT" {
		t.Errorf("expected block kind 'LUT', got '%s'", diag.Blocks[0].Kind)
	}

	if diag.Blocks[1].Label != "DFF_1" {
		t.Errorf("expected second block label 'DFF_1', got '%s'", diag.Blocks[1].Label)
	}

	if diag.Blocks[1].Name != "DFF Display" {
		t.Errorf("expected second block name 'DFF Display', got '%s'", diag.Blocks[1].Name)
	}

	if diag.Blocks[1].Hint != "show-ports" {
		t.Errorf("expected second block hint 'show-ports', got '%s'", diag.Blocks[1].Hint)
	}

	if len(diag.Wires) != 2 {
		t.Errorf("expected 2 wires, got %d", len(diag.Wires))
	}

	if len(diag.Layout) != 2 {
		t.Errorf("expected 2 layout entries, got %d", len(diag.Layout))
	}
}
