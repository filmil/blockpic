// SPDX-License-Identifier: Apache-2.0
package validation

import (
	"fmt"
	"github.com/filmil/hodos/pkg/ast"
)

func Validate(diag *ast.Diagram) error {
	blockMap := make(map[string]ast.Block)
	
	// 1. Check for duplicate block labels
	for _, b := range diag.Blocks {
		if _, ok := blockMap[b.Label]; ok {
			return fmt.Errorf("duplicate block label: %s", b.Label)
		}
		blockMap[b.Label] = b
	}

	// 2. Verify wires
	for _, w := range diag.Wires {
		// Source block
		srcBlock, ok := blockMap[w.Source.Block]
		if !ok {
			return fmt.Errorf("wire source block not found: %s", w.Source.Block)
		}
		// Source pin
		if w.Source.Pin != "" {
			if !hasPin(srcBlock.Outputs, w.Source.Pin) && !hasPin(srcBlock.Inputs, w.Source.Pin) {
				return fmt.Errorf("pin %s not found in block %s", w.Source.Pin, w.Source.Block)
			}
		}

		// Destination block
		dstBlock, ok := blockMap[w.Destination.Block]
		if !ok {
			return fmt.Errorf("wire destination block not found: %s", w.Destination.Block)
		}
		// Destination pin
		if w.Destination.Pin != "" {
			if !hasPin(dstBlock.Inputs, w.Destination.Pin) && !hasPin(dstBlock.Outputs, w.Destination.Pin) {
				return fmt.Errorf("pin %s not found in block %s", w.Destination.Pin, w.Destination.Block)
			}
		}
	}

	return nil
}

func hasPin(pins []ast.Pin, name string) bool {
	for _, p := range pins {
		if p.Name == name {
			return true
		}
	}
	return false
}
