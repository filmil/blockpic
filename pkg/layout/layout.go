// SPDX-License-Identifier: Apache-2.0
package layout

import (
	"github.com/filmil/hodos/pkg/ast"
	"github.com/monkey-works/cassowary"
)

// AutoLayout computes the layout using the Cassowary constraint solver.
func AutoLayout(diag *ast.Diagram, sizeFunc func(ast.Block) (float64, float64)) {
	layoutMap := make(map[string]bool)
	for _, l := range diag.Layout {
		layoutMap[l.Block] = true
	}

	adj := make(map[string][]string)
	inDegree := make(map[string]int)
	blocksToLayout := make(map[string]bool)

	for _, b := range diag.Blocks {
		if !layoutMap[b.Name] {
			blocksToLayout[b.Name] = true
			inDegree[b.Name] = 0
		}
	}

	if len(blocksToLayout) == 0 {
		return
	}

	for _, w := range diag.Wires {
		src := w.Source.Block
		dst := w.Destination.Block
		if blocksToLayout[src] && blocksToLayout[dst] {
			adj[src] = append(adj[src], dst)
			inDegree[dst]++
		}
	}

	layers := make(map[string]int)
	var queue []string

	for node := range blocksToLayout {
		if inDegree[node] == 0 {
			queue = append(queue, node)
			layers[node] = 0
		}
	}

	for len(queue) == 0 && len(blocksToLayout) > 0 {
		for node := range blocksToLayout {
			queue = append(queue, node)
			layers[node] = 0
			break
		}
	}

	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]

		for _, neighbor := range adj[curr] {
			inDegree[neighbor]--
			if layers[curr]+1 > layers[neighbor] {
				layers[neighbor] = layers[curr] + 1
			}
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
		if len(queue) == 0 {
			for node := range blocksToLayout {
				if _, visited := layers[node]; !visited {
					queue = append(queue, node)
					layers[node] = 0
					break
				}
			}
		}
	}

	layerGroups := make(map[int][]string)
	maxLayer := 0
	for node, layer := range layers {
		layerGroups[layer] = append(layerGroups[layer], node)
		if layer > maxLayer {
			maxLayer = layer
		}
	}

	solver := cassowary.NewSolver()

	type BlockVars struct {
		Left, Top, Right, Bottom *cassowary.Param
	}
	vars := make(map[string]BlockVars)

	for _, b := range diag.Blocks {
		if !blocksToLayout[b.Name] {
			continue
		}
		w, h := sizeFunc(b)
		v := BlockVars{
			Left:   cassowary.NewParam(0),
			Top:    cassowary.NewParam(0),
			Right:  cassowary.NewParam(0),
			Bottom: cassowary.NewParam(0),
		}
		vars[b.Name] = v

		// Right = Left + w
		solver.AddConstraint(v.Right.Equals(v.Left.Add(cassowary.CM(w))))
		// Bottom = Top + h
		solver.AddConstraint(v.Bottom.Equals(v.Top.Add(cassowary.CM(h))))
	}

	spacingX := 150.0
	spacingY := 100.0

	// Wire constraints (Left-to-right flow)
	for _, w := range diag.Wires {
		src := w.Source.Block
		dst := w.Destination.Block
		if !blocksToLayout[src] || !blocksToLayout[dst] {
			continue
		}
		vSrc := vars[src]
		vDst := vars[dst]

		// dst.Left >= src.Right + spacingX
		solver.AddConstraint(vDst.Left.GreaterThanOrEqualTo(vSrc.Right.Add(cassowary.CM(spacingX))))
		
		// Weak alignment constraint: dst.Top == src.Top
		alignConstraint := vDst.Top.Equals(vSrc.Top)
		alignConstraint.Priority = cassowary.PriorityWeak
		solver.AddConstraint(alignConstraint)
	}

	// Layer constraints (Non-overlapping within layer)
	for l := 0; l <= maxLayer; l++ {
		nodes := layerGroups[l]
		for i := 0; i < len(nodes)-1; i++ {
			v1 := vars[nodes[i]]
			v2 := vars[nodes[i+1]]
			// v2.Top >= v1.Bottom + spacingY
			solver.AddConstraint(v2.Top.GreaterThanOrEqualTo(v1.Bottom.Add(cassowary.CM(spacingY))))
		}
	}

	// Anchor constraints
	if len(layerGroups[0]) > 0 {
		firstNode := layerGroups[0][0]
		solver.AddConstraint(vars[firstNode].Left.Equals(cassowary.CM(20.0)))
		solver.AddConstraint(vars[firstNode].Top.Equals(cassowary.CM(40.0)))
	}

	solver.FlushUpdates()

	for _, b := range diag.Blocks {
		if !blocksToLayout[b.Name] {
			continue
		}
		v := vars[b.Name]
		diag.Layout = append(diag.Layout, ast.LayoutEntry{
			Block: b.Name,
			X:     v.Left.Value(),
			Y:     v.Top.Value(),
		})
	}
}
