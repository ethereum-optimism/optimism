package system

import (
	"context"
	"math/big"

	"github.com/ethereum-optimism/optimism/devnet-sdk/contracts/bindings"
	"github.com/ethereum-optimism/optimism/devnet-sdk/descriptors"
	"github.com/ethereum-optimism/optimism/devnet-sdk/interfaces"
	"github.com/ethereum-optimism/optimism/devnet-sdk/types"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	supervisorTypes "github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	coreTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/params"
)

type genSystem[T Chain, U L2Chain] interface {
	Identifier() string
	L1() T
	L2s() []U
}

// System represents a complete Optimism system with L1 and L2 chains
type System = genSystem[Chain, L2Chain]

type LowLevelSystem = genSystem[LowLevelChain, LowLevelL2Chain]

// Chain represents an Ethereum chain (L1 or L2)
type Chain interface {
	ID() types.ChainID
	ContractsRegistry() interfaces.ContractsRegistry
	SupportsEIP(ctx context.Context, eip uint64) bool
	Node() Node
	Config() (*params.ChainConfig, error)

	// The wallets and addresses below are for use on the chain that the instance represents.
	// If the instance also implements L2Chain, then the wallets and addresses below are still for the L2.
	Wallets() WalletMap
	Addresses() AddressMap
}

type L2Chain interface {
	Chain

	// The wallets and addresses below are for use on the L1 chain that this L2Chain instance settles to.
	L1Addresses() AddressMap
	L1Wallets() WalletMap
}

type Node interface {
	GasPrice(ctx context.Context) (*big.Int, error)
	GasLimit(ctx context.Context, tx TransactionData) (uint64, error)
	PendingNonceAt(ctx context.Context, address common.Address) (uint64, error)
	BlockByNumber(ctx context.Context, number *big.Int) (eth.BlockInfo, error)
}

type LowLevelChain interface {
	Chain
	RPCURL() string
	Client() (*sources.EthClient, error)
	GethClient() (*ethclient.Client, error)
}

type LowLevelL2Chain interface {
	L2Chain
	LowLevelChain
}

type WalletMap map[string]Wallet
type AddressMap descriptors.AddressMap

// Wallet represents a chain wallet.
// In particular it can process transactions.
type Wallet interface {
	PrivateKey() types.Key
	Address() types.Address
	SendETH(to types.Address, amount types.Balance) types.WriteInvocation[any]
	InitiateMessage(chainID types.ChainID, target common.Address, message []byte) types.WriteInvocation[any]
	ExecuteMessage(identifier bindings.Identifier, sentMessage []byte) types.WriteInvocation[any]
	Balance() types.Balance
	Nonce() uint64

	TransactionProcessor
}

// TransactionProcessor is a helper interface for signing and sending transactions.
type TransactionProcessor interface {
	Sign(tx Transaction) (Transaction, error)
	Send(ctx context.Context, tx Transaction) error
}

// Transaction interfaces:

// TransactionData is the input for a transaction creation.
type TransactionData interface {
	From() common.Address
	To() *common.Address
	Value() *big.Int
	Data() []byte
	AccessList() coreTypes.AccessList
}

// Transaction is the instantiated transaction object.
type Transaction interface {
	Type() uint8
	Hash() common.Hash
	TransactionData
}

type Receipt interface {
	BlockNumber() *big.Int
	Logs() []*coreTypes.Log
	TxHash() common.Hash
}

// RawTransaction is an optional interface that can be implemented by a Transaction
// to provide access to the raw transaction data.
// It is currently necessary to perform processing operations (signing, sending)
// on the transaction. We would need to do better here.
type RawTransaction interface {
	Raw() *coreTypes.Transaction
}

// Specialized interop interfaces:

// InteropSystem extends System with interoperability features
type InteropSystem interface {
	System
	InteropSet() InteropSet
	Supervisor(context.Context) (Supervisor, error)
}

// InteropSet provides access to L2 chains in an interop environment
type InteropSet interface {
	L2s() []L2Chain
}

// Supervisor provides access to the query interface of the supervisor
type Supervisor interface {
	LocalUnsafe(context.Context, eth.ChainID) (eth.BlockID, error)
	CrossSafe(context.Context, eth.ChainID) (supervisorTypes.DerivedIDPair, error)
	Finalized(context.Context, eth.ChainID) (eth.BlockID, error)
	FinalizedL1(context.Context) (eth.BlockRef, error)
	CrossDerivedFrom(context.Context, eth.ChainID, eth.BlockID) (eth.BlockRef, error)
	UpdateLocalUnsafe(context.Context, eth.ChainID, eth.BlockRef) error
	UpdateLocalSafe(context.Context, eth.ChainID, eth.L1BlockRef, eth.BlockRef) error
	SuperRootAtTimestamp(context.Context, hexutil.Uint64) (eth.SuperRootResponse, error)
	AllSafeDerivedAt(context.Context, eth.BlockID) (derived map[eth.ChainID]eth.BlockID, err error)
	SyncStatus(context.Context) (eth.SupervisorSyncStatus, error)
}
