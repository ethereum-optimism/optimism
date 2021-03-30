package rollup

import (
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

type Config struct {
	// Maximum calldata size for a Queue Origin Sequencer Tx
	MaxCallDataSize int
	// Number of confs before applying a L1 to L2 tx
	Eth1ConfirmationDepth uint64
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
	AddressManagerOwnerAddress    common.Address
	L1ETHGatewayAddress           common.Address
	// Deployment Height of the canonical transaction chain
	CanonicalTransactionChainDeployHeight *big.Int
	// Path to the state dump
	StateDumpPath string
	// Polling interval for rollup client
	PollInterval time.Duration
	// Interval for updating the timestamp
	TimestampRefreshThreshold time.Duration
	// The gas price to use when estimating L1 calldata publishing costs
	L1GasPrice *big.Int
}
