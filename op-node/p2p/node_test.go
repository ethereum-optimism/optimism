package p2p

import (
	"errors"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-node/metrics"
	"github.com/ethereum-optimism/optimism/op-node/p2p/store"
	metricstest "github.com/ethereum-optimism/optimism/op-service/metrics/test"
)

type banTestStore struct {
	store.ExtendedPeerstore
	setExpiry func(peer.ID, time.Time) error
}

func (s *banTestStore) SetPeerBanExpiration(id peer.ID, expiry time.Time) error {
	return s.setExpiry(id, expiry)
}

type banTestNetwork struct {
	network.Network
	closePeer func(peer.ID) error
}

func (n *banTestNetwork) ClosePeer(id peer.ID) error {
	return n.closePeer(id)
}

type banTestHost struct {
	host.Host
	network network.Network
}

func (h *banTestHost) Network() network.Network {
	return h.network
}

func TestNodeP2PBanPeerMetrics(t *testing.T) {
	expiry := time.Unix(10000, 0)
	storeErr := errors.New("store failure")
	closeErr := errors.New("disconnect failure")
	for _, test := range []struct {
		name       string
		expiry     time.Time
		storeErr   error
		closeErr   error
		noMetrics  bool
		wantBans   float64
		wantCloses int
		wantErr    error
	}{
		{name: "registered ban", expiry: expiry, wantBans: 1, wantCloses: 1},
		{name: "persistence failure", expiry: expiry, storeErr: storeErr, wantErr: storeErr},
		{name: "zero expiry clear", wantCloses: 1},
		{name: "disconnect failure", expiry: expiry, closeErr: closeErr, wantBans: 1, wantCloses: 1, wantErr: closeErr},
		{name: "optional metrics", expiry: expiry, noMetrics: true, wantCloses: 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			id := peer.ID("test-peer")
			m := metrics.NewMetrics("test", nil)
			banCount := func() float64 {
				return metricstest.NewMetricChecker(t, m.Registry()).FindByName("op_node_test_p2p_peer_bans").FindByLabels(nil).GetCounter().GetValue()
			}
			var persisted bool
			var writes, closes int
			n := &NodeP2P{
				metrics: m,
				store: &banTestStore{setExpiry: func(gotID peer.ID, gotExpiry time.Time) error {
					writes++
					require.Equal(t, id, gotID)
					require.Equal(t, test.expiry, gotExpiry)
					require.Zero(t, banCount(), "must persist before counting")
					persisted = test.storeErr == nil
					return test.storeErr
				}},
				host: &banTestHost{network: &banTestNetwork{closePeer: func(gotID peer.ID) error {
					closes++
					require.Equal(t, id, gotID)
					require.True(t, persisted, "must persist before disconnecting")
					require.Equal(t, test.wantBans, banCount(), "must count before disconnecting")
					return test.closeErr
				}}},
			}
			if test.noMetrics {
				n.metrics = nil
			}
			err := n.BanPeer(id, test.expiry)
			if test.wantErr != nil {
				require.ErrorIs(t, err, test.wantErr)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, 1, writes)
			require.Equal(t, test.wantCloses, closes)
			require.Equal(t, test.wantBans, banCount())
			require.Zero(t, metricstest.NewMetricChecker(t, m.Registry()).FindByName("op_node_test_p2p_peer_unbans").FindByLabels(nil).GetCounter().GetValue())
		})
	}
}
