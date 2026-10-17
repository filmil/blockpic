// SPDX-License-Identifier: Apache-2.0
package arrange

import (
	"math"
	"sort"

	"github.com/filmil/blockpic/pkg/yosys"
)

// commutative are the cell types whose inputs can trade places without
// changing what the cell computes. The skins draw none of their input
// pins' names, so a trade shows only in where the wires go.
var commutative = map[string]bool{
	"$and": true, "$nand": true,
	"$or": true, "$nor": true,
	"$xor": true, "$xnor": true,
	"$add": true, "$mul": true,
	"$eq": true, "$ne": true,
}

// Permute gives the inputs of each commutative cell to its wires in the
// order the wires arrive: the wire that comes in highest takes the
// topmost input. A wire comes in at the height it runs at before its last
// turn toward the pin, so a wire that loops around under the drawing
// comes in from below. Permute changes the netlist, and says which cells
// it changed; the graph is that of the netlist before the change.
func Permute(n *yosys.Netlist, g *Graph) []string {
	ports := indexPorts(g)
	into := make(map[string][]*Edge) // port id -> the edges that enter it
	for _, e := range g.Edges {
		into[e.TargetPortID()] = append(into[e.TargetPortID()], e)
	}

	var changed []string
	for _, mname := range sortedKeys(n.Modules) {
		mod := n.Modules[mname]
		for _, cname := range sortedKeys(mod.Cells) {
			cell := mod.Cells[cname]
			if !commutative[cell.Type] {
				continue
			}
			var ins []string
			ok := true
			for _, p := range sortedKeys(cell.PortDirections) {
				if cell.PortDirections[p] != "input" {
					continue
				}
				id := cname + "." + p
				if _, placed := ports[id]; !placed || len(into[id]) == 0 {
					ok = false
					break
				}
				ins = append(ins, p)
			}
			if !ok || len(ins) < 2 {
				continue
			}
			// The pins, top first, and the wires in the order they come.
			slots := append([]string(nil), ins...)
			sort.SliceStable(slots, func(i, j int) bool {
				return ports[cname+"."+slots[i]].port.Y < ports[cname+"."+slots[j]].port.Y
			})
			from := make(map[string]float64, len(ins))
			for _, p := range ins {
				from[p] = math.Inf(1)
				for _, e := range into[cname+"."+p] {
					from[p] = math.Min(from[p], approach(e))
				}
			}
			wires := append([]string(nil), slots...)
			sort.SliceStable(wires, func(i, j int) bool { return from[wires[i]] < from[wires[j]] })

			same := true
			for i := range slots {
				if slots[i] != wires[i] {
					same = false
				}
			}
			if same {
				continue
			}
			old := make(map[string][]int, len(ins))
			for _, p := range ins {
				old[p] = cell.Connections[p]
			}
			for i, s := range slots {
				cell.Connections[s] = old[wires[i]]
			}
			mod.Cells[cname] = cell
			changed = append(changed, cname)
		}
		n.Modules[mname] = mod
	}
	return changed
}

// approach is the height an edge runs at before its last turn toward its
// target, or the target's own height when the edge runs straight in.
func approach(e *Edge) float64 {
	pts := e.Points()
	if len(pts) == 0 {
		return math.Inf(1)
	}
	end := pts[len(pts)-1]
	for i := len(pts) - 2; i >= 0; i-- {
		if pts[i].Y != end.Y {
			return pts[i].Y
		}
	}
	return end.Y
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
