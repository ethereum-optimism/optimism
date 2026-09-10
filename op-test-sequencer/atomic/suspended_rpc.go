package atomic

import (
	"context"
	"fmt"
	"sync"

	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/trie"
)

// RPCPrefixSnapshot reads a retained, executed prefix block by hash. The caller
// must ensure its end state is the intended pre-transaction state: no unaccounted
// post-block mutations, transactions or system calls may intervene. This is not
// an adapter from a parent block to the next block's prefix.
// The trusted local RPC must retain this state until BuildSuspended returns.
type RPCPrefixSnapshot struct {
	rpc       client.RPC
	header    *types.Header
	ancestors map[uint64]common.Hash
	mu        sync.Mutex
}

func NewRPCPrefixSnapshot(rpcClient client.RPC, header *types.Header) (*RPCPrefixSnapshot, error) {
	if rpcClient == nil || header == nil || header.Number == nil || !header.Number.IsUint64() || header.Number.Sign() <= 0 {
		return nil, fmt.Errorf("invalid pinned prefix header")
	}
	return &RPCPrefixSnapshot{rpc: rpcClient, header: types.CopyHeader(header), ancestors: map[uint64]common.Hash{bigs.Uint64Strict(header.Number) - 1: header.ParentHash}}, nil
}

func (s *RPCPrefixSnapshot) Account(ctx context.Context, address common.Address) (*PrefixAccount, error) {
	var proof eth.AccountResult
	pin := rpc.BlockNumberOrHashWithHash(s.header.Hash(), false)
	if err := s.rpc.CallContext(ctx, &proof, "eth_getProof", address, []common.Hash{}, pin); err != nil {
		return nil, err
	}
	// Decode the authenticated account itself, including non-existence; inferring
	// existence from balance/nonce/code loses existing empty accounts.
	db := memorydb.New()
	for _, node := range proof.AccountProof {
		if err := db.Put(crypto.Keccak256(node), node); err != nil {
			return nil, err
		}
	}
	encoded, err := trie.VerifyProof(s.header.Root, crypto.Keccak256(address[:]), db)
	if err != nil {
		return nil, err
	}
	if encoded == nil {
		return nil, nil
	}
	var account types.StateAccount
	if err := rlp.DecodeBytes(encoded, &account); err != nil {
		return nil, err
	}
	var code hexutil.Bytes
	if err := s.rpc.CallContext(ctx, &code, "eth_getCode", address, pin); err != nil {
		return nil, err
	}
	if crypto.Keccak256Hash(code) != common.BytesToHash(account.CodeHash) {
		return nil, fmt.Errorf("pinned account code hash mismatch")
	}
	return &PrefixAccount{Balance: (*hexutil.Big)(account.Balance.ToBig()), Nonce: account.Nonce, Code: code}, nil
}
func (s *RPCPrefixSnapshot) Storage(ctx context.Context, address common.Address, index common.Hash) (common.Hash, error) {
	var value common.Hash
	err := s.rpc.CallContext(ctx, &value, "eth_getStorageAt", address, index, rpc.BlockNumberOrHashWithHash(s.header.Hash(), false))
	return value, err
}
func (s *RPCPrefixSnapshot) BlockHash(ctx context.Context, number uint64) (common.Hash, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	height := bigs.Uint64Strict(s.header.Number)
	if number >= height || height-number > 256 {
		return common.Hash{}, nil
	}
	for n := height - 1; n > number; n-- {
		if _, ok := s.ancestors[n-1]; ok {
			continue
		}
		var h *types.Header
		if err := s.rpc.CallContext(ctx, &h, "eth_getBlockByHash", s.ancestors[n], false); err != nil {
			return common.Hash{}, err
		}
		if h == nil || h.Number == nil || !h.Number.IsUint64() || bigs.Uint64Strict(h.Number) != n || h.Hash() != s.ancestors[n] {
			return common.Hash{}, fmt.Errorf("missing pinned ancestor")
		}
		s.ancestors[n-1] = h.ParentHash
	}
	return s.ancestors[number], nil
}
