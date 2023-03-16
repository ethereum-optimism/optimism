package sources

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"github.com/holiman/uint256"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/trie"

	"github.com/ethereum-optimism/optimism/op-node/eth"
)

type BatchCallContextFn func(ctx context.Context, b []rpc.BatchElem) error

type CallContextFn func(ctx context.Context, result any, method string, args ...any) error

// Note: these types are used, instead of the geth types, to enable:
// - batched calls of many block requests (standard bindings do extra uncle-header fetches, cannot be batched nicely)
// - ignore uncle data (does not even exist anymore post-Merge)
// - use cached block hash, if we trust the RPC.
// - verify transactions list matches tx-root, to ensure consistency with block-hash, if we do not trust the RPC
// - verify block contents are compatible with Post-Merge ExecutionPayload format
//
// Transaction-sender data from the RPC is not cached, since ethclient.setSenderFromServer is private,
// and we only need to compute the sender for transactions into the inbox.
//
// This way we minimize RPC calls, enable batching, and can choose to verify what the RPC gives us.

// HeaderInfo contains all the header-info required to implement the eth.BlockInfo interface,
// used in the rollup state-transition, with pre-computed block hash.
type HeaderInfo struct {
	hash        common.Hash
	parentHash  common.Hash
	coinbase    common.Address
	root        common.Hash
	number      uint64
	time        uint64
	mixDigest   common.Hash // a.k.a. the randomness field post-merge.
	baseFee     *big.Int
	txHash      common.Hash
	receiptHash common.Hash
	gasUsed     uint64

	// withdrawalsRoot was added in Shapella and is thus optional
	withdrawalsRoot *common.Hash
}

var _ eth.BlockInfo = (*HeaderInfo)(nil)

func (info *HeaderInfo) Hash() common.Hash {
	return info.hash
}

func (info *HeaderInfo) ParentHash() common.Hash {
	return info.parentHash
}

func (info *HeaderInfo) Coinbase() common.Address {
	return info.coinbase
}

func (info *HeaderInfo) Root() common.Hash {
	return info.root
}

func (info *HeaderInfo) NumberU64() uint64 {
	return info.number
}

func (info *HeaderInfo) Time() uint64 {
	return info.time
}

func (info *HeaderInfo) MixDigest() common.Hash {
	return info.mixDigest
}

func (info *HeaderInfo) BaseFee() *big.Int {
	return info.baseFee
}

func (info *HeaderInfo) ID() eth.BlockID {
	return eth.BlockID{Hash: info.hash, Number: info.number}
}

func (info *HeaderInfo) ReceiptHash() common.Hash {
	return info.receiptHash
}

func (info *HeaderInfo) GasUsed() uint64 {
	return info.gasUsed
}

type rpcHeader struct {
	ParentHash  common.Hash      `json:"parentHash"`
	UncleHash   common.Hash      `json:"sha3Uncles"`
	Coinbase    common.Address   `json:"miner"`
	Root        common.Hash      `json:"stateRoot"`
	TxHash      common.Hash      `json:"transactionsRoot"`
	ReceiptHash common.Hash      `json:"receiptsRoot"`
	Bloom       eth.Bytes256     `json:"logsBloom"`
	Difficulty  hexutil.Big      `json:"difficulty"`
	Number      hexutil.Uint64   `json:"number"`
	GasLimit    hexutil.Uint64   `json:"gasLimit"`
	GasUsed     hexutil.Uint64   `json:"gasUsed"`
	Time        hexutil.Uint64   `json:"timestamp"`
	Extra       hexutil.Bytes    `json:"extraData"`
	MixDigest   common.Hash      `json:"mixHash"`
	Nonce       types.BlockNonce `json:"nonce"`

	// BaseFee was added by EIP-1559 and is ignored in legacy headers.
	BaseFee *hexutil.Big `json:"baseFeePerGas"`

	// WithdrawalsRoot was added by EIP-4895 and is ignored in legacy headers.
	WithdrawalsRoot *common.Hash `json:"withdrawalsRoot"`

	// untrusted info included by RPC, may have to be checked
	Hash common.Hash `json:"hash"`
}

// checkPostMerge checks that the block header meets all criteria to be a valid ExecutionPayloadHeader,
// see EIP-3675 (block header changes) and EIP-4399 (mixHash usage for prev-randao)
func (hdr *rpcHeader) checkPostMerge() error {
	// TODO: the genesis block has a non-zero difficulty number value.
	// Either this block needs to change, or we special case it. This is not valid w.r.t. EIP-3675.
	if hdr.Number != 0 && (*big.Int)(&hdr.Difficulty).Cmp(common.Big0) != 0 {
		return fmt.Errorf("post-merge block header requires zeroed difficulty field, but got: %s", &hdr.Difficulty)
	}
	if hdr.Nonce != (types.BlockNonce{}) {
		return fmt.Errorf("post-merge block header requires zeroed block nonce field, but got: %s", hdr.Nonce)
	}
	if hdr.BaseFee == nil {
		return fmt.Errorf("post-merge block header requires EIP-1559 basefee field, but got %s", hdr.BaseFee)
	}
	if len(hdr.Extra) > 32 {
		return fmt.Errorf("post-merge block header requires 32 or less bytes of extra data, but got %d", len(hdr.Extra))
	}
	if hdr.UncleHash != types.EmptyUncleHash {
		return fmt.Errorf("post-merge block header requires uncle hash to be of empty uncle list, but got %s", hdr.UncleHash)
	}
	return nil
}

func (hdr *rpcHeader) computeBlockHash() common.Hash {
	gethHeader := types.Header{
		ParentHash:      hdr.ParentHash,
		UncleHash:       hdr.UncleHash,
		Coinbase:        hdr.Coinbase,
		Root:            hdr.Root,
		TxHash:          hdr.TxHash,
		ReceiptHash:     hdr.ReceiptHash,
		Bloom:           types.Bloom(hdr.Bloom),
		Difficulty:      (*big.Int)(&hdr.Difficulty),
		Number:          new(big.Int).SetUint64(uint64(hdr.Number)),
		GasLimit:        uint64(hdr.GasLimit),
		GasUsed:         uint64(hdr.GasUsed),
		Time:            uint64(hdr.Time),
		Extra:           hdr.Extra,
		MixDigest:       hdr.MixDigest,
		Nonce:           hdr.Nonce,
		BaseFee:         (*big.Int)(hdr.BaseFee),
		WithdrawalsHash: hdr.WithdrawalsRoot,
	}
	return gethHeader.Hash()
}

func (hdr *rpcHeader) Info(trustCache bool, mustBePostMerge bool) (*HeaderInfo, error) {
	if mustBePostMerge {
		if err := hdr.checkPostMerge(); err != nil {
			return nil, err
		}
	}
	if !trustCache {
		if computed := hdr.computeBlockHash(); computed != hdr.Hash {
			return nil, fmt.Errorf("failed to verify block hash: computed %s but RPC said %s", computed, hdr.Hash)
		}
	}

	info := HeaderInfo{
		hash:            hdr.Hash,
		parentHash:      hdr.ParentHash,
		coinbase:        hdr.Coinbase,
		root:            hdr.Root,
		number:          uint64(hdr.Number),
		time:            uint64(hdr.Time),
		mixDigest:       hdr.MixDigest,
		baseFee:         (*big.Int)(hdr.BaseFee),
		txHash:          hdr.TxHash,
		receiptHash:     hdr.ReceiptHash,
		gasUsed:         uint64(hdr.GasUsed),
		withdrawalsRoot: hdr.WithdrawalsRoot,
	}
	return &info, nil
}

type rpcBlock struct {
	rpcHeader
	Transactions []*types.Transaction `json:"transactions"`
}

func (block *rpcBlock) verify() error {
	if computed := block.computeBlockHash(); computed != block.Hash {
		return fmt.Errorf("failed to verify block hash: computed %s but RPC said %s", computed, block.Hash)
	}
	if computed := types.DeriveSha(types.Transactions(block.Transactions), trie.NewStackTrie(nil)); block.TxHash != computed {
		return fmt.Errorf("failed to verify transactions list: computed %s but RPC said %s", computed, block.TxHash)
	}
	return nil
}

func (block *rpcBlock) Info(trustCache bool, mustBePostMerge bool) (*HeaderInfo, types.Transactions, error) {
	if mustBePostMerge {
		if err := block.checkPostMerge(); err != nil {
			return nil, nil, err
		}
	}
	if !trustCache {
		if err := block.verify(); err != nil {
			return nil, nil, err
		}
	}

	// verify the header data
	info, err := block.rpcHeader.Info(trustCache, mustBePostMerge)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to verify block from RPC: %w", err)
	}

	return info, block.Transactions, nil
}

func (block *rpcBlock) ExecutionPayload(trustCache bool) (*eth.ExecutionPayload, error) {
	if err := block.checkPostMerge(); err != nil {
		return nil, err
	}
	if !trustCache {
		if err := block.verify(); err != nil {
			return nil, err
		}
	}
	var baseFee uint256.Int
	baseFee.SetFromBig((*big.Int)(block.BaseFee))

	// Unfortunately eth_getBlockByNumber either returns full transactions, or only tx-hashes.
	// There is no option for encoded transactions.
	opaqueTxs := make([]hexutil.Bytes, len(block.Transactions))
	for i, tx := range block.Transactions {
		data, err := tx.MarshalBinary()
		if err != nil {
			return nil, fmt.Errorf("failed to encode tx %d from RPC: %w", i, err)
		}
		opaqueTxs[i] = data
	}

	return &eth.ExecutionPayload{
		ParentHash:    block.ParentHash,
		FeeRecipient:  block.Coinbase,
		StateRoot:     eth.Bytes32(block.Root),
		ReceiptsRoot:  eth.Bytes32(block.ReceiptHash),
		LogsBloom:     block.Bloom,
		PrevRandao:    eth.Bytes32(block.MixDigest), // mix-digest field is used for prevRandao post-merge
		BlockNumber:   block.Number,
		GasLimit:      block.GasLimit,
		GasUsed:       block.GasUsed,
		Timestamp:     block.Time,
		ExtraData:     eth.BytesMax32(block.Extra),
		BaseFeePerGas: baseFee,
		BlockHash:     block.Hash,
		Transactions:  opaqueTxs,
	}, nil
}

// blockHashParameter is used as "block parameter":
// Some Nethermind and Alchemy RPC endpoints require an object to identify a block, instead of a string.
type blockHashParameter struct {
	BlockHash common.Hash `json:"blockHash"`
}

// unusableMethod identifies if an error indicates that the RPC method cannot be used as expected:
// if it's an unknown method, or if parameters were invalid.
func unusableMethod(err error) bool {
	if rpcErr, ok := err.(rpc.Error); ok {
		code := rpcErr.ErrorCode()
		// method not found, or invalid params
		if code == -32601 || code == -32602 {
			return true
		}
	} else {
		errText := strings.ToLower(err.Error())
		if strings.Contains(errText, "unknown method") || strings.Contains(errText, "invalid param") || strings.Contains(errText, "is not available") {
			return true
		}
	}
	return false
}
