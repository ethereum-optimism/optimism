package batcher

import (
	"errors"
	"math/big"
	"math/rand"
	"testing"
	"time"

	optypes "github.com/ethereum-optimism/optimism/op-core/types"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/queue"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestPrivatePublicationValidatesQueuedTerminal(t *testing.T) {
	for _, scenario := range []string{"canonical", "replaced", "missing", "rpc error", "discarded", "discarded during lookup", "public"} {
		t.Run(scenario, func(t *testing.T) {
			bs, ep := setup(t, nil)
			bs.Config.NetworkTimeout = time.Second
			bs.PublicProjection = &fakeFollower{}
			terminal := eth.BlockID{Hash: common.Hash{6}, Number: 600}
			frame := frameData{id: frameID{chID: derive.ChannelID{1}}, data: []byte{1}}
			data := singleFrameTxData(frame)
			ch := &channel{
				ChannelBuilder: &ChannelBuilder{latestL2: terminal, frames: queue.Queue[frameData]{frame}, frameCursor: 1},
				log:            bs.Log, metr: bs.Metr, pendingTransactions: map[string]txData{data.ID().String(): data},
			}
			bs.channelMgr.txChannels[data.ID().String()] = ch
			bs.channelMgr.channelQueue = []*channel{ch}
			encoder := &PrivateInteropEncoder{prepared: map[common.Hash]optypes.Receipts{terminal.Hash: {}}}
			bs.BlockEnricher = encoder
			payload := &eth.ExecutionPayloadEnvelope{ExecutionPayload: &eth.ExecutionPayload{
				BlockHash: terminal.Hash, BlockNumber: eth.Uint64Quantity(terminal.Number),
			}}
			var rpcErr error
			switch scenario {
			case "replaced":
				payload.ExecutionPayload.BlockHash = common.Hash{9}
			case "missing":
				payload = nil
			case "rpc error":
				rpcErr = errors.New("unavailable")
			case "discarded":
				bs.clearChannelState(eth.BlockID{})
			case "public":
				bs.PublicProjection = nil
			}
			if scenario == "discarded during lookup" {
				ep.ethClient.On("PayloadByNumber", uint64(600)).Once().Return(payload, &rpcErr).Run(func(mock.Arguments) {
					bs.channelMgrMutex.Lock()
					defer bs.channelMgrMutex.Unlock()
					bs.clearChannelState(eth.BlockID{})
				})
			} else if scenario != "public" && scenario != "discarded" {
				ep.ethClient.ExpectPayloadByNumber(600, payload, rpcErr)
			}
			err := bs.validatePrivatePublication(t.Context(), data)
			switch scenario {
			case "canonical", "public":
				require.NoError(t, err)
				require.Same(t, ch, bs.channelMgr.txChannels[data.ID().String()])
			case "replaced":
				require.ErrorIs(t, err, ErrReorg)
				require.Empty(t, bs.channelMgr.channelQueue)
				require.Empty(t, encoder.prepared)
			case "missing", "rpc error":
				require.Error(t, err)
				require.Equal(t, 0, ch.frameCursor, "retry the frame after a transient lookup failure")
				require.Len(t, bs.channelMgr.channelQueue, 1)
			case "discarded":
				require.ErrorContains(t, err, "no longer queued")
			case "discarded during lookup":
				require.ErrorContains(t, err, "changed during validation")
				require.Empty(t, bs.channelMgr.txChannels)
			}
			ep.ethClient.AssertExpectations(t)
		})
	}
}

func TestPrivatePublicationCursor(t *testing.T) {
	bs, ep := setup(t, nil)
	bs.Config.NetworkTimeout = time.Second
	status := eth.SyncStatus{
		LocalSafeL2: eth.L2BlockRef{Hash: common.Hash{8}, Number: 8},
		UnsafeL2:    eth.L2BlockRef{Hash: common.Hash{146}, Number: 146},
	}
	original := status
	// Public chains keep exactly the private/local safe cursor and need no RPC.
	cursor, _, reset, err := bs.publicationCursor(t.Context(), &status)
	require.NoError(t, err)
	require.Equal(t, status.LocalSafeL2, cursor)
	require.False(t, reset)

	follower := &fakeFollower{safe: piRenderedBlock(143), blocks: map[uint64]*PublicProjectionBlock{143: piRenderedBlock(143)}}
	bs.PublicProjection = follower
	_, _, _, err = bs.publicationCursor(t.Context(), &status)
	require.ErrorContains(t, err, "has not reconciled")
	require.Equal(t, original, status, "an unsafe private suffix does not authorize publication after fallback")
	status.LocalSafeL2 = eth.L2BlockRef{Hash: common.Hash{144}, Number: 144}
	l1InfoTx, err := derive.L1InfoDepositBytes(bs.RollupConfig, params.MergedTestChainConfig,
		eth.SystemConfig{}, 3, testutils.RandomBlockInfo(rand.New(rand.NewSource(789))), 1000)
	require.NoError(t, err)
	expectBlock := func(n uint64) {
		ep.ethClient.ExpectPayloadByNumber(n, &eth.ExecutionPayloadEnvelope{ExecutionPayload: &eth.ExecutionPayload{
			BlockHash: common.Hash{byte(n)}, BlockNumber: hexutil.Uint64(n), Timestamp: 1000,
			Transactions: []eth.Data{l1InfoTx},
		}}, nil)
	}
	expectBlock(143)
	cursor, id, reset, err := bs.publicationCursor(t.Context(), &status)
	require.NoError(t, err)
	require.Equal(t, uint64(143), cursor.Number)
	require.Equal(t, common.Hash{143}, cursor.Hash, "queue comparisons must use private block identity")
	require.NotEqual(t, id.Hash, cursor.Hash)
	require.False(t, reset)
	bs.projectionHead = eth.BlockID{Hash: id.Hash, Number: id.Number}

	// Repeated polls preserve the cursor; a same-height projection reorg resets it.
	expectBlock(143)
	_, _, reset, err = bs.publicationCursor(t.Context(), &status)
	require.NoError(t, err)
	require.False(t, reset)
	follower.blocks[143] = &PublicProjectionBlock{Hash: common.Hash{99}, Number: 143}
	follower.safe = follower.blocks[143]
	expectBlock(143)
	_, _, reset, err = bs.publicationCursor(t.Context(), &status)
	require.NoError(t, err)
	require.True(t, reset)

	// A regression reopens previously skipped positions instead of retaining a high-water mark.
	follower.safe = piRenderedBlock(140)
	follower.blocks[140] = follower.safe
	expectBlock(140)
	cursor, _, reset, err = bs.publicationCursor(t.Context(), &status)
	require.NoError(t, err)
	require.True(t, reset)
	require.Equal(t, uint64(140), cursor.Number)

	// A projection rollback below private safety must reopen those positions too.
	status.LocalSafeL2 = eth.L2BlockRef{Hash: common.Hash{144}, Number: 144}
	expectBlock(140)
	cursor, _, reset, err = bs.publicationCursor(t.Context(), &status)
	require.NoError(t, err)
	require.True(t, reset)
	require.Equal(t, uint64(140), cursor.Number)
	require.Equal(t, uint64(144), status.LocalSafeL2.Number)
	status = original

	follower.safe = piRenderedBlock(147)
	follower.blocks[147] = follower.safe
	_, _, _, err = bs.publicationCursor(t.Context(), &status)
	require.ErrorContains(t, err, "has not caught up")
	follower.safe = piRenderedBlock(143)
	_, _, _, err = bs.publicationCursor(t.Context(), &status)
	require.ErrorContains(t, err, "heads disagree")
	follower.safe = nil
	_, _, _, err = bs.publicationCursor(t.Context(), &status)
	require.ErrorContains(t, err, "no safe block")
	require.Equal(t, original, status, "publication must never promote private safety")
	ep.ethClient.AssertExpectations(t)
}

func TestPrivatePublicationPrunesExpiredPrefix(t *testing.T) {
	bs, _ := setup(t, nil)
	bs.PublicProjection = &fakeFollower{}
	encoder := &PrivateInteropEncoder{prepared: make(map[common.Hash]optypes.Receipts)}
	bs.BlockEnricher = encoder
	status := eth.SyncStatus{
		HeadL1: eth.L1BlockRef{Number: 100}, CurrentL1: eth.L1BlockRef{Number: 100},
		LocalSafeL2: eth.L2BlockRef{Hash: common.Hash{8}, Number: 8},
		UnsafeL2:    eth.L2BlockRef{Number: 146},
	}
	cursor := eth.L2BlockRef{Number: 143, Hash: common.Hash{143}}
	projection := PublicProjectionBlock{Number: 143, Hash: common.Hash{43}}
	load := bs.syncAndPrune(&status, cursor, projection, false)
	require.Equal(t, &inclusiveBlockRange{144, 146}, load, "skip expired blocks before expensive enrichment")

	block := func(n int64) SizedBlock {
		return mustSizedBlockFromGeth(types.NewBlockWithHeader(&types.Header{Number: big.NewInt(n), BaseFee: big.NewInt(7)}))
	}
	bs.channelMgr.blocks = queue.Queue[SizedBlock]{block(144), block(145), block(146)}
	bs.channelMgr.blockCursor = 0
	for _, b := range bs.channelMgr.blocks {
		encoder.prepared[b.Hash()] = optypes.Receipts{}
	}
	load = bs.syncAndPrune(&status, cursor, projection, false)
	require.Nil(t, load)
	require.Len(t, bs.channelMgr.blocks, 3, "private safe 8 must not clear the queue on the next poll")

	cursor.Number, cursor.Hash = 144, bs.channelMgr.blocks[0].Hash()
	load = bs.syncAndPrune(&status, cursor, PublicProjectionBlock{Number: 144, Hash: common.Hash{44}}, false)
	require.Nil(t, load)
	require.Len(t, bs.channelMgr.blocks, 2)
	require.Len(t, encoder.prepared, 2, "prune unencoded prepared receipts too")

	cursor.Number, cursor.Hash = 140, common.Hash{140}
	load = bs.syncAndPrune(&status, cursor, PublicProjectionBlock{Number: 140, Hash: common.Hash{40}}, true)
	require.Equal(t, &inclusiveBlockRange{141, 146}, load)
	require.Empty(t, bs.channelMgr.blocks)
	require.Empty(t, encoder.prepared)
	require.Equal(t, uint64(8), status.LocalSafeL2.Number)
}

func TestPrivatePublicationRebuildsPartlyExpiredRange(t *testing.T) {
	bs, _ := setup(t, nil)
	bs.PublicProjection = &fakeFollower{}
	status := eth.SyncStatus{
		HeadL1: eth.L1BlockRef{Number: 100}, CurrentL1: eth.L1BlockRef{Number: 100},
		LocalSafeL2: eth.L2BlockRef{Number: 8}, UnsafeL2: eth.L2BlockRef{Number: 146},
	}
	// This queued range was never accepted: projection fallback consumed 141..143.
	bs.channelMgr.channelQueue = []*channel{{ChannelBuilder: &ChannelBuilder{
		oldestL2: eth.BlockID{Number: 141}, latestL2: eth.BlockID{Number: 144},
	}}}
	load := bs.syncAndPrune(&status, eth.L2BlockRef{Number: 143}, PublicProjectionBlock{Number: 143}, false)
	require.Equal(t, &inclusiveBlockRange{144, 146}, load)
	require.Empty(t, bs.channelMgr.channelQueue, "rebuild the suffix with its own claim and new projection parent")
}

type projectionStatusAPI struct{ status eth.SyncStatus }

func (s *projectionStatusAPI) SyncStatus() eth.SyncStatus { return s.status }

func TestProjectionFollowerUsesCompleteLocalSpan(t *testing.T) {
	server := rpc.NewServer()
	require.NoError(t, server.RegisterName("optimism", &projectionStatusAPI{status: eth.SyncStatus{
		CurrentL1:   eth.L1BlockRef{Hash: common.Hash{30}, Number: 30},
		LocalSafeL2: eth.L2BlockRef{Hash: common.Hash{12}, Number: 12},
		SafeL2:      eth.L2BlockRef{Hash: common.Hash{9}, Number: 9},
	}}))
	client := rpc.DialInProc(server)
	t.Cleanup(client.Close)
	t.Cleanup(server.Stop)
	follower := &rpcPublicProjectionFollower{rollupRPC: client, timeout: time.Second}
	head, err := follower.SafeBlock(t.Context())
	require.NoError(t, err)
	require.Equal(t, &PublicProjectionBlock{Hash: common.Hash{12}, Number: 12, CurrentL1: eth.L1BlockRef{Hash: common.Hash{30}, Number: 30}}, head,
		"cross-safe block 9 would split accepted span 9..12")
}

func TestPrivatePublicationWaitsForProjectionDerivation(t *testing.T) {
	bs, _ := setup(t, nil)
	block := mustSizedBlockFromGeth(types.NewBlockWithHeader(&types.Header{Number: big.NewInt(100), BaseFee: big.NewInt(7)}))
	status := eth.SyncStatus{
		HeadL1: eth.L1BlockRef{Number: 200}, CurrentL1: eth.L1BlockRef{Number: 200},
		LocalSafeL2: eth.L2BlockRef{Number: 8}, UnsafeL2: eth.L2BlockRef{Number: 100},
	}
	channels := []testChannelStatuser{{latestL2: eth.ToBlockID(block), inclusionBlock: 199, fullySubmitted: true}}
	// Private derivation is past inclusion, but projection derivation has not
	// processed that L1 block yet. Do not rebuild an otherwise valid channel.
	actions, outOfSync := computeSyncActionsWithCursor(status, eth.L2BlockRef{Number: 99},
		eth.L1BlockRef{Number: 198}, eth.L1BlockRef{}, queue.Queue[SizedBlock]{block}, channels, bs.Log)
	require.False(t, outOfSync)
	require.Nil(t, actions.clearState)
	actions, outOfSync = computeSyncActionsWithCursor(status, eth.L2BlockRef{Number: 99},
		eth.L1BlockRef{Number: 200}, eth.L1BlockRef{}, queue.Queue[SizedBlock]{block}, channels, bs.Log)
	require.False(t, outOfSync)
	require.NotNil(t, actions.clearState, "a processed but unaccepted channel must still be retried")
}
