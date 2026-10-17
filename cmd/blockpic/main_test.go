// SPDX-License-Identifier: Apache-2.0
package main

import (
	"reflect"
	"testing"

	"github.com/filmil/blockpic/pkg/arrange"
	"github.com/filmil/blockpic/pkg/ast"
	"github.com/filmil/blockpic/pkg/yosys"
)

func TestCLI_Basic(t *testing.T) {
	// Simple test to ensure package builds
	if false {
		main()
	}
}

// Room is kept under a block whose name refine writes under it: one drawn
// as a generic cell, with a name of its own. A block the skin has a symbol
// for, or whose name is its kind, gets none.
func TestBelow(t *testing.T) {
	diag := &ast.Diagram{Blocks: []ast.Block{
		{Label: "sm", Kind: "generic", Name: "Switch Matrix"},
		{Label: "plus", Kind: "ADD", Name: "+1"},
		{Label: "ff", Kind: "FDRE", Name: "FDRE"},
		{Label: "c", Kind: "CONST", Name: "4'b0001"},
	}}
	nl := &yosys.Netlist{Modules: map[string]yosys.Module{"m": {Cells: map[string]yosys.Cell{
		"sm": {Type: "generic"}, "plus": {Type: "$add"}, "ff": {Type: "FDRE"}, "c": {Type: "CONST"},
	}}}}
	skin := []byte(`<g s:type="add"><s:alias val="$add" /></g><g s:type="CONST"><s:alias val="CONST" /></g>`)
	got := below(diag, nl, skin)
	if want := (arrange.Below{"sm": nameRoom}); !reflect.DeepEqual(got, want) {
		t.Errorf("below() = %v, want %v", got, want)
	}
}
