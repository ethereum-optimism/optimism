package driver

import (
	"context"
	"errors"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources"
)

var errL2FollowSourceNotEnabled = errors.New("L2 follow source not enabled")

// L1FollowSource provides access to L1 block references for upstream following.
type L1FollowSource interface {
	L1BlockRefByNumber(ctx context.Context, num uint64) (eth.L1BlockRef, error)
}

// UpstreamFollowSource combines L1 and L2 follow sources.
// L2 following may be optionally disabled.
type UpstreamFollowSource interface {
	L1FollowSource
	CanFollowL2() bool
	SafeL2(ctx context.Context) (eth.L2BlockRef, error)
	FinalizedL2(ctx context.Context) (eth.L2BlockRef, error)
	CurrentL1(ctx context.Context) (eth.L1BlockRef, error)
}

type L2FollowSource struct {
	l2Source *sources.FollowClient
	l1Source L1FollowSource
}

var _ UpstreamFollowSource = (*L2FollowSource)(nil)

func NewL2FollowSource(client *sources.FollowClient, l1Source L1FollowSource) *L2FollowSource {
	if l1Source == nil {
		panic("NewL2FollowSource: l1Source must not be nil")
	}
	return &L2FollowSource{l2Source: client, l1Source: l1Source}
}

func (fs *L2FollowSource) CanFollowL2() bool {
	return fs.l2Source != nil
}

func (fs *L2FollowSource) SafeL2(ctx context.Context) (eth.L2BlockRef, error) {
	if fs.l2Source == nil {
		return eth.L2BlockRef{}, errL2FollowSourceNotEnabled
	}
	return fs.l2Source.SafeL2(ctx)
}

func (fs *L2FollowSource) FinalizedL2(ctx context.Context) (eth.L2BlockRef, error) {
	if fs.l2Source == nil {
		return eth.L2BlockRef{}, errL2FollowSourceNotEnabled
	}
	return fs.l2Source.FinalizedL2(ctx)
}

func (fs *L2FollowSource) CurrentL1(ctx context.Context) (eth.L1BlockRef, error) {
	if fs.l2Source == nil {
		return eth.L1BlockRef{}, errL2FollowSourceNotEnabled
	}
	return fs.l2Source.CurrentL1(ctx)
}

func (fs *L2FollowSource) L1BlockRefByNumber(ctx context.Context, num uint64) (eth.L1BlockRef, error) {
	return fs.l1Source.L1BlockRefByNumber(ctx, num)
}
