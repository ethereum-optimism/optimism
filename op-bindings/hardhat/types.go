package hardhat

import (
	"encoding/json"

	"github.com/ethereum-optimism/optimism/op-bindings/solc"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// Deployment represents a hardhat-deploy artifact file
type Deployment struct {
	Name             string
	Abi              abi.ABI            `json:"abi"`
	Address          common.Address     `json:"address"`
	Args             []any              `json:"args"`
	Bytecode         hexutil.Bytes      `json:"bytecode"`
	DeployedBytecode hexutil.Bytes      `json:"deployedBytecode"`
	Devdoc           json.RawMessage    `json:"devdoc"`
	Metadata         string             `json:"metadata"`
	Receipt          json.RawMessage    `json:"receipt"`
	SolcInputHash    string             `json:"solcInputHash"`
	StorageLayout    solc.StorageLayout `json:"storageLayout"`
	TransactionHash  common.Hash        `json:"transactionHash"`
	Userdoc          json.RawMessage    `json:"userdoc"`
}

// Receipt represents the receipt held in a hardhat-deploy
// artifact file
type Receipt struct {
	To                *common.Address `json:"to"`
	From              common.Address  `json:"from"`
	ContractAddress   *common.Address `json:"contractAddress"`
	TransactionIndex  uint            `json:"transactionIndex"`
	GasUsed           uint            `json:"gasUsed,string"`
	LogsBloom         hexutil.Bytes   `json:"logsBloom"`
	BlockHash         common.Hash     `json:"blockHash"`
	TransactionHash   common.Hash     `json:"transactionHash"`
	Logs              []Log           `json:"logs"`
	BlockNumber       uint            `json:"blockNumber"`
	CumulativeGasUsed uint            `json:"cumulativeGasUsed,string"`
	Status            uint            `json:"status"`
	Byzantium         bool            `json:"byzantium"`
}

// Log represents the logs in the hardhat deploy artifact receipt
type Log struct {
	TransactionIndex uint           `json:"transactionIndex"`
	BlockNumber      uint           `json:"blockNumber"`
	TransactionHash  common.Hash    `json:"transactionHash"`
	Address          common.Address `json:"address"`
	Topics           []common.Hash  `json:"topics"`
	Data             hexutil.Bytes  `json:"data"`
	LogIndex         uint           `json:"logIndex"`
	Blockhash        common.Hash    `json:"blockHash"`
}

// Artifact represents a hardhat compilation artifact
// The Bytecode and DeployedBytecode are not guaranteed
// to be hexutil.Bytes when there are link references.
// In the future, custom json marshalling can be used
// to place the link reference values in the correct location.
type Artifact struct {
	Format                 string         `json:"_format"`
	ContractName           string         `json:"contractName"`
	SourceName             string         `json:"sourceName"`
	Abi                    abi.ABI        `json:"abi"`
	Bytecode               hexutil.Bytes  `json:"bytecode"`
	DeployedBytecode       hexutil.Bytes  `json:"deployedBytecode"`
	LinkReferences         LinkReferences `json:"linkReferences"`
	DeployedLinkReferences LinkReferences `json:"deployedLinkReferences"`
}

// LinkReferences represents the linked contracts
type LinkReferences map[string]LinkReference

// LinkReference represents a single linked contract
type LinkReference map[string][]LinkReferenceOffset

// LinkReferenceOffset represents the offsets in a link reference
type LinkReferenceOffset struct {
	Length uint `json:"length"`
	Start  uint `json:"start"`
}

// DebugFile represents the debug file that contains the path
// to the build info file
type DebugFile struct {
	Format    string `json:"_format"`
	BuildInfo string `json:"buildInfo"`
}

// BuildInfo represents a hardhat build info artifact that is created
// after compilation
type BuildInfo struct {
	Format          string              `json:"_format"`
	Id              string              `json:"id"`
	SolcVersion     string              `json:"solcVersion"`
	SolcLongVersion string              `json:"solcLongVersion"`
	Input           solc.CompilerInput  `json:"input"`
	Output          solc.CompilerOutput `json:"output"`
}
