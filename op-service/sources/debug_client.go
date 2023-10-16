package sources

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/rawdb"
)

type DebugClient struct {
	callContext CallContextFn
}

func NewDebugClient(callContext CallContextFn) *DebugClient {
	return &DebugClient{callContext}
}

func (o *DebugClient) NodeByHash(ctx context.Context, hash common.Hash) ([]byte, error) {
	// MPT nodes are stored as the hash of the node (with no prefix)
	node, err := o.dbGet(ctx, hash[:])
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve state MPT node: %w", err)
	}
	return node, nil
}

func (o *DebugClient) CodeByHash(ctx context.Context, hash common.Hash) ([]byte, error) {
	// First try retrieving with the new code prefix
	code, err := o.dbGet(ctx, append(append(make([]byte, 0), rawdb.CodePrefix...), hash[:]...))
	if err != nil {
		// Fallback to the legacy un-prefixed version
		code, err = o.dbGet(ctx, hash[:])
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve contract code, using new and legacy keys, with codehash %s: %w", hash, err)
		}
	}
	return code, nil
}

func (o *DebugClient) dbGet(ctx context.Context, key []byte) ([]byte, error) {
	var node hexutil.Bytes
	err := o.callContext(ctx, &node, "debug_dbGet", hexutil.Encode(key))
	if err != nil {
		return nil, fmt.Errorf("fetch error %x: %w", key, err)
	}
	return node, nil
}
