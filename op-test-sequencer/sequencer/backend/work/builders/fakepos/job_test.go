package fakepos

import (
	"context"
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
)

type forkHeaders struct {
	canonical map[int64]*types.Header
	byHash    map[common.Hash]*types.Header
}

func (f *forkHeaders) HeaderByNumber(_ context.Context, n *big.Int) (*types.Header, error) {
	number := int64(rpc.LatestBlockNumber)
	if n != nil {
		number = n.Int64()
	}
	if h, ok := f.canonical[number]; ok {
		return h, nil
	}
	return nil, fmt.Errorf("unknown header %d", number)
}

func (f *forkHeaders) HeaderByHash(_ context.Context, hash common.Hash) (*types.Header, error) {
	if h, ok := f.byHash[hash]; ok {
		return h, nil
	}
	return nil, fmt.Errorf("unknown header %s", hash)
}

func TestForkchoiceSafetyUsesSelectedParentAncestry(t *testing.T) {
	chain := &forkHeaders{canonical: make(map[int64]*types.Header), byHash: make(map[common.Hash]*types.Header)}
	parent := common.Hash{}
	for n := int64(0); n <= 25; n++ {
		h := &types.Header{Number: big.NewInt(n), ParentHash: parent, Extra: []byte("original")}
		chain.canonical[n], chain.byHash[h.Hash()], parent = h, h, h.Hash()
	}
	chain.canonical[int64(rpc.LatestBlockNumber)] = chain.canonical[25]
	chain.canonical[int64(rpc.SafeBlockNumber)] = chain.canonical[15]
	chain.canonical[int64(rpc.FinalizedBlockNumber)] = chain.canonical[3]
	parent = chain.canonical[5].Hash()
	var head, safe *types.Header
	for n := int64(6); n <= 23; n++ {
		h := &types.Header{Number: big.NewInt(n), ParentHash: parent, Extra: []byte("replacement")}
		chain.byHash[h.Hash()], parent, head = h, h.Hash(), h
		if n == 13 {
			safe = h
		}
	}
	job := &Job{parent: head.Hash(), b: &Builder{blockchain: chain, genesis: chain.canonical[0], safeDistance: 10, finalizedDistance: 20}}
	job.setHeadSafeAndFinalized()
	require.Equal(t, head.Hash(), job.head.Hash())
	require.Equal(t, safe.Hash(), job.safe.Hash(), "safe must be an ancestor of the selected fork, not the old canonical block at that height")
	require.Equal(t, chain.canonical[3].Hash(), job.finalized.Hash())
}
