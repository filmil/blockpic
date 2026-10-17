// SPDX-License-Identifier: Apache-2.0

// Package arrange works on the layout of a diagram between ELK, which
// places the blocks and routes the wires, and netlistsvg, which draws
// them. It puts the blocks of a group in a box of their own, gives the
// inputs of a commutative block to the wires in the order they arrive,
// and lines up a block that has a single wire with the pin at its other
// end.
package arrange

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
)

// Graph is the ELK JSON graph that netlistsvg builds and ELK lays out.
// Its root is a Node; after Flatten, every block is a child of the root.
type Graph = Node

// Node is a block, or the root of the graph, or a group's box while ELK
// lays it out.
type Node struct {
	ID     string  `json:"id"`
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
	Ports  []*Port `json:"ports,omitempty"`
	// Labels is a pointer, so that an empty list is kept: netlistsvg
	// reads the labels of every block it draws.
	Labels        *[]json.RawMessage `json:"labels,omitempty"`
	LayoutOptions map[string]any     `json:"layoutOptions,omitempty"`
	Children      []*Node            `json:"children,omitempty"`
	Edges         []*Edge            `json:"edges,omitempty"`
	// Groups are the boxes of the groups, once the graph is flattened.
	// netlistsvg ignores them; refine draws them.
	Groups []Box `json:"blockpicGroups,omitempty"`
}

// Port is a pin of a block, placed relative to the block.
type Port struct {
	ID     string             `json:"id"`
	X      float64            `json:"x"`
	Y      float64            `json:"y"`
	Width  float64            `json:"width"`
	Height float64            `json:"height"`
	Labels *[]json.RawMessage `json:"labels,omitempty"`
}

// Edge is a wire from one pin to another. netlistsvg writes most edges
// with sources and targets, the ports at their ends, and the edges of a
// net with several drivers with source, sourcePort, target and
// targetPort.
type Edge struct {
	ID             string         `json:"id"`
	Sources        []string       `json:"sources,omitempty"`
	Targets        []string       `json:"targets,omitempty"`
	Source         string         `json:"source,omitempty"`
	SourcePort     string         `json:"sourcePort,omitempty"`
	Target         string         `json:"target,omitempty"`
	TargetPort     string         `json:"targetPort,omitempty"`
	LayoutOptions  map[string]any `json:"layoutOptions,omitempty"`
	Sections       []*Section     `json:"sections,omitempty"`
	JunctionPoints []Point        `json:"junctionPoints,omitempty"`
}

// Section is the route of an edge, from its source to its target.
type Section struct {
	ID         string  `json:"id,omitempty"`
	StartPoint Point   `json:"startPoint"`
	EndPoint   Point   `json:"endPoint"`
	BendPoints []Point `json:"bendPoints,omitempty"`
}

// Point is a point of a route.
type Point struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// Box is the box of a group, in the coordinates of the drawing. Group is
// the group's index among the diagram's groups.
type Box struct {
	Group  int     `json:"group"`
	Name   string  `json:"name"`
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

// SourcePortID is the port that the edge leaves.
func (e *Edge) SourcePortID() string {
	if len(e.Sources) > 0 {
		return e.Sources[0]
	}
	return e.SourcePort
}

// TargetPortID is the port that the edge enters.
func (e *Edge) TargetPortID() string {
	if len(e.Targets) > 0 {
		return e.Targets[0]
	}
	return e.TargetPort
}

// Points are the points of the edge's route, from source to target. An
// edge that ELK has not routed has none.
func (e *Edge) Points() []Point {
	if len(e.Sections) == 0 {
		return nil
	}
	s := e.Sections[0]
	pts := make([]Point, 0, len(s.BendPoints)+2)
	pts = append(pts, s.StartPoint)
	pts = append(pts, s.BendPoints...)
	return append(pts, s.EndPoint)
}

// Parse reads a graph.
func Parse(data []byte) (*Graph, error) {
	var g Graph
	if err := json.Unmarshal(data, &g); err != nil {
		return nil, fmt.Errorf("arrange: reading the ELK graph: %w", err)
	}
	return &g, nil
}

// Marshal writes a graph.
func (g *Graph) Marshal() ([]byte, error) {
	return json.MarshalIndent(g, "", " ")
}

// clone copies a graph, so that a change can be tried and dropped.
func (g *Graph) clone() *Graph {
	data, err := json.Marshal(g)
	if err != nil {
		panic(fmt.Sprintf("arrange: copying a graph: %v", err))
	}
	c, err := Parse(data)
	if err != nil {
		panic(err)
	}
	return c
}

// portIndex finds every port of a flat graph, with the block it is on.
type portIndex map[string]portAt

type portAt struct {
	node *Node
	port *Port
}

// at is the port's place in the drawing.
func (p portAt) at() Point {
	return Point{p.node.X + p.port.X, p.node.Y + p.port.Y}
}

// sideSlack is how far, in pixels, a pin may sit from the side of its
// block and still be on it: a skin may set a pin a pixel out.
const sideSlack = 1

// onLeft says whether the port is on its block's left side.
func (p portAt) onLeft() bool {
	return p.port.X <= sideSlack
}

// onRight says whether the port is on its block's right side.
func (p portAt) onRight() bool {
	return p.port.X >= p.node.Width-sideSlack
}

func indexPorts(g *Graph) portIndex {
	idx := make(portIndex)
	for _, n := range g.Children {
		for _, p := range n.Ports {
			idx[p.ID] = portAt{n, p}
		}
	}
	return idx
}

// label is the part of a label that says where it is.
type label struct {
	X      number `json:"x"`
	Y      number `json:"y"`
	Width  number `json:"width"`
	Height number `json:"height"`
}

// number is a coordinate that netlistsvg writes as a number or, from a
// skin's attribute, as a string.
type number float64

func (n *number) UnmarshalJSON(b []byte) error {
	var v any
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	switch v := v.(type) {
	case float64:
		*n = number(v)
	case string:
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			*n = 0
			return nil
		}
		*n = number(f)
	}
	return nil
}

// extent is the space a block takes in the drawing: its body, any room
// kept under it, and the labels netlistsvg sets around it.
func (n *Node) extent() rect {
	r := rect{n.X, n.Y, n.X + n.Width, n.Y + n.Height}
	if n.Labels == nil {
		return r
	}
	for _, raw := range *n.Labels {
		var l label
		if json.Unmarshal(raw, &l) != nil {
			continue
		}
		x, y := n.X+float64(l.X), n.Y+float64(l.Y)
		r = r.union(rect{x, y, x + float64(l.Width), y + float64(l.Height)})
	}
	return r
}

// rect is an area of the drawing, from its top left to its bottom right.
type rect struct {
	x0, y0, x1, y1 float64
}

func (r rect) union(o rect) rect {
	return rect{math.Min(r.x0, o.x0), math.Min(r.y0, o.y0), math.Max(r.x1, o.x1), math.Max(r.y1, o.y1)}
}

func (r rect) grow(d float64) rect {
	return rect{r.x0 - d, r.y0 - d, r.x1 + d, r.y1 + d}
}

// overlaps says whether the two areas share any point inside both.
func (r rect) overlaps(o rect) bool {
	return r.x0 < o.x1 && o.x0 < r.x1 && r.y0 < o.y1 && o.y0 < r.y1
}

// contains says whether o lies wholly in r.
func (r rect) contains(o rect) bool {
	return r.x0 <= o.x0 && o.x1 <= r.x1 && r.y0 <= o.y0 && o.y1 <= r.y1
}
