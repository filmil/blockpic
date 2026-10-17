// SPDX-License-Identifier: Apache-2.0
package arrange

import "sort"

// Unjog removes the jogs that ELK leaves in a wire: a run of the wire
// that steps aside by a few pixels and back, or that could join the run
// before or after it. A wire that crosses a group's border is routed in
// two parts, outside the group and inside it, and the two seldom meet at
// one height.
//
// A middle run of a wire, one that ends at no pin, slides sideways until
// it lines up with the run two before or two after it, going the same
// way. The run between them shrinks to nothing, and two corners go. A
// slide is kept only if the wire then runs through no block and no
// block's label, and the drawing has no more crossings than before. Only
// a wire that is its net's only wire slides, so that no other wire loses
// the trunk it shares. Unjog works on a flat graph, and says how many
// runs it slid.
func Unjog(g *Graph) int {
	slid := 0
	for {
		best := Measure(g)
		done := true
		for _, i := range unjogCandidates(g) {
			if tryUnjog(g, i, best) {
				slid++
				done = false
				break
			}
		}
		if done {
			return slid
		}
	}
}

// unjogCandidates are the edges, by index, that are their net's only edge
// and have a middle run.
func unjogCandidates(g *Graph) []int {
	fanout := make(map[string]int)
	for _, e := range g.Edges {
		fanout[e.SourcePortID()]++
	}
	var out []int
	for i, e := range g.Edges {
		if fanout[e.SourcePortID()] == 1 && len(e.Points()) >= 4 {
			out = append(out, i)
		}
	}
	sort.SliceStable(out, func(a, b int) bool { return g.Edges[out[a]].ID < g.Edges[out[b]].ID })
	return out
}

// tryUnjog tries each slide of a middle run of edge i, and keeps the first
// that helps.
func tryUnjog(g *Graph, i int, best Score) bool {
	pts := g.Edges[i].Points()
	// Run k goes from pts[k] to pts[k+1]; the first and the last end at
	// pins and stay.
	for k := 1; k+1 < len(pts)-1; k++ {
		for _, other := range []int{k - 2, k + 2} {
			if other < 0 || other+1 >= len(pts) {
				continue
			}
			moved, ok := slide(pts, k, other)
			if !ok {
				continue
			}
			trial := g.clone()
			e := trial.Edges[i]
			e.Sections = []*Section{{ID: e.Sections[0].ID, StartPoint: moved[0], EndPoint: moved[len(moved)-1], BendPoints: moved[1 : len(moved)-1]}}
			if len(e.Sections[0].BendPoints) == 0 {
				e.Sections[0].BendPoints = nil
			}
			e.JunctionPoints = nil
			if blocked(trial, e) {
				continue
			}
			if s := Measure(trial); s.Crossings <= best.Crossings && s.Bends < best.Bends {
				*g = *trial
				return true
			}
		}
	}
	return false
}

// slide moves run k of a route onto the line of run other, two runs away,
// when both runs lie the same way along that line, and gives the route
// with the run between them gone.
func slide(pts []Point, k, other int) ([]Point, bool) {
	a, b := pts[k], pts[k+1]
	c, d := pts[other], pts[other+1]
	horizontal := a.Y == b.Y
	if horizontal != (c.Y == d.Y) {
		return nil, false
	}
	moved := append([]Point(nil), pts...)
	if horizontal {
		if (b.X > a.X) != (d.X > c.X) {
			return nil, false
		}
		moved[k].Y, moved[k+1].Y = c.Y, c.Y
	} else {
		if (b.Y > a.Y) != (d.Y > c.Y) {
			return nil, false
		}
		moved[k].X, moved[k+1].X = c.X, c.X
	}
	return simplify(moved), true
}

// simplify drops the points of a route that repeat the one before, or lie
// on the line between their neighbours.
func simplify(pts []Point) []Point {
	var out []Point
	for _, p := range pts {
		if n := len(out); n > 0 && out[n-1] == p {
			continue
		}
		out = append(out, p)
	}
	for i := 1; i+1 < len(out); {
		a, b, c := out[i-1], out[i], out[i+1]
		if (a.X == b.X && b.X == c.X) || (a.Y == b.Y && b.Y == c.Y) {
			out = append(out[:i], out[i+1:]...)
			continue
		}
		i++
	}
	return out
}

// blocked says whether the route of e runs through a block, or through a
// label of a block it does not end at.
func blocked(g *Graph, e *Edge) bool {
	ports := indexPorts(g)
	ends := make(map[*Node]bool)
	for _, p := range []string{e.SourcePortID(), e.TargetPortID()} {
		if at, ok := ports[p]; ok {
			ends[at.node] = true
		}
	}
	pts := e.Points()
	for i := 1; i < len(pts); i++ {
		p, q := pts[i-1], pts[i]
		run := rect{min(p.X, q.X), min(p.Y, q.Y), max(p.X, q.X), max(p.Y, q.Y)}
		for _, n := range g.Children {
			area := rect{n.X, n.Y, n.X + n.Width, n.Y + n.Height}
			if !ends[n] {
				area = n.extent()
			}
			if crosses(run, area) {
				return true
			}
		}
	}
	return false
}

// crosses says whether a run, a rect of no width or no height, passes
// through the inside of an area.
func crosses(run, area rect) bool {
	if run.y0 == run.y1 {
		return area.y0 < run.y0 && run.y0 < area.y1 && run.x0 < area.x1 && area.x0 < run.x1
	}
	return area.x0 < run.x0 && run.x0 < area.x1 && run.y0 < area.y1 && area.y0 < run.y1
}
