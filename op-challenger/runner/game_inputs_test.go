package runner

import (
	"context"
	"errors"
	"testing"
	"time"

	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

type stubSuperRootProvider struct {
	resp      eth.SuperRootAtTimestampResponse
	err       error
	requested []uint64
}

func (s *stubSuperRootProvider) SuperRootAtTimestamp(_ context.Context, timestamp uint64) (eth.SuperRootAtTimestampResponse, error) {
	s.requested = append(s.requested, timestamp)
	return s.resp, s.err
}

// Interop inputs come from superroot_atTimestamp, which op-node serves for single-chain
// rollups, rather than the supernode-only supernode_syncStatus.
func TestCreateGameInputsInteropReadsStatusFromSuperRoot(t *testing.T) {
	logger := testlog.Logger(t, log.LvlInfo)
	before := uint64(time.Now().Unix())
	client := &stubSuperRootProvider{resp: eth.SuperRootAtTimestampResponse{
		CurrentFinalizedTimestamp: 0, // stop before any trace work
		CurrentL1:                 eth.BlockID{Number: 100, Hash: common.Hash{0xaa}},
	}}

	_, err := createGameInputsInterop(context.Background(), logger, client, "test")
	require.ErrorContains(t, err, "finalized timestamp is 0")

	require.Len(t, client.requested, 1, "should read status with a single super root call")
	require.GreaterOrEqual(t, client.requested[0], before, "should request the current timestamp")
}

func TestCreateGameInputsInteropStatusFailure(t *testing.T) {
	logger := testlog.Logger(t, log.LvlInfo)
	boom := errors.New("boom")
	client := &stubSuperRootProvider{err: boom}

	_, err := createGameInputsInterop(context.Background(), logger, client, "test")
	require.ErrorIs(t, err, boom)
	require.ErrorContains(t, err, "super root status")
}

// A finalized timestamp of zero means nothing is finalized yet: there is no super root to
// dispute, and claimTimestamp-1 would underflow.
func TestCreateGameInputsInteropRejectsZeroFinalizedTimestamp(t *testing.T) {
	logger := testlog.Logger(t, log.LvlInfo)
	client := &stubSuperRootProvider{resp: eth.SuperRootAtTimestampResponse{
		CurrentFinalizedTimestamp: 0,
		CurrentL1:                 eth.BlockID{Number: 100},
	}}

	_, err := createGameInputsInterop(context.Background(), logger, client, "test")
	require.ErrorContains(t, err, "finalized timestamp is 0")
}

func TestCreateGameInputsInteropRejectsZeroL1Head(t *testing.T) {
	logger := testlog.Logger(t, log.LvlInfo)
	client := &stubSuperRootProvider{resp: eth.SuperRootAtTimestampResponse{
		CurrentFinalizedTimestamp: 5000,
		CurrentL1:                 eth.BlockID{Number: 0},
	}}

	_, err := createGameInputsInterop(context.Background(), logger, client, "test")
	require.ErrorContains(t, err, "l1 head is 0")
}

// Super root games need a source; without one the run fails rather than silently falling
// back to single-chain inputs.
func TestCreateGameInputsRequiresSuperRootSource(t *testing.T) {
	logger := testlog.Logger(t, log.LvlInfo)

	_, err := createGameInputs(context.Background(), logger, nil, nil, nil, "test", gameTypes.SuperCannonKonaGameType, false)
	require.ErrorContains(t, err, "requires super root RPC to be set")
}

func TestCreateGameInputsRequiresRollupClient(t *testing.T) {
	logger := testlog.Logger(t, log.LvlInfo)

	_, err := createGameInputs(context.Background(), logger, nil, nil, nil, "test", gameTypes.CannonKonaGameType, false)
	require.ErrorContains(t, err, "requires rollup rpc to be set")
}
