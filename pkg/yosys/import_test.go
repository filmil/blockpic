// SPDX-License-Identifier: Apache-2.0
package yosys

import (
	"os"
	"strings"
	"testing"

	"github.com/filmil/blockpic/pkg/ast"
	"github.com/filmil/blockpic/pkg/format"
	"github.com/filmil/blockpic/pkg/parser"
	"github.com/filmil/blockpic/pkg/scanner"
	"github.com/filmil/blockpic/pkg/validation"
)

func findBlock(t *testing.T, d *ast.Diagram, label string) ast.Block {
	t.Helper()
	for _, b := range d.Blocks {
		if b.Label == label {
			return b
		}
	}
	t.Fatalf("no block %s in %v", label, d.Blocks)
	return ast.Block{}
}

func hasWire(d *ast.Diagram, src, dst string, thickness int) bool {
	for _, w := range d.Wires {
		if w.Source.Block+"."+w.Source.Pin == src && w.Destination.Block+"."+w.Destination.Pin == dst && w.Thickness == thickness {
			return true
		}
	}
	return false
}

// up_counter.json is the top module of rules_openxc7's blinky, as
// vivado_synthesis writes it, with the "src" attributes taken out.
func TestImportOpenXC7Blinky(t *testing.T) {
	data, err := os.ReadFile("testdata/up_counter.json")
	if err != nil {
		t.Fatal(err)
	}
	d, err := Import(data, "")
	if err != nil {
		t.Fatal(err)
	}
	if d.Name != "up_counter" {
		t.Errorf("name = %s, want up_counter", d.Name)
	}
	if err := validation.Validate(d); err != nil {
		t.Errorf("Validate: %v", err)
	}

	if b := findBlock(t, d, "clk"); b.Kind != "INPUT_PIN" {
		t.Errorf("clk kind = %s, want INPUT_PIN", b.Kind)
	}
	if b := findBlock(t, d, "out"); b.Kind != "OUTPUT_PIN" {
		t.Errorf("out kind = %s, want OUTPUT_PIN", b.Kind)
	}
	// The flip-flops are hidden cells, named after the bits they hold.
	if b := findBlock(t, d, "count_7"); b.Kind != "FDRE" || b.Name != "count[7]" {
		t.Errorf("count_7 = %s %q, want FDRE count[7]", b.Kind, b.Name)
	}
	if b := findBlock(t, d, "CARRY4_1"); b.Name != "CARRY4" {
		t.Errorf("CARRY4_1 name = %q, want CARRY4", b.Name)
	}

	for _, w := range []struct {
		src, dst  string
		thickness int
	}{
		{"clk.out", "IBUF_1.I", 0},
		{"count_7.Q", "OBUF_1.I", 0},
		{"OBUF_1.O", "out.in", 0},
		// CO[3] of the low slice is CI of the high one: CO is split.
		{"CARRY4_1.CO_3", "CARRY4_2.CI", 0},
		{"CARRY4_1.O_1", "count_1.D", 0},
		// Four bits, each from another flip-flop: S is split.
		{"count_4.Q", "CARRY4_2.S_0", 0},
	} {
		if !hasWire(d, w.src, w.dst, w.thickness) {
			t.Errorf("no wire %s -> %s with thickness %d", w.src, w.dst, w.thickness)
		}
	}
	// DI of the high slice is all zeros: one 4-bit wire from GND.
	var di *ast.Wire
	for i, w := range d.Wires {
		if w.Destination.Block == "CARRY4_2" && w.Destination.Pin == "DI" {
			di = &d.Wires[i]
		}
	}
	if di == nil || di.Thickness != 4 || findBlock(t, d, di.Source.Block).Kind != "GND" {
		t.Errorf("CARRY4_2.DI is not driven by GND: %+v", di)
	}

	// DI of the low slice is 4'b0001.
	for _, w := range d.Wires {
		if w.Destination.Block == "CARRY4_1" && w.Destination.Pin == "DI" {
			if b := findBlock(t, d, w.Source.Block); b.Kind != "CONST" || b.Name != "4'b0001" || w.Thickness != 4 {
				t.Errorf("CARRY4_1.DI is driven by %s %q over %d bits, want CONST 4'b0001 over 4", b.Kind, b.Name, w.Thickness)
			}
		}
	}

	// The diagram survives the trip through blockpic source.
	src := format.Format(d)
	p := parser.New(scanner.New(string(src)))
	back := p.ParseDiagram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parse errors %v in:\n%s", p.Errors(), src)
	}
	if len(back.Blocks) != len(d.Blocks) || len(back.Wires) != len(d.Wires) {
		t.Errorf("parsed back %d blocks and %d wires, want %d and %d",
			len(back.Blocks), len(back.Wires), len(d.Blocks), len(d.Wires))
	}
	if _, err := Convert(back); err != nil {
		t.Errorf("Convert: %v", err)
	}
}

func TestImportPicksModule(t *testing.T) {
	const two = `{"modules": {
		"a": {"ports": {}, "cells": {"x": {"type": "INV", "connections": {}}}},
		"b": {"ports": {}, "cells": {"y": {"type": "INV", "connections": {}}}}
	}}`
	if _, err := Import([]byte(two), ""); err == nil || !strings.Contains(err.Error(), "name one of: a, b") {
		t.Errorf("Import with two candidates: err = %v", err)
	}
	d, err := Import([]byte(two), "b")
	if err != nil || d.Name != "b" {
		t.Errorf("Import -top b = %v, %v", d, err)
	}
	if _, err := Import([]byte(two), "c"); err == nil {
		t.Error("Import -top c: no error")
	}
}

func TestImportBuses(t *testing.T) {
	// a[1:0] goes whole into a two-bit cell, whose bits leave for two
	// one-bit outputs of a two-bit port.
	const nl = `{"modules": {"top": {"attributes": {"top": 1},
		"ports": {
			"a": {"direction": "input", "bits": [2, 3]},
			"y": {"direction": "output", "bits": [5, 4]}
		},
		"cells": {"\\c": {"type": "BOX",
			"port_directions": {"A": "input", "Y": "output"},
			"connections": {"A": [2, 3], "Y": [4, 5]}}}
	}}}`
	d, err := Import([]byte(nl), "")
	if err != nil {
		t.Fatal(err)
	}
	if err := validation.Validate(d); err != nil {
		t.Errorf("Validate: %v", err)
	}
	for _, w := range []struct {
		src, dst  string
		thickness int
	}{
		{"a.out", "c.A", 2},
		{"c.Y_1", "y_0.in", 0},
		{"c.Y_0", "y_1.in", 0},
	} {
		if !hasWire(d, w.src, w.dst, w.thickness) {
			t.Errorf("no wire %s -> %s with thickness %d in %+v", w.src, w.dst, w.thickness, d.Wires)
		}
	}
	if b := findBlock(t, d, "y_1"); b.Name != "y[1]" {
		t.Errorf("y_1 name = %q, want y[1]", b.Name)
	}
	for _, b := range d.Blocks {
		if b.Label == "y" {
			t.Errorf("the split port y still has a block of its own")
		}
	}
}

func TestImportDirectionFromCellModule(t *testing.T) {
	// A cell with no port_directions takes them from its type's module.
	const nl = `{"modules": {
		"LUT1": {"attributes": {"blackbox": "00000000000000000000000000000001"},
			"ports": {"I0": {"direction": "input", "bits": [2]}, "O": {"direction": "output", "bits": [3]}}},
		"top": {"attributes": {"top": 1},
			"ports": {"a": {"direction": "input", "bits": [2]}, "y": {"direction": "output", "bits": [3]}},
			"cells": {"\\lut": {"type": "LUT1", "connections": {"I0": [2], "O": [3]}}}}
	}}`
	d, err := Import([]byte(nl), "")
	if err != nil {
		t.Fatal(err)
	}
	if !hasWire(d, "a.out", "lut.I0", 0) || !hasWire(d, "lut.O", "y.in", 0) {
		t.Errorf("wires = %+v", d.Wires)
	}
}

func TestIdentifier(t *testing.T) {
	for in, want := range map[string]string{
		`\count_reg[3]`: "count_reg_3",
		"$abc$123$lut":  "abc_123_lut",
		"9lives":        "n9lives",
		"name":          "name_",
		"a.b  c":        "a_b_c",
		"":              "n",
	} {
		if got := identifier(in); got != want {
			t.Errorf("identifier(%q) = %q, want %q", in, got, want)
		}
	}
}
