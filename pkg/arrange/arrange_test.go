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

func TestMeasureJoinsDriversAndCountsOverlaps(t *testing.T) {
	g := &Graph{
		Edges: []*Edge{
			// Two drivers of one net, joined at netlistsvg's node $d_0:
			// they meet, and do not cross.
			wire("e0", "a.Y", "$d_0.p", pt(0, 10), pt(50, 10), pt(50, 30)),
			wire("e1", "b.Y", "$d_0.p", pt(0, 40), pt(50, 40), pt(50, 20)),
			// Another net along the first one's run at y=10.
			wire("e2", "c.Y", "d.A", pt(20, 10), pt(80, 10)),
		},
	}
	if got, want := Measure(g), (Score{Crossings: 1, Bends: 2}); got != want {
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

func TestNestBlockListedTwice(t *testing.T) {
	g := &Graph{Children: []*Node{block("a", 0, 0, 10, 10, nil)}}
	if nested := Nest(g, []ast.Group{{Name: "g", Blocks: []string{"a", "a"}}}); !reflect.DeepEqual(nested, []int{0}) {
		t.Errorf("Nest() = %v, want the group nested once", nested)
	}
	if n := len(g.Children[0].Children); n != 1 {
		t.Errorf("group holds %d blocks, want 1", n)
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

func TestExtentTakesTheLabelEitherWay(t *testing.T) {
	// A title anchored at its middle, x=15, 30 wide, over a block 30 wide.
	labels := []json.RawMessage{json.RawMessage(`{"id":"a.label.ref","text":"title","x":"15","y":-16,"width":30,"height":11}`)}
	n := &Node{ID: "a", X: 100, Y: 100, Width: 30, Height: 20, Labels: &labels}
	if r := n.extent(); r.x0 != 85 || r.x1 != 145 || r.y0 != 84 || r.y1 != 120 {
		t.Errorf("extent = %+v, want from x=85 to 145 and y=84 to 120", r)
	}
	// A coordinate that is not a number leaves the label out.
	bad := []json.RawMessage{json.RawMessage(`{"id":"a.label.ref","x":"left","y":-16,"width":30,"height":11}`)}
	n.Labels = &bad
	if r := n.extent(); r != (rect{100, 100, 130, 120}) {
		t.Errorf("extent with an unreadable label = %+v, want the body alone", r)
	}
}

// jogged is a graph with one wire that steps down 2.5 pixels before its
// pin, as ELK routes a wire into a group, and the blocks given.
func jogged(blocks ...*Node) *Graph {
	g := &Graph{Width: 300, Height: 200, Children: blocks}
	g.Children = append(g.Children,
		block("a", 0, 75, 10, 10, map[string]Point{"Y": pt(10, 5)}),
		block("b", 151, 104, 20, 20, map[string]Point{"B": pt(0, 10)}))
	g.Edges = []*Edge{wire("e0", "a.Y", "b.B",
		pt(10, 80), pt(20, 80), pt(20, 116.5), pt(126, 116.5), pt(126, 114), pt(151, 114))}
	return g
}

func TestUnjog(t *testing.T) {
	for _, tc := range []struct {
		desc   string
		blocks []*Node
		want   []Point
	}{
		{"the middle run joins the first", nil,
			[]Point{pt(10, 80), pt(126, 80), pt(126, 114), pt(151, 114)}},
		{"a block on the first run's line: the middle run joins the last",
			[]*Node{block("x", 40, 70, 60, 20, nil)},
			[]Point{pt(10, 80), pt(20, 80), pt(20, 114), pt(151, 114)}},
	} {
		g := jogged(tc.blocks...)
		if n := Unjog(g); n != 1 {
			t.Errorf("%s: Unjog() = %d, want 1", tc.desc, n)
		}
		if got := g.Edges[0].Points(); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%s: wire = %v, want %v", tc.desc, got, tc.want)
		}
	}
}

func TestUnjogLeaves(t *testing.T) {
	for _, tc := range []struct {
		desc   string
		change func(g *Graph)
	}{
		{"blocks on both lines", func(g *Graph) {
			g.Children = append(g.Children, block("x", 40, 70, 60, 20, nil), block("y", 40, 108, 60, 12, nil))
		}},
		{"a wire that shares its net", func(g *Graph) {
			g.Children = append(g.Children, block("c", 151, 150, 20, 20, map[string]Point{"A": pt(0, 10)}))
			g.Edges = append(g.Edges, wire("e1", "a.Y", "c.A", pt(10, 80), pt(20, 80), pt(20, 160), pt(151, 160)))
		}},
	} {
		g := jogged()
		tc.change(g)
		before := g.Edges[0].Points()
		if n := Unjog(g); n != 0 {
			t.Errorf("%s: Unjog() = %d, want 0", tc.desc, n)
		}
		if got := g.Edges[0].Points(); !reflect.DeepEqual(got, before) {
			t.Errorf("%s: wire = %v, want it unchanged", tc.desc, got)
		}
	}
}

func TestSide(t *testing.T) {
	for _, tc := range []struct {
		desc string
		n    *Node
		want byte
	}{
		{"an adder's input, 2 px inside its circle", block("plus", 0, 0, 25, 25, map[string]Point{"B": pt(2, 20)}), 'L'},
		{"an input port's pin", block("in", 0, 0, 20, 10, map[string]Point{"Y": pt(20, 5)}), 'R'},
		{"a buffer's enable", block("buf", 0, 0, 30, 30, map[string]Point{"EN": pt(15, 0)}), 'T'},
		{"a multiplexer's select", block("mux", 0, 0, 20, 40, map[string]Point{"S": pt(10, 40)}), 'B'},
	} {
		p := portAt{tc.n, tc.n.Ports[0]}
		if got := p.side(); got != tc.want {
			t.Errorf("%s: side() = %c, want %c", tc.desc, got, tc.want)
		}
	}
}
