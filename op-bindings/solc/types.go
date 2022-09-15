package solc

import (
	"encoding/json"

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

// TODO(tynes): ignoring devdoc and userdoc for now
type CompilerOutputContract struct {
	Abi           abi.ABI           `json:"abi"`
	Evm           CompilerOutputEvm `json:"evm"`
	Metadata      string            `json:"metadata"`
	StorageLayout StorageLayout     `json:"storageLayout"`
}

type StorageLayout struct {
	Storage []StorageLayoutEntry         `json:"storage"`
	Types   map[string]StorageLayoutType `json:"types"`
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
	Encoding      string `json:"encoding"`
	Label         string `json:"label"`
	NumberOfBytes uint   `json:"numberOfBytes,string"`
	Key           string `json:"key,omitempty"`
	Value         string `json:"value,omitempty"`
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
