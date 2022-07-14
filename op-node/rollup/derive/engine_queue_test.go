package derive

import (
	"context"
	"fmt"
	"io"
	"testing"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

func L2Chain(l2Times []uint64, origin eth.BlockID) []eth.L2BlockRef {
	var out []eth.L2BlockRef
	var parentHash [32]byte
	for i, time := range l2Times {
		hash := [32]byte{byte(i)}
		out = append(out, eth.L2BlockRef{
			Hash:           hash,
			Number:         uint64(i),
			ParentHash:     parentHash,
			Time:           time,
			L1Origin:       origin,
			SequenceNumber: uint64(i),
		})
		parentHash = hash
	}
	return out
}

func (f *fakeL1Fetcher) L1BlockRefByHash(context.Context, common.Hash) (eth.L1BlockRef, error) {
	return f.l1[0], nil
}

func (f *fakeL1Fetcher) Fetch(ctx context.Context, blockHash common.Hash) (eth.L1Info, types.Transactions, types.Receipts, error) {
	return nil, nil, nil, nil
}

func (f *fakeL1Fetcher) InfoAndTxsByHash(ctx context.Context, hash common.Hash) (eth.L1Info, types.Transactions, error) {
	return nil, nil, nil
}

type fakeL2Engine struct {
	l2 []eth.L2BlockRef
}

func (f *fakeL2Engine) L2BlockRefHead(_ context.Context) (eth.L2BlockRef, error) {
	return f.l2[0], nil
}

func (f *fakeL2Engine) L2BlockRefByHash(ctx context.Context, l2Hash common.Hash) (eth.L2BlockRef, error) {
	return f.l2[0], nil
}

func (f *fakeL2Engine) GetPayload(_ context.Context, _ eth.PayloadID) (*eth.ExecutionPayload, error) {
	return nil, nil
}

func (f *fakeL2Engine) ForkchoiceUpdate(ctx context.Context, state *eth.ForkchoiceState, attr *eth.PayloadAttributes) (*eth.ForkchoiceUpdatedResult, error) {
	return nil, nil
}

func (f *fakeL2Engine) NewPayload(ctx context.Context, payload *eth.ExecutionPayload) (*eth.PayloadStatusV1, error) {
	return nil, nil
}

func (f *fakeL2Engine) PayloadByHash(context.Context, common.Hash) (*eth.ExecutionPayload, error) {
	return nil, nil
}

func (f *fakeL2Engine) PayloadByNumber(context.Context, uint64) (*eth.ExecutionPayload, error) {
	return nil, nil
}

func TestResetStepGenesis(t *testing.T) {
	log := testlog.Logger(t, log.LvlTrace)
	cfg := &rollup.Config{
		Genesis: rollup.Genesis{
			L2Time: 10,
		},
		BlockTime:         2,
		MaxSequencerDrift: 600,
		SeqWindowSize:     30,
	}

	emptyL2Ref := eth.L2BlockRef{}

	// setup l1 and l2 chain info to stage some genesis step from the l2
	l1 := L1Chain([]uint64{10})
	l2 := L2Chain([]uint64{10}, l1[0].ID())
	fetcher := fakeL1Fetcher{l1: l1}
	engine := fakeL2Engine{l2: l2}

	// make and configure engine queue
	engineQueue := NewEngineQueue(log, cfg, &engine)
	require.Equal(t, engineQueue.safeHead, emptyL2Ref)
	require.Equal(t, engineQueue.unsafeHead, emptyL2Ref)

	// run reset step
	engineQueue.ResetStep(context.Background(), &fetcher)

	require.Equal(t, engineQueue.unsafeHead, l2[0])
	require.Equal(t, engineQueue.safeHead, l2[0])

	engineQueue.resetting = true
	ret := engineQueue.ResetStep(context.Background(), &fetcher)

	require.Equal(t, engineQueue.unsafeHead, l2[0])
	require.Equal(t, engineQueue.safeHead, l2[0])
	require.Equal(t, ret, io.EOF)
}

func TestResetStepGenesisFailure(t *testing.T) {
	log := testlog.Logger(t, log.LvlTrace)
	cfg := &rollup.Config{
		Genesis: rollup.Genesis{
			L2Time: 10,
		},
		BlockTime:         2,
		MaxSequencerDrift: 600,
		SeqWindowSize:     30,
	}

	emptyL2Ref := eth.L2BlockRef{}

	// setup l1 and l2 chain info to stage some genesis step from the l2
	l1 := L1Chain([]uint64{15})
	l2 := L2Chain([]uint64{20}, l1[0].ID())
	fetcher := fakeL1Fetcher{l1: l1}
	engine := fakeL2Engine{l2: l2}

	// make and configure engine queue
	engineQueue := NewEngineQueue(log, cfg, &engine)
	require.Equal(t, engineQueue.safeHead, emptyL2Ref)
	require.Equal(t, engineQueue.unsafeHead, emptyL2Ref)

	// run reset step
	engineQueue.ResetStep(context.Background(), &fetcher)

	require.Equal(t, engineQueue.unsafeHead, l2[0])
	require.Equal(t, engineQueue.safeHead, l2[0])

	engineQueue.resetting = true
	ret := engineQueue.ResetStep(context.Background(), &fetcher)

	require.Equal(t, engineQueue.unsafeHead, l2[0])
	require.Equal(t, engineQueue.safeHead, l2[0])
	require.Equal(t, ret, fmt.Errorf("cannot reset block derivation to start at L2 block 0x0000000000000000000000000000000000000000000000000000000000000000:0 with time 20 older than its L1 origin 0x0000000000000000000000000000000000000000000000000000000000000000:0 with time 15, time invariant is broken"))
}
