package p2p_test

import (
	"testing"
	"time"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	peer "github.com/libp2p/go-libp2p/core/peer"
	suite "github.com/stretchr/testify/suite"

	log "github.com/ethereum/go-ethereum/log"

	p2p "github.com/ethereum-optimism/optimism/op-node/p2p"
	p2pMocks "github.com/ethereum-optimism/optimism/op-node/p2p/mocks"
	"github.com/ethereum-optimism/optimism/op-node/p2p/store"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

// PeerScorerTestSuite tests peer parameterization.
type PeerScorerTestSuite struct {
	suite.Suite

	mockStore    *p2pMocks.Peerstore
	mockMetricer *p2pMocks.ScoreMetrics
	logger       log.Logger
}

// SetupTest sets up the test suite.
func (testSuite *PeerScorerTestSuite) SetupTest() {
	testSuite.mockStore = &p2pMocks.Peerstore{}
	testSuite.mockMetricer = &p2pMocks.ScoreMetrics{}
	testSuite.logger = testlog.Logger(testSuite.T(), log.LevelError)
}

// TestPeerScorer runs the PeerScorerTestSuite.
func TestPeerScorer(t *testing.T) {
	suite.Run(t, new(PeerScorerTestSuite))
}

// TestScorer_SnapshotHook tests running the snapshot hook on the peer scorer.
func (testSuite *PeerScorerTestSuite) TestScorer_SnapshotHook() {
	scorer := p2p.NewScorer(
		testSuite.mockStore,
		testSuite.mockMetricer,
		&p2p.NoopApplicationScorer{},
		testSuite.logger,
	)
	inspectFn := scorer.SnapshotHook()

	scores := store.PeerScores{Gossip: store.GossipScores{Total: 3}}
	// Expect updating the peer store
	testSuite.mockStore.On("SetScore", peer.ID("peer1"), &store.GossipScores{Total: float64(-100)}).Return(scores, nil).Once()

	// The metricer should then be called with the peer score band map
	testSuite.mockMetricer.On("SetPeerScores", []store.PeerScores{scores}).Return(nil).Once()

	// Apply the snapshot
	snapshotMap := map[peer.ID]*pubsub.PeerScoreSnapshot{
		peer.ID("peer1"): {
			Score: -100,
		},
	}
	inspectFn(snapshotMap)

	// Expect updating the peer store
	testSuite.mockStore.On("SetScore", peer.ID("peer1"), &store.GossipScores{
		Total: 0,
		Blocks: store.TopicScores{
			// expect the maximum time in mesh
			TimeInMesh: 200.0, // float in seconds
			// expect the sum of deliveries
			FirstMessageDeliveries:   111,
			MeshMessageDeliveries:    222,
			InvalidMessageDeliveries: 333,
		},
	}).Return(scores, nil).Once()

	// The metricer should then be called with the peer score band map
	testSuite.mockMetricer.On("SetPeerScores", []store.PeerScores{scores}).Return(nil).Once()

	// Apply the snapshot
	snapshotMap = map[peer.ID]*pubsub.PeerScoreSnapshot{
		peer.ID("peer1"): {
			Score: 0,
			Topics: map[string]*pubsub.TopicScoreSnapshot{
				"foo": {
					TimeInMesh:               time.Second * 100,
					FirstMessageDeliveries:   1,
					MeshMessageDeliveries:    2,
					InvalidMessageDeliveries: 3,
				},
				"bar": {
					TimeInMesh:               time.Second * 200,
					FirstMessageDeliveries:   10,
					MeshMessageDeliveries:    20,
					InvalidMessageDeliveries: 30,
				},
				"baz": {
					TimeInMesh:               time.Second * 50,
					FirstMessageDeliveries:   100,
					MeshMessageDeliveries:    200,
					InvalidMessageDeliveries: 300,
				},
			},
		},
	}
	inspectFn(snapshotMap)
}
