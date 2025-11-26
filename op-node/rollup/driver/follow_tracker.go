package driver

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources"
)

type L1SourceForL2FollowTracker interface {
	L1BlockRefByNumber(ctx context.Context, num uint64) (eth.L1BlockRef, error)
}

type ELFollowTracker struct {
	l2Source *sources.L2Client
	l1Source L1SourceForL2FollowTracker
}

func NewELFollowTracker(client *sources.L2Client, l1Source L1SourceForL2FollowTracker) *ELFollowTracker {
	return &ELFollowTracker{l2Source: client, l1Source: l1Source}
}

func (ft *ELFollowTracker) L2BlockRefByLabel(ctx context.Context, label eth.BlockLabel) (eth.L2BlockRef, error) {
	return ft.l2Source.L2BlockRefByLabel(ctx, label)
}

func (ft *ELFollowTracker) L2BlockRefByNumber(ctx context.Context, num uint64) (eth.L2BlockRef, error) {
	return ft.l2Source.L2BlockRefByNumber(ctx, num)
}

func (ft *ELFollowTracker) L1BlockRefByNumber(ctx context.Context, num uint64) (eth.L1BlockRef, error) {
	return ft.l1Source.L1BlockRefByNumber(ctx, num)
}
