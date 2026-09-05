package batcher

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
)

type projectionCursorRPC struct {
	t      *testing.T
	number uint64
	err    error
}

func (p *projectionCursorRPC) GetBlockByNumber(_ context.Context, label rpc.BlockNumber, full bool) (*rpcBlock, error) {
	require.Equal(p.t, rpc.LatestBlockNumber, label)
	require.False(p.t, full)
	return &rpcBlock{Number: hexutil.Uint64(p.number), Hash: common.Hash{0xff}}, p.err
}

func TestPrivateInteropPublicationCursor(t *testing.T) {
	for _, tc := range []struct {
		name          string
		publicDerived uint64
		wantSafe      uint64
		publicError   error
		privateErr    error
		wrongBlock    bool
	}{
		{name: "skip expired fallback", publicDerived: 920, wantSafe: 920},
		{name: "wait for private catchup", publicDerived: 940, wantSafe: 930},
		{name: "keep ordinary local safe progress", publicDerived: 904, wantSafe: 905},
		{name: "projection unavailable", publicError: errors.New("projection unavailable")},
		{name: "private payload unavailable", publicDerived: 920, privateErr: errors.New("private payload unavailable")},
		{name: "private payload at wrong height", publicDerived: 920, wrongBlock: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := rpc.NewServer()
			require.NoError(t, server.RegisterName("eth", &projectionCursorRPC{t, tc.publicDerived, tc.publicError}))
			t.Cleanup(server.Stop)
			rpcClient := rpc.DialInProc(server)
			t.Cleanup(rpcClient.Close)
			base := newEndpointProvider()
			status := &eth.SyncStatus{
				HeadL1:      eth.L1BlockRef{Hash: common.Hash{1}, Number: 100},
				CurrentL1:   eth.L1BlockRef{Hash: common.Hash{1}, Number: 100},
				LocalSafeL2: eth.L2BlockRef{Hash: common.Hash{2}, Number: 905},
				UnsafeL2:    eth.L2BlockRef{Hash: common.Hash{3}, Number: 930},
			}
			base.rollupClient.ExpectSyncStatus(status, nil)
			number := min(tc.publicDerived, status.UnsafeL2.Number)
			var expected eth.L2BlockRef
			if tc.publicError == nil && number > status.LocalSafeL2.Number {
				payload := piPayload(t, number)
				if tc.wrongBlock {
					payload.BlockNumber++
				}
				base.ethClient.ExpectPayloadByNumber(number, &eth.ExecutionPayloadEnvelope{ExecutionPayload: payload}, tc.privateErr)
				var err error
				expected, err = derive.PayloadToBlockRef(piRollupCfg(), payload)
				require.NoError(t, err)
			}
			provider := &privateInteropEndpoints{
				L2EndpointProvider: base,
				projection:         &rpcPublicProjectionFollower{rpc: rpcClient, timeout: time.Second},
				rollup:             piRollupCfg(),
			}
			client, err := provider.RollupClient(t.Context())
			require.NoError(t, err)
			got, err := client.SyncStatus(t.Context())
			if tc.publicError != nil || tc.privateErr != nil || tc.wrongBlock {
				require.Error(t, err)
				require.Nil(t, got)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.wantSafe, got.LocalSafeL2.Number)
				require.Equal(t, uint64(905), status.LocalSafeL2.Number, "must not mutate the private CL's status")
				require.Equal(t, status.UnsafeL2, got.UnsafeL2)
				require.Equal(t, status.SafeL2, got.SafeL2)
				if number > status.LocalSafeL2.Number {
					require.Equal(t, expected, got.LocalSafeL2, "stock pruning needs the private payload hash")
				}
				actions, outOfSync := computeSyncActions[channelStatuser](*got, eth.L1BlockRef{}, nil, nil, log.New())
				require.False(t, outOfSync)
				if tc.wantSafe < status.UnsafeL2.Number {
					require.Equal(t, &inclusiveBlockRange{tc.wantSafe + 1, status.UnsafeL2.Number}, actions.blocksToLoad)
				} else {
					require.Nil(t, actions.blocksToLoad)
				}
			}
			base.ethClient.AssertExpectations(t)
			base.rollupClient.AssertExpectations(t)
		})
	}
}
