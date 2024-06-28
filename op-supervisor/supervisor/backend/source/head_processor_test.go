package source

import (
	"context"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

func TestHeadUpdateProcessor(t *testing.T) {
	t.Run("NotifyUnsafeHeadProcessors", func(t *testing.T) {
		logger := testlog.Logger(t, log.LvlInfo)
		processed := make([]eth.L1BlockRef, 3)
		makeProcessor := func(idx int) HeadProcessor {
			return HeadProcessorFn(func(_ context.Context, head eth.L1BlockRef) {
				processed[idx] = head
			})
		}
		headUpdates := newHeadUpdateProcessor(logger, []HeadProcessor{makeProcessor(0), makeProcessor(1), makeProcessor(2)}, nil, nil)
		block := eth.L1BlockRef{Number: 110, Hash: common.Hash{0xaa}}
		headUpdates.OnNewUnsafeHead(context.Background(), block)
		require.Equal(t, []eth.L1BlockRef{block, block, block}, processed)
	})

	t.Run("NotifySafeHeadProcessors", func(t *testing.T) {
		logger := testlog.Logger(t, log.LvlInfo)
		processed := make([]eth.L1BlockRef, 3)
		makeProcessor := func(idx int) HeadProcessor {
			return HeadProcessorFn(func(_ context.Context, head eth.L1BlockRef) {
				processed[idx] = head
			})
		}
		headUpdates := newHeadUpdateProcessor(logger, nil, []HeadProcessor{makeProcessor(0), makeProcessor(1), makeProcessor(2)}, nil)
		block := eth.L1BlockRef{Number: 110, Hash: common.Hash{0xaa}}
		headUpdates.OnNewSafeHead(context.Background(), block)
		require.Equal(t, []eth.L1BlockRef{block, block, block}, processed)
	})

	t.Run("NotifyFinalizedHeadProcessors", func(t *testing.T) {
		logger := testlog.Logger(t, log.LvlInfo)
		processed := make([]eth.L1BlockRef, 3)
		makeProcessor := func(idx int) HeadProcessor {
			return HeadProcessorFn(func(_ context.Context, head eth.L1BlockRef) {
				processed[idx] = head
			})
		}
		headUpdates := newHeadUpdateProcessor(logger, nil, nil, []HeadProcessor{makeProcessor(0), makeProcessor(1), makeProcessor(2)})
		block := eth.L1BlockRef{Number: 110, Hash: common.Hash{0xaa}}
		headUpdates.OnNewFinalizedHead(context.Background(), block)
		require.Equal(t, []eth.L1BlockRef{block, block, block}, processed)
	})
}
