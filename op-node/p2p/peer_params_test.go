package p2p

import (
	"sort"
	"testing"

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

// TestAvailablePeerScoreParams validates the available peer score parameters.
func (testSuite *PeerParamsTestSuite) TestAvailablePeerScoreParams() {
	available := AvailablePeerScoreParams()
	sort.Strings(available)
	expected := []string{"light", "none"}
	testSuite.Equal(expected, available)
}
