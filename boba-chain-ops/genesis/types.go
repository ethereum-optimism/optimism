package genesis

import (
	"fmt"
	"math/big"
	"reflect"

	"github.com/ledgerwatch/erigon-lib/chain"
	"github.com/ledgerwatch/erigon-lib/common"
	"github.com/ledgerwatch/erigon-lib/common/hexutil"
	"github.com/ledgerwatch/erigon-lib/common/hexutility"
	"github.com/ledgerwatch/erigon/core/types"
)

type Genesis struct {
	// The L1 block that the rollup starts *after* (no derived transactions)
	L1 BlockID `json:"l1"`
	// The L2 block the rollup starts from (no transactions, pre-configured state)
	L2 BlockID `json:"l2"`
	// Timestamp of L2 block
	L2Time uint64 `json:"l2_time"`
	// Initial system configuration values.
	// The L2 genesis block may not include transactions, and thus cannot encode the config values,
	// unlike later L2 blocks.
	SystemConfig SystemConfig `json:"system_config"`
}

type Config struct {
	// Genesis anchor point of the rollup
	Genesis Genesis `json:"genesis"`
	// Seconds per L2 block
	BlockTime uint64 `json:"block_time"`
	// Sequencer batches may not be more than MaxSequencerDrift seconds after
	// the L1 timestamp of the sequencing window end.
	//
	// Note: When L1 has many 1 second consecutive blocks, and L2 grows at fixed 2 seconds,
	// the L2 time may still grow beyond this difference.
	MaxSequencerDrift uint64 `json:"max_sequencer_drift"`
	// Number of epochs (L1 blocks) per sequencing window, including the epoch L1 origin block itself
	SeqWindowSize uint64 `json:"seq_window_size"`
	// Number of L1 blocks between when a channel can be opened and when it must be closed by.
	ChannelTimeout uint64 `json:"channel_timeout"`
	// Required to verify L1 signatures
	L1ChainID *big.Int `json:"l1_chain_id"`
	// Required to identify the L2 network and create p2p signatures unique for this chain.
	L2ChainID *big.Int `json:"l2_chain_id"`

	// RegolithTime sets the activation time of the Regolith network-upgrade:
	// a pre-mainnet Bedrock change that addresses findings of the Sherlock contest related to deposit attributes.
	// "Regolith" is the loose deposited rock that sits on top of Bedrock.
	// Active if RegolithTime != nil && L2 block timestamp >= *RegolithTime, inactive otherwise.
	RegolithTime *uint64 `json:"regolith_time,omitempty"`

	// CanyonTime sets the activation time of the Canyon network upgrade.
	// Active if CanyonTime != nil && L2 block timestamp >= *CanyonTime, inactive otherwise.
	CanyonTime *uint64 `json:"canyon_time,omitempty"`

	// Note: below addresses are part of the block-derivation process,
	// and required to be the same network-wide to stay in consensus.

	// L1 address that batches are sent to.
	BatchInboxAddress common.Address `json:"batch_inbox_address"`
	// L1 Deposit Contract Address
	DepositContractAddress common.Address `json:"deposit_contract_address"`
	// L1 System Config Address
	L1SystemConfigAddress common.Address `json:"l1_system_config_address"`
}

type BlockID struct {
	Hash   common.Hash `json:"hash"`
	Number uint64      `json:"number"`
}

func (id BlockID) String() string {
	return fmt.Sprintf("%s:%d", id.Hash.String(), id.Number)
}

// TerminalString implements log.TerminalStringer, formatting a string for console
// output during logging.
func (id BlockID) TerminalString() string {
	return fmt.Sprintf("%s:%d", id.Hash.TerminalString(), id.Number)
}

type L2BlockRef struct {
	Hash           common.Hash `json:"hash"`
	Number         uint64      `json:"number"`
	ParentHash     common.Hash `json:"parentHash"`
	Time           uint64      `json:"timestamp"`
	L1Origin       BlockID     `json:"l1origin"`
	SequenceNumber uint64      `json:"sequenceNumber"` // distance to first block of epoch
}

func (id L2BlockRef) String() string {
	return fmt.Sprintf("%s:%d", id.Hash.String(), id.Number)
}

// TerminalString implements log.TerminalStringer, formatting a string for console
// output during logging.
func (id L2BlockRef) TerminalString() string {
	return fmt.Sprintf("%s:%d", id.Hash.TerminalString(), id.Number)
}

type L1BlockRef struct {
	Hash       common.Hash `json:"hash"`
	Number     uint64      `json:"number"`
	ParentHash common.Hash `json:"parentHash"`
	Time       uint64      `json:"timestamp"`
}

func (id L1BlockRef) String() string {
	return fmt.Sprintf("%s:%d", id.Hash.String(), id.Number)
}

// TerminalString implements log.TerminalStringer, formatting a string for console
// output during logging.
func (id L1BlockRef) TerminalString() string {
	return fmt.Sprintf("%s:%d", id.Hash.TerminalString(), id.Number)
}

func (id L1BlockRef) ID() BlockID {
	return BlockID{
		Hash:   id.Hash,
		Number: id.Number,
	}
}

func (id L1BlockRef) ParentID() BlockID {
	n := id.ID().Number
	// Saturate at 0 with subtraction
	if n > 0 {
		n -= 1
	}
	return BlockID{
		Hash:   id.ParentHash,
		Number: n,
	}
}

func (id L2BlockRef) ID() BlockID {
	return BlockID{
		Hash:   id.Hash,
		Number: id.Number,
	}
}

func (id L2BlockRef) ParentID() BlockID {
	n := id.ID().Number
	// Saturate at 0 with subtraction
	if n > 0 {
		n -= 1
	}
	return BlockID{
		Hash:   id.ParentHash,
		Number: n,
	}
}

// SystemConfig represents the rollup system configuration that carries over in every L2 block,
// and may be changed through L1 system config events.
// The initial SystemConfig at rollup genesis is embedded in the rollup configuration.
type SystemConfig struct {
	// BatcherAddr identifies the batch-sender address used in batch-inbox data-transaction filtering.
	BatcherAddr common.Address `json:"batcherAddr"`
	// Overhead identifies the L1 fee overhead, and is passed through opaquely to op-geth.
	Overhead Bytes32 `json:"overhead"`
	// Scalar identifies the L1 fee scalar, and is passed through opaquely to op-geth.
	Scalar Bytes32 `json:"scalar"`
	// GasLimit identifies the L2 block gas limit
	GasLimit uint64 `json:"gasLimit"`
	// More fields can be added for future SystemConfig versions.
}

type Bytes32 [32]byte

func (b *Bytes32) UnmarshalJSON(text []byte) error {
	return hexutility.UnmarshalFixedJSON(reflect.TypeOf(b), text, b[:])
}

func (b *Bytes32) UnmarshalText(text []byte) error {
	return hexutility.UnmarshalFixedText("Bytes32", text, b[:])
}

func (b Bytes32) MarshalText() ([]byte, error) {
	return hexutility.Bytes(b[:]).MarshalText()
}

func (b Bytes32) String() string {
	return hexutility.Encode(b[:])
}

// TerminalString implements log.TerminalStringer, formatting a string for console
// output during logging.
func (b Bytes32) TerminalString() string {
	return fmt.Sprintf("%x..%x", b[:3], b[29:])
}

// The struct type of genesis file is different from Genesis in types.go
// It doesn't have the following fields:
// AuRaStep 	uint64         `json:"auRaStep"`
// AuRaSeal 	[]byte         `json:"auRaSeal"`
type GenesisOutput struct {
	Config        *chain.Config      `json:"config"`
	Nonce         hexutil.Uint64     `json:"nonce"`
	Timestamp     hexutil.Uint64     `json:"timestamp"`
	ExtraData     hexutility.Bytes   `json:"extraData"`
	GasLimit      hexutil.Uint64     `json:"gasLimit"   gencodec:"required"`
	Difficulty    *hexutil.Big       `json:"difficulty" gencodec:"required"`
	Mixhash       common.Hash        `json:"mixHash"`
	Coinbase      common.Address     `json:"coinbase"`
	BaseFee       *hexutil.Big       `json:"baseFeePerGas"`
	ExcessDataGas *hexutil.Big       `json:"excessDataGas"`
	Alloc         types.GenesisAlloc `json:"alloc"      gencodec:"required"`

	// These fields are used for consensus tests. Please don't use them
	// in actual genesis blocks.
	Number     hexutil.Uint64 `json:"number"`
	GasUsed    hexutil.Uint64 `json:"gasUsed"`
	ParentHash common.Hash    `json:"parentHash"`
}

func (g GenesisOutput) PerformOutput(genesis *types.Genesis) GenesisOutput {
	var excessDataGas *hexutil.Big
	if genesis.ExcessBlobGas != nil {
		excessDataGas = (*hexutil.Big)(big.NewInt(int64(*genesis.ExcessBlobGas)))
	}
	return GenesisOutput{
		Config:        genesis.Config,
		Nonce:         hexutil.Uint64(genesis.Nonce),
		Timestamp:     hexutil.Uint64(genesis.Timestamp),
		ExtraData:     genesis.ExtraData,
		GasLimit:      hexutil.Uint64(genesis.GasLimit),
		Difficulty:    (*hexutil.Big)(genesis.Difficulty),
		Mixhash:       genesis.Mixhash,
		Coinbase:      genesis.Coinbase,
		BaseFee:       (*hexutil.Big)(genesis.BaseFee),
		ExcessDataGas: excessDataGas,
		Alloc:         genesis.Alloc,

		Number:     hexutil.Uint64(genesis.Number),
		GasUsed:    hexutil.Uint64(genesis.GasUsed),
		ParentHash: genesis.ParentHash,
	}
}
