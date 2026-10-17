// SPDX-License-Identifier: Apache-2.0
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"

	"github.com/filmil/blockpic/pkg/arrange"
	"github.com/filmil/blockpic/pkg/ast"
	"github.com/filmil/blockpic/pkg/parser"
	"github.com/filmil/blockpic/pkg/scanner"
	"github.com/filmil/blockpic/pkg/yosys"
)

// permuteRounds bounds how often the inputs of commutative cells are
// reordered and the diagram laid out again.
const permuteRounds = 3

// nameRoom is the room kept under a block whose name refine writes under
// it.
const nameRoom = 14

// layoutCmd lays out a rendered diagram for `nsv draw`:
//
//	blockpic layout <netlist.json> -src <file.blockpic> -skin <skin.svg>
//	    -nsv <nsv> [-o layout.json] [-netlist arranged.json] [-v]
//
// It writes the laid-out ELK graph, with the boxes of the diagram's
// groups, and the netlist to draw with it, whose commutative cells may
// have their inputs in another order. See package arrange.
func layoutCmd() {
	fl := flag.NewFlagSet("layout", flag.ExitOnError)
	src := fl.String("src", "", "The .blockpic file that the netlist was rendered from")
	skin := fl.String("skin", "", "The netlistsvg skin to draw with")
	nsv := fl.String("nsv", "", "The nsv tool, //tools/netlistsvg:nsv")
	out := fl.String("o", "layout.json", "The laid-out ELK graph to write")
	netOut := fl.String("netlist", "arranged.json", "The netlist to write, for drawing with the layout")
	verbose := fl.Bool("v", false, "Say what the layout changed")

	usage := "Usage: blockpic layout <netlist.json> -src <file.blockpic> -skin <skin.svg> -nsv <nsv> [-o layout.json] [-netlist arranged.json] [-v]"
	if len(os.Args) < 3 {
		fmt.Println(usage)
		os.Exit(1)
	}
	in := os.Args[2]
	fl.Parse(os.Args[3:])
	if *src == "" || *skin == "" || *nsv == "" {
		fmt.Println(usage)
		os.Exit(1)
	}
	if err := runLayout(in, *src, *skin, *nsv, *out, *netOut, *verbose); err != nil {
		fmt.Fprintf(os.Stderr, "blockpic layout: %v\n", err)
		os.Exit(1)
	}
}

func runLayout(in, src, skin, nsv, out, netOut string, verbose bool) error {
	content, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	p := parser.New(scanner.New(string(content)))
	diag := p.ParseDiagram()
	if len(p.Errors()) > 0 {
		return fmt.Errorf("parsing %s: %v", src, p.Errors())
	}
	data, err := os.ReadFile(in)
	if err != nil {
		return err
	}
	var nl yosys.Netlist
	if err := json.Unmarshal(data, &nl); err != nil {
		return fmt.Errorf("reading %s: %w", in, err)
	}

	tmp, err := os.MkdirTemp("", "blockpic-layout-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	skinData, err := os.ReadFile(skin)
	if err != nil {
		return err
	}
	room := below(diag, &nl, skinData)
	l := layouter{nsv: nsv, skin: skin, dir: tmp, groups: diag.Groups, below: room}

	g, err := l.lay(&nl, "0")
	if err != nil {
		return err
	}
	score := arrange.Measure(g)
	if verbose {
		fmt.Printf("laid out: %v\n", score)
	}
	for i := 1; i <= permuteRounds; i++ {
		trial, err := copyNetlist(&nl)
		if err != nil {
			return err
		}
		changed := arrange.Permute(trial, g)
		if len(changed) == 0 {
			break
		}
		tg, err := l.lay(trial, strconv.Itoa(i))
		if err != nil {
			return err
		}
		s := arrange.Measure(tg)
		if verbose {
			fmt.Printf("inputs reordered on %v: %v\n", changed, s)
		}
		if !s.Better(score) {
			break
		}
		nl, g, score = *trial, tg, s
	}
	moved := arrange.Straighten(g)
	if verbose && len(moved) > 0 {
		fmt.Printf("straightened %v: %v\n", moved, arrange.Measure(g))
	}
	if n := arrange.Unjog(g); verbose && n > 0 {
		fmt.Printf("unjogged %d runs: %v\n", n, arrange.Measure(g))
	}

	laid, err := g.Marshal()
	if err != nil {
		return err
	}
	if err := os.WriteFile(out, laid, 0o644); err != nil {
		return err
	}
	arranged, err := json.MarshalIndent(nl, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(netOut, arranged, 0o644)
}

// layouter lays a netlist out with nsv, with the diagram's groups as
// boxes of their own.
type layouter struct {
	nsv, skin, dir string
	groups         []ast.Group
	below          arrange.Below
}

func (l *layouter) lay(nl *yosys.Netlist, tag string) (*arrange.Graph, error) {
	path := func(name string) string { return filepath.Join(l.dir, tag+"."+name) }
	data, err := json.MarshalIndent(nl, "", "  ")
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(path("netlist.json"), data, 0o644); err != nil {
		return nil, err
	}
	if err := l.run("graph", path("netlist.json"), l.skin, path("graph.json")); err != nil {
		return nil, err
	}
	// A graph with nothing to add goes to ELK as nsv wrote it.
	toLay := path("graph.json")
	if len(l.groups) > 0 || len(l.below) > 0 {
		data, err := os.ReadFile(toLay)
		if err != nil {
			return nil, err
		}
		g, err := arrange.Parse(data)
		if err != nil {
			return nil, err
		}
		arrange.Reserve(g, l.below)
		arrange.Nest(g, l.groups)
		if data, err = g.Marshal(); err != nil {
			return nil, err
		}
		toLay = path("arranged.json")
		if err := os.WriteFile(toLay, data, 0o644); err != nil {
			return nil, err
		}
	}
	if err := l.run("layout", toLay, path("laid.json")); err != nil {
		return nil, err
	}
	data, err = os.ReadFile(path("laid.json"))
	if err != nil {
		return nil, err
	}
	g, err := arrange.Parse(data)
	if err != nil {
		return nil, err
	}
	if err := arrange.Flatten(g, l.groups); err != nil {
		return nil, err
	}
	return g, nil
}

func (l *layouter) run(args ...string) error {
	cmd := exec.Command(l.nsv, args...)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("nsv %s: %w", args[0], err)
	}
	return nil
}

func copyNetlist(nl *yosys.Netlist) (*yosys.Netlist, error) {
	data, err := json.Marshal(nl)
	if err != nil {
		return nil, err
	}
	var c yosys.Netlist
	return &c, json.Unmarshal(data, &c)
}

// below keeps room under each block whose name refine writes there: a
// block drawn as a generic cell, for which the skin has no symbol, that
// has a name of its own. refine writes the name of any other block in
// place of its title.
func below(diag *ast.Diagram, nl *yosys.Netlist, skin []byte) arrange.Below {
	symbols := make(map[string]bool)
	for _, m := range skinAlias.FindAllSubmatch(skin, -1) {
		symbols[string(m[1])] = true
	}
	b := make(arrange.Below)
	for _, mod := range nl.Modules {
		for _, blk := range diag.Blocks {
			cell, ok := mod.Cells[blk.Label]
			if !ok || symbols[cell.Type] {
				continue
			}
			if blk.Name != "" && blk.Name != blk.Kind {
				b[blk.Label] = nameRoom
			}
		}
	}
	return b
}

// skinAlias finds a cell type that a skin has a symbol for.
var skinAlias = regexp.MustCompile(`<s:alias val="([^"]*)"`)
