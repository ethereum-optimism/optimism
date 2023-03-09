package p2p

import (
	"math"
	"sort"
	"testing"
	"time"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/stretchr/testify/suite"
)

// PeerParamsTestSuite tests peer parameterization.
type PeerParamsTestSuite struct {
	suite.Suite
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

// TestGetPeerScoreParams validates the peer score parameters.
func (testSuite *PeerParamsTestSuite) TestGetPeerScoreParams() {
	params, err := GetPeerScoreParams("light", 1)
	testSuite.NoError(err)
	expected := LightPeerScoreParams(1)
	testSuite.Equal(expected.DecayInterval, params.DecayInterval)
	testSuite.Equal(time.Duration(1)*time.Second, params.DecayInterval)

	params, err = GetPeerScoreParams("none", 1)
	testSuite.NoError(err)
	expected = DisabledPeerScoreParams(1)
	testSuite.Equal(expected.DecayInterval, params.DecayInterval)
	testSuite.Equal(time.Duration(1)*time.Second, params.DecayInterval)

	_, err = GetPeerScoreParams("invalid", 1)
	testSuite.Error(err)
}

// TestLightPeerScoreParams validates the light peer score params.
func (testSuite *PeerParamsTestSuite) TestLightPeerScoreParams() {
	blockTime := uint64(1)
	slot := time.Duration(blockTime) * time.Second
	epoch := 6 * slot
	oneHundredEpochs := 100 * epoch

	// calculate the behavior penalty decay
	duration := 10 * epoch
	decay := math.Pow(DecayToZero, 1/float64(duration/slot))
	testSuite.Equal(0.9261187281287935, decay)

	// Test the params
	params, err := GetPeerScoreParams("light", blockTime)
	testSuite.NoError(err)
	testSuite.Equal(params.Topics, make(map[string]*pubsub.TopicScoreParams))
	testSuite.Equal(params.TopicScoreCap, float64(34))
	// testSuite.Equal(params.AppSpecificScore("alice"), float(0))
	testSuite.Equal(params.AppSpecificWeight, float64(1))
	testSuite.Equal(params.IPColocationFactorWeight, float64(-35))
	testSuite.Equal(params.IPColocationFactorThreshold, int(10))
	testSuite.Nil(params.IPColocationFactorWhitelist)
	testSuite.Equal(params.BehaviourPenaltyWeight, float64(-16))
	testSuite.Equal(params.BehaviourPenaltyThreshold, float64(6))
	testSuite.Equal(params.BehaviourPenaltyDecay, decay)
	testSuite.Equal(params.DecayInterval, slot)
	testSuite.Equal(params.DecayToZero, DecayToZero)
	testSuite.Equal(params.RetainScore, oneHundredEpochs)
}

// TestDisabledPeerScoreParams validates the disabled peer score params.
func (testSuite *PeerParamsTestSuite) TestDisabledPeerScoreParams() {
	blockTime := uint64(1)
	slot := time.Duration(blockTime) * time.Second
	epoch := 6 * slot
	oneHundredEpochs := 100 * epoch

	// calculate the behavior penalty decay
	duration := 10 * epoch
	decay := math.Pow(DecayToZero, 1/float64(duration/slot))
	testSuite.Equal(0.9261187281287935, decay)

	// Test the params
	params, err := GetPeerScoreParams("none", blockTime)
	testSuite.NoError(err)
	testSuite.Equal(params.Topics, make(map[string]*pubsub.TopicScoreParams))
	testSuite.Equal(params.TopicScoreCap, float64(0))
	testSuite.Equal(params.AppSpecificWeight, float64(1))
	testSuite.Equal(params.IPColocationFactorWeight, float64(0))
	testSuite.Nil(params.IPColocationFactorWhitelist)
	testSuite.Equal(params.BehaviourPenaltyWeight, float64(0))
	testSuite.Equal(params.BehaviourPenaltyDecay, decay)
	testSuite.Equal(params.DecayInterval, slot)
	testSuite.Equal(params.DecayToZero, DecayToZero)
	testSuite.Equal(params.RetainScore, oneHundredEpochs)
}

// TestParamsZeroBlockTime validates peer score params use default slot for 0 block time.
func (testSuite *PeerParamsTestSuite) TestParamsZeroBlockTime() {
	slot := 2 * time.Second
	params, err := GetPeerScoreParams("none", uint64(0))
	testSuite.NoError(err)
	testSuite.Equal(params.DecayInterval, slot)
	params, err = GetPeerScoreParams("light", uint64(0))
	testSuite.NoError(err)
	testSuite.Equal(params.DecayInterval, slot)
}
