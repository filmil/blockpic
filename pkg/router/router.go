// SPDX-License-Identifier: Apache-2.0
package router

import (
	"errors"
	"math"
	"sort"
	"github.com/filmil/hodos/pkg/ast"
)

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

type Obstacle struct {
	Name       string
	X, Y, W, H float64
}

type Segment struct {
	P1, P2 Point
}

// Route calculates orthogonal paths between two points, avoiding obstacles.
func Route(x1, y1 float64, d1 Direction, x2, y2 float64, d2 Direction, hint ast.RoutingHint, obstacles []Obstacle, routedSegments []Segment) []Point {
	p1 := Point{x1, y1}
	p2 := Point{x2, y2}

	if hint == ast.HintStraight {
		return []Point{p1, p2}
	}

	stubLen := 16.0

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
		exitP.X += stubLen
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
		enterP.X -= stubLen
	}

	// Try A* routing first
	path, err := routeAStar(exitP, enterP, obstacles, routedSegments)
	if err == nil {
		// Prepend start point and append end point
		fullPath := append([]Point{p1}, path...)
		fullPath = append(fullPath, p2)
		return cleanConsecutiveDuplicates(fullPath)
	}

	// Fallback to Z-shape default
	var fallbackPath []Point
	fallbackPath = append(fallbackPath, p1, exitP)
	if hint == ast.HintLShape {
		fallbackPath = append(fallbackPath, Point{exitP.X, enterP.Y})
	} else {
		midX := (exitP.X + enterP.X) / 2
		fallbackPath = append(fallbackPath, Point{midX, exitP.Y}, Point{midX, enterP.Y})
	}
	fallbackPath = append(fallbackPath, enterP, p2)

	return cleanConsecutiveDuplicates(fallbackPath)
}

func cleanConsecutiveDuplicates(path []Point) []Point {
	var cleanPath []Point
	for i, p := range path {
		if i == 0 || math.Abs(p.X-path[i-1].X) > 0.1 || math.Abs(p.Y-path[i-1].Y) > 0.1 {
			cleanPath = append(cleanPath, p)
		}
	}
	return cleanPath
}

func routeAStar(start, end Point, obstacles []Obstacle, routedSegments []Segment) ([]Point, error) {
	margin := 12.0

	// 1. Gather all coordinate candidates to build the routing grid
	xs := []float64{start.X, end.X}
	ys := []float64{start.Y, end.Y}

	for _, obs := range obstacles {
		xs = append(xs, obs.X-margin, obs.X+obs.W+margin)
		ys = append(ys, obs.Y-margin, obs.Y+obs.H+margin)
	}

	for _, seg := range routedSegments {
		if math.Abs(seg.P1.X-seg.P2.X) < 0.1 { // Vertical segment
			xs = append(xs, seg.P1.X-8.0, seg.P1.X+8.0)
		} else if math.Abs(seg.P1.Y-seg.P2.Y) < 0.1 { // Horizontal segment
			ys = append(ys, seg.P1.Y-8.0, seg.P1.Y+8.0)
		}
	}

	xs = uniqueSorted(xs)
	ys = uniqueSorted(ys)

	// Map coordinates to grid indices
	findXIdx := func(x float64) int {
		for i, v := range xs {
			if math.Abs(v-x) < 0.1 {
				return i
			}
		}
		return -1
	}
	findYIdx := func(y float64) int {
		for i, v := range ys {
			if math.Abs(v-y) < 0.1 {
				return i
			}
		}
		return -1
	}

	startIX := findXIdx(start.X)
	startIY := findYIdx(start.Y)
	endIX := findXIdx(end.X)
	endIY := findYIdx(end.Y)

	if startIX == -1 || startIY == -1 || endIX == -1 || endIY == -1 {
		return nil, errors.New("coordinate mapping failed")
	}

	// 2. A* algorithm definitions
	type astarNode struct {
		ix, iy int
		gScore float64
		fScore float64
		parent *astarNode
		dir    Direction
	}

	manhattanDist := func(x1, y1, x2, y2 float64) float64 {
		return math.Abs(x1-x2) + math.Abs(y1-y2)
	}

	// Open set as a list (since grid is small, linear scan is fine)
	openSet := []*astarNode{
		{
			ix:     startIX,
			iy:     startIY,
			gScore: 0,
			fScore: manhattanDist(start.X, start.Y, end.X, end.Y),
			dir:    DirNone,
		},
	}

	// Track closed set
	closedSet := make(map[int]map[int]bool)

	for len(openSet) > 0 {
		// Get node with lowest fScore
		bestIdx := 0
		for i, node := range openSet {
			if node.fScore < openSet[bestIdx].fScore {
				bestIdx = i
			}
		}

		curr := openSet[bestIdx]
		openSet = append(openSet[:bestIdx], openSet[bestIdx+1:]...)

		if curr.ix == endIX && curr.iy == endIY {
			// Reconstruct path
			var path []Point
			p := curr
			for p != nil {
				path = append([]Point{{xs[p.ix], ys[p.iy]}}, path...)
				p = p.parent
			}
			return path, nil
		}

		if closedSet[curr.ix] == nil {
			closedSet[curr.ix] = make(map[int]bool)
		}
		closedSet[curr.ix][curr.iy] = true

		// Check neighbors
		dirs := []struct {
			dx, dy int
			dir    Direction
		}{
			{1, 0, DirRight},
			{-1, 0, DirLeft},
			{0, 1, DirBottom},
			{0, -1, DirTop},
		}

		for _, d := range dirs {
			nix := curr.ix + d.dx
			niy := curr.iy + d.dy

			if nix < 0 || nix >= len(xs) || niy < 0 || niy >= len(ys) {
				continue
			}

			if closedSet[nix] != nil && closedSet[nix][niy] {
				continue
			}

			cx, cy := xs[curr.ix], ys[curr.iy]
			nx, ny := xs[nix], ys[niy]

			// Collision check
			if intersectsObstacles(cx, cy, nx, ny, obstacles, margin) {
				continue
			}
			if intersectsSegments(cx, cy, nx, ny, routedSegments) {
				continue
			}

			// Calculate cost
			stepCost := manhattanDist(cx, cy, nx, ny)
			turnPenalty := 0.0
			if curr.dir != DirNone && curr.dir != d.dir {
				turnPenalty = 80.0 // penalize changes in direction to keep lines straight
			}

			gScore := curr.gScore + stepCost + turnPenalty

			// Check if already in openSet
			var existing *astarNode
			for _, node := range openSet {
				if node.ix == nix && node.iy == niy {
					existing = node
					break
				}
			}

			if existing == nil {
				newNode := &astarNode{
					ix:     nix,
					iy:     niy,
					gScore: gScore,
					fScore: gScore + manhattanDist(nx, ny, end.X, end.Y),
					parent: curr,
					dir:    d.dir,
				}
				openSet = append(openSet, newNode)
			} else if gScore < existing.gScore {
				existing.gScore = gScore
				existing.fScore = gScore + manhattanDist(nx, ny, end.X, end.Y)
				existing.parent = curr
				existing.dir = d.dir
			}
		}
	}

	return nil, errors.New("path not found")
}

func uniqueSorted(arr []float64) []float64 {
	if len(arr) == 0 {
		return arr
	}
	sort.Float64s(arr)
	res := []float64{arr[0]}
	for i := 1; i < len(arr); i++ {
		if math.Abs(arr[i]-arr[i-1]) > 0.1 {
			res = append(res, arr[i])
		}
	}
	return res
}

func intersectsObstacles(x1, y1, x2, y2 float64, obstacles []Obstacle, margin float64) bool {
	minX := math.Min(x1, x2)
	maxX := math.Max(x1, x2)
	minY := math.Min(y1, y2)
	maxY := math.Max(y1, y2)

	for _, obs := range obstacles {
		// Define the obstacle box with margin
		obsLeft := obs.X - margin
		obsRight := obs.X + obs.W + margin
		obsTop := obs.Y - margin
		obsBottom := obs.Y + obs.H + margin

		if x1 == x2 { // Vertical segment
			if x1 > obsLeft && x1 < obsRight {
				// Check if the Y interval overlaps
				if !(maxY <= obsTop || minY >= obsBottom) {
					return true
				}
			}
		} else if y1 == y2 { // Horizontal segment
			if y1 > obsTop && y1 < obsBottom {
				// Check if the X interval overlaps
				if !(maxX <= obsLeft || minX >= obsRight) {
					return true
				}
			}
		}
	}
	return false
}

func intersectsSegments(x1, y1, x2, y2 float64, segments []Segment) bool {
	minX := math.Min(x1, x2)
	maxX := math.Max(x1, x2)
	minY := math.Min(y1, y2)
	maxY := math.Max(y1, y2)

	for _, seg := range segments {
		sMinX := math.Min(seg.P1.X, seg.P2.X)
		sMaxX := math.Max(seg.P1.X, seg.P2.X)
		sMinY := math.Min(seg.P1.Y, seg.P2.Y)
		sMaxY := math.Max(seg.P1.Y, seg.P2.Y)

		if math.Abs(x1-x2) < 0.1 && math.Abs(seg.P1.X-seg.P2.X) < 0.1 { // Both are vertical
			if math.Abs(x1-seg.P1.X) < 0.1 { // On the same X coordinate
				// Check if Y intervals overlap
				if !(maxY <= sMinY || minY >= sMaxY) {
					return true
				}
			}
		} else if math.Abs(y1-y2) < 0.1 && math.Abs(seg.P1.Y-seg.P2.Y) < 0.1 { // Both are horizontal
			if math.Abs(y1-seg.P1.Y) < 0.1 { // On the same Y coordinate
				// Check if X intervals overlap
				if !(maxX <= sMinX || minX >= sMaxX) {
					return true
				}
			}
		}
	}
	return false
}
