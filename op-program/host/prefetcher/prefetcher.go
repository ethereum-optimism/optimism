package prefetcher

import (
	"errors"
	"fmt"
	"strings"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-program/client/l1"
	"github.com/ethereum-optimism/optimism/op-program/client/l2"
	"github.com/ethereum-optimism/optimism/op-program/client/mpt"
	"github.com/ethereum-optimism/optimism/op-program/host/kvstore"
	"github.com/ethereum-optimism/optimism/op-program/preimage"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
)

type Prefetcher struct {
	l1Fetcher l1.Oracle
	l2Fetcher l2.Oracle
	lastHint  string
	kvStore   kvstore.KV
}

func NewPrefetcher(l1Fetcher l1.Oracle, l2Fetcher l2.Oracle, kvStore kvstore.KV) *Prefetcher {
	return &Prefetcher{
		l1Fetcher: l1Fetcher,
		l2Fetcher: l2Fetcher,
		kvStore:   kvStore,
	}
}

func (p *Prefetcher) Hint(hint string) error {
	p.lastHint = hint
	return nil
}

func (p *Prefetcher) GetPreimage(key common.Hash) ([]byte, error) {
	pre, err := p.kvStore.Get(key)
	if errors.Is(err, kvstore.ErrNotFound) && p.lastHint != "" {
		hint := p.lastHint
		p.lastHint = ""
		if err := p.prefetch(hint); err != nil {
			return nil, fmt.Errorf("prefetch failed: %w", err)
		}
		// Should now be available
		return p.kvStore.Get(key)
	}
	return pre, err
}

func (p *Prefetcher) prefetch(hint string) error {
	hintType, hash, err := parseHint(hint)
	if err != nil {
		return err
	}
	switch hintType {
	case l1.HintL1BlockHeader:
		header := p.l1Fetcher.HeaderByBlockHash(hash)
		data, err := header.HeaderRLP()
		if err != nil {
			return fmt.Errorf("marshall header: %w", err)
		}

		return p.kvStore.Put(preimage.Keccak256Key(hash).PreimageKey(), data)
	case l1.HintL1Transactions:
		_, txs := p.l1Fetcher.TransactionsByBlockHash(hash)
		return p.storeTransactions(txs)
	case l1.HintL1Receipts:
		_, rcpts := p.l1Fetcher.ReceiptsByBlockHash(hash)
		opaqueRcpts, err := eth.EncodeReceipts(rcpts)
		if err != nil {
			return err
		}
		return p.storeTrieNodes(opaqueRcpts)
	case l2.HintL2BlockHeader:
		// Pre-fetch both block and transactions
		block := p.l2Fetcher.BlockByHash(hash)
		data, err := rlp.EncodeToBytes(block.Header())
		if err != nil {
			return fmt.Errorf("marshall header: %w", err)
		}
		err = p.kvStore.Put(preimage.Keccak256Key(hash).PreimageKey(), data)
		if err != nil {
			return err
		}
		return p.storeTransactions(block.Transactions())
	case l2.HintL2StateNode:
		node := p.l2Fetcher.NodeByHash(hash)
		return p.kvStore.Put(preimage.Keccak256Key(hash).PreimageKey(), node)
	case l2.HintL2Code:
		code := p.l2Fetcher.CodeByHash(hash)
		return p.kvStore.Put(preimage.Keccak256Key(hash).PreimageKey(), code)
	}
	return fmt.Errorf("unknown hint type: %v", hintType)
}

func (p *Prefetcher) storeTransactions(txs types.Transactions) error {
	opaqueTxs, err := eth.EncodeTransactions(txs)
	if err != nil {
		return err
	}
	return p.storeTrieNodes(opaqueTxs)
}

func (p *Prefetcher) storeTrieNodes(values []hexutil.Bytes) error {
	_, nodes := mpt.WriteTrie(values)
	for _, node := range nodes {
		err := p.kvStore.Put(preimage.Keccak256Key(crypto.Keccak256Hash(node)).PreimageKey(), node)
		if err != nil {
			return fmt.Errorf("failed to store node: %w", err)
		}
	}
	return nil
}

// parseHint parses a hint string in wire protocol. Returns the hint type, requested hash and error (if any).
func parseHint(hint string) (string, common.Hash, error) {
	hintType, hashStr, found := strings.Cut(hint, " ")
	if !found {
		return "", common.Hash{}, fmt.Errorf("unsupported hint: %s", hint)
	}
	hash := common.HexToHash(hashStr)
	if hash == (common.Hash{}) {
		return "", common.Hash{}, fmt.Errorf("invalid hash: %s", hashStr)
	}
	return hintType, hash, nil
}
