// SPDX-License-Identifier: Apache-2.0
package ast

type Diagram struct {
	Name   string
	Blocks []Block
	Wires  []Wire
	Layout []LayoutEntry
}

type BlockHint string

const (
	HintShowPorts BlockHint = "show-ports"
	HintHidePorts BlockHint = "hide-ports"
	HintNone      BlockHint = ""
)

type Block struct {
	Label   string
	Name    string
	Kind    string
	Params  map[string]string
	Inputs  []Pin
	Outputs []Pin
	Size    *Size
	Hint    BlockHint
}

type Pin struct {
	Name     string
	Position string // left, right, top, bottom
}

type Wire struct {
	Source      PinRef
	Destination PinRef
	Thickness   int
	Crossline   int
	Hint        string
}

type PinRef struct {
	Block string
	Pin   string
}

type LayoutEntry struct {
	Block string
	X, Y  float64
}

type Size struct {
	W, H float64
}
