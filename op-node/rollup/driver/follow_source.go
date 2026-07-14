package driver

import (
	"context"
	"fmt"
	"sync"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"golang.org/x/sync/errgroup"
)

// L1FollowSource provides access to L1 block references for upstream following.
type L1FollowSource interface {
	L1BlockRefByNumber(ctx context.Context, num uint64) (eth.L1BlockRef, error)
}

// UpstreamFollowSource combines L1 and L2 follow sources.
type UpstreamFollowSource interface {
	L1FollowSource
	GetFollowStatus(ctx context.Context) (*sources.FollowStatus, error)
}

type L2FollowSource struct {
	l2Source *sources.FollowClient
	l1Source L1FollowSource
}

var _ UpstreamFollowSource = (*L2FollowSource)(nil)

func NewL2FollowSource(client *sources.FollowClient, l1Source L1FollowSource) *L2FollowSource {
	if client == nil || l1Source == nil {
		panic("NewL2FollowSource: sources must not be nil")
	}
	return &L2FollowSource{l2Source: client, l1Source: l1Source}
}

func (fs *L2FollowSource) GetFollowStatus(ctx context.Context) (*sources.FollowStatus, error) {
	return fs.l2Source.GetFollowStatus(ctx)
}

func (fs *L2FollowSource) L1BlockRefByNumber(ctx context.Context, num uint64) (eth.L1BlockRef, error) {
	return fs.l1Source.L1BlockRefByNumber(ctx, num)
}

// fetchL1BlockRefs fetches each unique L1 block number concurrently.
func fetchL1BlockRefs(ctx context.Context, source L1FollowSource, numbers ...uint64) (map[uint64]eth.L1BlockRef, error) {
	unique := make(map[uint64]struct{}, len(numbers))
	for _, number := range numbers {
		unique[number] = struct{}{}
	}

	refs := make(map[uint64]eth.L1BlockRef, len(unique))
	var mu sync.Mutex
	group, groupCtx := errgroup.WithContext(ctx)
	for number := range unique {
		number := number
		group.Go(func() error {
			ref, err := source.L1BlockRefByNumber(groupCtx, number)
			if err != nil {
				return fmt.Errorf("fetch L1 block %d: %w", number, err)
			}
			mu.Lock()
			refs[number] = ref
			mu.Unlock()
			return nil
		})
	}
	if err := group.Wait(); err != nil {
		return nil, err
	}
	return refs, nil
}
