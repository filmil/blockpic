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
