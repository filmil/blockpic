// SPDX-License-Identifier: Apache-2.0
package router

type Point struct {
	X, Y float64
}

func Route(x1, y1, x2, y2 float64, hint string) []Point {
	switch hint {
	case "straight":
		return []Point{{x1, y1}, {x2, y2}}
	case "L-shape":
		return []Point{{x1, y1}, {x2, y1}, {x2, y2}}
	case "Z-shape":
		midX := (x1 + x2) / 2
		return []Point{{x1, y1}, {midX, y1}, {midX, y2}, {x2, y2}}
	default:
		// Default to Z-shape for orthogonal flow
		midX := (x1 + x2) / 2
		return []Point{{x1, y1}, {midX, y1}, {midX, y2}, {x2, y2}}
	}
}
