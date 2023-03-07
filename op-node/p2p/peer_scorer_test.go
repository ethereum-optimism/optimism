package p2p_test

import (
	"testing"

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
