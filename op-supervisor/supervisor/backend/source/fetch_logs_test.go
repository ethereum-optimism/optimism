package source

import (
	"context"
	"errors"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestFetchLogs(t *testing.T) {
	ctx := context.Background()
	rcpts := types.Receipts{&types.Receipt{Type: 3}, &types.Receipt{Type: 4}}

	t.Run("Success", func(t *testing.T) {
		client := &stubLogSource{
			rcpts: rcpts,
		}
		var processed []types.Receipts
		processor := ReceiptProcessorFn(func(ctx context.Context, block eth.L1BlockRef, rcpts types.Receipts) error {
			processed = append(processed, rcpts)
			return nil
		})
		fetcher := newLogFetcher(client, processor)
		block := eth.L1BlockRef{Number: 11, Hash: common.Hash{0xaa}}

		err := fetcher.ProcessBlock(ctx, block)
		require.NoError(t, err)

		require.Equal(t, []types.Receipts{rcpts}, processed)
	})

	t.Run("ReceiptFetcherError", func(t *testing.T) {
		client := &stubLogSource{
			err: errors.New("boom"),
		}
		processor := ReceiptProcessorFn(func(ctx context.Context, block eth.L1BlockRef, rcpts types.Receipts) error {
			t.Fatal("should not be called")
			return nil
		})
		fetcher := newLogFetcher(client, processor)
		block := eth.L1BlockRef{Number: 11, Hash: common.Hash{0xaa}}

		err := fetcher.ProcessBlock(ctx, block)
		require.ErrorIs(t, err, client.err)
	})

	t.Run("ProcessorError", func(t *testing.T) {
		expectedErr := errors.New("boom")
		client := &stubLogSource{
			rcpts: rcpts,
		}
		processor := ReceiptProcessorFn(func(ctx context.Context, block eth.L1BlockRef, rcpts types.Receipts) error {
			return expectedErr
		})
		fetcher := newLogFetcher(client, processor)
		block := eth.L1BlockRef{Number: 11, Hash: common.Hash{0xaa}}

		err := fetcher.ProcessBlock(ctx, block)
		require.ErrorIs(t, err, expectedErr)
	})
}

type stubLogSource struct {
	err   error
	rcpts types.Receipts
}

func (s *stubLogSource) FetchReceipts(_ context.Context, _ common.Hash) (eth.BlockInfo, types.Receipts, error) {
	if s.err != nil {
		return nil, nil, s.err
	}
	return nil, s.rcpts, nil
}
