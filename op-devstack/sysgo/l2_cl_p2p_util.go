package sysgo

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/retry"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-service/testreq"
)

func GetP2PClient(ctx context.Context, logger log.Logger, l2CLNode L2CLNode) (*sources.P2PClient, error) {
	rpcClient, err := client.NewRPC(ctx, logger, l2CLNode.UserRPC(), client.WithLazyDial())
	if err != nil {
		return nil, fmt.Errorf("failed to initialize rpc client for p2p client: %w", err)
	}
	return sources.NewP2PClient(rpcClient), nil
}

func GetPeerInfo(ctx context.Context, p2pClient *sources.P2PClient) (*apis.PeerInfo, error) {
	peerInfo, err := retry.Do(ctx, 3, retry.Exponential(), func() (*apis.PeerInfo, error) {
		return p2pClient.Self(ctx)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get peer info: %w", err)
	}
	return peerInfo, nil
}

func GetPeers(ctx context.Context, p2pClient *sources.P2PClient) (*apis.PeerDump, error) {
	peerDump, err := retry.Do(ctx, 3, retry.Exponential(), func() (*apis.PeerDump, error) {
		return p2pClient.Peers(ctx, true)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get peers: %w", err)
	}
	return peerDump, nil
}

type p2pClientsAndPeers struct {
	client1   *sources.P2PClient
	client2   *sources.P2PClient
	peerInfo1 *apis.PeerInfo
	peerInfo2 *apis.PeerInfo
}

func getP2PClientsAndPeers(ctx context.Context, logger log.Logger,
	require *testreq.Assertions, l2CL1, l2CL2 L2CLNode) *p2pClientsAndPeers {
	p2pClient1, err := GetP2PClient(ctx, logger, l2CL1)
	require.NoError(err)
	p2pClient2, err := GetP2PClient(ctx, logger, l2CL2)
	require.NoError(err)

	peerInfo1, err := GetPeerInfo(ctx, p2pClient1)
	require.NoError(err)
	peerInfo2, err := GetPeerInfo(ctx, p2pClient2)
	require.NoError(err)

	require.True(len(peerInfo1.Addresses) > 0 && len(peerInfo2.Addresses) > 0, "malformed peer info")

	return &p2pClientsAndPeers{
		client1:   p2pClient1,
		client2:   p2pClient2,
		peerInfo1: peerInfo1,
		peerInfo2: peerInfo2,
	}
}

// WithL2CLP2PConnection connects P2P between two L2CLs
func WithL2CLP2PConnection(l2CL1ID, l2CL2ID stack.L2CLNodeID) stack.Option[*Orchestrator] {
	return stack.AfterDeploy(func(orch *Orchestrator) {
		require := orch.P().Require()

		l2CL1, ok := orch.l2CLs.Get(l2CL1ID)
		require.True(ok, "looking for L2 CL node 1 to connect p2p")
		l2CL2, ok := orch.l2CLs.Get(l2CL2ID)
		require.True(ok, "looking for L2 CL node 2 to connect p2p")
		require.Equal(l2CL1ID.ChainID(), l2CL2ID.ChainID(), "must be same l2 chain")

		ctx := orch.P().Ctx()
		logger := orch.P().Logger()

		p := getP2PClientsAndPeers(ctx, logger, require, l2CL1, l2CL2)

		connectPeer := func(p2pClient *sources.P2PClient, multiAddress string) {
			err := retry.Do0(ctx, 6, retry.Exponential(), func() error {
				return p2pClient.ConnectPeer(ctx, multiAddress)
			})
			require.NoError(err, "failed to connect peer")
		}

		connectPeer(p.client1, p.peerInfo2.Addresses[0])
		connectPeer(p.client2, p.peerInfo1.Addresses[0])

		check := func(peerDump *apis.PeerDump, peerInfo *apis.PeerInfo) {
			multiAddress := peerInfo.PeerID.String()
			_, ok := peerDump.Peers[multiAddress]
			require.True(ok, "peer register invalid")
		}

		peerDump1, err := GetPeers(ctx, p.client1)
		require.NoError(err)
		peerDump2, err := GetPeers(ctx, p.client2)
		require.NoError(err)

		check(peerDump1, p.peerInfo2)
		check(peerDump2, p.peerInfo1)
	})
}
