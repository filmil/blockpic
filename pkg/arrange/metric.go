// SPDX-License-Identifier: Apache-2.0
package arrange

import (
	"fmt"
	"sort"
)

// Score is how hard a drawing is to read: the places where the wires of
// two nets cross, and the corners of the wires.
type Score struct {
	Crossings int
	Bends     int
}

func (s Score) String() string {
	return fmt.Sprintf("%d crossings, %d bends", s.Crossings, s.Bends)
}

// Better says whether s is easier to read than o: fewer crossings, or as
// many and fewer bends.
func (s Score) Better(o Score) bool {
	if s.Crossings != o.Crossings {
		return s.Crossings < o.Crossings
	}
	return s.Bends < o.Bends
}

// segment is a straight run of a net: horizontal at y from a to b, or
// vertical at x from a to b, with a < b.
type segment struct {
	at, a, b float64
}

// netRoutes are the runs of one net, merged where its edges share a
// path, as the drawing shows them.
type netRoutes struct {
	horizontal []segment
	vertical   []segment
	bends      map[Point]bool
}

// Measure scores a laid-out, flat graph. The edges that leave one port are
// one net: ELK routes them along a shared trunk, which is drawn once.
func Measure(g *Graph) Score {
	nets := make(map[string]*netRoutes)
	var order []string
	for _, e := range g.Edges {
		pts := e.Points()
		if len(pts) < 2 {
			continue
		}
		key := e.SourcePortID()
		n, ok := nets[key]
		if !ok {
			n = &netRoutes{bends: make(map[Point]bool)}
			nets[key] = n
			order = append(order, key)
		}
		for i := 1; i < len(pts); i++ {
			p, q := pts[i-1], pts[i]
			switch {
			case p.Y == q.Y && p.X != q.X:
				n.horizontal = append(n.horizontal, ordered(p.Y, p.X, q.X))
			case p.X == q.X && p.Y != q.Y:
				n.vertical = append(n.vertical, ordered(p.X, p.Y, q.Y))
			}
		}
		for i := 1; i+1 < len(pts); i++ {
			if turns(pts[i-1], pts[i], pts[i+1]) {
				n.bends[pts[i]] = true
			}
		}
	}
	var s Score
	for _, k := range order {
		n := nets[k]
		n.horizontal = merge(n.horizontal)
		n.vertical = merge(n.vertical)
		s.Bends += len(n.bends)
	}
	for _, a := range order {
		for _, b := range order {
			if a == b {
				continue
			}
			for _, h := range nets[a].horizontal {
				for _, v := range nets[b].vertical {
					if h.a < v.at && v.at < h.b && v.a < h.at && h.at < v.b {
						s.Crossings++
					}
				}
			}
		}
	}
	return s
}

func ordered(at, a, b float64) segment {
	if a > b {
		a, b = b, a
	}
	return segment{at, a, b}
}

// turns says whether a route turns at q, on its way from p to r.
func turns(p, q, r Point) bool {
	return (p.X == q.X) != (q.X == r.X) || (p.Y == q.Y) != (q.Y == r.Y)
}

// merge joins the segments on one line that overlap or touch.
func merge(segs []segment) []segment {
	sort.Slice(segs, func(i, j int) bool {
		if segs[i].at != segs[j].at {
			return segs[i].at < segs[j].at
		}
		return segs[i].a < segs[j].a
	})
	var out []segment
	for _, s := range segs {
		if n := len(out); n > 0 && out[n-1].at == s.at && s.a <= out[n-1].b {
			if s.b > out[n-1].b {
				out[n-1].b = s.b
			}
			continue
		}
		out = append(out, s)
	}
	return out
}
