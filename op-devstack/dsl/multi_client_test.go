package dsl

import (
	"context"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

type mockHeaderProvider struct {
	latestBlockNum int
	blocksByNum    map[int]*testutils.MockBlockInfo
}

func (m mockHeaderProvider) InfoByNumber(ctx context.Context, number uint64) (eth.BlockInfo, error) {
	var idx int
	if number == 0 {
		idx = m.latestBlockNum
	} else {
		idx = int(number)
	}
	block, exists := m.blocksByNum[idx]
	if !exists {
		return nil, nil
	}
	return block, nil
}

func (m mockHeaderProvider) InfoByLabel(ctx context.Context, label eth.BlockLabel) (eth.BlockInfo, error) {
	return m.InfoByNumber(ctx, uint64(m.latestBlockNum))
}

func (m mockHeaderProvider) InfoByHash(ctx context.Context, hash common.Hash) (eth.BlockInfo, error) {
	return m.InfoByNumber(ctx, uint64(m.latestBlockNum))
}

func TestDetectsFork(t *testing.T) {
	leader := mockHeaderProvider{latestBlockNum: 0, blocksByNum: map[int]*testutils.MockBlockInfo{
		0: {InfoHash: common.HexToHash("0x0"), InfoNum: 0},
		1: {InfoHash: common.HexToHash("0x1"), InfoNum: 1},
	}}

	followerA := mockHeaderProvider{latestBlockNum: 0, blocksByNum: map[int]*testutils.MockBlockInfo{
		0: {InfoHash: common.HexToHash("0x0"), InfoNum: 0}, // in sync with leader
		1: {InfoHash: common.HexToHash("0xb"), InfoNum: 1}, // forks off from leader
	}}

	followerB := mockHeaderProvider{latestBlockNum: 0, blocksByNum: map[int]*testutils.MockBlockInfo{
		0: {InfoHash: common.HexToHash("0x0"), InfoNum: 0}, // forks off from leader
		1: {InfoHash: common.HexToHash("0xb"), InfoNum: 1}, // forks off from leader
	}}

	// First scenario: leader and follower are in sync initially, but then split
	secondCheck, firstErr := checkForChainFork(context.Background(), []HeaderProvider{&leader, &followerA}, testlog.Logger(t, log.LevelDebug))
	require.NoError(t, firstErr)
	leader.latestBlockNum = 1    // advance the chain head
	followerA.latestBlockNum = 1 // advance the chain head
	require.Error(t, secondCheck(false), "expected chain split error")

	// Second scenario: leader and follower are forked immediately
	_, firstErr = checkForChainFork(context.Background(), []HeaderProvider{&leader, &followerB}, testlog.Logger(t, log.LevelDebug))
	require.Error(t, firstErr, "expected chain split error")
}

func TestDetectsHealthy(t *testing.T) {
	leader := mockHeaderProvider{latestBlockNum: 0, blocksByNum: map[int]*testutils.MockBlockInfo{
		0: {InfoHash: common.HexToHash("0x0"), InfoNum: 0},
		1: {InfoHash: common.HexToHash("0x1"), InfoNum: 1},
	}}

	followerA := mockHeaderProvider{latestBlockNum: 0, blocksByNum: map[int]*testutils.MockBlockInfo{
		0: {InfoHash: common.HexToHash("0x0"), InfoNum: 0}, // in sync with leader
		1: {InfoHash: common.HexToHash("0x1"), InfoNum: 1}, // in sync with leader
	}}

	followerB := mockHeaderProvider{latestBlockNum: 0, blocksByNum: map[int]*testutils.MockBlockInfo{
		0: {InfoHash: common.HexToHash("0x0"), InfoNum: 0}, // in sync with leader
		1: {InfoHash: common.HexToHash("0x1"), InfoNum: 1}, // in sync with leader
	}}

	secondCheck, firstErr := checkForChainFork(context.Background(), []HeaderProvider{&leader, &followerA, &followerB}, testlog.Logger(t, log.LevelDebug))
	require.NoError(t, firstErr)
	leader.latestBlockNum = 1    // advance the chain head
	followerA.latestBlockNum = 1 // advance the chain head
	followerB.latestBlockNum = 1 // advance the chain head
	require.NoError(t, secondCheck(false), "did not expect chain split error")
}
