// SPDX-License-Identifier: Apache-2.0
package yosys

import (
	"encoding/json"
	"testing"

	"github.com/filmil/blockpic/pkg/ast"
)

func TestConvert(t *testing.T) {
	diag := &ast.Diagram{
		Name: "TestDiag",
		Blocks: []ast.Block{
			{
				Label: "A",
				Kind: "AND",
				Inputs: []ast.Pin{{Name: "in1"}, {Name: "in2"}},
				Outputs: []ast.Pin{{Name: "out"}},
			},
			{
				Label: "B",
				Kind: "NOT",
				Inputs: []ast.Pin{{Name: "in"}},
				Outputs: []ast.Pin{{Name: "out"}},
			},
		},
		Wires: []ast.Wire{
			{
				Source: ast.PinRef{Block: "A", Pin: "out"},
				Destination: ast.PinRef{Block: "B", Pin: "in"},
			},
		},
	}

	b, err := Convert(diag)
	if err != nil {
		t.Fatalf("Convert failed: %v", err)
	}

	var netlist Netlist
	if err := json.Unmarshal(b, &netlist); err != nil {
		t.Fatalf("Failed to parse output JSON: %v", err)
	}

	mod, ok := netlist.Modules["TestDiag"]
	if !ok {
		t.Fatalf("Module TestDiag not found in JSON")
	}

	if len(mod.Cells) != 2 {
		t.Errorf("Expected 2 cells, got %d", len(mod.Cells))
	}

	cellA := mod.Cells["A"]
	cellB := mod.Cells["B"]

	if cellA.Type != "$and" {
		t.Errorf("Expected A type $and, got %s", cellA.Type)
	}
	
	outBit := cellA.Connections["Y"][0]
	inBit := cellB.Connections["A"][0]

	if outBit != inBit {
		t.Errorf("Expected connection bits to match, A.out=%d, B.in=%d", outBit, inBit)
	}
}
