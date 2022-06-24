package derive

import (
	"context"
	"math/rand"
	"testing"

	"github.com/ethereum-optimism/optimism/op-node/testutils"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/testlog"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

type mockUnsafeSource struct {
	blocks []*eth.ExecutionPayload
}

func (m *mockUnsafeSource) PayloadByHash(ctx context.Context, hash common.Hash) (*eth.ExecutionPayload, error) {
	for _, b := range m.blocks {
		if b.BlockHash == hash {
			return b, nil
		}
	}
	return nil, ethereum.NotFound
}

func (m *mockUnsafeSource) UnsafeBlockIDs(ctx context.Context, safeHead eth.BlockID, max uint64) (out []eth.BlockID, err error) {
	for _, b := range m.blocks {
		if uint64(len(out)) >= max {
			return
		}
		if uint64(b.BlockNumber) < safeHead.Number {
			continue
		}
		out = append(out, b.ID())
	}
	return
}

var _ UnsafeBlocksSource = (*mockUnsafeSource)(nil)

func TestOutput(t *testing.T) {
	// TODO more helper funcs to create mock data for better testing
	randomData := func(size int) []byte {
		out := make([]byte, size)
		rand.Read(out[:])
		return out
	}

	rng := rand.New(rand.NewSource(1234))
	randInfoTx := func() []byte {
		l1Info := testutils.RandomL1Info(rng)
		l1InfoTx, err := L1InfoDepositBytes(rng.Uint64(), l1Info)
		require.NoError(t, err)
		return l1InfoTx
	}
	src := &mockUnsafeSource{blocks: []*eth.ExecutionPayload{
		&eth.ExecutionPayload{
			BlockNumber:  1,
			BlockHash:    common.Hash{1},
			Transactions: []eth.Data{randInfoTx(), randomData(5000), randomData(3000)},
		},
		&eth.ExecutionPayload{
			BlockNumber:  2,
			BlockHash:    common.Hash{2},
			Transactions: []eth.Data{randInfoTx(), randomData(4000)}, // will be partially in previous tx, and part in the next
		},
		&eth.ExecutionPayload{
			BlockNumber:  3,
			BlockHash:    common.Hash{3},
			Transactions: []eth.Data{randInfoTx(), randomData(3000), randomData(3000)},
		},
		&eth.ExecutionPayload{
			BlockNumber:  4,
			BlockHash:    common.Hash{4},
			Transactions: []eth.Data{randInfoTx(), randomData(5000), randomData(6000)},
		},
	}}
	cfg := &rollup.Config{
		Genesis: rollup.Genesis{
			L1:     eth.BlockID{Hash: common.Hash{0xff, 1}},
			L2:     eth.BlockID{Hash: common.Hash{0xff, 2}},
			L2Time: 2,
		},
		ChannelTimeout: 20,
		// the other fields don't matter in this test
	}
	og := NewChannelEmitter(testlog.Logger(t, log.LvlDebug), cfg, src)

	l1Time := uint64(123)
	og.SetL1Time(l1Time)

	history := map[ChannelID]uint64{}
	minSize := uint64(1000) // TODO min size param
	maxSize := uint64(10_000)
	maxBlocksPerChannel := uint64(20)

	// produce some outputs
	for i := 0; i < 3; i++ {
		out, err := og.Output(context.Background(), history, minSize, maxSize, maxBlocksPerChannel)
		require.NoError(t, err)
		require.Less(t, 0, len(out.Channels), "expecting at least one new channel to be opened")
		// update history by merging in the results
		for chID, frameNr := range out.Channels {
			require.Equal(t, chID.Time, l1Time)
			history[chID] = frameNr
		}
	}
}
