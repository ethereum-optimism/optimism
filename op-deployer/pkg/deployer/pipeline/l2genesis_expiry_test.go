package pipeline

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCheckL2ToL2MessageExpiryPeriodOverride(t *testing.T) {
	for l1ChainID := range publicL1ChainIDs {
		require.NoError(t, checkL2ToL2MessageExpiryPeriodOverride(l1ChainID, 0), "the production period is always allowed")
		require.ErrorContains(t, checkL2ToL2MessageExpiryPeriodOverride(l1ChainID, 30), "test networks only")
	}
	require.NoError(t, checkL2ToL2MessageExpiryPeriodOverride(900, 30), "a local devnet may shorten the period")
}
