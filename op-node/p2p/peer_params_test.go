package p2p

import (
	"sort"
	"testing"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/stretchr/testify/suite"
)

// PeerParamsTestSuite tests peer parameterization.
type PeerParamsTestSuite struct {
	suite.Suite
}

// SetupTest sets up the test suite.
func (testSuite *PeerParamsTestSuite) SetupTest() {
	// TODO:
}

// TestPeerParams runs the PeerParamsTestSuite.
func TestPeerParams(t *testing.T) {
	suite.Run(t, new(PeerParamsTestSuite))
}

// TestPeerScoreConstants validates the peer score constants.
func (testSuite *PeerParamsTestSuite) TestPeerScoreConstants() {
	testSuite.Equal(0.01, DecayToZero)
}

// TestAvailablePeerScoreParams validates the available peer score parameters.
func (testSuite *PeerParamsTestSuite) TestAvailablePeerScoreParams() {
	available := AvailablePeerScoreParams()
	sort.Strings(available)
	expected := []string{"light", "none"}
	testSuite.Equal(expected, available)
}

// TestNewPeerScoreThresholds validates the peer score thresholds.
//
// This is tested to ensure that the thresholds are not modified and missed in review.
func (testSuite *PeerParamsTestSuite) TestNewPeerScoreThresholds() {
	thresholds := NewPeerScoreThresholds()
	expected := pubsub.PeerScoreThresholds{
		SkipAtomicValidation:        false,
		GossipThreshold:             -10,
		PublishThreshold:            -40,
		GraylistThreshold:           -40,
		AcceptPXThreshold:           20,
		OpportunisticGraftThreshold: 0.05,
	}
	testSuite.Equal(expected, thresholds)
}
