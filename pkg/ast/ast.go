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

type RoutingHint string

const (
	HintStraight RoutingHint = "straight"
	HintLShape   RoutingHint = "L-shape"
	HintZShape   RoutingHint = "Z-shape"
)

type PinPosition string

const (
	PositionLeft   PinPosition = "left"
	PositionRight  PinPosition = "right"
	PositionTop    PinPosition = "top"
	PositionBottom PinPosition = "bottom"
	PositionNone   PinPosition = ""
)

type Pin struct {
	Name     string
	Position PinPosition
}

type Wire struct {
	Source      PinRef
	Destination PinRef
	Thickness   int
	Crossline   int
	Hint        RoutingHint
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
