package p2p

import (
	"context"
	"fmt"
	"math/big"
	"math/rand"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-node/rollup"

	p2pMocks "github.com/ethereum-optimism/optimism/op-node/p2p/mocks"
	"github.com/ethereum-optimism/optimism/op-node/p2p/store"
	testlog "github.com/ethereum-optimism/optimism/op-node/testlog"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	log "github.com/ethereum/go-ethereum/log"
	ds "github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/sync"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	host "github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	peer "github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	bhost "github.com/libp2p/go-libp2p/p2p/host/blank"
	"github.com/libp2p/go-libp2p/p2p/host/peerstore/pstoreds"
	tswarm "github.com/libp2p/go-libp2p/p2p/net/swarm/testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// PeerScoresTestSuite tests peer parameterization.
type PeerScoresTestSuite struct {
	suite.Suite

	mockStore    *p2pMocks.Peerstore
	mockMetricer *p2pMocks.ScoreMetrics
	logger       log.Logger
}

// SetupTest sets up the test suite.
func (testSuite *PeerScoresTestSuite) SetupTest() {
	testSuite.mockStore = &p2pMocks.Peerstore{}
	testSuite.mockMetricer = &p2pMocks.ScoreMetrics{}
	testSuite.logger = testlog.Logger(testSuite.T(), log.LvlError)
}

// TestPeerScores runs the PeerScoresTestSuite.
func TestPeerScores(t *testing.T) {
	suite.Run(t, new(PeerScoresTestSuite))
}

type customPeerstoreNetwork struct {
	network.Network
	ps peerstore.Peerstore
}

func (c *customPeerstoreNetwork) Peerstore() peerstore.Peerstore {
	return c.ps
}

func (c *customPeerstoreNetwork) Close() error {
	_ = c.ps.Close()
	return c.Network.Close()
}

// getNetHosts generates a slice of hosts using the [libp2p/go-libp2p] library.
func getNetHosts(testSuite *PeerScoresTestSuite, ctx context.Context, n int) []host.Host {
	var out []host.Host
	log := testlog.Logger(testSuite.T(), log.LvlError)
	for i := 0; i < n; i++ {
		swarm := tswarm.GenSwarm(testSuite.T())
		eps, err := store.NewExtendedPeerstore(ctx, log, clock.SystemClock, swarm.Peerstore(), sync.MutexWrap(ds.NewMapDatastore()), 1*time.Hour)
		netw := &customPeerstoreNetwork{swarm, eps}
		require.NoError(testSuite.T(), err)
		h := bhost.NewBlankHost(netw)
		testSuite.T().Cleanup(func() { h.Close() })
		out = append(out, h)
	}
	return out
}

func newGossipSubs(testSuite *PeerScoresTestSuite, ctx context.Context, hosts []host.Host) []*pubsub.PubSub {
	var psubs []*pubsub.PubSub

	logger := testlog.Logger(testSuite.T(), log.LvlCrit)

	// For each host, create a default gossipsub router.
	for _, h := range hosts {
		rt := pubsub.DefaultGossipSubRouter(h)
		opts := []pubsub.Option{}

		dataStore := sync.MutexWrap(ds.NewMapDatastore())
		peerStore, err := pstoreds.NewPeerstore(context.Background(), dataStore, pstoreds.DefaultOpts())
		require.NoError(testSuite.T(), err)
		extPeerStore, err := store.NewExtendedPeerstore(context.Background(), logger, clock.SystemClock, peerStore, dataStore, 1*time.Hour)
		require.NoError(testSuite.T(), err)

		scorer := NewScorer(
			&rollup.Config{L2ChainID: big.NewInt(123)},
			extPeerStore, testSuite.mockMetricer, logger)
		opts = append(opts, ConfigurePeerScoring(&Config{
			PeerScoring: &pubsub.PeerScoreParams{
				AppSpecificScore: func(p peer.ID) float64 {
					if p == hosts[0].ID() {
						return -1000
					} else {
						return 0
					}
				},
				AppSpecificWeight: 1,
				DecayInterval:     time.Second,
				DecayToZero:       0.01,
			},
		}, scorer, logger)...)
		ps, err := pubsub.NewGossipSubWithRouter(ctx, h, rt, opts...)
		if err != nil {
			panic(err)
		}
		psubs = append(psubs, ps)
	}

	return psubs
}

func connectHosts(t *testing.T, hosts []host.Host, d int) {
	for i, a := range hosts {
		for j := 0; j < d; j++ {
			n := rand.Intn(len(hosts))
			if n == i {
				j--
				continue
			}

			b := hosts[n]

			pinfo := a.Peerstore().PeerInfo(a.ID())
			err := b.Connect(context.Background(), pinfo)
			if err != nil {
				t.Fatal(err)
			}
		}
	}
}

// TestNegativeScores tests blocking peers with negative scores.
//
// This follows the testing done in libp2p's gossipsub_test.go [TestGossipsubNegativeScore] function.
func (testSuite *PeerScoresTestSuite) TestNegativeScores() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	testSuite.mockMetricer.On("SetPeerScores", mock.Anything, mock.Anything).Return(nil)

	// Construct 20 hosts using the [getNetHosts] function.
	hosts := getNetHosts(testSuite, ctx, 20)
	testSuite.Equal(20, len(hosts))

	// Construct 20 gossipsub routers using the [newGossipSubs] function.
	pubsubs := newGossipSubs(testSuite, ctx, hosts)
	testSuite.Equal(20, len(pubsubs))

	// Connect the hosts in a dense network
	connectHosts(testSuite.T(), hosts, 10)

	// Create subscriptions
	var subs []*pubsub.Subscription
	var topics []*pubsub.Topic
	for _, ps := range pubsubs {
		topic, err := ps.Join("test")
		testSuite.NoError(err)
		sub, err := topic.Subscribe()
		testSuite.NoError(err)
		subs = append(subs, sub)
		topics = append(topics, topic)
	}

	// Wait and then publish messages
	time.Sleep(3 * time.Second)
	for i := 0; i < 20; i++ {
		msg := []byte(fmt.Sprintf("message %d", i))
		topic := topics[i]
		err := topic.Publish(ctx, msg)
		testSuite.NoError(err)
		time.Sleep(20 * time.Millisecond)
	}

	// Allow gossip to propagate
	time.Sleep(2 * time.Second)

	// Collects all messages from a subscription
	collectAll := func(sub *pubsub.Subscription) []*pubsub.Message {
		var res []*pubsub.Message
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()
		for {
			msg, err := sub.Next(ctx)
			if err != nil {
				break
			}
			res = append(res, msg)
		}
		return res
	}

	// Collect messages for the first host subscription
	// This host should only receive 1 message from itself
	count := len(collectAll(subs[0]))
	testSuite.Equal(1, count)

	// Validate that all messages were received from the first peer
	for _, sub := range subs[1:] {
		all := collectAll(sub)
		for _, m := range all {
			testSuite.NotEqual(hosts[0].ID(), m.ReceivedFrom)
		}
	}
}
