// SPDX-License-Identifier: Apache-2.0
package router

import (
	"math"
	"testing"
)

func TestRoute_OverlapAvoidance(t *testing.T) {
	// We want to route two parallel connections from left to right.
	// Wire 1: (10, 50) -> (300, 50)
	// Wire 2: (10, 70) -> (300, 70)
	// We will force them to use the same horizontal space.
	obstacles := []Obstacle{
		{Name: "OBS", X: 100, Y: 10, W: 100, H: 80}, // Blocks top, forcing wire 1 to go down
	}

	// Route first wire
	path1 := Route(10, 50, DirRight, 300, 50, DirLeft, "Z-shape", obstacles, nil)
	if len(path1) == 0 {
		t.Fatalf("path1 is empty")
	}

	// Convert path1 to segments
	var routedSegments []Segment
	for i := 0; i < len(path1)-1; i++ {
		routedSegments = append(routedSegments, Segment{
			P1: path1[i],
			P2: path1[i+1],
		})
	}

	// Now route the second wire. It starts at (10, 70) and goes to (300, 70).
	// Under normal conditions, it might want to share path1's segments.
	// But it must avoid them!
	path2 := Route(10, 70, DirRight, 300, 70, DirLeft, "Z-shape", obstacles, routedSegments)
	if len(path2) == 0 {
		t.Fatalf("path2 is empty")
	}

	// Verify that no segment in path2 overlaps with any segment in path1
	for i := 0; i < len(path1)-1; i++ {
		p1_1 := path1[i]
		p1_2 := path1[i+1]
		for j := 0; j < len(path2)-1; j++ {
			p2_1 := path2[j]
			p2_2 := path2[j+1]

			if hasOverlap(p1_1, p1_2, p2_1, p2_2) {
				t.Errorf("Overlapping segments found:\nSegment 1: %v -> %v\nSegment 2: %v -> %v", p1_1, p1_2, p2_1, p2_2)
			}
		}
	}
}

func hasOverlap(p1, p2, q1, q2 Point) bool {
	minX1 := math.Min(p1.X, p2.X)
	maxX1 := math.Max(p1.X, p2.X)
	minY1 := math.Min(p1.Y, p2.Y)
	maxY1 := math.Max(p1.Y, p2.Y)

	minX2 := math.Min(q1.X, q2.X)
	maxX2 := math.Max(q1.X, q2.X)
	minY2 := math.Min(q1.Y, q2.Y)
	maxY2 := math.Max(q1.Y, q2.Y)

	if math.Abs(p1.X-p2.X) < 0.1 && math.Abs(q1.X-q2.X) < 0.1 { // Both vertical
		if math.Abs(p1.X-q1.X) < 0.1 { // Same X
			// Check overlap (excluding single-point touch)
			return !(maxY1 <= minY2 || minY1 >= maxY2)
		}
	} else if math.Abs(p1.Y-p2.Y) < 0.1 && math.Abs(q1.Y-q2.Y) < 0.1 { // Both horizontal
		if math.Abs(p1.Y-q1.Y) < 0.1 { // Same Y
			// Check overlap (excluding single-point touch)
			return !(maxX1 <= minX2 || minX1 >= maxX2)
		}
	}
	return false
}
