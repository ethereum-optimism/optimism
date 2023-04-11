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
func (testSuite *PeerGaterTestSuite) TestPeerGater_UpdateBansPeers() {
	gater := p2p.NewPeerGater(
		testSuite.mockGater,
		testSuite.logger,
		true,
	)

	// Return an empty list of already blocked peers
	testSuite.mockGater.On("ListBlockedPeers").Return([]peer.ID{}).Once()

	// Mock a connection gater peer block call
	// Since the peer score is below the [PeerScoreThreshold] of -100,
	// the [BlockPeer] method should be called
	testSuite.mockGater.On("BlockPeer", peer.ID("peer1")).Return(nil).Once()

	// The peer should initially be unblocked
	testSuite.False(gater.IsBlocked(peer.ID("peer1")))

	// Apply the peer gater update
	gater.Update(peer.ID("peer1"), float64(-101))

	// The peer should be considered blocked
	testSuite.True(gater.IsBlocked(peer.ID("peer1")))

	// Now let's unblock the peer
	testSuite.mockGater.On("UnblockPeer", peer.ID("peer1")).Return(nil).Once()
	gater.Update(peer.ID("peer1"), float64(0))

	// The peer should be considered unblocked
	testSuite.False(gater.IsBlocked(peer.ID("peer1")))
}

// TestPeerGaterUpdateNoBanning tests the peer gater update hook without banning set
func (testSuite *PeerGaterTestSuite) TestPeerGater_UpdateNoBanning() {
	gater := p2p.NewPeerGater(
		testSuite.mockGater,
		testSuite.logger,
		false,
	)

	// Return an empty list of already blocked peers
	testSuite.mockGater.On("ListBlockedPeers").Return([]peer.ID{})

	// Notice: [BlockPeer] should not be called since banning is not enabled
	// even though the peer score is way below the [PeerScoreThreshold] of -100
	gater.Update(peer.ID("peer1"), float64(-100000))

	// The peer should be unblocked
	testSuite.False(gater.IsBlocked(peer.ID("peer1")))

	// Make sure that if we then "unblock" the peer, nothing happens
	gater.Update(peer.ID("peer1"), float64(0))

	// The peer should still be unblocked
	testSuite.False(gater.IsBlocked(peer.ID("peer1")))
}
