// SPDX-License-Identifier: Apache-2.0
package yosys

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/filmil/blockpic/pkg/ast"
)

// The JSON netlist as Yosys's write_json writes it. A bit is either a net
// number or a constant: "0", "1", "x" or "z".
type inNetlist struct {
	Modules map[string]inModule `json:"modules"`
}

type inModule struct {
	Attributes map[string]any       `json:"attributes"`
	Ports      map[string]inPort    `json:"ports"`
	Cells      map[string]inCell    `json:"cells"`
	Netnames   map[string]inNetname `json:"netnames"`
}

type inPort struct {
	Direction string `json:"direction"`
	Bits      []any  `json:"bits"`
}

type inCell struct {
	HideName       int               `json:"hide_name"`
	Type           string            `json:"type"`
	PortDirections map[string]string `json:"port_directions"`
	Connections    map[string][]any  `json:"connections"`
}

type inNetname struct {
	HideName   int            `json:"hide_name"`
	Bits       []any          `json:"bits"`
	Attributes map[string]any `json:"attributes"`
}

// bit is one bit of a connection: a net number, or a constant when net is
// zero.
type bit struct {
	net   int
	value byte
}

func parseBit(v any) (bit, error) {
	switch b := v.(type) {
	case float64:
		return bit{net: int(b)}, nil
	case string:
		if len(b) == 1 && strings.Contains("01xz", b) {
			return bit{value: b[0]}, nil
		}
	}
	return bit{}, fmt.Errorf("unrecognised bit %v", v)
}

func parseBits(vs []any) ([]bit, error) {
	bits := make([]bit, len(vs))
	for i, v := range vs {
		b, err := parseBit(v)
		if err != nil {
			return nil, err
		}
		bits[i] = b
	}
	return bits, nil
}

// Import reads a Yosys JSON netlist, such as the one rules_openxc7's
// vivado_synthesis writes, and returns a diagram of one module in it.
//
// data is the netlist. top names the module to draw; when it is empty,
// the module with the "top" attribute is drawn, or the only module that
// has cells.
//
// Each cell becomes a block of the cell's type, and each module port
// becomes an INPUT_PIN or OUTPUT_PIN block (an inout port is drawn as an
// input). A wire joins a driving pin to a driven pin; when it carries
// several bits, its thickness is their number, and its type is the
// typedef of the net that holds exactly those bits, when Yosys recorded
// one (the wiretype attribute of a SystemVerilog wire). A driven port whose bits
// come from more than one pin is split into one pin per bit, named
// PORT_i. A constant is a block of its own for each driven pin: a GND or
// VCC cell when its bits are all zeros or all ones, or else a CONST cell
// named by the value, such as 4'b0001. The skins draw these as tie-offs,
// the value beside a short stub at the pin. Undriven bits are left out.
//
// The returned diagram passes validation.Validate. An error is returned
// when the netlist does not parse or no module can be chosen.
func Import(data []byte, top string) (*ast.Diagram, error) {
	var nl inNetlist
	if err := json.Unmarshal(data, &nl); err != nil {
		return nil, fmt.Errorf("parsing the netlist: %w", err)
	}
	name, err := pickModule(&nl, top)
	if err != nil {
		return nil, err
	}
	im := &importer{
		nl:        &nl,
		mod:       nl.Modules[name],
		labels:    make(map[string]bool),
		drivers:   make(map[int]ast.PinRef),
		wireIdx:   make(map[[2]ast.PinRef]int),
		typeCount: make(map[string]int),
		diag:      &ast.Diagram{Name: identifier(name)},
	}
	if err := im.run(); err != nil {
		return nil, fmt.Errorf("module %s: %w", name, err)
	}
	return im.diag, nil
}

func pickModule(nl *inNetlist, top string) (string, error) {
	if top != "" {
		if _, ok := nl.Modules[top]; !ok {
			return "", fmt.Errorf("no module %q in the netlist", top)
		}
		return top, nil
	}
	var tops, withCells []string
	for name, m := range nl.Modules {
		if isTrue(m.Attributes["top"]) {
			tops = append(tops, name)
		}
		if len(m.Cells) > 0 && !isTrue(m.Attributes["blackbox"]) {
			withCells = append(withCells, name)
		}
	}
	sort.Strings(tops)
	sort.Strings(withCells)
	switch {
	case len(tops) == 1:
		return tops[0], nil
	case len(tops) > 1:
		return "", fmt.Errorf("several top modules (%s); name one", strings.Join(tops, ", "))
	case len(withCells) == 1:
		return withCells[0], nil
	}
	return "", fmt.Errorf("no top module marked; name one of: %s", strings.Join(withCells, ", "))
}

// isTrue reads a Yosys attribute value, which write_json gives as a
// binary string of 32 digits or, in older versions, as a number.
func isTrue(v any) bool {
	switch a := v.(type) {
	case float64:
		return a != 0
	case string:
		n, err := strconv.ParseUint(a, 2, 64)
		return err == nil && n != 0
	}
	return false
}

type importer struct {
	nl      *inNetlist
	mod     inModule
	diag    *ast.Diagram
	labels  map[string]bool
	drivers map[int]ast.PinRef
	// wireIdx finds the wire already joining two pins, to count its bits.
	wireIdx map[[2]ast.PinRef]int
	// typeCount numbers the unnamed blocks of each kind.
	typeCount map[string]int
	// busTypes finds the type of a bus by its bits, from the wiretype
	// attribute that Yosys gives a wire declared with a typedef.
	busTypes map[string]string
}

// port is a connection of a block: a cell's port, or a module port
// drawn as an INPUT_PIN or OUTPUT_PIN block.
type port struct {
	block int
	name  string
	bits  []bit
}

func (im *importer) run() error {
	im.findBusTypes()
	// sources drive bits; sinks are driven: cell inputs and module
	// outputs.
	var sources, sinks []port

	for _, name := range sortedKeys(im.mod.Ports) {
		p := im.mod.Ports[name]
		bits, err := parseBits(p.Bits)
		if err != nil {
			return fmt.Errorf("port %s: %w", name, err)
		}
		kind := "INPUT_PIN"
		if p.Direction == "output" {
			kind = "OUTPUT_PIN"
		}
		b := im.addBlock(ast.Block{Label: im.label(name), Name: name, Kind: kind})
		if kind == "OUTPUT_PIN" {
			sinks = append(sinks, port{block: b, name: name, bits: bits})
		} else {
			sources = append(sources, port{block: b, name: name, bits: bits})
		}
	}

	names := im.cellNames()
	for _, cname := range sortedKeys(im.mod.Cells) {
		c := im.mod.Cells[cname]
		if c.Type == "$scopeinfo" {
			continue
		}
		display, base := names[cname], names[cname]
		if display == "" {
			display = c.Type
			base = im.numbered(c.Type)
		}
		b := im.addBlock(ast.Block{Label: im.label(base), Name: display, Kind: c.Type})
		for _, pname := range sortedKeys(c.Connections) {
			bits, err := parseBits(c.Connections[pname])
			if err != nil {
				return fmt.Errorf("cell %s, port %s: %w", cname, pname, err)
			}
			p := port{block: b, name: pname, bits: bits}
			if im.direction(c, pname) == "input" {
				sinks = append(sinks, p)
			} else {
				sources = append(sources, p)
			}
		}
	}

	for _, src := range sources {
		im.addSource(src, wholeSource(src, sinks))
	}
	for _, s := range sinks {
		im.connect(s)
	}

	// A split module port leaves its own block without pins.
	blocks := im.diag.Blocks[:0]
	for _, b := range im.diag.Blocks {
		if len(b.Inputs)+len(b.Outputs) > 0 || (b.Kind != "INPUT_PIN" && b.Kind != "OUTPUT_PIN") {
			blocks = append(blocks, b)
		}
	}
	im.diag.Blocks = blocks
	return nil
}

// wholeSource reports whether a source can be drawn as one pin: that is,
// whether every sink that takes any of its bits takes all of them, in
// order.
func wholeSource(src port, sinks []port) bool {
	if len(src.bits) < 2 {
		return true
	}
	mine := make(map[int]bool)
	for _, x := range src.bits {
		mine[x.net] = true
	}
	for _, s := range sinks {
		taken := 0
		for _, x := range s.bits {
			if x.net != 0 && mine[x.net] {
				taken++
			}
		}
		if taken == 0 {
			continue
		}
		if len(s.bits) != len(src.bits) {
			return false
		}
		for i := range s.bits {
			if s.bits[i] != src.bits[i] {
				return false
			}
		}
	}
	return true
}

// addSource gives a source its output pins and records which pin drives
// each of its bits. A source that is not drawn whole gets a pin for each
// bit, PORT_i; a split module input gets a block for each bit.
func (im *importer) addSource(src port, whole bool) {
	add := func(name, display string, bits []bit) {
		b := &im.diag.Blocks[src.block]
		pin := pinName(name)
		if b.Kind == "INPUT_PIN" {
			pin = "out"
			if !whole {
				i := im.addBlock(ast.Block{Label: im.label(name), Name: display, Kind: "INPUT_PIN"})
				b = &im.diag.Blocks[i]
			}
		}
		b.Outputs = append(b.Outputs, ast.Pin{Name: pin})
		for _, x := range bits {
			if x.net != 0 {
				im.drivers[x.net] = ast.PinRef{Block: b.Label, Pin: pin}
			}
		}
	}
	if whole {
		add(src.name, src.name, src.bits)
		return
	}
	for i, x := range src.bits {
		add(fmt.Sprintf("%s_%d", src.name, i), fmt.Sprintf("%s[%d]", src.name, i), []bit{x})
	}
}

// connect gives a sink its pins, and joins them to their drivers.
func (im *importer) connect(s port) {
	// The pin that drives each bit, with an empty block for a constant
	// or an undriven bit.
	from := make([]ast.PinRef, len(s.bits))
	same := true
	for i, x := range s.bits {
		if x.net != 0 {
			from[i] = im.drivers[x.net]
		}
		if from[i] != from[0] {
			same = false
		}
	}
	if same {
		im.connectPins(s, s.name, s.bits, from[0])
		return
	}
	for i, x := range s.bits {
		im.connectPins(s, fmt.Sprintf("%s_%d", s.name, i), []bit{x}, from[i])
	}
}

// connectPins adds a pin named port to the sink's block and joins it to
// src. bits are the bits it carries, all driven by src; with no src they
// are constants or undriven.
func (im *importer) connectPins(s port, name string, bits []bit, src ast.PinRef) {
	blk := &im.diag.Blocks[s.block]
	var dst ast.PinRef
	if blk.Kind == "OUTPUT_PIN" {
		if bit, ok := strings.CutPrefix(name, s.name+"_"); ok {
			// A split output port: a block per bit.
			b := im.addBlock(ast.Block{Label: im.label(name), Name: s.name + "[" + bit + "]", Kind: "OUTPUT_PIN"})
			blk = &im.diag.Blocks[b]
		}
		blk.Inputs = append(blk.Inputs, ast.Pin{Name: "in"})
		dst = ast.PinRef{Block: blk.Label, Pin: "in"}
	} else {
		pin := pinName(name)
		blk.Inputs = append(blk.Inputs, ast.Pin{Name: pin})
		dst = ast.PinRef{Block: blk.Label, Pin: pin}
	}

	if src.Block == "" {
		value, ok := constant(bits)
		if !ok {
			return
		}
		kind, pin := constantCell(value)
		// GND and VCC say their value; a CONST is named by it.
		name := kind
		if kind == "CONST" {
			name = value
		}
		b := im.addBlock(ast.Block{
			Label:   im.label(im.numbered(kind)),
			Name:    name,
			Kind:    kind,
			Outputs: []ast.Pin{{Name: pin}},
		})
		src = ast.PinRef{Block: im.diag.Blocks[b].Label, Pin: pin}
	}

	key := [2]ast.PinRef{src, dst}
	if i, ok := im.wireIdx[key]; ok {
		im.diag.Wires[i].Thickness += len(bits)
		return
	}
	im.wireIdx[key] = len(im.diag.Wires)
	w := ast.Wire{Source: src, Destination: dst}
	if len(bits) > 1 {
		w.Thickness = len(bits)
		w.Type = im.busTypes[bitsKey(bits)]
	}
	im.diag.Wires = append(im.diag.Wires, w)
}

// bitsKey makes a map key of a sequence of bits.
func bitsKey(bits []bit) string {
	var sb strings.Builder
	for _, x := range bits {
		fmt.Fprintf(&sb, "%d%c,", x.net, x.value)
	}
	return sb.String()
}

// findBusTypes maps the bits of every net that Yosys gave a wiretype
// attribute, a wire declared with a typedef, to the name of its type.
func (im *importer) findBusTypes() {
	im.busTypes = make(map[string]string)
	for _, n := range sortedKeys(im.mod.Netnames) {
		nn := im.mod.Netnames[n]
		t, ok := nn.Attributes["wiretype"].(string)
		if !ok || t == "" || len(nn.Bits) < 2 {
			continue
		}
		bits, err := parseBits(nn.Bits)
		if err != nil {
			continue
		}
		key := bitsKey(bits)
		if _, ok := im.busTypes[key]; !ok {
			im.busTypes[key] = strings.TrimPrefix(t, "\\")
		}
	}
}

// constant spells bits, least significant first, as a Verilog constant.
// It reports false when a bit is an undriven net.
func constant(bits []bit) (string, bool) {
	var sb strings.Builder
	for i := len(bits) - 1; i >= 0; i-- {
		if bits[i].net != 0 {
			return "", false
		}
		sb.WriteByte(bits[i].value)
	}
	if len(bits) == 1 {
		return sb.String(), true
	}
	return fmt.Sprintf("%d'b%s", len(bits), sb.String()), true
}

// constantCell names the cell that drives a constant, and its output: the
// Xilinx primitives GND and VCC for all zeros and all ones, or else CONST.
func constantCell(value string) (kind, pin string) {
	digits := value[strings.Index(value, "b")+1:]
	switch {
	case strings.Trim(digits, "0") == "":
		return "GND", "G"
	case strings.Trim(digits, "1") == "":
		return "VCC", "P"
	}
	return "CONST", "O"
}

// direction is the direction of a cell's port, from the cell or else
// from the cell type's module in the netlist. An unknown port is taken
// as an input, and an inout port as an output.
func (im *importer) direction(c inCell, port string) string {
	d := c.PortDirections[port]
	if d == "" {
		if m, ok := im.nl.Modules[c.Type]; ok {
			d = m.Ports[port].Direction
		}
	}
	if d == "" {
		return "input"
	}
	if d == "inout" {
		return "output"
	}
	return d
}

// cellNames gives a display name to every cell that has one: its own name
// when Yosys shows it, or else the name of a shown net it drives, such as
// count[3] for the flip-flop that holds that bit.
func (im *importer) cellNames() map[string]string {
	netName := make(map[int]string)
	for _, n := range sortedKeys(im.mod.Netnames) {
		nn := im.mod.Netnames[n]
		if nn.HideName != 0 {
			continue
		}
		for i, v := range nn.Bits {
			x, err := parseBit(v)
			if err != nil || x.net == 0 {
				continue
			}
			if _, ok := netName[x.net]; ok {
				continue
			}
			if len(nn.Bits) == 1 {
				netName[x.net] = n
			} else {
				netName[x.net] = fmt.Sprintf("%s[%d]", n, i)
			}
		}
	}
	// A module port's net is named after the port, which is drawn
	// already.
	for _, p := range im.mod.Ports {
		for _, v := range p.Bits {
			if x, err := parseBit(v); err == nil {
				delete(netName, x.net)
			}
		}
	}

	names := make(map[string]string)
	used := make(map[string]bool)
	for _, cname := range sortedKeys(im.mod.Cells) {
		c := im.mod.Cells[cname]
		if c.HideName == 0 {
			names[cname] = strings.TrimPrefix(cname, "\\")
			continue
		}
		for _, port := range sortedKeys(c.Connections) {
			if im.direction(c, port) != "output" {
				continue
			}
			for _, v := range c.Connections[port] {
				x, err := parseBit(v)
				if err != nil || x.net == 0 {
					continue
				}
				if n, ok := netName[x.net]; ok && !used[n] && names[cname] == "" {
					names[cname] = n
					used[n] = true
				}
			}
		}
	}
	return names
}

func (im *importer) addBlock(b ast.Block) int {
	if b.Params == nil {
		b.Params = make(map[string]string)
	}
	im.diag.Blocks = append(im.diag.Blocks, b)
	return len(im.diag.Blocks) - 1
}

// numbered gives the next name in a series for unnamed blocks of a
// kind: INV_1, INV_2 and so on.
func (im *importer) numbered(kind string) string {
	im.typeCount[kind]++
	return fmt.Sprintf("%s_%d", kind, im.typeCount[kind])
}

// label makes an unused block label from name.
func (im *importer) label(name string) string {
	base := identifier(name)
	l := base
	for i := 2; im.labels[l]; i++ {
		l = fmt.Sprintf("%s_%d", base, i)
	}
	im.labels[l] = true
	return l
}

// keywords are the words the blockpic scanner reserves.
var keywords = map[string]bool{
	"diagram": true, "block": true, "wire": true, "layout": true,
	"group": true, "kind": true, "name": true, "inputs": true,
	"outputs": true, "params": true, "size": true, "hint": true,
	"thickness": true, "crossline": true, "at": true, "type": true,
}

// identifier makes a blockpic identifier from a Yosys name: every run of
// characters an identifier cannot hold becomes one underscore.
func identifier(name string) string {
	var sb strings.Builder
	under := false
	for _, r := range strings.TrimPrefix(name, "\\") {
		if r < 128 && (r == '_' || ('a' <= r && r <= 'z') || ('A' <= r && r <= 'Z') || ('0' <= r && r <= '9')) {
			sb.WriteRune(r)
			under = r == '_'
			continue
		}
		if !under && sb.Len() > 0 {
			sb.WriteByte('_')
			under = true
		}
	}
	s := strings.TrimRight(sb.String(), "_")
	if s == "" || s[0] == '_' || ('0' <= s[0] && s[0] <= '9') {
		s = "n" + s
	}
	if keywords[s] {
		s += "_"
	}
	return s
}

// pinName makes a pin name from a port name.
func pinName(port string) string {
	return identifier(port)
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
