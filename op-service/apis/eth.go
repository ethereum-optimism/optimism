package apis

import (
	"context"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
)

type ChainID interface {
	// ChainID fetches the chain id of the internal RPC.
	ChainID(ctx context.Context) (*big.Int, error)
}

type EthBlockInfo interface {
	InfoByHash(ctx context.Context, hash common.Hash) (eth.BlockInfo, error)

	InfoByNumber(ctx context.Context, number uint64) (eth.BlockInfo, error)

	InfoByLabel(ctx context.Context, label eth.BlockLabel) (eth.BlockInfo, error)

	InfoAndTxsByHash(ctx context.Context, hash common.Hash) (eth.BlockInfo, types.Transactions, error)

	InfoAndTxsByNumber(ctx context.Context, number uint64) (eth.BlockInfo, types.Transactions, error)

	InfoAndTxsByLabel(ctx context.Context, label eth.BlockLabel) (eth.BlockInfo, types.Transactions, error)
}

type EthPayload interface {
	PayloadByHash(ctx context.Context, hash common.Hash) (*eth.ExecutionPayloadEnvelope, error)

	PayloadByNumber(ctx context.Context, number uint64) (*eth.ExecutionPayloadEnvelope, error)

	PayloadByLabel(ctx context.Context, label eth.BlockLabel) (*eth.ExecutionPayloadEnvelope, error)
}

type ReceiptsFetcher interface {
	// FetchReceipts returns a block info and all of the receipts associated with transactions in the block.
	// It verifies the receipt hash in the block header against the receipt hash of the fetched receipts
	// to ensure that the execution engine did not fail to return any receipts.
	FetchReceipts(ctx context.Context, blockHash common.Hash) (eth.BlockInfo, types.Receipts, error)
}

type ReceiptFetcher interface {
	// TransactionReceipt returns a receipt associated with transaction.
	TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error)
}

type ExecutionWitness interface {
	// PayloadExecutionWitness generates a block from a payload and returns execution witness data.
	PayloadExecutionWitness(ctx context.Context, parentHash common.Hash, payloadAttributes eth.PayloadAttributes) (*eth.ExecutionWitness, error)
}

type EthProof interface {
	// GetProof returns an account proof result, with any optional requested storage proofs.
	// The retrieval does sanity-check that storage proofs for the expected keys are present in the response,
	// but does not verify the result. Call accountResult.Verify(stateRoot) to verify the result.
	GetProof(ctx context.Context, address common.Address, storage []common.Hash, blockTag string) (*eth.AccountResult, error)
}

type EthStorage interface {
	// GetStorageAt returns the storage value at the given address and storage slot, **without verifying the correctness of the result**.
	// This should only ever be used as alternative to GetProof when the user opts in.
	// E.g. Erigon L1 node users may have to use this, since Erigon does not support eth_getProof, see https://github.com/ledgerwatch/erigon/issues/1349
	GetStorageAt(ctx context.Context, address common.Address, storageSlot common.Hash, blockTag string) (common.Hash, error)

	// ReadStorageAt is a convenience method to read a single storage value at the given slot in the given account.
	// The storage slot value is verified against the state-root of the given block if we do not trust the RPC provider, or directly retrieved without proof if we do trust the RPC.
	ReadStorageAt(ctx context.Context, address common.Address, storageSlot common.Hash, blockHash common.Hash) (common.Hash, error)
}

type EthBlockRef interface {
	// BlockRefByLabel returns the [eth.BlockRef] for the given block label.
	// Notice, we cannot cache a block reference by label because labels are not guaranteed to be unique.
	BlockRefByLabel(ctx context.Context, label eth.BlockLabel) (eth.BlockRef, error)

	// BlockRefByNumber returns an [eth.BlockRef] for the given block number.
	// Notice, we cannot cache a block reference by number because L1 re-orgs can invalidate the cached block reference.
	BlockRefByNumber(ctx context.Context, num uint64) (eth.BlockRef, error)

	// BlockRefByHash returns the [eth.BlockRef] for the given block hash.
	// We cache the block reference by hash as it is safe to assume collision will not occur.
	BlockRefByHash(ctx context.Context, hash common.Hash) (eth.BlockRef, error)
}

type Gas interface {
	// SuggestGasPrice retrieves the currently suggested gas price to allow a timely
	// execution of a transaction.
	SuggestGasPrice(ctx context.Context) (*big.Int, error)

	// EstimateGas tries to estimate the gas needed to execute a specific transaction.
	EstimateGas(ctx context.Context, msg ethereum.CallMsg) (uint64, error)
}

type EthCall interface {
	// Call executes a message call transaction but never mined into the blockchain.
	Call(ctx context.Context, msg ethereum.CallMsg, blockNumber rpc.BlockNumber) ([]byte, error)
}

type TransactionSender interface {
	// SendTransaction submits a signed transaction.
	SendTransaction(ctx context.Context, tx *types.Transaction) error
}

type EthNonce interface {
	// PendingNonceAt returns the account nonce of the given account in the pending state.
	PendingNonceAt(ctx context.Context, account common.Address) (uint64, error)
	// NonceAt returns the account nonce of the given account in the state at the given block number.
	// A nil block number may be used to get the latest state.
	NonceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (uint64, error)
}

type EthBalance interface {
	// BalanceAt returns the wei balance of the given account.
	BalanceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (*big.Int, error)
}

type EthCode interface {
	CodeAtHash(ctx context.Context, account common.Address, blockHash common.Hash) ([]byte, error)
}

type EthMultiCaller interface {
	NewMultiCaller(batchSize int) *batching.MultiCaller
}

type RPCCaller interface {
	RPC() client.RPC
}

type EthClient interface {
	ChainID
	EthBlockInfo
	ReceiptFetcher
	ExecutionWitness
	EthProof
	EthStorage
	EthBlockRef
	Gas
	EthCall
	TransactionSender
	EthNonce
	EthBalance
	EthCode
	EthMultiCaller
	RPCCaller
}

type EthExtendedClient interface {
	EthClient
	EthPayload
	ReceiptsFetcher
}
