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

// L2FollowSource provides access to L2 block references for upstream following.
type L2FollowSource interface {
	L2BlockRefByLabel(ctx context.Context, label eth.BlockLabel) (eth.L2BlockRef, error)
	L2BlockRefByNumber(ctx context.Context, num uint64) (eth.L2BlockRef, error)
}

// FollowUpstreamSource combines L1 and L2 follow sources.
// L2 following may be optionally disabled.
type FollowUpstreamSource interface {
	L2FollowSource
	L1FollowSource
	CanFollowL2() bool
}

type L2ELFollowSource struct {
	l2Source *sources.L2Client
	l1Source L1FollowSource
}

var _ FollowUpstreamSource = (*L2ELFollowSource)(nil)

func NewL2ELFollowSource(client *sources.L2Client, l1Source L1FollowSource) *L2ELFollowSource {
	if l1Source == nil {
		panic("NewL2ELFollowSource: l1Source must not be nil")
	}
	return &L2ELFollowSource{l2Source: client, l1Source: l1Source}
}

func (fs *L2ELFollowSource) CanFollowL2() bool {
	return fs.l2Source != nil
}

func (fs *L2ELFollowSource) L2BlockRefByLabel(ctx context.Context, label eth.BlockLabel) (eth.L2BlockRef, error) {
	if fs.l2Source == nil {
		return eth.L2BlockRef{}, errL2FollowSourceNotEnabled
	}
	return fs.l2Source.L2BlockRefByLabel(ctx, label)
}

func (fs *L2ELFollowSource) L2BlockRefByNumber(ctx context.Context, num uint64) (eth.L2BlockRef, error) {
	if fs.l2Source == nil {
		return eth.L2BlockRef{}, errL2FollowSourceNotEnabled
	}
	return fs.l2Source.L2BlockRefByNumber(ctx, num)
}

func (fs *L2ELFollowSource) L1BlockRefByNumber(ctx context.Context, num uint64) (eth.L1BlockRef, error) {
	return fs.l1Source.L1BlockRefByNumber(ctx, num)
}
