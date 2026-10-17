// SPDX-License-Identifier: Apache-2.0
package refine

import (
	"strings"
	"testing"

	"github.com/filmil/blockpic/pkg/ast"
)

// Two thick wires from one pin are one net, and its lines are thickened
// once: a repeated attribute is not well-formed XML.
func TestThickFanOutOnce(t *testing.T) {
	diag := &ast.Diagram{
		Name: "D",
		Wires: []ast.Wire{
			{Source: ast.PinRef{Block: "a", Pin: "out"}, Destination: ast.PinRef{Block: "x", Pin: "in"}, Thickness: 4},
			{Source: ast.PinRef{Block: "a", Pin: "out"}, Destination: ast.PinRef{Block: "y", Pin: "in"}, Thickness: 4},
		},
	}
	svg := `<svg><line x1="0" x2="1" y1="0" y2="0" class="net_2"/></svg>`
	out, err := Process(diag, []byte(svg))
	if err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(string(out), "stroke-width"); n != 1 {
		t.Errorf("stroke-width appears %d times, want 1, in %s", n, out)
	}
}

func TestDrawNames(t *testing.T) {
	diag := &ast.Diagram{
		Name: "D",
		Blocks: []ast.Block{
			{Label: "count_3", Name: "count[3]", Kind: "FDRE"},
			{Label: "plus", Name: "+1 & <b>", Kind: "ADD"},
			{Label: "IBUF_1", Name: "IBUF", Kind: "IBUF"},
			{Label: "led_0", Name: "led[0]", Kind: "OUTPUT_PIN"},
		},
	}
	svg := `<svg>
  <g s:type="generic" transform="translate(100,50)" s:width="30" s:height="40" id="cell_count_3">
    <text x="15" y="-4" class="nodelabel cell_count_3" s:attribute="ref">FDRE</text>
    <rect width="30" height="80" s:generic="body" class="cell_count_3"/>
  </g>
  <g s:type="generic" transform="translate(0,0)" s:width="30" s:height="40" id="cell_IBUF_1">
    <text x="15" y="-4" class="nodelabel cell_IBUF_1" s:attribute="ref">IBUF</text>
    <rect width="30" height="20" s:generic="body" class="cell_IBUF_1"/>
  </g>
  <g s:type="outputExt" transform="translate(9,9)" s:width="20" s:height="10" id="cell_led_0">
    <text x="23" y="8" class="pinIn cell_led_0" s:attribute="ref">led_0</text>
  </g>
  <g s:type="add" transform="translate(10,10)" s:width="25" s:height="25" id="cell_plus">
    <text x="15" y="-4" class="nodelabel cell_plus" s:attribute="ref">plus</text>
  </g>
</svg>`
	out, err := Process(diag, []byte(svg))
	if err != nil {
		t.Fatal(err)
	}
	got := string(out)
	// A generic cell keeps its kind as the title and gets its name under it.
	if !strings.Contains(got, `s:attribute="ref">FDRE</text>`) {
		t.Errorf("the FDRE title is gone:\n%s", got)
	}
	if !strings.Contains(got, `<text x="115.0" y="141.0" class="nodelabel cell_count_3" text-anchor="middle">count[3]</text>`) {
		t.Errorf("no count[3] under the FDRE:\n%s", got)
	}
	// A skinned cell is titled with its name, escaped.
	if !strings.Contains(got, `s:attribute="ref">+1 &amp; &lt;b&gt;</text>`) {
		t.Errorf("the adder is not titled with its name:\n%s", got)
	}
	// Whatever class the skin gives the title.
	if !strings.Contains(got, `s:attribute="ref">led[0]</text>`) {
		t.Errorf("the port is not titled with its name:\n%s", got)
	}
	// A name equal to the kind is not drawn again.
	if strings.Count(got, ">IBUF<") != 1 {
		t.Errorf("IBUF drawn %d times, want 1:\n%s", strings.Count(got, ">IBUF<"), got)
	}
}

func TestFitPortLabels(t *testing.T) {
	// An output named beside its flag at the right edge, and an input
	// named left of its flag at the left edge.
	svg := `<svg xmlns="http://www.w3.org/2000/svg" width="100" height="50">
  <g s:type="outputExt" transform="translate(80,10)" s:width="20" s:height="10" id="cell_led">
    <text x="23" y="8" class="pinIn cell_led" s:attribute="ref">led</text>
  </g>
  <g s:type="inputExt" transform="translate(0,10)" s:width="20" s:height="10" id="cell_clk">
    <text x="-3" y="8" class="inputPortLabel cell_clk" s:attribute="ref">clk</text>
  </g>
</svg>`
	got := fitPortLabels(svg)
	// led ends at 80+23+3*5.5+2 = 121.5; clk starts at 0-3-16.5-2 = -21.5.
	want := `<svg xmlns="http://www.w3.org/2000/svg" width="143" height="50" viewBox="-21.5 0 143 50"`
	if !strings.Contains(got, want) {
		t.Errorf("got %s, want it to start with %s", got, want)
	}
	// A drawing whose names fit is left alone.
	fits := `<svg xmlns="http://www.w3.org/2000/svg" width="100" height="50"></svg>`
	if got := fitPortLabels(fits); got != fits {
		t.Errorf("fitPortLabels changed a drawing that fits: %s", got)
	}
}
