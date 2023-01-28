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
	// SYSCOIN
	SyscoinNode sources.SyscoinClient

	// Where to send the batch txs to.
	BatchInboxAddress common.Address

	// Channel creation parameters
	Channel ChannelConfig

	// Chain ID of the L1 chain to submit txs to.
	ChainID *big.Int

	PollInterval time.Duration
}
