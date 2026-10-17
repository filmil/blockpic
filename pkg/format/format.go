// SPDX-License-Identifier: Apache-2.0

// Package format writes a diagram as blockpic source.
package format

import (
	"fmt"
	"sort"
	"strings"

	"github.com/filmil/blockpic/pkg/ast"
)

// Format returns diag as blockpic source text, which the parser reads
// back into an equal diagram.
//
// Block labels, pin names and parameter keys must already be identifiers.
// A string cannot hold a double quote in blockpic, so each one in a kind,
// name or parameter value is written as a single quote.
func Format(diag *ast.Diagram) []byte {
	var sb strings.Builder
	fmt.Fprintf(&sb, "diagram %s {\n", diag.Name)
	for _, b := range diag.Blocks {
		fmt.Fprintf(&sb, "    block %s {\n", b.Label)
		if b.Kind != "" {
			fmt.Fprintf(&sb, "        kind: %s;\n", quote(b.Kind))
		}
		if b.Name != "" {
			fmt.Fprintf(&sb, "        name: %s;\n", quote(b.Name))
		}
		if b.Hint != ast.HintNone {
			fmt.Fprintf(&sb, "        hint: %s;\n", quote(string(b.Hint)))
		}
		if len(b.Inputs) > 0 {
			fmt.Fprintf(&sb, "        inputs: %s;\n", pins(b.Inputs))
		}
		if len(b.Outputs) > 0 {
			fmt.Fprintf(&sb, "        outputs: %s;\n", pins(b.Outputs))
		}
		if len(b.Params) > 0 {
			keys := make([]string, 0, len(b.Params))
			for k := range b.Params {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			parts := make([]string, len(keys))
			for i, k := range keys {
				parts[i] = fmt.Sprintf("%s: %s", k, quote(b.Params[k]))
			}
			fmt.Fprintf(&sb, "        params: { %s };\n", strings.Join(parts, ", "))
		}
		if b.Size != nil {
			fmt.Fprintf(&sb, "        size: (%g, %g);\n", b.Size.W, b.Size.H)
		}
		sb.WriteString("    }\n")
	}
	for _, w := range diag.Wires {
		fmt.Fprintf(&sb, "    wire %s -> %s", pinRef(w.Source), pinRef(w.Destination))
		var attrs []string
		if w.Thickness > 0 {
			attrs = append(attrs, fmt.Sprintf("thickness: %d;", w.Thickness))
		}
		if w.Crossline > 0 {
			attrs = append(attrs, fmt.Sprintf("crossline: %d;", w.Crossline))
		}
		if w.Hint != "" {
			attrs = append(attrs, fmt.Sprintf("hint: %s;", quote(string(w.Hint))))
		}
		if len(attrs) > 0 {
			fmt.Fprintf(&sb, " { %s }", strings.Join(attrs, " "))
		}
		sb.WriteString("\n")
	}
	for _, g := range diag.Groups {
		fmt.Fprintf(&sb, "    group %s { %s }\n", quote(g.Name), strings.Join(g.Blocks, "; "))
	}
	if len(diag.Layout) > 0 {
		sb.WriteString("    layout {\n")
		for _, l := range diag.Layout {
			fmt.Fprintf(&sb, "        %s at (%g, %g)\n", l.Block, l.X, l.Y)
		}
		sb.WriteString("    }\n")
	}
	sb.WriteString("}\n")
	return []byte(sb.String())
}

func quote(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, "'") + `"`
}

func pins(ps []ast.Pin) string {
	parts := make([]string, len(ps))
	for i, p := range ps {
		parts[i] = p.Name
		if p.Position != ast.PositionNone {
			parts[i] += " [" + string(p.Position) + "]"
		}
	}
	return strings.Join(parts, ", ")
}

func pinRef(r ast.PinRef) string {
	if r.Pin == "" {
		return r.Block
	}
	return r.Block + "." + r.Pin
}
