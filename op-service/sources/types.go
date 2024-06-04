package sources

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/rlp"
	"github.com/holiman/uint256"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/consensus/misc/eip4844"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/trie"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

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

// headerInfo is a conversion type of types.Header turning it into a
// BlockInfo, but using a cached hash value.
type headerInfo struct {
	hash common.Hash
	*types.Header
}

var _ eth.BlockInfo = (*headerInfo)(nil)

func (h headerInfo) Hash() common.Hash {
	return h.hash
}

func (h headerInfo) ParentHash() common.Hash {
	return h.Header.ParentHash
}

func (h headerInfo) Coinbase() common.Address {
	return h.Header.Coinbase
}

func (h headerInfo) Root() common.Hash {
	return h.Header.Root
}

func (h headerInfo) NumberU64() uint64 {
	return h.Header.Number.Uint64()
}

func (h headerInfo) Time() uint64 {
	return h.Header.Time
}

func (h headerInfo) MixDigest() common.Hash {
	return h.Header.MixDigest
}

func (h headerInfo) BaseFee() *big.Int {
	return h.Header.BaseFee
}

func (h headerInfo) BlobBaseFee() *big.Int {
	if h.Header.ExcessBlobGas == nil {
		return nil
	}
	return eip4844.CalcBlobFee(*h.Header.ExcessBlobGas)
}

func (h headerInfo) ReceiptHash() common.Hash {
	return h.Header.ReceiptHash
}

func (h headerInfo) GasUsed() uint64 {
	return h.Header.GasUsed
}

func (h headerInfo) GasLimit() uint64 {
	return h.Header.GasLimit
}

func (h headerInfo) ParentBeaconRoot() *common.Hash {
	return h.Header.ParentBeaconRoot
}

func (h headerInfo) HeaderRLP() ([]byte, error) {
	return rlp.EncodeToBytes(h.Header)
}

type RPCHeader struct {
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
	WithdrawalsRoot *common.Hash `json:"withdrawalsRoot,omitempty"`

	// BlobGasUsed was added by EIP-4844 and is ignored in legacy headers.
	BlobGasUsed *hexutil.Uint64 `json:"blobGasUsed,omitempty"`

	// ExcessBlobGas was added by EIP-4844 and is ignored in legacy headers.
	ExcessBlobGas *hexutil.Uint64 `json:"excessBlobGas,omitempty"`

	// ParentBeaconRoot was added by EIP-4788 and is ignored in legacy headers.
	ParentBeaconRoot *common.Hash `json:"parentBeaconBlockRoot,omitempty"`

	// untrusted info included by RPC, may have to be checked
	Hash common.Hash `json:"hash"`
}

// checkPostMerge checks that the block header meets all criteria to be a valid ExecutionPayloadHeader,
// see EIP-3675 (block header changes) and EIP-4399 (mixHash usage for prev-randao)
func (hdr *RPCHeader) checkPostMerge() error {
	// TODO: the genesis block has a non-zero difficulty number value.
	// Either this block needs to change, or we special case it. This is not valid w.r.t. EIP-3675.
	if hdr.Number != 0 && (*big.Int)(&hdr.Difficulty).Cmp(common.Big0) != 0 {
		return fmt.Errorf("post-merge block header requires zeroed difficulty field, but got: %s", &hdr.Difficulty)
	}
	if hdr.Nonce != (types.BlockNonce{}) {
		return fmt.Errorf("post-merge block header requires zeroed block nonce field, but got: %s", hdr.Nonce)
	}
	if hdr.BaseFee == nil {
		return fmt.Errorf("post-merge block header requires EIP-1559 base fee field, but got %s", hdr.BaseFee)
	}
	if len(hdr.Extra) > 32 {
		return fmt.Errorf("post-merge block header requires 32 or less bytes of extra data, but got %d", len(hdr.Extra))
	}
	if hdr.UncleHash != types.EmptyUncleHash {
		return fmt.Errorf("post-merge block header requires uncle hash to be of empty uncle list, but got %s", hdr.UncleHash)
	}
	return nil
}

func (hdr *RPCHeader) computeBlockHash() common.Hash {
	gethHeader := hdr.createGethHeader()
	return gethHeader.Hash()
}

func (hdr *RPCHeader) createGethHeader() *types.Header {
	return &types.Header{
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
		// Cancun
		BlobGasUsed:      (*uint64)(hdr.BlobGasUsed),
		ExcessBlobGas:    (*uint64)(hdr.ExcessBlobGas),
		ParentBeaconRoot: hdr.ParentBeaconRoot,
	}
}

func (hdr *RPCHeader) Info(trustCache bool, mustBePostMerge bool) (eth.BlockInfo, error) {
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
	return &headerInfo{hdr.Hash, hdr.createGethHeader()}, nil
}

func (hdr *RPCHeader) BlockID() eth.BlockID {
	return eth.BlockID{
		Hash:   hdr.Hash,
		Number: uint64(hdr.Number),
	}
}

type RPCBlock struct {
	RPCHeader
	Transactions []*types.Transaction `json:"transactions"`
	Withdrawals  *types.Withdrawals   `json:"withdrawals,omitempty"`
}

func (block *RPCBlock) verify() error {
	if computed := block.computeBlockHash(); computed != block.Hash {
		return fmt.Errorf("failed to verify block hash: computed %s but RPC said %s", computed, block.Hash)
	}
	for i, tx := range block.Transactions {
		if tx == nil {
			return fmt.Errorf("block tx %d is nil", i)
		}
	}
	if computed := types.DeriveSha(types.Transactions(block.Transactions), trie.NewStackTrie(nil)); block.TxHash != computed {
		return fmt.Errorf("failed to verify transactions list: computed %s but RPC said %s", computed, block.TxHash)
	}
	if block.WithdrawalsRoot != nil {
		if block.Withdrawals == nil {
			return fmt.Errorf("expected withdrawals")
		}
		for i, w := range *block.Withdrawals {
			if w == nil {
				return fmt.Errorf("block withdrawal %d is null", i)
			}
		}
		if computed := types.DeriveSha(*block.Withdrawals, trie.NewStackTrie(nil)); *block.WithdrawalsRoot != computed {
			return fmt.Errorf("failed to verify withdrawals list: computed %s but RPC said %s", computed, block.WithdrawalsRoot)
		}
	} else {
		if block.Withdrawals != nil {
			return fmt.Errorf("expected no withdrawals due to missing withdrawals-root, but got %d", len(*block.Withdrawals))
		}
	}
	return nil
}

func (block *RPCBlock) Info(trustCache bool, mustBePostMerge bool) (eth.BlockInfo, types.Transactions, error) {
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
	info, err := block.RPCHeader.Info(trustCache, mustBePostMerge)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to verify block from RPC: %w", err)
	}

	return info, block.Transactions, nil
}

func (block *RPCBlock) ExecutionPayloadEnvelope(trustCache bool) (*eth.ExecutionPayloadEnvelope, error) {
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

	payload := &eth.ExecutionPayload{
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
		BaseFeePerGas: eth.Uint256Quantity(baseFee),
		BlockHash:     block.Hash,
		Transactions:  opaqueTxs,
		Withdrawals:   block.Withdrawals,
		BlobGasUsed:   block.BlobGasUsed,
		ExcessBlobGas: block.ExcessBlobGas,
	}

	return &eth.ExecutionPayloadEnvelope{
		ParentBeaconBlockRoot: block.ParentBeaconRoot,
		ExecutionPayload:      payload,
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
		// invalid request, method not found, or invalid params
		if code == -32600 || code == -32601 || code == -32602 {
			return true
		}
	}
	errText := strings.ToLower(err.Error())
	return strings.Contains(errText, "unsupported method") || // alchemy -32600 message
		strings.Contains(errText, "unknown method") ||
		strings.Contains(errText, "invalid param") ||
		strings.Contains(errText, "is not available") ||
		strings.Contains(errText, "rpc method is not whitelisted") // proxyd -32001 error code
}
