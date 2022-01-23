package rollup

import (
	"math/big"
	"time"

	"github.com/ethereum-optimism/optimism/l2geth/common"
)

type Config struct {
	// Maximum calldata size for a Queue Origin Sequencer Tx
	MaxCallDataSize int
	// Verifier mode
	IsVerifier bool
	// Enable the sync service
	Eth1SyncServiceEnable bool
	// Gas Limit
	GasLimit uint64
	// HTTP endpoint of the data transport layer
	RollupClientHttp string
	// Owner of the GasPriceOracle contract
	GasPriceOracleOwnerAddress common.Address
	// Turns on checking of state for L2 gas price
	EnableL2GasPolling bool
	// Deployment Height of the canonical transaction chain
	CanonicalTransactionChainDeployHeight *big.Int
	// Polling interval for rollup client
	PollInterval time.Duration
	// Interval for updating the timestamp
	TimestampRefreshThreshold time.Duration
	// Represents the source of the transactions that is being synced
	Backend Backend
	// Only accept transactions with fees
	EnforceFees bool
	// Allow fees within a buffer upwards or downwards
	// to take fee volatility into account between being
	// quoted and the transaction being executed
	FeeThresholdDown *big.Float
	FeeThresholdUp   *big.Float
	// HTTP endpoint of the sequencer
	SequencerClientHttp string
}
