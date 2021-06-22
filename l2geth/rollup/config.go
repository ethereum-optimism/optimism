package rollup

import (
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

type Config struct {
	// Maximum calldata size for a Queue Origin Sequencer Tx
	MaxCallDataSize int
	// Verifier mode
	IsVerifier bool
	// Enable the sync service
	Eth1SyncServiceEnable bool
	// Ensure that the correct layer 1 chain is being connected to
	Eth1ChainId uint64
	// Gas Limit
	GasLimit uint64
	// HTTP endpoint of the data transport layer
	RollupClientHttp              string
	L1CrossDomainMessengerAddress common.Address
	L1FeeWalletAddress            common.Address
	AddressManagerOwnerAddress    common.Address
	GasPriceOracleOwnerAddress    common.Address
	L1StandardBridgeAddress       common.Address
	// Turns on checking of state for L2 gas price
	EnableL2GasPolling bool
	// Deployment Height of the canonical transaction chain
	CanonicalTransactionChainDeployHeight *big.Int
	// Path to the state dump
	StateDumpPath string
	// Polling interval for rollup client
	PollInterval time.Duration
	// Interval for updating the timestamp
	TimestampRefreshThreshold time.Duration
	// Represents the source of the transactions that is being synced
	Backend Backend
	// Only accept transactions with fees
	EnforceFees bool
}
