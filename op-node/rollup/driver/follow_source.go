package driver

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources"
)

// L1FollowSource provides access to L1 block references for upstream following.
type L1FollowSource interface {
	L1BlockRefByNumber(ctx context.Context, num uint64) (eth.L1BlockRef, error)
}

// UpstreamFollowSource combines L1 and L2 follow sources.
// L2 following may be optionally disabled.
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
	if l1Source == nil || client == nil {
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
