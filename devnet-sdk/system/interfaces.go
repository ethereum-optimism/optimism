package system

import (
	"context"
	"math/big"

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

type genSystem[T Chain] interface {
	Identifier() string
	L1() T
	L2s() []T
}

// System represents a complete Optimism system with L1 and L2 chains
type System = genSystem[Chain]

type LowLevelSystem = genSystem[LowLevelChain]

// Chain represents an Ethereum chain (L1 or L2)
type Chain interface {
	ID() types.ChainID
	// If an instance of an implementation this interface represents an L1 chain,
	// then then the wallets returned should be either validator wallets or test wallets,
	// both useful in the context of sending transactions on the L1.
	//
	// If an instance of an implementation of this interface represents an L2 chain,
	// then the wallets returned should be a combination of:
	// 1. L2 admin wallets: wallets with admin priviledges for administrating an
	//      L2's bridge contracts, etc on L1. Despite inclusion on the L2 wallet list, these wallets
	//      are not useful for sending transactions on the L2 and do not control any L2 balance.
	// 2. L2 test wallets: wallets controlling balance on the L2 for purposes of
	//      testing. The balance on these wallets will originate unbacked L2 ETH from
	//      the L2 genesis definition which cannot be withdrawn without maybe "stealing"
	//      the backing from other deposits.
	Wallets(ctx context.Context) ([]Wallet, error)
	ContractsRegistry() interfaces.ContractsRegistry
	SupportsEIP(ctx context.Context, eip uint64) bool
	Node() Node
	Config() (*params.ChainConfig, error)
	Addresses() descriptors.AddressMap
}

type Node interface {
	GasPrice(ctx context.Context) (*big.Int, error)
	GasLimit(ctx context.Context, tx TransactionData) (uint64, error)
	PendingNonceAt(ctx context.Context, address common.Address) (uint64, error)
	BlockByNumber(ctx context.Context, number *big.Int) (eth.BlockInfo, error)
}

// LowLevelChain is a Chain that gives direct access to the low level RPC client.
type LowLevelChain interface {
	Chain
	RPCURL() string
	Client() (*sources.EthClient, error)
	GethClient() (*ethclient.Client, error)
}

// Wallet represents a chain wallet.
// In particular it can process transactions.
type Wallet interface {
	PrivateKey() types.Key
	Address() types.Address
	SendETH(to types.Address, amount types.Balance) types.WriteInvocation[any]
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
}

// Transaction is the instantiated transaction object.
type Transaction interface {
	Type() uint8
	Hash() common.Hash
	TransactionData
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
	L2s() []Chain
}

// Supervisor provides access to the query interface of the supervisor
type Supervisor interface {
	CheckMessage(context.Context, supervisorTypes.Identifier, common.Hash, supervisorTypes.ExecutingDescriptor) (supervisorTypes.SafetyLevel, error)
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
