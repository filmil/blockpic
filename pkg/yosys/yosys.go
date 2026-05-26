// SPDX-License-Identifier: Apache-2.0
package yosys

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/filmil/hodos/pkg/ast"
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

		for _, param := range []string{"WIDTH", "INIT", "TYPE"} {
			if val, ok := b.Params[param]; ok {
				cell.Attributes[param] = val
			}
		}

		mapPort := func(pinName string, isOutput bool, idx int) string {
			if cell.Type == "$and" || cell.Type == "$or" || cell.Type == "$xor" || cell.Type == "$nand" || cell.Type == "$nor" || cell.Type == "$xnor" {
				if isOutput { return "Y" }
				if idx == 0 { return "A" }
				if idx == 1 { return "B" }
			}
			if cell.Type == "$not" {
				if isOutput { return "Y" }
				return "A"
			}
			if cell.Type == "$mux" {
				if isOutput { return "Y" }
				if idx == 0 { return "A" }
				if idx == 1 { return "B" }
				if idx == 2 { return "S" }
			}
			if cell.Type == "$dff" {
				if isOutput { return "Q" }
				if idx == 0 { return "D" }
				if idx == 1 { return "CLK" }
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
	case "and", "nand", "or", "nor", "xor", "xnor", "not", "mux", "dff", "add", "sub":
		return "$" + k
	case "register", "reg":
		return "$dff"
	default:
		// Unknown types render as generic blocks
		return kind
	}
}
