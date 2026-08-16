// SPDX-License-Identifier: Apache-2.0
package yosys

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/filmil/blockpic/pkg/ast"
)

type Netlist struct {
	Modules map[string]Module `json:"modules"`
}

type Module struct {
	Ports    map[string]Port    `json:"ports"`
	Cells    map[string]Cell    `json:"cells"`
	Netnames map[string]Netname `json:"netnames"`
}

type Port struct {
	Direction string `json:"direction"`
	Bits      []int  `json:"bits"`
}

type Cell struct {
	HideName       int              `json:"hide_name"`
	Type           string           `json:"type"`
	PortDirections map[string]string `json:"port_directions"`
	Connections    map[string][]int  `json:"connections"`
	Attributes     map[string]string `json:"attributes"`
}

type Netname struct {
	HideName   int               `json:"hide_name"`
	Bits       []int             `json:"bits"`
	Attributes map[string]string `json:"attributes"`
}

func Convert(diag *ast.Diagram) ([]byte, error) {
	mod := Module{
		Ports:    make(map[string]Port),
		Cells:    make(map[string]Cell),
		Netnames: make(map[string]Netname),
	}

	bitCounter := 2 // Start bits from 2 (netlistsvg convention often avoids 0/1 for nets, 0=0, 1=1)

	// Map to keep track of the bit assigned to each Source (Block.Pin)
	sourceBits := make(map[string]int)

	// First pass: assign bits to all sources of wires
	for _, w := range diag.Wires {
		srcKey := fmt.Sprintf("%s.%s", w.Source.Block, w.Source.Pin)
		if _, exists := sourceBits[srcKey]; !exists {
			sourceBits[srcKey] = bitCounter
			bitCounter++
		}
	}

	// Function to get the bit for a specific pin
	getBit := func(blockName, pinName string, isOutput bool) int {
		key := fmt.Sprintf("%s.%s", blockName, pinName)
		if isOutput {
			if bit, exists := sourceBits[key]; exists {
				return bit
			}
			// Unconnected output
			b := bitCounter
			bitCounter++
			return b
		} else {
			// Input pin - find wire where this is the destination
			for _, w := range diag.Wires {
				if w.Destination.Block == blockName && w.Destination.Pin == pinName {
					srcKey := fmt.Sprintf("%s.%s", w.Source.Block, w.Source.Pin)
					return sourceBits[srcKey]
				}
			}
			// Unconnected input
			b := bitCounter
			bitCounter++
			return b
		}
	}

	for _, b := range diag.Blocks {
		if strings.ToUpper(b.Kind) == "INPUT_PIN" {
			pinName := "out"
			if len(b.Outputs) > 0 {
				pinName = b.Outputs[0].Name
				if pinName == "" {
					pinName = "out"
				}
			}
			bit := getBit(b.Label, pinName, true)
			mod.Ports[b.Label] = Port{
				Direction: "input",
				Bits:      []int{bit},
			}
			continue
		}

		if strings.ToUpper(b.Kind) == "OUTPUT_PIN" {
			pinName := "in"
			if len(b.Inputs) > 0 {
				pinName = b.Inputs[0].Name
				if pinName == "" {
					pinName = "in"
				}
			}
			bit := getBit(b.Label, pinName, false)
			mod.Ports[b.Label] = Port{
				Direction: "output",
				Bits:      []int{bit},
			}
			continue
		}

		cell := Cell{
			Type:           mapKindToYosysType(b.Kind),
			PortDirections: make(map[string]string),
			Connections:    make(map[string][]int),
			Attributes:     make(map[string]string),
		}
		
		if b.Name != "" {
			cell.Attributes["label"] = b.Name
		} else {
			cell.Attributes["label"] = b.Label
		}

		for k, v := range b.Params {
			// Convert ~TEXT~ to unicode overbars
			processedVal := v
			if strings.HasPrefix(v, "~") && strings.HasSuffix(v, "~") {
				coreText := v[1 : len(v)-1]
				var overbar strings.Builder
				for _, char := range coreText {
					overbar.WriteRune(char)
					overbar.WriteRune('\u0305') // Combining overline
				}
				processedVal = overbar.String()
			}
			cell.Attributes[k] = processedVal
		}

		mapPort := func(pinName string, isOutput bool, idx int) string {
			switch cell.Type {
			case "$and", "$or", "$xor", "$nand", "$nor", "$xnor", "$andnot", "$ornot",
				"$add", "$sub", "$mul", "$div", "$mod", "$pow",
				"$eq", "$ne", "$lt", "$le", "$ge", "$gt",
				"$shr", "$shl", "$sshr", "$sshl":
				if isOutput {
					return "Y"
				}
				if idx == 0 {
					return "A"
				}
				if idx == 1 {
					return "B"
				}
			case "$not", "$buf", "$pos", "$neg", "$reduce_and", "$reduce_or", "$reduce_xor", "$reduce_xnor", "$reduce_bool", "$logic_not", "$reduce_nor", "$reduce_nxor":
				if isOutput {
					return "Y"
				}
				return "A"
			case "$mux", "$pmux", "$mux-bus", "$pmux-bus":
				if isOutput {
					return "Y"
				}
				if idx == 0 {
					return "A"
				}
				if idx == 1 {
					return "B"
				}
				if idx == 2 {
					return "S"
				}
			case "$tribuf":
				if isOutput {
					return "Y"
				}
				if idx == 0 {
					return "A"
				}
				if idx == 1 {
					return "EN"
				}
			case "$dff", "$dff-bus", "$dffn", "$dffn-bus":
				if isOutput {
					return "Q"
				}
				if idx == 0 {
					return "D"
				}
				if idx == 1 {
					return "CLK"
				}
			case "$dlatch", "$dlatch-bus", "$dlatchn", "$dlatchn-bus":
				if isOutput {
					return "Q"
				}
				if idx == 0 {
					return "D"
				}
				if idx == 1 {
					return "EN"
				}
			case "$_AOI3_", "$_OAI3_":
				if isOutput {
					return "Y"
				}
				if idx == 0 { return "A" }
				if idx == 1 { return "B" }
				if idx == 2 { return "C" }
			case "$_AOI4_", "$_OAI4_":
				if isOutput {
					return "Y"
				}
				if idx == 0 { return "A" }
				if idx == 1 { return "B" }
				if idx == 2 { return "C" }
				if idx == 3 { return "D" }
			}
			return pinName
		}

		for i, p := range b.Inputs {
			pinName := p.Name
			if pinName == "" {
				pinName = "in"
			}
			yosysName := mapPort(pinName, false, i)
			cell.PortDirections[yosysName] = "input"
			cell.Connections[yosysName] = []int{getBit(b.Label, p.Name, false)}
		}

		for i, p := range b.Outputs {
			pinName := p.Name
			if pinName == "" {
				pinName = "out"
			}
			yosysName := mapPort(pinName, true, i)
			cell.PortDirections[yosysName] = "output"
			cell.Connections[yosysName] = []int{getBit(b.Label, p.Name, true)}
		}

		mod.Cells[b.Label] = cell
	}

	netlist := Netlist{
		Modules: map[string]Module{
			diag.Name: mod,
		},
	}

	return json.MarshalIndent(netlist, "", "  ")
}

func mapKindToYosysType(kind string) string {
	k := strings.ToLower(kind)
	switch k {
	case "and", "nand", "or", "nor", "xor", "xnor", "not", "mux", "add", "sub",
		"andnot", "ornot", "tribuf", "buf", "pos", "neg", "eq", "ne", "lt", "le", "ge", "gt",
		"mul", "div", "mod", "pow", "shr", "shl", "sshr", "sshl",
		"reduce_nor", "reduce_xor", "reduce_nxor",
		"dff", "dffn", "dlatch", "dlatchn",
		"split", "join", "constant":
		return "$" + k
	case "mux_bus", "dff_bus", "dffn_bus", "dlatch_bus", "dlatchn_bus":
		return "$" + strings.ReplaceAll(k, "_", "-")
	case "register", "reg":
		return "$dff"
	case "aoi3", "oai3", "aoi4", "oai4":
		return "$_" + strings.ToUpper(k) + "_"
	default:
		// Unknown types render as generic blocks
		return kind
	}
}
