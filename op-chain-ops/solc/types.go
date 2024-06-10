package solc

import (
	"encoding/json"
	"fmt"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

type CompilerInput struct {
	Language string                       `json:"language"`
	Sources  map[string]map[string]string `json:"sources"`
	Settings CompilerSettings             `json:"settings"`
}

type CompilerSettings struct {
	Optimizer       OptimizerSettings              `json:"optimizer"`
	Metadata        CompilerInputMetadata          `json:"metadata"`
	OutputSelection map[string]map[string][]string `json:"outputSelection"`
	EvmVersion      string                         `json:"evmVersion,omitempty"`
	Libraries       map[string]map[string]string   `json:"libraries,omitempty"`
}

type OptimizerSettings struct {
	Enabled bool `json:"enabled"`
	Runs    uint `json:"runs"`
}

type CompilerInputMetadata struct {
	UseLiteralContent bool `json:"useLiteralContent"`
}

type CompilerOutput struct {
	Contracts map[string]CompilerOutputContracts `json:"contracts"`
	Sources   CompilerOutputSources              `json:"sources"`
}

type CompilerOutputContracts map[string]CompilerOutputContract

// CompilerOutputContract represents the solc compiler output for a contract.
// Ignoring some fields such as devdoc and userdoc.
type CompilerOutputContract struct {
	Abi           abi.ABI           `json:"abi"`
	Evm           CompilerOutputEvm `json:"evm"`
	Metadata      string            `json:"metadata"`
	StorageLayout StorageLayout     `json:"storageLayout"`
}

// StorageLayout represents the solc compilers output storage layout for
// a contract.
type StorageLayout struct {
	Storage []StorageLayoutEntry         `json:"storage"`
	Types   map[string]StorageLayoutType `json:"types"`
}

// GetStorageLayoutEntry returns the StorageLayoutEntry where the label matches
// the provided name.
func (s *StorageLayout) GetStorageLayoutEntry(name string) (StorageLayoutEntry, error) {
	for _, entry := range s.Storage {
		if entry.Label == name {
			return entry, nil
		}
	}
	return StorageLayoutEntry{}, fmt.Errorf("%s not found", name)
}

// GetStorageLayoutType returns the StorageLayoutType where the label matches
// the provided name.
func (s *StorageLayout) GetStorageLayoutType(name string) (StorageLayoutType, error) {
	if ty, ok := s.Types[name]; ok {
		return ty, nil
	}
	return StorageLayoutType{}, fmt.Errorf("%s not found", name)
}

type StorageLayoutEntry struct {
	AstId    uint   `json:"astId"`
	Contract string `json:"contract"`
	Label    string `json:"label"`
	Offset   uint   `json:"offset"`
	Slot     uint   `json:"slot,string"`
	Type     string `json:"type"`
}

type StorageLayoutType struct {
	Encoding      string               `json:"encoding"`
	Label         string               `json:"label"`
	NumberOfBytes uint                 `json:"numberOfBytes,string"`
	Key           string               `json:"key,omitempty"`
	Value         string               `json:"value,omitempty"`
	Base          string               `json:"base,omitempty"`
	Members       []StorageLayoutEntry `json:"members,omitempty"`
}

type CompilerOutputEvm struct {
	Bytecode          CompilerOutputBytecode       `json:"bytecode"`
	DeployedBytecode  CompilerOutputBytecode       `json:"deployedBytecode"`
	GasEstimates      map[string]map[string]string `json:"gasEstimates"`
	MethodIdentifiers map[string]string            `json:"methodIdentifiers"`
}

// Object must be a string because its not guaranteed to be
// a hex string
type CompilerOutputBytecode struct {
	Object         string         `json:"object"`
	Opcodes        string         `json:"opcodes"`
	SourceMap      string         `json:"sourceMap"`
	LinkReferences LinkReferences `json:"linkReferences"`
}

type LinkReferences map[string]LinkReference
type LinkReference map[string][]LinkReferenceOffset

type LinkReferenceOffset struct {
	Length uint `json:"length"`
	Start  uint `json:"start"`
}

type CompilerOutputSources map[string]CompilerOutputSource

type CompilerOutputSource struct {
	Id  uint `json:"id"`
	Ast Ast  `json:"ast"`
}

type Ast struct {
	AbsolutePath    string            `json:"absolutePath"`
	ExportedSymbols map[string][]uint `json:"exportedSymbols"`
	Id              uint              `json:"id"`
	License         string            `json:"license"`
	NodeType        string            `json:"nodeType"`
	Nodes           json.RawMessage   `json:"nodes"`
}
