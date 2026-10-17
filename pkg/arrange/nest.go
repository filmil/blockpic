// SPDX-License-Identifier: Apache-2.0
package arrange

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/filmil/blockpic/pkg/ast"
)

// groupPrefix starts the id of the node that holds a group's blocks while
// ELK lays them out. The rest of the id is the group's index.
const groupPrefix = "$group_"

// Room around the blocks of a group, inside its box. The top has a row
// more, for the group's name.
const (
	groupPad      = 20
	groupLabelRow = 10
)

// Nest puts the blocks of each group into a node of their own, so that ELK
// lays the group out as one box that no other block enters, and routes
// the wires in and out of it. A group that shares a block with another
// group cannot be one box, and is left out, as is a group with no block
// in the graph. Nest says which groups it nested, by index.
func Nest(g *Graph, groups []ast.Group) []int {
	byID := make(map[string]*Node, len(g.Children))
	for _, c := range g.Children {
		byID[c.ID] = c
	}
	owners := make(map[string][]int)
	for i, gr := range groups {
		for _, b := range gr.Blocks {
			if _, ok := byID[b]; ok {
				owners[b] = append(owners[b], i)
			}
		}
	}
	shared := make(map[int]bool)
	for _, gs := range owners {
		if len(gs) > 1 {
			for _, i := range gs {
				shared[i] = true
			}
		}
	}

	member := make(map[string]int)
	var nested []int
	var boxes []*Node
	for i, gr := range groups {
		if shared[i] {
			continue
		}
		box := &Node{
			ID: groupPrefix + strconv.Itoa(i),
			LayoutOptions: map[string]any{
				"org.eclipse.elk.padding": fmt.Sprintf("[top=%d,left=%d,bottom=%d,right=%d]",
					groupPad+groupLabelRow, groupPad, groupPad, groupPad),
			},
		}
		// A group is laid out as a graph of its own, with the root's
		// spacing and strategies.
		for k, v := range g.LayoutOptions {
			if strings.Contains(k, "spacing") || strings.Contains(k, "strategy") {
				box.LayoutOptions[k] = v
			}
		}
		for _, b := range gr.Blocks {
			if n, ok := byID[b]; ok {
				if _, dup := member[b]; dup {
					continue
				}
				member[b] = i
				box.Children = append(box.Children, n)
			}
		}
		if len(box.Children) == 0 {
			continue
		}
		nested = append(nested, i)
		boxes = append(boxes, box)
	}
	if len(nested) == 0 {
		return nil
	}

	var rest []*Node
	for _, c := range g.Children {
		if _, ok := member[c.ID]; !ok {
			rest = append(rest, c)
		}
	}
	g.Children = append(rest, boxes...)
	if g.LayoutOptions == nil {
		g.LayoutOptions = make(map[string]any)
	}
	// Edges stay at the root, and ELK routes those that cross a group's
	// border along with the rest.
	g.LayoutOptions["org.eclipse.elk.hierarchyHandling"] = "INCLUDE_CHILDREN"
	return nested
}

// Flatten undoes Nest on a laid-out graph. Every block becomes a child of
// the root again, placed in the drawing's coordinates, and each group's
// box is kept in Groups. ELK gives the route of an edge between two blocks
// of one group in the group's coordinates, and of any other edge in the
// root's; Flatten finds which by where the route starts, and moves it.
// A route starts within a pixel or so of its port, not on it, so the
// frame that puts the start nearer the port is the edge's.
func Flatten(g *Graph, groups []ast.Group) error {
	var flat []*Node
	offset := make(map[string]Point) // block id -> its group's corner
	var boxes []Box
	for _, c := range g.Children {
		if !strings.HasPrefix(c.ID, groupPrefix) {
			flat = append(flat, c)
			continue
		}
		i, err := strconv.Atoi(strings.TrimPrefix(c.ID, groupPrefix))
		if err != nil || i < 0 || i >= len(groups) {
			return fmt.Errorf("arrange: no group for the node %q", c.ID)
		}
		boxes = append(boxes, Box{Group: i, Name: groups[i].Name, X: c.X, Y: c.Y, Width: c.Width, Height: c.Height})
		for _, k := range c.Children {
			k.X += c.X
			k.Y += c.Y
			offset[k.ID] = Point{c.X, c.Y}
			flat = append(flat, k)
		}
		// ELK keeps the edges where they were given; any it put in the
		// group are in the group's coordinates.
		for _, e := range c.Edges {
			shift(e, Point{c.X, c.Y})
			g.Edges = append(g.Edges, e)
		}
	}
	if len(boxes) == 0 {
		return nil
	}
	g.Children = flat
	g.Groups = boxes

	ports := indexPorts(g)
	for _, e := range g.Edges {
		pts := e.Points()
		if len(pts) == 0 {
			continue
		}
		src, ok := ports[e.SourcePortID()]
		if !ok {
			continue
		}
		want := src.at()
		off := distance(pts[0], want)
		if d, inGroup := offset[src.node.ID]; inGroup {
			if moved := distance(Point{pts[0].X + d.X, pts[0].Y + d.Y}, want); moved < off {
				shift(e, d)
				off = moved
			}
		}
		if off > maxPortGap {
			return fmt.Errorf("arrange: edge %s starts at (%g,%g), away from its port %s at (%g,%g)",
				e.ID, pts[0].X, pts[0].Y, e.SourcePortID(), want.X, want.Y)
		}
	}
	return nil
}

// maxPortGap is how far, in pixels, a route may start from its port.
const maxPortGap = 3

// distance is how far apart two points are, along the axes.
func distance(p, q Point) float64 {
	return math.Abs(p.X-q.X) + math.Abs(p.Y-q.Y)
}

// shift moves the route of an edge by d.
func shift(e *Edge, d Point) {
	move := func(p *Point) { p.X += d.X; p.Y += d.Y }
	for _, s := range e.Sections {
		move(&s.StartPoint)
		move(&s.EndPoint)
		for i := range s.BendPoints {
			move(&s.BendPoints[i])
		}
	}
	for i := range e.JunctionPoints {
		move(&e.JunctionPoints[i])
	}
}
