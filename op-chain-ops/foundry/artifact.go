package foundry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/holiman/uint256"
	"golang.org/x/exp/maps"

	"github.com/ethereum-optimism/optimism/op-chain-ops/solc"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
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

// artifactMarshaling is a helper struct for marshaling and unmarshalling
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

// Bytecode represents the bytecode section of the solc compiler output.
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

type ForgeAllocs struct {
	Accounts types.GenesisAlloc
}

func (d *ForgeAllocs) Copy() *ForgeAllocs {
	out := make(types.GenesisAlloc, len(d.Accounts))
	maps.Copy(out, d.Accounts)
	return &ForgeAllocs{Accounts: out}
}

func (d *ForgeAllocs) UnmarshalJSON(b []byte) error {
	// forge, since integrating Alloy, likes to hex-encode everything.
	type forgeAllocAccount struct {
		Balance hexutil.U256                `json:"balance"`
		Nonce   hexutil.Uint64              `json:"nonce"`
		Code    hexutil.Bytes               `json:"code,omitempty"`
		Storage map[common.Hash]common.Hash `json:"storage,omitempty"`
	}
	var allocs map[common.Address]forgeAllocAccount
	if err := json.Unmarshal(b, &allocs); err != nil {
		return err
	}
	d.Accounts = make(types.GenesisAlloc, len(allocs))
	for addr, acc := range allocs {
		acc := acc
		d.Accounts[addr] = types.Account{
			Code:       acc.Code,
			Storage:    acc.Storage,
			Balance:    (*uint256.Int)(&acc.Balance).ToBig(),
			Nonce:      (uint64)(acc.Nonce),
			PrivateKey: nil,
		}
	}
	return nil
}

func LoadForgeAllocs(allocsPath string) (*ForgeAllocs, error) {
	path := filepath.Join(allocsPath)
	f, err := os.OpenFile(path, os.O_RDONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open forge allocs %q: %w", path, err)
	}
	defer f.Close()
	var out ForgeAllocs
	if err := json.NewDecoder(f).Decode(&out); err != nil {
		return nil, fmt.Errorf("failed to json-decode forge allocs %q: %w", path, err)
	}
	return &out, nil
}
