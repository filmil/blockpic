// SPDX-License-Identifier: Apache-2.0
package validation

import (
	"testing"
	"github.com/filmil/hodos/pkg/ast"
)

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		diag    *ast.Diagram
		wantErr bool
	}{
		{
			name: "valid diagram",
			diag: &ast.Diagram{
				Blocks: []ast.Block{
					{Name: "B1", Outputs: []ast.Pin{{Name: "O"}}},
					{Name: "B2", Inputs: []ast.Pin{{Name: "I"}}},
				},
				Wires: []ast.Wire{
					{
						Source:      ast.PinRef{Block: "B1", Pin: "O"},
						Destination: ast.PinRef{Block: "B2", Pin: "I"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "duplicate block",
			diag: &ast.Diagram{
				Blocks: []ast.Block{{Name: "B1"}, {Name: "B1"}},
			},
			wantErr: true,
		},
		{
			name: "missing source block",
			diag: &ast.Diagram{
				Blocks: []ast.Block{{Name: "B1"}},
				Wires: []ast.Wire{
					{Source: ast.PinRef{Block: "B2"}, Destination: ast.PinRef{Block: "B1"}},
				},
			},
			wantErr: true,
		},
		{
			name: "missing pin",
			diag: &ast.Diagram{
				Blocks: []ast.Block{
					{Name: "B1", Outputs: []ast.Pin{{Name: "O1"}}},
					{Name: "B2"},
				},
				Wires: []ast.Wire{
					{
						Source:      ast.PinRef{Block: "B1", Pin: "O2"},
						Destination: ast.PinRef{Block: "B2"},
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := Validate(tt.diag); (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
