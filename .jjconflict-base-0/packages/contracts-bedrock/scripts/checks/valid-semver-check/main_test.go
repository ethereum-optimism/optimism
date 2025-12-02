package main

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/solc"
	"github.com/stretchr/testify/assert"
)

func TestAssertValidSemver(t *testing.T) {
	tests := []struct {
		name    string
		version string
		wantErr bool
	}{
		{name: "Valid semver", version: "1.0.0", wantErr: false},
		{name: "Invalid semver", version: "1.0.0-beta.1", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := assertValidSemver(tt.version)
			assert.Equal(t, tt.wantErr, err != nil)
		})
	}
}

func TestGetVersion(t *testing.T) {
	tests := []struct {
		name     string
		artifact *solc.ForgeArtifact
		want     string
		wantErr  bool
	}{
		{
			name: "Semver constant definition found",
			artifact: &solc.ForgeArtifact{
				Ast: solc.Ast{
					Nodes: []solc.AstNode{
						{NodeType: "ContractDefinition", Nodes: []solc.AstNode{
							{NodeType: "VariableDeclaration", Name: "version", Visibility: "public", Mutability: "constant", Value: map[string]interface{}{"value": "1.0.0"}},
						}},
					},
				},
			},
			want:    "1.0.0",
			wantErr: false,
		},
		{
			name: "Semver function definition found",
			artifact: &solc.ForgeArtifact{
				Ast: solc.Ast{
					Nodes: []solc.AstNode{
						{NodeType: "ContractDefinition", Nodes: []solc.AstNode{
							{NodeType: "FunctionDefinition", Name: "version", Visibility: "public", Body: &solc.AstBlock{Statements: []solc.AstNode{{NodeType: "Return", Expression: &solc.Expression{Value: "1.0.0"}}}}},
						}},
					},
				},
			},
			want:    "1.0.0",
			wantErr: false,
		},
		{
			name: "Semver function definition not found",
			artifact: &solc.ForgeArtifact{
				Ast: solc.Ast{
					Nodes: []solc.AstNode{
						{NodeType: "ContractDefinition", Nodes: []solc.AstNode{}},
					},
				},
			},
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getVersion(tt.artifact)
			assert.Equal(t, tt.wantErr, err != nil)
			assert.Equal(t, tt.want, got)
		})
	}
}
