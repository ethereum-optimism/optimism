package forge_artifacts

import (
	_ "embed"

	"encoding/json"

	"github.com/ethereum-optimism/optimism/op-bindings/solc"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// Artifact represents a foundry compilation artifact.
// The Abi is specifically left as a json.RawMessage because
// round trip marshaling/unmarshalling of the abi.ABI type
// causes issues.
type Artifact struct {
	Abi              json.RawMessage    `json:"abi"`
	StorageLayout    solc.StorageLayout `json:"storageLayout"`
	DeployedBytecode DeployedBytecode   `json:"deployedBytecode"`
	Bytecode         Bytecode           `json:"bytecode"`
}

type DeployedBytecode struct {
	SourceMap           string          `json:"sourceMap"`
	Object              hexutil.Bytes   `json:"object"`
	LinkReferences      json.RawMessage `json:"linkReferences"`
	ImmutableReferences json.RawMessage `json:"immutableReferences"`
}

type Bytecode struct {
	SourceMap      string          `json:"sourceMap"`
	Object         hexutil.Bytes   `json:"object"`
	LinkReferences json.RawMessage `json:"linkReferences"`
}

//go:embed MIPS.sol/MIPS.json
var mips []byte

//go:embed PreimageOracle.sol/PreimageOracle.json
var preimateOracle []byte

func LoadMIPS() (*Artifact, error) {
	return loadArtifact(mips)
}

func LoadPreimageOracle() (*Artifact, error) {
	return loadArtifact(preimateOracle)
}

func loadArtifact(input []byte) (*Artifact, error) {
	artifact := Artifact{}
	err := json.Unmarshal(input, &artifact)
	if err != nil {
		return nil, err
	}
	return &artifact, nil
}
