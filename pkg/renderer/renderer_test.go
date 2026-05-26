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
			{Block: "B1", X: 10, Y: 20},
			{Block: "B2", X: 100, Y: 20},
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

func TestRenderer_AllShapes(t *testing.T) {
	diag := &ast.Diagram{
		Name: "AllShapes",
		Blocks: []ast.Block{
			{Name: "AND1", Kind: "AND"},
			{Name: "OR1", Kind: "OR"},
			{Name: "NOT1", Kind: "NOT"},
			{Name: "XOR1", Kind: "XOR"},
			{Name: "DFF1", Kind: "DFF"},
			{Name: "MUX1", Kind: "MUX"},
			{Name: "DEMUX1", Kind: "DEMUX"},
			{Name: "VCC1", Kind: "VCC"},
			{Name: "GND1", Kind: "GND"},
			{Name: "CLK1", Kind: "CLOCK"},
			{Name: "REG1", Kind: "REGISTER"},
		},
		Layout: []ast.LayoutEntry{
			{Block: "AND1", X: 10, Y: 20},
			{Block: "OR1", X: 80, Y: 20},
			{Block: "NOT1", X: 150, Y: 20},
			{Block: "XOR1", X: 220, Y: 20},
			{Block: "DFF1", X: 10, Y: 80},
			{Block: "MUX1", X: 80, Y: 80},
			{Block: "DEMUX1", X: 150, Y: 80},
			{Block: "VCC1", X: 220, Y: 80},
			{Block: "GND1", X: 10, Y: 140},
			{Block: "CLK1", X: 80, Y: 140},
			{Block: "REG1", X: 150, Y: 140},
		},
	}

	r := New(400, 250)
	img, err := r.Render(diag)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	if img == nil {
		t.Fatalf("Rendered image is nil")
	}
}

func TestRenderer_TextCollisionShift(t *testing.T) {
	diag := &ast.Diagram{
		Name: "CollisionShift",
		Blocks: []ast.Block{
			{Label: "A", Kind: "LUT", Size: &ast.Size{W: 20, H: 20}},
			{Label: "B1", Name: "B1_Collision_Test_Gate", Kind: "AND", Size: &ast.Size{W: 50, H: 50}},
			{Label: "C", Kind: "LUT", Size: &ast.Size{W: 20, H: 20}},
		},
		Wires: []ast.Wire{
			{
				Source:      ast.PinRef{Block: "A", Pin: "O"},
				Destination: ast.PinRef{Block: "C", Pin: "I"},
				Hint:        "straight",
			},
		},
		Layout: []ast.LayoutEntry{
			{Block: "A", X: 10, Y: 80},
			{Block: "B1", X: 100, Y: 100},
			{Block: "C", X: 300, Y: 80},
		},
	}

	r := New(400, 300)
	img, err := r.Render(diag)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	if img == nil {
		t.Fatalf("Rendered image is nil")
	}
}

func TestRenderer_PortVisibilityHints(t *testing.T) {
	r := New(100, 100)

	// Case 1: AND gate (small element) - should not show ports by default
	andBlock := ast.Block{Kind: "AND"}
	if r.shouldShowPins(andBlock) {
		t.Errorf("expected AND block to hide ports by default")
	}

	// Case 2: DFF (not small element) - should show ports by default
	dffBlock := ast.Block{Kind: "DFF"}
	if !r.shouldShowPins(dffBlock) {
		t.Errorf("expected DFF block to show ports by default")
	}

	// Case 3: AND gate with hint: "show-ports" - should show ports
	andShowBlock := ast.Block{Kind: "AND", Hint: ast.HintShowPorts}
	if !r.shouldShowPins(andShowBlock) {
		t.Errorf("expected AND block with HintShowPorts to show ports")
	}

	// Case 4: DFF with hint: "hide-ports" - should hide ports
	dffHideBlock := ast.Block{Kind: "DFF", Hint: ast.HintHidePorts}
	if r.shouldShowPins(dffHideBlock) {
		t.Errorf("expected DFF block with HintHidePorts to hide ports")
	}
}
