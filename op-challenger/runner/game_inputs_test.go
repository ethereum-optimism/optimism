package runner

import (
	"context"
	"errors"
	"math/big"
	"testing"
	"time"

	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

type stubL1Headers struct {
	requested []uint64
	err       error
}

func (s *stubL1Headers) header(number uint64) *ethTypes.Header {
	return &ethTypes.Header{Number: new(big.Int).SetUint64(number)}
}

func (s *stubL1Headers) HeaderByNumber(_ context.Context, number *big.Int) (*ethTypes.Header, error) {
	num := bigs.Uint64Strict(number)
	s.requested = append(s.requested, num)
	if s.err != nil {
		return nil, s.err
	}
	return s.header(num), nil
}

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

	_, err := createGameInputsInterop(context.Background(), logger, client, &stubL1Headers{}, "test")
	require.ErrorContains(t, err, "finalized timestamp is 0")

	require.Len(t, client.requested, 1, "should read status with a single super root call")
	require.GreaterOrEqual(t, client.requested[0], before, "should request the current timestamp")
}

func TestCreateGameInputsInteropStatusFailure(t *testing.T) {
	logger := testlog.Logger(t, log.LvlInfo)
	boom := errors.New("boom")
	client := &stubSuperRootProvider{err: boom}

	_, err := createGameInputsInterop(context.Background(), logger, client, &stubL1Headers{}, "test")
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

	_, err := createGameInputsInterop(context.Background(), logger, client, &stubL1Headers{}, "test")
	require.ErrorContains(t, err, "finalized timestamp is 0")
}

func TestCreateGameInputsInteropRejectsZeroL1Head(t *testing.T) {
	logger := testlog.Logger(t, log.LvlInfo)
	client := &stubSuperRootProvider{resp: eth.SuperRootAtTimestampResponse{
		CurrentFinalizedTimestamp: 5000,
		CurrentL1:                 eth.BlockID{Number: 0},
	}}

	_, err := createGameInputsInterop(context.Background(), logger, client, &stubL1Headers{}, "test")
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

// stubSyncedSuperNode answers like a fully synced single-chain op-node: a fixed CurrentL1
// for every timestamp, super root data up to the finalized timestamp, and no chain data
// beyond it.
type stubSyncedSuperNode struct {
	currentL1   eth.BlockID
	finalizedTs uint64
	// requiredL1 is the L1 data the disputed timestamps need. Defaults to well below
	// currentL1, as it is for a finalized timestamp on a live chain.
	requiredL1 eth.BlockID
}

func (s *stubSyncedSuperNode) SuperRootAtTimestamp(_ context.Context, timestamp uint64) (eth.SuperRootAtTimestampResponse, error) {
	chainID := eth.ChainIDFromUInt64(10)
	resp := eth.SuperRootAtTimestampResponse{
		CurrentL1:                 s.currentL1,
		CurrentFinalizedTimestamp: s.finalizedTs,
		ChainIDs:                  []eth.ChainID{chainID},
		OptimisticAtTimestamp:     map[eth.ChainID]eth.OutputWithRequiredL1{},
	}
	if timestamp > s.finalizedTs {
		return resp, nil
	}
	output := &eth.OutputV0{BlockHash: common.Hash{byte(timestamp)}}
	requiredL1 := s.requiredL1
	if requiredL1 == (eth.BlockID{}) {
		requiredL1 = eth.BlockID{Number: s.currentL1.Number - 100, Hash: common.Hash{0xbb}}
	}
	resp.OptimisticAtTimestamp[chainID] = eth.OutputWithRequiredL1{
		Output:     output,
		OutputRoot: eth.OutputRoot(output),
		RequiredL1: requiredL1,
	}
	super := eth.NewSuperV1(timestamp, eth.ChainIDAndOutput{ChainID: chainID, Output: eth.OutputRoot(output)})
	resp.Data = &eth.SuperRootResponseData{
		VerifiedRequiredL1: requiredL1,
		Super:              super,
		SuperRoot:          eth.SuperRoot(super),
	}
	return resp, nil
}

// The game L1 head must be one the node has fully processed, otherwise the trace provider's
// sync gate (which requires CurrentL1 > l1Head) rejects every claim as ErrNotInSync.
func TestCreateGameInputsInteropBuildsInputsFromSyncedNode(t *testing.T) {
	logger := testlog.Logger(t, log.LvlInfo)
	currentL1 := eth.BlockID{Number: 5000, Hash: common.Hash{0xaa}}
	client := &stubSyncedSuperNode{currentL1: currentL1, finalizedTs: 9000}

	// The run picks one of three trace positions at random; all must build inputs.
	for i := 0; i < 20; i++ {
		l1 := &stubL1Headers{}
		inputs, err := createGameInputsInterop(context.Background(), logger, client, l1, "test")
		require.NoError(t, err)
		require.NotEmpty(t, inputs.AgreedPreState)
		require.NotEqual(t, common.Hash{}, inputs.L2Claim)
		require.NotEqual(t, eth.InvalidTransitionHash, inputs.L2Claim,
			"a sentinel claim means the FPP is asked to prove nothing")
		require.Equal(t, new(big.Int).SetUint64(client.finalizedTs+10), inputs.L2SequenceNumber)
		require.Equal(t, []uint64{currentL1.Number - 1}, l1.requested,
			"game L1 head must be the highest fully processed block, not CurrentL1 itself")
		require.Equal(t, l1.header(currentL1.Number-1).Hash(), inputs.L1Head)
	}
}

// The other edge of the window: an L1 head below the L1 data the disputed timestamps need
// makes every claim the InvalidTransition sentinel, which the trace provider returns without
// error. The FPP would prove it trivially, so the run must fail rather than pass having
// verified nothing.
func TestCreateGameInputsInteropRejectsSentinelClaim(t *testing.T) {
	logger := testlog.Logger(t, log.LvlInfo)
	currentL1 := eth.BlockID{Number: 5000, Hash: common.Hash{0xaa}}
	client := &stubSyncedSuperNode{
		currentL1:   currentL1,
		finalizedTs: 9000,
		requiredL1:  eth.BlockID{Number: currentL1.Number + 5, Hash: common.Hash{0xcc}},
	}

	// Every one of the three random trace positions must be rejected.
	for i := 0; i < 20; i++ {
		_, err := createGameInputsInterop(context.Background(), logger, client, &stubL1Headers{}, "test")
		require.ErrorContains(t, err, "invalid transition sentinel")
	}
}

// The game L1 head needs a working L1 client to resolve its hash. A lookup failure must
// surface, not leave the run to fall back on an unusable head.
func TestCreateGameInputsInteropL1LookupFailure(t *testing.T) {
	logger := testlog.Logger(t, log.LvlInfo)
	client := &stubSyncedSuperNode{currentL1: eth.BlockID{Number: 5000}, finalizedTs: 9000}
	boom := errors.New("l1 unavailable")

	_, err := createGameInputsInterop(context.Background(), logger, client, &stubL1Headers{err: boom}, "test")
	require.ErrorIs(t, err, boom)
	require.ErrorContains(t, err, "failed to fetch l1 head at block 4999")
}
