package l1

import (
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/trie"
)

// Note: we do this ugly typing because we want the best, and the standard bindings are not sufficient:
// - batched calls of many block requests (standard bindings do extra uncle-header fetches, cannot be batched nicely)
// - ignore uncle data (does not even exist anymore post-Merge)
// - use cached block hash, if we trust the RPC.
// - verify transactions list matches tx-root, to ensure consistency with block-hash, if we do not trust the RPC
//
// Transaction-sender data from the RPC is not cached, since ethclient.setSenderFromServer is private,
// and we only need to compute the sender for transactions into the inbox.
//
// This way we minimize RPC calls, enable batching, and can choose to verify what the RPC gives us.

type HeaderInfo struct {
	hash        common.Hash
	parentHash  common.Hash
	root        common.Hash
	number      uint64
	time        uint64
	mixDigest   common.Hash // a.k.a. the randomness field post-merge.
	baseFee     *big.Int
	txHash      common.Hash
	receiptHash common.Hash
}

var _ eth.L1Info = (*HeaderInfo)(nil)

func (info *HeaderInfo) Hash() common.Hash {
	return info.hash
}

func (info *HeaderInfo) ParentHash() common.Hash {
	return info.parentHash
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

func (info *HeaderInfo) BlockRef() eth.L1BlockRef {
	return eth.L1BlockRef{
		Hash:       info.hash,
		Number:     info.number,
		ParentHash: info.parentHash,
		Time:       info.time,
	}
}

func (info *HeaderInfo) ReceiptHash() common.Hash {
	return info.receiptHash
}

type rpcHeaderCacheInfo struct {
	Hash common.Hash `json:"hash"`
}

type rpcHeader struct {
	cache  rpcHeaderCacheInfo
	header types.Header
}

func (header *rpcHeader) UnmarshalJSON(msg []byte) error {
	if err := json.Unmarshal(msg, &header.header); err != nil {
		return err
	}
	return json.Unmarshal(msg, &header.cache)
}

func (header *rpcHeader) Info(trustCache bool) (*HeaderInfo, error) {
	info := HeaderInfo{
		hash:        header.cache.Hash,
		parentHash:  header.header.ParentHash,
		root:        header.header.Root,
		number:      header.header.Number.Uint64(),
		time:        header.header.Time,
		mixDigest:   header.header.MixDigest,
		baseFee:     header.header.BaseFee,
		txHash:      header.header.TxHash,
		receiptHash: header.header.ReceiptHash,
	}
	if !trustCache {
		if computed := header.header.Hash(); computed != info.hash {
			return nil, fmt.Errorf("failed to verify block hash: computed %s but RPC said %s", computed, info.hash)
		}
	}
	return &info, nil
}

type rpcBlockCacheInfo struct {
	Transactions []*types.Transaction `json:"transactions"`
}

type rpcBlock struct {
	header rpcHeader
	extra  rpcBlockCacheInfo
}

func (block *rpcBlock) UnmarshalJSON(msg []byte) error {
	if err := json.Unmarshal(msg, &block.header); err != nil {
		return err
	}
	return json.Unmarshal(msg, &block.extra)
}

func (block *rpcBlock) Info(trustCache bool) (*HeaderInfo, types.Transactions, error) {
	// verify the header data
	info, err := block.header.Info(trustCache)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to verify block from RPC: %v", err)
	}

	if !trustCache { // verify the list of transactions matches the tx-root
		hasher := trie.NewStackTrie(nil)
		computed := types.DeriveSha(types.Transactions(block.extra.Transactions), hasher)
		if expected := info.txHash; expected != computed {
			return nil, nil, fmt.Errorf("failed to verify transactions list: expected transactions root %s but retrieved %s", expected, computed)
		}
	}
	return info, block.extra.Transactions, nil
}
