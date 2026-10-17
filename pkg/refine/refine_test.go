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
	// A name equal to the kind is not drawn again.
	if strings.Count(got, ">IBUF<") != 1 {
		t.Errorf("IBUF drawn %d times, want 1:\n%s", strings.Count(got, ">IBUF<"), got)
	}
}
