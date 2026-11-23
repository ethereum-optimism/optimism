package node

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	optypes "github.com/ethereum-optimism/optimism/op-core/types"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

// blockingReceiptsSource is an L1 source whose receipt fetches block until released.
type blockingReceiptsSource struct {
	L1Source
	fetches chan receiptsFetch
	release chan struct{}
}

type receiptsFetch struct {
	hash        common.Hash
	hasDeadline bool
}

func (s *blockingReceiptsSource) FetchReceipts(ctx context.Context, blockHash common.Hash) (eth.BlockInfo, optypes.Receipts, error) {
	_, hasDeadline := ctx.Deadline()
	s.fetches <- receiptsFetch{hash: blockHash, hasDeadline: hasDeadline}
	<-s.release
	return nil, nil, nil
}

func TestPrefetchL1Receipts(t *testing.T) {
	const heads = 100
	src := &blockingReceiptsSource{
		fetches: make(chan receiptsFetch, heads),
		release: make(chan struct{}),
	}
	n := &OpNode{log: testlog.Logger(t, log.LevelError), l1Source: src}

	// Only the first of many L1 heads that arrive while a fetch is running is fetched,
	// and its fetch has a deadline.
	for i := range heads {
		n.prefetchL1Receipts(context.Background(), eth.L1BlockRef{Hash: common.Hash{byte(i)}})
	}
	close(src.release)
	n.l1ReceiptsPrefetchGroup.Wait()
	require.Len(t, src.fetches, 1)
	require.Equal(t, receiptsFetch{hash: common.Hash{0}, hasDeadline: true}, <-src.fetches)

	// A head that arrives after the fetch completed is fetched again.
	n.prefetchL1Receipts(context.Background(), eth.L1BlockRef{Hash: common.Hash{0xaa}})
	n.l1ReceiptsPrefetchGroup.Wait()
	require.Len(t, src.fetches, 1)
	require.Equal(t, common.Hash{0xaa}, (<-src.fetches).hash)
}
