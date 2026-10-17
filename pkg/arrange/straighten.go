// SPDX-License-Identifier: Apache-2.0
package arrange

import (
	"math"
	"sort"
)

// clearance is the room, in pixels, kept between a moved block, with its
// labels, and any other block.
const clearance = 4

// Below gives the room to keep under a block, for text that is drawn there
// after the layout, such as the name refine writes under a generic cell.
// See Reserve.
type Below map[string]float64

// Straighten moves each block that has a single wire up or down, until the
// wire runs straight from pin to pin. ELK spaces the blocks of a layer
// evenly, so such a block, a constant or a port, often sits a few pixels
// off the pin it feeds, and its wire takes a dogleg to get there.
//
// A block moves only where it has room: clear of every other block and of
// every group it is not in, and inside its own group. Its wire may not run
// through a block, nor along a wire of another net. A move is kept only if
// it adds no crossing. Straighten works on a flat graph, and says which
// blocks it moved.
func Straighten(g *Graph) []string {
	var moved []string
	for {
		best := Measure(g)
		next := ""
		for _, id := range candidates(g) {
			trial := g.clone()
			if !straightenOne(trial, id) {
				continue
			}
			s := Measure(trial)
			if s.Crossings <= best.Crossings && s.Bends < best.Bends {
				*g = *trial
				next = id
				break
			}
		}
		if next == "" {
			return moved
		}
		moved = append(moved, next)
	}
}

// candidates are the blocks with one port and one edge, whose net is that
// edge alone, in the order of their ids.
func candidates(g *Graph) []string {
	edges := make(map[string]int)  // block id -> its edges
	fanout := make(map[string]int) // port id -> the edges that leave it
	ports := indexPorts(g)
	for _, e := range g.Edges {
		fanout[e.SourcePortID()]++
		for _, p := range []string{e.SourcePortID(), e.TargetPortID()} {
			if at, ok := ports[p]; ok {
				edges[at.node.ID]++
			}
		}
	}
	var ids []string
	for _, n := range g.Children {
		if len(n.Ports) != 1 || edges[n.ID] != 1 {
			continue
		}
		ids = append(ids, n.ID)
	}
	var out []string
	for _, id := range ids {
		e := edgeOf(g, id, ports)
		if e != nil && fanout[e.SourcePortID()] == 1 {
			out = append(out, id)
		}
	}
	sort.Strings(out)
	return out
}

// edgeOf is the one edge at a block.
func edgeOf(g *Graph, id string, ports portIndex) *Edge {
	for _, e := range g.Edges {
		for _, p := range []string{e.SourcePortID(), e.TargetPortID()} {
			if at, ok := ports[p]; ok && at.node.ID == id {
				return e
			}
		}
	}
	return nil
}

// straightenOne moves the block id so that its wire runs straight, and
// reroutes the wire. It says whether the block could move. The wire keeps
// the end that ELK gave it at the other block, and the block moves by as
// much as the wire's own end must: a route ends within a pixel or so of a
// port, not on it.
func straightenOne(g *Graph, id string) bool {
	ports := indexPorts(g)
	e := edgeOf(g, id, ports)
	if e == nil {
		return false
	}
	pts := e.Points()
	if len(pts) < 2 {
		return false
	}
	src, ok1 := ports[e.SourcePortID()]
	dst, ok2 := ports[e.TargetPortID()]
	if !ok1 || !ok2 || src.node == dst.node {
		return false
	}
	a, b := pts[0], pts[len(pts)-1]
	// A straight wire leaves a pin on its source's right side and enters
	// one on its target's left side, to the right of it. A pin on the top
	// or the bottom of a symbol, such as a buffer's enable, takes its
	// wire from above or below.
	if src.side() != 'R' || dst.side() != 'L' || a.X >= b.X {
		return false
	}
	n := src.node
	dy := b.Y - a.Y
	y := b.Y
	if dst.node.ID == id {
		n = dst.node
		dy = a.Y - b.Y
		y = a.Y
	}
	if dy == 0 {
		return false
	}

	old := n.extent()
	n.Y += dy
	area := n.extent()
	if area.y0 < 0 || area.y1 > g.Height {
		return false
	}
	for _, o := range g.Children {
		if o != n && area.grow(clearance).overlaps(o.extent()) {
			return false
		}
	}
	for _, box := range g.Groups {
		r := rect{box.X, box.Y, box.X + box.Width, box.Y + box.Height}
		inside := r.contains(old)
		if inside && !r.contains(area) || !inside && area.overlaps(r) {
			return false
		}
	}

	// The straight wire, and what it may not touch.
	a.Y, b.Y = y, y
	for _, o := range g.Children {
		if o == src.node || o == dst.node {
			continue
		}
		if o.X < b.X && a.X < o.X+o.Width && o.Y <= y && y <= o.Y+o.Height {
			return false
		}
	}
	for _, f := range g.Edges {
		if f == e {
			continue
		}
		fp := f.Points()
		for i := 1; i < len(fp); i++ {
			p, q := fp[i-1], fp[i]
			if p.Y == y && q.Y == y && math.Min(p.X, q.X) < b.X && a.X < math.Max(p.X, q.X) {
				return false
			}
		}
	}
	e.Sections = []*Section{{ID: e.Sections[0].ID, StartPoint: a, EndPoint: b}}
	e.JunctionPoints = nil
	return true
}

// Reserve makes each block with room kept under it that much taller,
// before the layout, so that ELK keeps the room clear of wires and of
// other blocks. netlistsvg draws a block from its symbol, not from the
// layout's size, so the block itself looks the same. ELK keeps no room
// for a label it is given at a fixed place, which is how netlistsvg gives
// it a block's title.
func Reserve(g *Graph, below Below) {
	for _, n := range g.Children {
		n.Height += below[n.ID]
	}
}
