package p2p_test

import (
	"testing"

	peer "github.com/libp2p/go-libp2p/core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	p2p "github.com/ethereum-optimism/optimism/op-node/p2p"
	log "github.com/ethereum/go-ethereum/log"
	node "github.com/ethereum-optimism/optimism/op-node/node"
	suite "github.com/stretchr/testify/suite"
	p2pMocks "github.com/ethereum-optimism/optimism/op-node/p2p/mocks"
)

// PeerScorerTestSuite tests peer parameterization.
type PeerScorerTestSuite struct {
	suite.Suite

	mockGater *p2pMocks.ConnectionGater
	mockStore *p2pMocks.Peerstore
	mockMetricer *p2pMocks.GossipMetricer
	logger log.Logger
}

// SetupTest sets up the test suite.
func (testSuite *PeerScorerTestSuite) SetupTest() {
	testSuite.mockGater = &p2pMocks.ConnectionGater{}
	testSuite.mockStore = &p2pMocks.Peerstore{}
	testSuite.mockMetricer = &p2pMocks.GossipMetricer{}
	logger := node.DefaultLogConfig()
	testSuite.logger = logger.NewLogger()
}

// TestPeerScorer runs the PeerScorerTestSuite.
func TestPeerScorer(t *testing.T) {
	suite.Run(t, new(PeerScorerTestSuite))
}

// TestPeerScoreConstants validates the peer score constants.
func (testSuite *PeerScorerTestSuite) TestPeerScoreConstants() {
	testSuite.Equal(-10, p2p.ConnectionFactor)
	testSuite.Equal(-100, p2p.PeerScoreThreshold)
}

// TestPeerScorerOnConnect ensures we can call the OnConnect method on the peer scorer.
func (testSuite *PeerScorerTestSuite) TestPeerScorerOnConnect() {
	scorer := p2p.NewScorer(
		testSuite.mockGater,
		testSuite.mockStore,
		testSuite.mockMetricer,
		testSuite.logger,
	)
	scorer.OnConnect()
}

// TestPeerScorerOnDisconnect ensures we can call the OnDisconnect method on the peer scorer.
func (testSuite *PeerScorerTestSuite) TestPeerScorerOnDisconnect() {
	scorer := p2p.NewScorer(
		testSuite.mockGater,
		testSuite.mockStore,
		testSuite.mockMetricer,
		testSuite.logger,
	)
	scorer.OnDisconnect()
}

// TestSnapshotHook tests running the snapshot hook on the peer scorer.
func (testSuite *PeerScorerTestSuite) TestSnapshotHook() {
	scorer := p2p.NewScorer(
		testSuite.mockGater,
		testSuite.mockStore,
		testSuite.mockMetricer,
		testSuite.logger,
	)
	inspectFn := scorer.SnapshotHook()

	// Mock the snapshot updates
	// This doesn't return anything
	testSuite.mockMetricer.On("RecordPeerScoring", peer.ID("peer1"), float64(-100)).Return(nil)

	// Since the peer score is not below the [PeerScoreThreshold] of -100,
	// no connection gater method should be called since the peer isn't already blocked

	// Apply the snapshot
	snapshotMap := map[peer.ID]*pubsub.PeerScoreSnapshot{
		peer.ID("peer1"): &pubsub.PeerScoreSnapshot{
			Score: -100,
		},
	}
	inspectFn(snapshotMap)
}

// TestSnapshotHookBlockPeer tests running the snapshot hook on the peer scorer with a peer score below the threshold.
// This implies that the peer should be blocked.
func (testSuite *PeerScorerTestSuite) TestSnapshotHookBlockPeer() {
	scorer := p2p.NewScorer(
		testSuite.mockGater,
		testSuite.mockStore,
		testSuite.mockMetricer,
		testSuite.logger,
	)
	inspectFn := scorer.SnapshotHook()

	// Mock the snapshot updates
	// This doesn't return anything
	testSuite.mockMetricer.On("RecordPeerScoring", peer.ID("peer1"), float64(-101)).Return(nil)

	// Mock a connection gater peer block call
	// Since the peer score is below the [PeerScoreThreshold] of -100,
	// the [BlockPeer] method should be called
	testSuite.mockGater.On("BlockPeer", peer.ID("peer1")).Return(nil)

	// Apply the snapshot
	snapshotMap := map[peer.ID]*pubsub.PeerScoreSnapshot{
		peer.ID("peer1"): &pubsub.PeerScoreSnapshot{
			Score: -101,
		},
	}
	inspectFn(snapshotMap)
}
