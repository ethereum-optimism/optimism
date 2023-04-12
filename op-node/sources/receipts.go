package sources

import (
	"context"
	"fmt"
	"io"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/trie"

	"github.com/ethereum-optimism/optimism/op-node/eth"
)

// validateReceipts validates that the receipt contents are valid.
// Warning: contractAddress is not verified, since it is a more expensive operation for data we do not use.
// See go-ethereum/crypto.CreateAddress to verify contract deployment address data based on sender and tx nonce.
func validateReceipts(block eth.BlockID, receiptHash common.Hash, txHashes []common.Hash, receipts []*types.Receipt) error {
	if len(receipts) != len(txHashes) {
		return fmt.Errorf("got %d receipts but expected %d", len(receipts), len(txHashes))
	}
	if len(txHashes) == 0 {
		if receiptHash != types.EmptyRootHash {
			return fmt.Errorf("no transactions, but got non-empty receipt trie root: %s", receiptHash)
		}
	}
	// We don't trust the RPC to provide consistent cached receipt info that we use for critical rollup derivation work.
	// Let's check everything quickly.
	logIndex := uint(0)
	cumulativeGas := uint64(0)
	for i, r := range receipts {
		if r == nil { // on reorgs or other cases the receipts may disappear before they can be retrieved.
			return fmt.Errorf("receipt of tx %d returns nil on retrieval", i)
		}
		if r.TransactionIndex != uint(i) {
			return fmt.Errorf("receipt %d has unexpected tx index %d", i, r.TransactionIndex)
		}
		if r.BlockNumber == nil {
			return fmt.Errorf("receipt %d has unexpected nil block number, expected %d", i, block.Number)
		}
		if r.BlockNumber.Uint64() != block.Number {
			return fmt.Errorf("receipt %d has unexpected block number %d, expected %d", i, r.BlockNumber, block.Number)
		}
		if r.BlockHash != block.Hash {
			return fmt.Errorf("receipt %d has unexpected block hash %s, expected %s", i, r.BlockHash, block.Hash)
		}
		if expected := r.CumulativeGasUsed - cumulativeGas; r.GasUsed != expected {
			return fmt.Errorf("receipt %d has invalid gas used metadata: %d, expected %d", i, r.GasUsed, expected)
		}
		for j, log := range r.Logs {
			if log.Index != logIndex {
				return fmt.Errorf("log %d (%d of tx %d) has unexpected log index %d", logIndex, j, i, log.Index)
			}
			if log.TxIndex != uint(i) {
				return fmt.Errorf("log %d has unexpected tx index %d", log.Index, log.TxIndex)
			}
			if log.BlockHash != block.Hash {
				return fmt.Errorf("log %d of block %s has unexpected block hash %s", log.Index, block.Hash, log.BlockHash)
			}
			if log.BlockNumber != block.Number {
				return fmt.Errorf("log %d of block %d has unexpected block number %d", log.Index, block.Number, log.BlockNumber)
			}
			if log.TxHash != txHashes[i] {
				return fmt.Errorf("log %d of tx %s has unexpected tx hash %s", log.Index, txHashes[i], log.TxHash)
			}
			if log.Removed {
				return fmt.Errorf("canonical log (%d) must never be removed due to reorg", log.Index)
			}
			logIndex++
		}
		cumulativeGas = r.CumulativeGasUsed
		// Note: 3 non-consensus L1 receipt fields are ignored:
		// PostState - not part of L1 ethereum anymore since EIP 658 (part of Byzantium)
		// ContractAddress - we do not care about contract deployments
		// And Optimism L1 fee meta-data in the receipt is ignored as well
	}

	// Sanity-check: external L1-RPC sources are notorious for not returning all receipts,
	// or returning them out-of-order. Verify the receipts against the expected receipt-hash.
	hasher := trie.NewStackTrie(nil)
	computed := types.DeriveSha(types.Receipts(receipts), hasher)
	if receiptHash != computed {
		return fmt.Errorf("failed to fetch list of receipts: expected receipt root %s but computed %s from retrieved receipts", receiptHash, computed)
	}
	return nil
}

func makeReceiptRequest(txHash common.Hash) (*types.Receipt, rpc.BatchElem) {
	out := new(types.Receipt)
	return out, rpc.BatchElem{
		Method: "eth_getTransactionReceipt",
		Args:   []any{txHash},
		Result: &out, // receipt may become nil, double pointer is intentional
	}
}

// Cost break-down sources:
// Alchemy: https://docs.alchemy.com/reference/compute-units
// QuickNode: https://www.quicknode.com/docs/ethereum/api_credits
// Infura: no pricing table available.
//
// Receipts are encoded the same everywhere:
//
//     blockHash, blockNumber, transactionIndex, transactionHash, from, to, cumulativeGasUsed, gasUsed,
//     contractAddress, logs, logsBloom, status, effectiveGasPrice, type.
//
// Note that Alchemy/Geth still have a "root" field for legacy reasons,
// but ethereum does not compute state-roots per tx anymore, so quicknode and others do not serve this data.

// RPCProviderKind identifies an RPC provider, used to hint at the optimal receipt fetching approach.
type RPCProviderKind string

const (
	RPCKindAlchemy    RPCProviderKind = "alchemy"
	RPCKindQuickNode  RPCProviderKind = "quicknode"
	RPCKindInfura     RPCProviderKind = "infura"
	RPCKindParity     RPCProviderKind = "parity"
	RPCKindNethermind RPCProviderKind = "nethermind"
	RPCKindDebugGeth  RPCProviderKind = "debug_geth"
	RPCKindErigon     RPCProviderKind = "erigon"
	RPCKindBasic      RPCProviderKind = "basic" // try only the standard most basic receipt fetching
	RPCKindAny        RPCProviderKind = "any"   // try any method available
)

var RPCProviderKinds = []RPCProviderKind{
	RPCKindAlchemy,
	RPCKindQuickNode,
	RPCKindInfura,
	RPCKindParity,
	RPCKindNethermind,
	RPCKindDebugGeth,
	RPCKindErigon,
	RPCKindBasic,
	RPCKindAny,
}

func (kind RPCProviderKind) String() string {
	return string(kind)
}

func (kind *RPCProviderKind) Set(value string) error {
	if !ValidRPCProviderKind(RPCProviderKind(value)) {
		return fmt.Errorf("unknown rpc kind: %q", value)
	}
	*kind = RPCProviderKind(value)
	return nil
}

func ValidRPCProviderKind(value RPCProviderKind) bool {
	for _, k := range RPCProviderKinds {
		if k == value {
			return true
		}
	}
	return false
}

// ReceiptsFetchingMethod is a bitfield with 1 bit for each receipts fetching type.
// Depending on errors, tx counts and preferences the code may select different sets of fetching methods.
type ReceiptsFetchingMethod uint64

func (r ReceiptsFetchingMethod) String() string {
	out := ""
	x := r
	addMaybe := func(m ReceiptsFetchingMethod, v string) {
		if x&m != 0 {
			out += v
			x ^= x & m
		}
		if x != 0 { // add separator if there are entries left
			out += ", "
		}
	}
	addMaybe(EthGetTransactionReceiptBatch, "eth_getTransactionReceipt (batched)")
	addMaybe(AlchemyGetTransactionReceipts, "alchemy_getTransactionReceipts")
	addMaybe(DebugGetRawReceipts, "debug_getRawReceipts")
	addMaybe(ParityGetBlockReceipts, "parity_getBlockReceipts")
	addMaybe(EthGetBlockReceipts, "eth_getBlockReceipts")
	addMaybe(^ReceiptsFetchingMethod(0), "unknown") // if anything is left, describe it as unknown
	return out
}

const (
	// EthGetTransactionReceiptBatch is standard per-tx receipt fetching with JSON-RPC batches.
	// Available in: standard, everywhere.
	//   - Alchemy: 15 CU / tx
	//   - Quicknode: 2 credits / tx
	// Method: eth_getTransactionReceipt
	// See: https://ethereum.github.io/execution-apis/api-documentation/
	EthGetTransactionReceiptBatch ReceiptsFetchingMethod = 1 << iota
	// AlchemyGetTransactionReceipts is a special receipt fetching method provided by Alchemy.
	// Available in:
	//   - Alchemy: 250 CU total
	// Method: alchemy_getTransactionReceipts
	// Params:
	//   - object with "blockNumber" or "blockHash" field
	// Returns: "array of receipts" - docs lie, array is wrapped in a struct with single "receipts" field
	// See: https://docs.alchemy.com/reference/alchemy-gettransactionreceipts#alchemy_gettransactionreceipts
	AlchemyGetTransactionReceipts
	// DebugGetRawReceipts is a debug method from Geth, faster by avoiding serialization and metadata overhead.
	// Ideal for fast syncing from a local geth node.
	// Available in:
	//   - Geth: free
	//   - QuickNode: 22 credits maybe? Unknown price, undocumented ("debug_getblockreceipts" exists in table though?)
	// Method: debug_getRawReceipts
	// Params:
	//   - string presenting a block number or hash
	// Returns: list of strings, hex encoded RLP of receipts data. "consensus-encoding of all receipts in a single block"
	// See: https://geth.ethereum.org/docs/rpc/ns-debug#debug_getrawreceipts
	DebugGetRawReceipts
	// ParityGetBlockReceipts is an old parity method, which has been adopted by Nethermind and some RPC providers.
	// Available in:
	//   - Alchemy: 500 CU total
	//   - QuickNode: 59 credits - docs are wrong, not actually available anymore.
	//   - Any open-ethereum/parity legacy: free
	//   - Nethermind: free
	// Method: parity_getBlockReceipts
	// Params:
	//   Parity: "quantity or tag"
	//   Alchemy: string with block hash, number in hex, or block tag.
	//   Nethermind: very flexible: tag, number, hex or object with "requireCanonical"/"blockHash" fields.
	// Returns: array of receipts
	// See:
	//   - Parity: https://openethereum.github.io/JSONRPC-parity-module#parity_getblockreceipts
	//   - QuickNode: undocumented.
	//   - Alchemy: https://docs.alchemy.com/reference/eth-getblockreceipts
	//   - Nethermind: https://docs.nethermind.io/nethermind/ethereum-client/json-rpc/parity#parity_getblockreceipts
	ParityGetBlockReceipts
	// EthGetBlockReceipts is a non-standard receipt fetching method in the eth namespace,
	// supported by some RPC platforms and Erigon.
	// Available in:
	//   - Alchemy: 500 CU total  (and deprecated)
	//   - Erigon: free
	//   - QuickNode: 59 credits total       (does not seem to work with block hash arg, inaccurate docs)
	// Method: eth_getBlockReceipts
	// Params:
	//   - QuickNode: string, "quantity or tag", docs say incl. block hash, but API does not actually accept it.
	//   - Alchemy: string, block hash / num (hex) / block tag
	// Returns: array of receipts
	// See:
	//   - QuickNode: https://www.quicknode.com/docs/ethereum/eth_getBlockReceipts
	//   - Alchemy: https://docs.alchemy.com/reference/eth-getblockreceipts
	EthGetBlockReceipts

	// Other:
	//  - 250 credits, not supported, strictly worse than other options. In quicknode price-table.
	// qn_getBlockWithReceipts - in price table, ? undocumented, but in quicknode "Single Flight RPC" description
	// qn_getReceipts          - in price table, ? undocumented, but in quicknode "Single Flight RPC" description
	// debug_getBlockReceipts  - ? undocumented, shows up in quicknode price table, not available.
)

// AvailableReceiptsFetchingMethods selects receipt fetching methods based on the RPC provider kind.
func AvailableReceiptsFetchingMethods(kind RPCProviderKind) ReceiptsFetchingMethod {
	switch kind {
	case RPCKindAlchemy:
		return AlchemyGetTransactionReceipts | EthGetBlockReceipts | EthGetTransactionReceiptBatch
	case RPCKindQuickNode:
		return DebugGetRawReceipts | EthGetBlockReceipts | EthGetTransactionReceiptBatch
	case RPCKindInfura:
		// Infura is big, but sadly does not support more optimized receipts fetching methods (yet?)
		return EthGetTransactionReceiptBatch
	case RPCKindParity:
		return ParityGetBlockReceipts | EthGetTransactionReceiptBatch
	case RPCKindNethermind:
		return ParityGetBlockReceipts | EthGetTransactionReceiptBatch
	case RPCKindDebugGeth:
		return DebugGetRawReceipts | EthGetTransactionReceiptBatch
	case RPCKindErigon:
		return EthGetBlockReceipts | EthGetTransactionReceiptBatch
	case RPCKindBasic:
		return EthGetTransactionReceiptBatch
	case RPCKindAny:
		// if it's any kind of RPC provider, then try all methods
		return AlchemyGetTransactionReceipts | EthGetBlockReceipts |
			DebugGetRawReceipts | ParityGetBlockReceipts | EthGetTransactionReceiptBatch
	default:
		return EthGetTransactionReceiptBatch
	}
}

// PickBestReceiptsFetchingMethod selects an RPC method that is still available,
// and optimal for fetching the given number of tx receipts from the specified provider kind.
func PickBestReceiptsFetchingMethod(kind RPCProviderKind, available ReceiptsFetchingMethod, txCount uint64) ReceiptsFetchingMethod {
	// If we have optimized methods available, it makes sense to use them, but only if the cost is
	// lower than fetching transactions one by one with the standard receipts RPC method.
	if kind == RPCKindAlchemy {
		if available&AlchemyGetTransactionReceipts != 0 && txCount > 250/15 {
			return AlchemyGetTransactionReceipts
		}
		if available&EthGetBlockReceipts != 0 && txCount > 500/15 {
			return EthGetBlockReceipts
		}
		return EthGetTransactionReceiptBatch
	} else if kind == RPCKindQuickNode {
		if available&DebugGetRawReceipts != 0 {
			return DebugGetRawReceipts
		}
		if available&EthGetBlockReceipts != 0 && txCount > 59/2 {
			return EthGetBlockReceipts
		}
		return EthGetTransactionReceiptBatch
	}
	// in order of preference (based on cost): check available methods
	if available&AlchemyGetTransactionReceipts != 0 {
		return AlchemyGetTransactionReceipts
	}
	if available&DebugGetRawReceipts != 0 {
		return DebugGetRawReceipts
	}
	if available&EthGetBlockReceipts != 0 {
		return EthGetBlockReceipts
	}
	if available&ParityGetBlockReceipts != 0 {
		return ParityGetBlockReceipts
	}
	// otherwise fall back on per-tx fetching
	return EthGetTransactionReceiptBatch
}

type rpcClient interface {
	CallContext(ctx context.Context, result any, method string, args ...any) error
	BatchCallContext(ctx context.Context, b []rpc.BatchElem) error
}

// receiptsFetchingJob runs the receipt fetching for a specific block,
// and can re-run and adapt based on the fetching method preferences and errors communicated with the requester.
type receiptsFetchingJob struct {
	m sync.Mutex

	requester ReceiptsRequester

	client       rpcClient
	maxBatchSize int

	block       eth.BlockID
	receiptHash common.Hash
	txHashes    []common.Hash

	fetcher *IterativeBatchCall[common.Hash, *types.Receipt]

	result types.Receipts
}

func NewReceiptsFetchingJob(requester ReceiptsRequester, client rpcClient, maxBatchSize int, block eth.BlockID,
	receiptHash common.Hash, txHashes []common.Hash) *receiptsFetchingJob {
	return &receiptsFetchingJob{
		requester:    requester,
		client:       client,
		maxBatchSize: maxBatchSize,
		block:        block,
		receiptHash:  receiptHash,
		txHashes:     txHashes,
	}
}

// ReceiptsRequester helps determine which receipts fetching method can be used,
// and is given feedback upon receipt fetching errors to adapt the choice of method.
type ReceiptsRequester interface {
	PickReceiptsMethod(txCount uint64) ReceiptsFetchingMethod
	OnReceiptsMethodErr(m ReceiptsFetchingMethod, err error)
}

// runFetcher retrieves the result by continuing previous batched receipt fetching work,
// and starting this work if necessary.
func (job *receiptsFetchingJob) runFetcher(ctx context.Context) error {
	if job.fetcher == nil {
		// start new work
		job.fetcher = NewIterativeBatchCall[common.Hash, *types.Receipt](
			job.txHashes,
			makeReceiptRequest,
			job.client.BatchCallContext,
			job.client.CallContext,
			job.maxBatchSize,
		)
	}
	// Fetch all receipts
	for {
		if err := job.fetcher.Fetch(ctx); err == io.EOF {
			break
		} else if err != nil {
			return err
		}
	}
	result, err := job.fetcher.Result()
	if err != nil { // errors if results are not available yet, should never happen.
		return err
	}
	if err := validateReceipts(job.block, job.receiptHash, job.txHashes, result); err != nil {
		job.fetcher.Reset() // if results are fetched but invalid, try restart all the fetching to try and get valid data.
		return err
	}
	// Remember the result, and don't keep the fetcher and tx hashes around for longer than needed
	job.result = result
	job.fetcher = nil
	job.txHashes = nil
	return nil
}

// receiptsWrapper is a decoding type util. Alchemy in particular wraps the receipts array result.
type receiptsWrapper struct {
	Receipts []*types.Receipt `json:"receipts"`
}

// runAltMethod retrieves the result by fetching all receipts at once,
// using the given non-standard receipt fetching method.
func (job *receiptsFetchingJob) runAltMethod(ctx context.Context, m ReceiptsFetchingMethod) error {
	var result []*types.Receipt
	var err error
	switch m {
	case AlchemyGetTransactionReceipts:
		var tmp receiptsWrapper
		err = job.client.CallContext(ctx, &tmp, "alchemy_getTransactionReceipts", blockHashParameter{BlockHash: job.block.Hash})
		result = tmp.Receipts
	case DebugGetRawReceipts:
		var rawReceipts []hexutil.Bytes
		err = job.client.CallContext(ctx, &rawReceipts, "debug_getRawReceipts", job.block.Hash)
		if err == nil {
			if len(rawReceipts) == len(job.txHashes) {
				result = make([]*types.Receipt, len(rawReceipts))
				totalIndex := uint(0)
				prevCumulativeGasUsed := uint64(0)
				for i, r := range rawReceipts {
					var x types.Receipt
					_ = x.UnmarshalBinary(r) // safe to ignore, we verify receipts against the receipts hash later
					x.TxHash = job.txHashes[i]
					x.BlockHash = job.block.Hash
					x.BlockNumber = new(big.Int).SetUint64(job.block.Number)
					x.TransactionIndex = uint(i)
					x.GasUsed = x.CumulativeGasUsed - prevCumulativeGasUsed
					// contract address meta-data is not computed.
					prevCumulativeGasUsed = x.CumulativeGasUsed
					for _, l := range x.Logs {
						l.BlockNumber = job.block.Number
						l.TxHash = x.TxHash
						l.TxIndex = uint(i)
						l.BlockHash = job.block.Hash
						l.Index = totalIndex
						totalIndex += 1
					}
					result[i] = &x
				}
			} else {
				err = fmt.Errorf("got %d raw receipts, but expected %d", len(rawReceipts), len(job.txHashes))
			}
		}
	case ParityGetBlockReceipts:
		err = job.client.CallContext(ctx, &result, "parity_getBlockReceipts", job.block.Hash)
	case EthGetBlockReceipts:
		err = job.client.CallContext(ctx, &result, "eth_getBlockReceipts", job.block.Hash)
	default:
		err = fmt.Errorf("unknown receipt fetching method: %d", uint64(m))
	}
	if err != nil {
		job.requester.OnReceiptsMethodErr(m, err)
		return err
	} else {
		if err := validateReceipts(job.block, job.receiptHash, job.txHashes, result); err != nil {
			return err
		}
		job.result = result
		return nil
	}
}

// Fetch makes the job fetch the receipts, and returns the results, if any.
// An error may be returned if the fetching is not successfully completed,
// and fetching may be continued/re-attempted by calling Fetch again.
// The job caches the result, so repeated Fetches add no additional cost.
// Fetch is safe to be called concurrently, and will lock to avoid duplicate work or internal inconsistency.
func (job *receiptsFetchingJob) Fetch(ctx context.Context) (types.Receipts, error) {
	job.m.Lock()
	defer job.m.Unlock()

	if job.result != nil {
		return job.result, nil
	}

	m := job.requester.PickReceiptsMethod(uint64(len(job.txHashes)))

	if m == EthGetTransactionReceiptBatch {
		if err := job.runFetcher(ctx); err != nil {
			return nil, err
		}
	} else {
		if err := job.runAltMethod(ctx, m); err != nil {
			return nil, err
		}
	}

	return job.result, nil
}
