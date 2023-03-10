package p2p_test

import (
	"testing"

	p2p "github.com/ethereum-optimism/optimism/op-node/p2p"
	p2pMocks "github.com/ethereum-optimism/optimism/op-node/p2p/mocks"
	testlog "github.com/ethereum-optimism/optimism/op-node/testlog"
	log "github.com/ethereum/go-ethereum/log"
	peer "github.com/libp2p/go-libp2p/core/peer"
	suite "github.com/stretchr/testify/suite"
)

// PeerGaterTestSuite tests peer parameterization.
type PeerGaterTestSuite struct {
	suite.Suite

	mockGater *p2pMocks.ConnectionGater
	logger    log.Logger
}

// SetupTest sets up the test suite.
func (testSuite *PeerGaterTestSuite) SetupTest() {
	testSuite.mockGater = &p2pMocks.ConnectionGater{}
	testSuite.logger = testlog.Logger(testSuite.T(), log.LvlError)
}

// TestPeerGater runs the PeerGaterTestSuite.
func TestPeerGater(t *testing.T) {
	suite.Run(t, new(PeerGaterTestSuite))
}

// TestPeerScoreConstants validates the peer score constants.
func (testSuite *PeerGaterTestSuite) TestPeerScoreConstants() {
	testSuite.Equal(-10, p2p.ConnectionFactor)
	testSuite.Equal(-100, p2p.PeerScoreThreshold)
}

// TestPeerGaterUpdate tests the peer gater update hook.
func (testSuite *PeerGaterTestSuite) TestPeerGaterUpdate() {
	gater := p2p.NewPeerGater(
		testSuite.mockGater,
		testSuite.logger,
		true,
	)

	// Mock a connection gater peer block call
	// Since the peer score is below the [PeerScoreThreshold] of -100,
	// the [BlockPeer] method should be called
	testSuite.mockGater.On("BlockPeer", peer.ID("peer1")).Return(nil)

	// Apply the peer gater update
	gater.Update(peer.ID("peer1"), float64(-100))
}

// TestPeerGaterUpdateNoBanning tests the peer gater update hook without banning set
func (testSuite *PeerGaterTestSuite) TestPeerGaterUpdateNoBanning() {
	gater := p2p.NewPeerGater(
		testSuite.mockGater,
		testSuite.logger,
		false,
	)

	// Notice: [BlockPeer] should not be called since banning is not enabled
	gater.Update(peer.ID("peer1"), float64(-100000))
}
