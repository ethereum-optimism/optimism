package processors

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/event"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestFailover(t *testing.T) {
	log := testlog.Logger(t, slog.LevelDebug)
	ctx := context.Background()
	chainID := eth.ChainID{1}
	source := &mockSource{}
	processor := &mockProcessor{}
	rewinder := &mockRewinder{}
	chainProc := NewChainProcessor(ctx, log, chainID, processor, rewinder)
	chainProc.AttachEmitter(&mockEmitter{})

	chainProc.AddSource(source)
	require.Equal(t, source, chainProc.activeClient)

	badSource := &mockSource{}
	badSource.l2blockRefFunc = func(ctx context.Context, number uint64) (eth.L2BlockRef, error) {
		return eth.L2BlockRef{}, errors.New("bad source")
	}
	// after adding the second source, the activeClient hasn't changed
	chainProc.AddSource(badSource)
	require.Equal(t, source, chainProc.activeClient)

	// when no error, the activeClient should be unchanged
	chainProc.target = 2
	chainProc.index()
	require.Equal(t, source, chainProc.activeClient)

	// force the activeClient to be the bad source
	chainProc.nextActiveClient()
	require.Equal(t, badSource, chainProc.activeClient)

	// when the bad source errors, the activeClient should be back to the first source
	chainProc.target = 2
	chainProc.index()
	require.Equal(t, source, chainProc.activeClient)
}

type mockEmitter struct{}

func (m *mockEmitter) Emit(ctx context.Context, ev event.Event) {
}

type mockSource struct {
	l2blockRefFunc func(ctx context.Context, number uint64) (eth.L2BlockRef, error)
}

func (m *mockSource) L2BlockRefByNumber(ctx context.Context, number uint64) (eth.L2BlockRef, error) {
	if m.l2blockRefFunc != nil {
		return m.l2blockRefFunc(ctx, number)
	}
	return eth.L2BlockRef{}, nil
}
func (m *mockSource) FetchReceipts(ctx context.Context, blockHash common.Hash) (gethtypes.Receipts, error) {
	return gethtypes.Receipts{}, nil
}

type mockProcessor struct {
}

func (m *mockProcessor) ProcessLogs(ctx context.Context, block eth.BlockRef, receipts gethtypes.Receipts) error {
	return nil
}

type mockRewinder struct {
}

func (m *mockRewinder) Rewind(chain eth.ChainID, headBlock eth.BlockID) error {
	return nil
}
func (m *mockRewinder) LatestBlockNum(chain eth.ChainID) (num uint64, ok bool) {
	return 0, true
}
func (m *mockRewinder) AcceptedBlock(chainID eth.ChainID, id eth.BlockID) error {
	return nil
}
