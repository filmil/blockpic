// SPDX-License-Identifier: Apache-2.0
package arrange

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/filmil/blockpic/pkg/ast"
	"github.com/filmil/blockpic/pkg/yosys"
)

// block is a node of the given place and size, with ports at the given
// offsets, named by their pins.
func block(id string, x, y, w, h float64, ports map[string]Point) *Node {
	n := &Node{ID: id, X: x, Y: y, Width: w, Height: h}
	for _, pin := range sortedKeys(ports) {
		p := ports[pin]
		n.Ports = append(n.Ports, &Port{ID: id + "." + pin, X: p.X, Y: p.Y})
	}
	return n
}

// wire is an edge from port src to port dst along the given points.
func wire(id, src, dst string, pts ...Point) *Edge {
	e := &Edge{ID: id, Sources: []string{src}, Targets: []string{dst}}
	if len(pts) >= 2 {
		e.Sections = []*Section{{ID: id + "_s0", StartPoint: pts[0], EndPoint: pts[len(pts)-1], BendPoints: pts[1 : len(pts)-1]}}
	}
	return e
}

func pt(x, y float64) Point { return Point{x, y} }

func TestMeasure(t *testing.T) {
	g := &Graph{
		Edges: []*Edge{
			// Net a: a dogleg, with a vertical run at x=50.
			wire("e0", "a.Y", "b.A", pt(0, 10), pt(50, 10), pt(50, 30), pt(100, 30)),
			// Net a again, along the same trunk: drawn once.
			wire("e1", "a.Y", "c.A", pt(0, 10), pt(50, 10), pt(50, 60), pt(100, 60)),
			// Net d: straight across net a's trunk.
			wire("e2", "d.Y", "e.A", pt(0, 40), pt(100, 40)),
		},
	}
	got := Measure(g)
	want := Score{Crossings: 1, Bends: 3}
	if got != want {
		t.Errorf("Measure() = %v, want %v", got, want)
	}
}

func TestScoreBetter(t *testing.T) {
	for _, tc := range []struct {
		a, b Score
		want bool
	}{
		{Score{0, 9}, Score{1, 2}, true},
		{Score{1, 2}, Score{0, 9}, false},
		{Score{1, 2}, Score{1, 3}, true},
		{Score{1, 3}, Score{1, 3}, false},
	} {
		if got := tc.a.Better(tc.b); got != tc.want {
			t.Errorf("%v.Better(%v) = %v, want %v", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestNest(t *testing.T) {
	g := &Graph{
		LayoutOptions: map[string]any{
			"org.eclipse.elk.spacing.nodeNode":          "35",
			"org.eclipse.elk.layered.layering.strategy": "LONGEST_PATH",
			"org.eclipse.elk.direction":                 "RIGHT",
		},
		Children: []*Node{
			block("in", 0, 0, 20, 10, nil),
			block("plus", 0, 0, 25, 25, nil),
			block("reg", 0, 0, 40, 40, nil),
			block("x", 0, 0, 20, 20, nil),
			block("y", 0, 0, 20, 20, nil),
		},
	}
	groups := []ast.Group{
		{Name: "counter", Blocks: []string{"plus", "reg", "missing"}},
		{Name: "left", Blocks: []string{"x"}},
		{Name: "right", Blocks: []string{"x", "y"}}, // shares x with "left"
		{Name: "empty", Blocks: []string{"missing"}},
	}
	nested := Nest(g, groups)
	if want := []int{0}; !reflect.DeepEqual(nested, want) {
		t.Fatalf("Nest() nested %v, want %v", nested, want)
	}
	var ids []string
	for _, c := range g.Children {
		ids = append(ids, c.ID)
	}
	if want := []string{"in", "x", "y", "$group_0"}; !reflect.DeepEqual(ids, want) {
		t.Errorf("root children = %v, want %v", ids, want)
	}
	box := g.Children[3]
	if len(box.Children) != 2 || box.Children[0].ID != "plus" || box.Children[1].ID != "reg" {
		t.Errorf("group children = %v, want plus and reg", box.Children)
	}
	if got := g.LayoutOptions["org.eclipse.elk.hierarchyHandling"]; got != "INCLUDE_CHILDREN" {
		t.Errorf("hierarchyHandling = %v, want INCLUDE_CHILDREN", got)
	}
	if _, ok := box.LayoutOptions["org.eclipse.elk.spacing.nodeNode"]; !ok {
		t.Errorf("group options %v lack the root's spacing", box.LayoutOptions)
	}
	if _, ok := box.LayoutOptions["org.eclipse.elk.direction"]; ok {
		t.Errorf("group options %v carry an option other than spacing and strategy", box.LayoutOptions)
	}
	if !strings.Contains(box.LayoutOptions["org.eclipse.elk.padding"].(string), "top=30") {
		t.Errorf("group padding = %v, want a row for the name at the top", box.LayoutOptions["org.eclipse.elk.padding"])
	}
}

func TestNestWithoutGroups(t *testing.T) {
	g := &Graph{Children: []*Node{block("a", 0, 0, 10, 10, nil)}}
	if nested := Nest(g, nil); nested != nil {
		t.Errorf("Nest() = %v, want nil", nested)
	}
	if g.LayoutOptions != nil {
		t.Errorf("Nest() set options %v on a graph without groups", g.LayoutOptions)
	}
}

// laidOutGroup is what ELK gives back for a group at (100, 50) holding
// blocks p and q, with block s outside it.
func laidOutGroup() *Graph {
	group := block("$group_0", 100, 50, 200, 100, nil)
	group.Children = []*Node{
		block("p", 20, 30, 20, 20, map[string]Point{"A": pt(0, 10), "Y": pt(20, 10)}),
		block("q", 120, 30, 20, 20, map[string]Point{"A": pt(0, 10)}),
	}
	return &Graph{
		Width: 400, Height: 200,
		Children: []*Node{block("s", 10, 80, 20, 10, map[string]Point{"Y": pt(20, 5)}), group},
		Edges: []*Edge{
			// Into the group: in the root's coordinates, off by a pixel.
			wire("e0", "s.Y", "p.A", pt(31, 85), pt(60, 85), pt(60, 90), pt(120, 90)),
			// Inside the group: in the group's coordinates.
			wire("e1", "p.Y", "q.A", pt(40, 40), pt(120, 40)),
		},
	}
}

func TestFlatten(t *testing.T) {
	g := laidOutGroup()
	groups := []ast.Group{{Name: "core", Blocks: []string{"p", "q"}}}
	if err := Flatten(g, groups); err != nil {
		t.Fatalf("Flatten() = %v", err)
	}
	place := make(map[string]Point)
	for _, c := range g.Children {
		place[c.ID] = pt(c.X, c.Y)
	}
	want := map[string]Point{"s": pt(10, 80), "p": pt(120, 80), "q": pt(220, 80)}
	if !reflect.DeepEqual(place, want) {
		t.Errorf("blocks at %v, want %v", place, want)
	}
	if want := []Box{{Group: 0, Name: "core", X: 100, Y: 50, Width: 200, Height: 100}}; !reflect.DeepEqual(g.Groups, want) {
		t.Errorf("Groups = %v, want %v", g.Groups, want)
	}
	if got, want := g.Edges[0].Points(), []Point{pt(31, 85), pt(60, 85), pt(60, 90), pt(120, 90)}; !reflect.DeepEqual(got, want) {
		t.Errorf("edge into the group = %v, want it unmoved, %v", got, want)
	}
	if got, want := g.Edges[1].Points(), []Point{pt(140, 90), pt(220, 90)}; !reflect.DeepEqual(got, want) {
		t.Errorf("edge inside the group = %v, want %v", got, want)
	}
}

func TestFlattenRejectsAStrayEdge(t *testing.T) {
	g := laidOutGroup()
	g.Edges[0].Sections[0].StartPoint = pt(500, 500)
	err := Flatten(g, []ast.Group{{Name: "core", Blocks: []string{"p", "q"}}})
	if err == nil || !strings.Contains(err.Error(), "away from its port") {
		t.Errorf("Flatten() = %v, want an error about the edge e0", err)
	}
}

// adder is a netlist of one cell of the given type, whose inputs A and B
// take nets 2 and 3, and the graph of its layout: A is the upper pin. The
// wire into A comes around from below, and the wire into B runs straight in.
func adder(typ string) (*yosys.Netlist, *Graph) {
	nl := &yosys.Netlist{Modules: map[string]yosys.Module{"m": {
		Cells: map[string]yosys.Cell{"plus": {
			Type:           typ,
			PortDirections: map[string]string{"A": "input", "B": "input", "Y": "output"},
			Connections:    map[string][]int{"A": {2}, "B": {3}, "Y": {4}},
		}},
	}}}
	g := &Graph{
		Children: []*Node{
			block("plus", 100, 20, 25, 25, map[string]Point{"A": pt(0, 5), "B": pt(0, 20), "Y": pt(25, 12.5)}),
			block("reg", 200, 20, 40, 40, map[string]Point{"Q": pt(40, 10)}),
			block("one", 10, 35, 20, 10, map[string]Point{"Y": pt(20, 5)}),
		},
		Edges: []*Edge{
			wire("e0", "reg.Q", "plus.A", pt(240, 30), pt(250, 30), pt(250, 100), pt(50, 100), pt(50, 25), pt(100, 25)),
			wire("e1", "one.Y", "plus.B", pt(30, 40), pt(100, 40)),
		},
	}
	return nl, g
}

func TestPermute(t *testing.T) {
	nl, g := adder("$add")
	changed := Permute(nl, g)
	if want := []string{"plus"}; !reflect.DeepEqual(changed, want) {
		t.Fatalf("Permute() = %v, want %v", changed, want)
	}
	got := nl.Modules["m"].Cells["plus"].Connections
	want := map[string][]int{"A": {3}, "B": {2}, "Y": {4}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("connections = %v, want %v", got, want)
	}
}

func TestPermuteLeavesOthers(t *testing.T) {
	// A subtractor's inputs cannot trade places.
	nl, g := adder("$sub")
	if changed := Permute(nl, g); changed != nil {
		t.Errorf("Permute() on $sub = %v, want nothing", changed)
	}
	// Inputs already in the order their wires come in stay.
	nl, g = adder("$and")
	g.Edges[0].Sections[0] = &Section{StartPoint: pt(240, 30), EndPoint: pt(100, 25)}
	if changed := Permute(nl, g); changed != nil {
		t.Errorf("Permute() on wires in order = %v, want nothing", changed)
	}
}

// tieOff is a graph with a constant at (10, 50), whose pin at its right
// side is at y=55, wired with a dogleg to the left pin of a block whose
// pin is at y=40.
func tieOff() *Graph {
	return &Graph{
		Width: 300, Height: 200,
		Children: []*Node{
			block("vcc", 10, 50, 40, 10, map[string]Point{"P": pt(40, 5)}),
			block("ff", 150, 20, 60, 40, map[string]Point{"CE": pt(0, 20), "Q": pt(60, 10)}),
		},
		Edges: []*Edge{
			wire("e0", "vcc.P", "ff.CE", pt(50, 55), pt(100, 55), pt(100, 40), pt(150, 40)),
		},
	}
}

func TestStraighten(t *testing.T) {
	g := tieOff()
	moved := Straighten(g)
	if want := []string{"vcc"}; !reflect.DeepEqual(moved, want) {
		t.Fatalf("Straighten() = %v, want %v", moved, want)
	}
	if y := g.Children[0].Y; y != 35 {
		t.Errorf("vcc at y=%v, want 35", y)
	}
	if got, want := g.Edges[0].Points(), []Point{pt(50, 40), pt(150, 40)}; !reflect.DeepEqual(got, want) {
		t.Errorf("wire = %v, want %v", got, want)
	}
}

func TestStraightenNeedsRoom(t *testing.T) {
	for _, tc := range []struct {
		desc   string
		change func(g *Graph)
	}{
		{"another block where it would go", func(g *Graph) {
			g.Children = append(g.Children, block("other", 5, 20, 40, 12, nil))
		}},
		{"a block in the straight wire's way", func(g *Graph) {
			g.Children = append(g.Children, block("other", 90, 30, 20, 20, nil))
		}},
		{"another net's wire along the straight wire", func(g *Graph) {
			g.Edges = append(g.Edges, wire("e1", "ff.Q", "ff.CE", pt(60, 40), pt(90, 40)))
		}},
		{"a group it is not in", func(g *Graph) {
			g.Groups = []Box{{Name: "core", X: 0, Y: 0, Width: 60, Height: 45}}
		}},
		{"a pin on the top of the target", func(g *Graph) {
			g.Children[1].Ports[0].X = 30
		}},
	} {
		g := tieOff()
		tc.change(g)
		if moved := Straighten(g); moved != nil {
			t.Errorf("%s: Straighten() = %v, want nothing moved", tc.desc, moved)
		}
	}
}

func TestReserve(t *testing.T) {
	g := tieOff()
	Reserve(g, Below{"ff": 14})
	if h := g.Children[1].Height; h != 54 {
		t.Errorf("ff height = %v, want 54", h)
	}
	if h := g.Children[0].Height; h != 10 {
		t.Errorf("vcc height = %v, want 10", h)
	}
}

func TestRoundTrip(t *testing.T) {
	// Labels netlistsvg writes, with a coordinate given as a string, and
	// an empty list, must come back as they were.
	in := `{"id":"root","x":0,"y":0,"width":10,"height":10,"children":[` +
		`{"id":"a","x":1,"y":2,"width":3,"height":4,"labels":[{"id":"a.label.ref","text":"a","x":"15","y":-10}]},` +
		`{"id":"b","x":1,"y":2,"width":3,"height":4,"labels":[]}],` +
		`"edges":[{"id":"e0","source":"a","sourcePort":"a.Y","target":"b","targetPort":"b.A"}]}`
	g, err := Parse([]byte(in))
	if err != nil {
		t.Fatal(err)
	}
	out, err := g.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	var got, want any
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatal(err)
	}
	json.Unmarshal([]byte(in), &want)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("round trip:\n got %s\nwant %s", out, in)
	}
	if e := g.Edges[0]; e.SourcePortID() != "a.Y" || e.TargetPortID() != "b.A" {
		t.Errorf("ports of an edge with sourcePort and targetPort = %q, %q", e.SourcePortID(), e.TargetPortID())
	}
	if r := g.Children[0].extent(); r.x0 != 1 || r.y0 != -8 || r.x1 != 16 || r.y1 != 6 {
		t.Errorf("extent with a label above = %+v", r)
	}
}
