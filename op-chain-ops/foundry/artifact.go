package foundry

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/ethereum-optimism/optimism/op-chain-ops/solc"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// Artifact represents a foundry compilation artifact.
// JSON marshaling logic is implemented to maintain the ability
// to roundtrip serialize an artifact
type Artifact struct {
	ABI              abi.ABI
	abi              json.RawMessage
	StorageLayout    solc.StorageLayout
	DeployedBytecode DeployedBytecode
	Bytecode         Bytecode
}

func (a *Artifact) UnmarshalJSON(data []byte) error {
	artifact := artifactMarshaling{}
	if err := json.Unmarshal(data, &artifact); err != nil {
		return err
	}
	parsed, err := abi.JSON(strings.NewReader(string(artifact.ABI)))
	if err != nil {
		return err
	}
	a.ABI = parsed
	a.abi = artifact.ABI
	a.StorageLayout = artifact.StorageLayout
	a.DeployedBytecode = artifact.DeployedBytecode
	a.Bytecode = artifact.Bytecode
	return nil
}

func (a Artifact) MarshalJSON() ([]byte, error) {
	artifact := artifactMarshaling{
		ABI:              a.abi,
		StorageLayout:    a.StorageLayout,
		DeployedBytecode: a.DeployedBytecode,
		Bytecode:         a.Bytecode,
	}
	return json.Marshal(artifact)
}

// artifactMarshaling is a helper struct for marshaling and unmarshaling
// foundry artifacts.
type artifactMarshaling struct {
	ABI              json.RawMessage    `json:"abi"`
	StorageLayout    solc.StorageLayout `json:"storageLayout"`
	DeployedBytecode DeployedBytecode   `json:"deployedBytecode"`
	Bytecode         Bytecode           `json:"bytecode"`
}

// DeployedBytecode represents the deployed bytecode section of the solc compiler output.
type DeployedBytecode struct {
	SourceMap           string          `json:"sourceMap"`
	Object              hexutil.Bytes   `json:"object"`
	LinkReferences      json.RawMessage `json:"linkReferences"`
	ImmutableReferences json.RawMessage `json:"immutableReferences,omitempty"`
}

// DeployedBytecode represents the bytecode section of the solc compiler output.
type Bytecode struct {
	SourceMap           string          `json:"sourceMap"`
	Object              hexutil.Bytes   `json:"object"`
	LinkReferences      json.RawMessage `json:"linkReferences"`
	ImmutableReferences json.RawMessage `json:"immutableReferences,omitempty"`
}

// ReadArtifact will read an artifact from disk given a path.
func ReadArtifact(path string) (*Artifact, error) {
	file, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("artifact at %s not found: %w", path, err)
	}
	artifact := Artifact{}
	if err := json.Unmarshal(file, &artifact); err != nil {
		return nil, err
	}
	return &artifact, nil
}
