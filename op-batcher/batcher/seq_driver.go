package batcher

import (
	"math/big"
	"time"

	"github.com/ethereum-optimism/optimism/op-node/sources"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
)

type DriverConfig struct {
	Log  log.Logger
	Name string

	// API to submit txs to
	L1Client *ethclient.Client

	// API to hit for batch data
	L2Client *ethclient.Client

	RollupNode *sources.RollupClient

	// Limit the size of txs
	MinL1TxSize uint64
	MaxL1TxSize uint64

	// Where to send the batch txs to.
	BatchInboxAddress common.Address

	// The batcher can decide to set it shorter than the actual timeout,
	//  since submitting continued channel data to L1 is not instantaneous.
	//  It's not worth it to work with nearly timed-out channels.
	ChannelTimeout uint64

	// Chain ID of the L1 chain to submit txs to.
	ChainID *big.Int

	PollInterval time.Duration
}
