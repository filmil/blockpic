// SPDX-License-Identifier: Apache-2.0
package format

import (
	"reflect"
	"testing"

	"github.com/filmil/blockpic/pkg/ast"
	"github.com/filmil/blockpic/pkg/parser"
	"github.com/filmil/blockpic/pkg/scanner"
)

func TestFormatParsesBack(t *testing.T) {
	want := &ast.Diagram{
		Name: "RoundTrip",
		Blocks: []ast.Block{
			{
				Label:   "clk",
				Name:    "clk",
				Kind:    "INPUT_PIN",
				Params:  map[string]string{},
				Outputs: []ast.Pin{{Name: "out"}},
			},
			{
				Label:   "reg1",
				Name:    "count[0]",
				Kind:    "FDRE",
				Params:  map[string]string{"INIT": "1'b0", "mode": "fast"},
				Inputs:  []ast.Pin{{Name: "C", Position: ast.PositionLeft}, {Name: "D"}},
				Outputs: []ast.Pin{{Name: "Q", Position: ast.PositionRight}},
				Size:    &ast.Size{W: 40, H: 60},
				Hint:    ast.HintShowPorts,
			},
		},
		Wires: []ast.Wire{
			{Source: ast.PinRef{Block: "clk", Pin: "out"}, Destination: ast.PinRef{Block: "reg1", Pin: "C"}},
			{
				Source:      ast.PinRef{Block: "reg1", Pin: "Q"},
				Destination: ast.PinRef{Block: "reg1", Pin: "D"},
				Thickness:   4,
				Crossline:   4,
				Type:        "nibble_t",
				Hint:        ast.HintZShape,
			},
		},
		Groups: []ast.Group{{Name: "regs", Blocks: []string{"reg1"}}},
		Layout: []ast.LayoutEntry{{Block: "reg1", X: 20, Y: 30}},
	}

	src := Format(want)
	p := parser.New(scanner.New(string(src)))
	got := p.ParseDiagram()
	if errs := p.Errors(); len(errs) > 0 {
		t.Fatalf("parse errors %v in:\n%s", errs, src)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("parsed back\n%#v\nwant\n%#v\nfrom:\n%s", got, want, src)
	}
}

func TestFormatQuote(t *testing.T) {
	d := &ast.Diagram{Name: "Q", Blocks: []ast.Block{{Label: "a", Name: `say "hi"`}}}
	p := parser.New(scanner.New(string(Format(d))))
	got := p.ParseDiagram()
	if got.Blocks[0].Name != "say 'hi'" {
		t.Errorf("name = %q, want %q", got.Blocks[0].Name, "say 'hi'")
	}
}
