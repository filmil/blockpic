// SPDX-License-Identifier: Apache-2.0
package layout

import (
	"github.com/filmil/hodos/pkg/ast"
)

func AutoLayout(diag *ast.Diagram) {
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

	spacingX := 200.0
	spacingY := 150.0
	offsetX := 20.0
	offsetY := 40.0

	for l := 0; l <= maxLayer; l++ {
		nodes := layerGroups[l]
		for i, node := range nodes {
			diag.Layout = append(diag.Layout, ast.LayoutEntry{
				Block: node,
				X:     offsetX + float64(l)*spacingX,
				Y:     offsetY + float64(i)*spacingY,
			})
		}
	}
}
