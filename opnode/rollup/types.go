package rollup

import (
	"math/big"

	"github.com/ethereum-optimism/optimistic-specs/opnode/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type Genesis struct {
	// The L1 block that the rollup starts *after* (no derived transactions)
	L1 eth.BlockID
	// The L2 block the rollup starts from (no transactions, pre-configured state)
	L2 eth.BlockID
	// Timestamp of L2 block
	L2Time uint64
}

type Config struct {
	// Genesis anchor point of the rollup
	Genesis Genesis
	// Seconds per L2 block
	BlockTime uint64
	// Sequencer batches may not be more than MaxSequencerTimeDiff seconds after
	// the L1 timestamp of the sequencing window end.
	//
	// Note: When L1 has many 1 second consecutive blocks, and L2 grows at fixed 2 seconds,
	// the L2 time may still grow beyond this difference.
	MaxSequencerTimeDiff uint64
	// Number of epochs (L1 blocks) per sequencing window
	SeqWindowSize uint64
	// Required to verify L1 signatures
	L1ChainID *big.Int

	// L2 address receiving all L2 transaction fees
	FeeRecipientAddress common.Address
	// L1 address that batches are sent to
	BatchInboxAddress common.Address
	// Acceptable batch-sender address
	BatchSenderAddress common.Address
}

func (c *Config) L1Signer() types.Signer {
	return types.NewLondonSigner(c.L1ChainID)
}

type Epoch uint64
