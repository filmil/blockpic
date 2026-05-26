// SPDX-License-Identifier: Apache-2.0
package router

import "math"

type Point struct {
	X, Y float64
}

// Direction represents the side of the block the pin is on.
type Direction string

const (
	DirRight  Direction = "right"
	DirLeft   Direction = "left"
	DirTop    Direction = "top"
	DirBottom Direction = "bottom"
	DirNone   Direction = ""
)

// Route calculates orthogonal paths between two points, considering their exit/enter directions.
func Route(x1, y1 float64, d1 Direction, x2, y2 float64, d2 Direction, hint string) []Point {
	// Simple straight line fallback
	if hint == "straight" {
		return []Point{{x1, y1}, {x2, y2}}
	}

	// Always add a small stub to ensure the wire exits perpendicular to the pin
	stubLen := 10.0
	
	p1 := Point{x1, y1}
	p2 := Point{x2, y2}

	// Calculate exit point
	exitP := p1
	switch d1 {
	case DirRight:
		exitP.X += stubLen
	case DirLeft:
		exitP.X -= stubLen
	case DirTop:
		exitP.Y -= stubLen
	case DirBottom:
		exitP.Y += stubLen
	default:
		exitP.X += stubLen // Default assume exit right
	}

	// Calculate enter point
	enterP := p2
	switch d2 {
	case DirRight:
		enterP.X += stubLen
	case DirLeft:
		enterP.X -= stubLen
	case DirTop:
		enterP.Y -= stubLen
	case DirBottom:
		enterP.Y += stubLen
	default:
		enterP.X -= stubLen // Default assume enter left
	}

	// Now route between exitP and enterP
	var path []Point
	path = append(path, p1, exitP)

	if hint == "L-shape" {
		path = append(path, Point{exitP.X, enterP.Y})
	} else {
		// Z-shape default
		midX := (exitP.X + enterP.X) / 2
		path = append(path, Point{midX, exitP.Y}, Point{midX, enterP.Y})
	}

	path = append(path, enterP, p2)

	// Clean up duplicate consecutive points
	var cleanPath []Point
	for i, p := range path {
		if i == 0 || math.Abs(p.X-path[i-1].X) > 0.1 || math.Abs(p.Y-path[i-1].Y) > 0.1 {
			cleanPath = append(cleanPath, p)
		}
	}

	return cleanPath
}
