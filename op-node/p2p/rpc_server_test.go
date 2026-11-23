package p2p

import (
	"context"
	"testing"

	"github.com/libp2p/go-libp2p/core/host"
	mocknet "github.com/libp2p/go-libp2p/p2p/net/mock"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-node/p2p/gating"
	"github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

// gaterlessNode is a p2p node without a connection gater.
type gaterlessNode struct {
	Node
	host host.Host
}

func (n gaterlessNode) Host() host.Host {
	return n.host
}

func (n gaterlessNode) ConnectionGater() gating.BlockingConnectionGater {
	return nil
}

func TestDisconnectPeerWithoutConnectionGater(t *testing.T) {
	mn, err := mocknet.FullMeshConnected(2)
	require.NoError(t, err)
	hosts := mn.Hosts()
	backend := NewP2PAPIBackend(gaterlessNode{host: hosts[0]}, testlog.Logger(t, log.LevelError))

	require.NoError(t, backend.DisconnectPeer(context.Background(), hosts[1].ID()))
	require.Empty(t, hosts[0].Network().ConnsToPeer(hosts[1].ID()))
}
