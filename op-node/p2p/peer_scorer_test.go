package p2p_test

import (
	"testing"

	p2p "github.com/ethereum-optimism/optimism/op-node/p2p"
	p2pMocks "github.com/ethereum-optimism/optimism/op-node/p2p/mocks"
	"github.com/ethereum-optimism/optimism/op-node/testlog"
	log "github.com/ethereum/go-ethereum/log"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	peer "github.com/libp2p/go-libp2p/core/peer"
	suite "github.com/stretchr/testify/suite"
)

// PeerScorerTestSuite tests peer parameterization.
type PeerScorerTestSuite struct {
	suite.Suite

	mockStore    *p2pMocks.Peerstore
	mockMetricer *p2pMocks.GossipMetricer
	bandScorer   *p2p.BandScoreThresholds
	logger       log.Logger
}

// SetupTest sets up the test suite.
func (testSuite *PeerScorerTestSuite) SetupTest() {
	testSuite.mockStore = &p2pMocks.Peerstore{}
	testSuite.mockMetricer = &p2pMocks.GossipMetricer{}
	bandScorer, err := p2p.NewBandScorer("-40:graylist;0:friend;")
	testSuite.NoError(err)
	testSuite.bandScorer = bandScorer
	testSuite.logger = testlog.Logger(testSuite.T(), log.LvlError)
}

// TestPeerScorer runs the PeerScorerTestSuite.
func TestPeerScorer(t *testing.T) {
	suite.Run(t, new(PeerScorerTestSuite))
}

// TestScorer_OnConnect ensures we can call the OnConnect method on the peer scorer.
func (testSuite *PeerScorerTestSuite) TestScorer_OnConnect() {
	scorer := p2p.NewScorer(
		testSuite.mockStore,
		testSuite.mockMetricer,
		testSuite.bandScorer,
		testSuite.logger,
	)
	scorer.OnConnect(peer.ID("alice"))
}

// TestScorer_OnDisconnect ensures we can call the OnDisconnect method on the peer scorer.
func (testSuite *PeerScorerTestSuite) TestScorer_OnDisconnect() {
	scorer := p2p.NewScorer(
		testSuite.mockStore,
		testSuite.mockMetricer,
		testSuite.bandScorer,
		testSuite.logger,
	)
	scorer.OnDisconnect(peer.ID("alice"))
}

// TestScorer_SnapshotHook tests running the snapshot hook on the peer scorer.
func (testSuite *PeerScorerTestSuite) TestScorer_SnapshotHook() {
	scorer := p2p.NewScorer(
		testSuite.mockStore,
		testSuite.mockMetricer,
		testSuite.bandScorer,
		testSuite.logger,
	)
	inspectFn := scorer.SnapshotHook()

	// The metricer should then be called with the peer score band map
	testSuite.mockMetricer.On("SetPeerScores", map[string]float64{
		"friend":   0,
		"graylist": 1,
	}).Return(nil).Once()

	// Apply the snapshot
	snapshotMap := map[peer.ID]*pubsub.PeerScoreSnapshot{
		peer.ID("peer1"): {
			Score: -100,
		},
	}
	inspectFn(snapshotMap)

	// The metricer should then be called with the peer score band map
	testSuite.mockMetricer.On("SetPeerScores", map[string]float64{
		"friend":   1,
		"graylist": 0,
	}).Return(nil).Once()

	// Apply the snapshot
	snapshotMap = map[peer.ID]*pubsub.PeerScoreSnapshot{
		peer.ID("peer1"): {
			Score: 0,
		},
	}
	inspectFn(snapshotMap)
}

// TestScorer_SnapshotHookBlocksPeer tests running the snapshot hook on the peer scorer with a peer score below the threshold.
// This implies that the peer should be blocked.
func (testSuite *PeerScorerTestSuite) TestScorer_SnapshotHookBlocksPeer() {
	scorer := p2p.NewScorer(
		testSuite.mockStore,
		testSuite.mockMetricer,
		testSuite.bandScorer,
		testSuite.logger,
	)
	inspectFn := scorer.SnapshotHook()

	// The metricer should then be called with the peer score band map
	testSuite.mockMetricer.On("SetPeerScores", map[string]float64{
		"friend":   0,
		"graylist": 1,
	}).Return(nil)

	// Apply the snapshot
	snapshotMap := map[peer.ID]*pubsub.PeerScoreSnapshot{
		peer.ID("peer1"): {
			Score: -101,
		},
	}
	inspectFn(snapshotMap)
}
