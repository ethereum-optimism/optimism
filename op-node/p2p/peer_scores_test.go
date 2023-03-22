package p2p_test

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
	"time"

	p2p "github.com/ethereum-optimism/optimism/op-node/p2p"
	p2pMocks "github.com/ethereum-optimism/optimism/op-node/p2p/mocks"
	testlog "github.com/ethereum-optimism/optimism/op-node/testlog"

	"github.com/stretchr/testify/mock"
	suite "github.com/stretchr/testify/suite"

	log "github.com/ethereum/go-ethereum/log"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	host "github.com/libp2p/go-libp2p/core/host"
	peer "github.com/libp2p/go-libp2p/core/peer"
	bhost "github.com/libp2p/go-libp2p/p2p/host/blank"
	tswarm "github.com/libp2p/go-libp2p/p2p/net/swarm/testing"
)

// PeerScoresTestSuite tests peer parameterization.
type PeerScoresTestSuite struct {
	suite.Suite

	mockGater    *p2pMocks.ConnectionGater
	mockStore    *p2pMocks.Peerstore
	mockMetricer *p2pMocks.GossipMetricer
	bandScorer   p2p.BandScoreThresholds
	logger       log.Logger
}

// SetupTest sets up the test suite.
func (testSuite *PeerScoresTestSuite) SetupTest() {
	testSuite.mockGater = &p2pMocks.ConnectionGater{}
	testSuite.mockStore = &p2pMocks.Peerstore{}
	testSuite.mockMetricer = &p2pMocks.GossipMetricer{}
	bandScorer, err := p2p.NewBandScorer("0:graylist;")
	testSuite.NoError(err)
	testSuite.bandScorer = *bandScorer
	testSuite.logger = testlog.Logger(testSuite.T(), log.LvlError)
}

// TestPeerScores runs the PeerScoresTestSuite.
func TestPeerScores(t *testing.T) {
	suite.Run(t, new(PeerScoresTestSuite))
}

// getNetHosts generates a slice of hosts using the [libp2p/go-libp2p] library.
func getNetHosts(testSuite *PeerScoresTestSuite, ctx context.Context, n int) []host.Host {
	var out []host.Host
	for i := 0; i < n; i++ {
		netw := tswarm.GenSwarm(testSuite.T())
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
		opts = append(opts, p2p.ConfigurePeerScoring(h, testSuite.mockGater, &p2p.Config{
			BandScoreThresholds: testSuite.bandScorer,
			PeerScoring: pubsub.PeerScoreParams{
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
		}, testSuite.mockMetricer, logger)...)
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

	testSuite.mockMetricer.On("SetPeerScores", mock.Anything).Return(nil)

	testSuite.mockGater.On("ListBlockedPeers").Return([]peer.ID{})

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
