package rpc

import (
	"context"
	"log/slog"
	"testing"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

// stubConductor is a conductor that only reports its leader field.
type stubConductor struct {
	conductor
	leader bool
}

func (c *stubConductor) Leader(context.Context) bool { return c.leader }

// TestProxyRefusesOnFollowerWhenSequencerFails runs every leader-gated proxy
// method against a sequencer that serves no methods, so each forwarded call
// fails. A follower must still refuse with ErrNotLeader: its sequencer may lag
// the leader and fail on the leader's newest blocks, and returning that
// failure would hide that the request reached a non-leader.
func TestProxyRefusesOnFollowerWhenSequencerFails(t *testing.T) {
	srv := rpc.NewServer()
	t.Cleanup(srv.Stop)
	rpcClient := rpc.DialInProc(srv)
	t.Cleanup(rpcClient.Close)
	rollupClient := sources.NewRollupClient(client.NewBaseRPCClient(rpcClient))
	execClient := ethclient.NewClient(rpcClient)
	logger := testlog.Logger(t, slog.LevelInfo)

	proxiedCalls := func(con conductor) map[string]func(context.Context) error {
		execProxy := NewExecutionProxyBackend(logger, con, execClient)
		nodeProxy := NewNodeProxyBackend(logger, con, rollupClient)
		nodeAdminProxy := NewNodeAdminProxyBackend(logger, con, rollupClient)
		return map[string]func(context.Context) error{
			"eth_getBlockByNumber": func(ctx context.Context) error {
				_, err := execProxy.GetBlockByNumber(ctx, rpc.LatestBlockNumber, false)
				return err
			},
			"optimism_syncStatus": func(ctx context.Context) error {
				_, err := nodeProxy.SyncStatus(ctx)
				return err
			},
			"optimism_outputAtBlock": func(ctx context.Context) error {
				_, err := nodeProxy.OutputAtBlock(ctx, "0x7")
				return err
			},
			"optimism_rollupConfig": func(ctx context.Context) error {
				_, err := nodeProxy.RollupConfig(ctx)
				return err
			},
			"admin_sequencerActive": func(ctx context.Context) error {
				_, err := nodeAdminProxy.SequencerActive(ctx)
				return err
			},
		}
	}

	for method, call := range proxiedCalls(&stubConductor{leader: false}) {
		t.Run("follower refuses "+method, func(t *testing.T) {
			require.ErrorIs(t, call(t.Context()), ErrNotLeader)
		})
	}
	for method, call := range proxiedCalls(&stubConductor{leader: true}) {
		t.Run("leader returns the sequencer error for "+method, func(t *testing.T) {
			require.ErrorContains(t, call(t.Context()), "does not exist")
		})
	}
}

// losesLeadershipNode answers optimism_syncStatus and drops the conductor's
// leadership while doing so.
type losesLeadershipNode struct{ con *stubConductor }

func (n *losesLeadershipNode) SyncStatus() *eth.SyncStatus {
	n.con.leader = false
	return &eth.SyncStatus{}
}

// TestProxyChecksLeadershipAfterForwarding checks that a conductor that loses
// leadership while the forwarded call is in flight does not return the answer.
func TestProxyChecksLeadershipAfterForwarding(t *testing.T) {
	con := &stubConductor{leader: true}
	srv := rpc.NewServer()
	t.Cleanup(srv.Stop)
	require.NoError(t, srv.RegisterName("optimism", &losesLeadershipNode{con: con}))
	rpcClient := rpc.DialInProc(srv)
	t.Cleanup(rpcClient.Close)
	proxy := NewNodeProxyBackend(testlog.Logger(t, slog.LevelInfo), con,
		sources.NewRollupClient(client.NewBaseRPCClient(rpcClient)))

	_, err := proxy.SyncStatus(t.Context())
	require.ErrorIs(t, err, ErrNotLeader)
}
