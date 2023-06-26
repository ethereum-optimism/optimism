package foundry

import (
	"encoding/json"

	"github.com/ethereum-optimism/optimism/op-bindings/solc"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

type Artifact struct {
	Abi              abi.ABI            `json:"abi"`
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
