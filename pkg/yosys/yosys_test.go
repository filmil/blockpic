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

// A tie-off's one output is named as the skins' tie-off has it, whatever
// the block calls it, so that the wire from it finds the pin.
func TestTieOffPins(t *testing.T) {
	diag := &ast.Diagram{
		Name: "D",
		Blocks: []ast.Block{
			{Label: "v", Kind: "VCC", Outputs: []ast.Pin{{Name: "V"}}},
			{Label: "g", Kind: "GND", Outputs: []ast.Pin{{Name: "out"}}},
			{Label: "c", Kind: "CONST", Name: "4'b0001", Outputs: []ast.Pin{{Name: "O"}}},
			{Label: "x", Kind: "AND", Inputs: []ast.Pin{{Name: "a"}, {Name: "b"}}, Outputs: []ast.Pin{{Name: "y"}}},
		},
		Wires: []ast.Wire{
			{Source: ast.PinRef{Block: "v", Pin: "V"}, Destination: ast.PinRef{Block: "x", Pin: "a"}},
			{Source: ast.PinRef{Block: "g", Pin: "out"}, Destination: ast.PinRef{Block: "x", Pin: "b"}},
		},
	}
	b, err := Convert(diag)
	if err != nil {
		t.Fatal(err)
	}
	var netlist Netlist
	if err := json.Unmarshal(b, &netlist); err != nil {
		t.Fatal(err)
	}
	cells := netlist.Modules["D"].Cells
	for label, pin := range map[string]string{"v": "P", "g": "G", "c": "O"} {
		if _, ok := cells[label].Connections[pin]; !ok || len(cells[label].Connections) != 1 {
			t.Errorf("%s connections = %v, want just %s", label, cells[label].Connections, pin)
		}
	}
	if cells["v"].Connections["P"][0] != cells["x"].Connections["A"][0] {
		t.Errorf("VCC's net %v does not reach the AND's input %v", cells["v"].Connections, cells["x"].Connections)
	}
}
