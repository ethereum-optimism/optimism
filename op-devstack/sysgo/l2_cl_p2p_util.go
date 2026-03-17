package sysgo

import (
	"context"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/log"

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
		self, err := p2pClient.Self(ctx)
		if err != nil {
			return nil, err
		}
		if len(self.Addresses) == 0 {
			return nil, fmt.Errorf("no address found for peer")
		}
		if strings.HasPrefix(self.Addresses[0], "/p2p/") {
			return nil, fmt.Errorf("malformed multiaddr which starts with /p2p/")
		}
		return self, nil
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
