package buidl

import (
	"context"
	"math/rand"
	"testing"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/l2"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-node/testlog"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

type mockUnsafeSource struct {
	blocks []*l2.ExecutionPayload
}

func (m *mockUnsafeSource) Block(ctx context.Context, id eth.BlockID) (*l2.ExecutionPayload, error) {
	for _, b := range m.blocks {
		if b.ID() == id {
			return b, nil
		}
	}
	return nil, ethereum.NotFound
}

func (m *mockUnsafeSource) UnsafeBlockIDs(ctx context.Context, max uint64) (out []eth.BlockID, err error) {
	for i, b := range m.blocks {
		if uint64(i) > max {
			return
		}
		out = append(out, b.ID())
	}
	return
}

var _ UnsafeBlocksSource = (*mockUnsafeSource)(nil)

func TestOutput(t *testing.T) {
	// TODO more helper funcs to create mock data for better testing
	head := eth.L1BlockRef{
		Hash:       common.Hash{5},
		Number:     5,
		ParentHash: common.Hash{4},
		Time:       100,
	}
	randomData := func(size int) []byte {
		out := make([]byte, size)
		rand.Read(out[:])
		return out
	}

	// TODO: not exposed, but need that testing util
	var l1Info derive.L1Info // derive.randomL1Info()
	l1InfoTx, err := derive.L1InfoDepositBytes(2, l1Info)
	require.NoError(t, err)
	src := &mockUnsafeSource{blocks: []*l2.ExecutionPayload{
		&l2.ExecutionPayload{
			BlockNumber:  1,
			BlockHash:    common.Hash{1},
			Transactions: []l2.Data{l1InfoTx, randomData(5000), randomData(3000)},
		},
		&l2.ExecutionPayload{
			BlockNumber:  2,
			BlockHash:    common.Hash{2},
			Transactions: []l2.Data{l1InfoTx, randomData(4000)}, // will be partially in previous tx, and part in the next
		},
		&l2.ExecutionPayload{
			BlockNumber:  3,
			BlockHash:    common.Hash{3},
			Transactions: []l2.Data{l1InfoTx, randomData(3000), randomData(3000)},
		},
		&l2.ExecutionPayload{
			BlockNumber:  4,
			BlockHash:    common.Hash{4},
			Transactions: []l2.Data{l1InfoTx, randomData(5000), randomData(6000)},
		},
	}}
	cfg := &rollup.Config{
		Genesis: rollup.Genesis{
			L1:     eth.BlockID{Hash: common.Hash{0xff, 1}},
			L2:     eth.BlockID{Hash: common.Hash{0xff, 2}},
			L2Time: 2,
		},
		// the other fields don't matter in this test
	}
	og := NewChannelEmitter(testlog.Logger(t, log.LvlDebug), cfg, src, head)
	history := map[ChannelID]uint64{}
	maxSize := uint64(10_000)
	maxBlocksPerChannel := uint64(20)

	// produce some outputs
	for i := 0; i < 3; i++ {
		out, err := og.Output(context.Background(), history, maxSize, maxBlocksPerChannel)
		require.NoError(t, err)
		require.Less(t, 1, len(out.Data), "expecting at least a version byte and some frame data")
		require.Less(t, 0, len(out.Channels), "expecting at least one new channel to be opened")
		// update history by merging in the results
		for chID, frameNr := range out.Channels {
			history[chID] = frameNr
		}
	}
}
