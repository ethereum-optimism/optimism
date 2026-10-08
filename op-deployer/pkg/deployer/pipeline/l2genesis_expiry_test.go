package pipeline

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/stretchr/testify/require"
)

func TestCheckL2ToL2MessageExpiryPeriodOverride(t *testing.T) {
	for l1ChainID := range publicL1ChainIDs {
		require.NoError(t, checkL2ToL2MessageExpiryPeriodOverride(state.IntentTypeCustom, l1ChainID, 0),
			"the production period is always allowed")
		require.ErrorContains(t, checkL2ToL2MessageExpiryPeriodOverride(state.IntentTypeCustom, l1ChainID, 30),
			"test networks only")
	}
	for _, configType := range []state.IntentType{state.IntentTypeStandard, state.IntentTypeStandardOverrides} {
		require.NoError(t, checkL2ToL2MessageExpiryPeriodOverride(configType, 900, 0))
		require.ErrorContains(t, checkL2ToL2MessageExpiryPeriodOverride(configType, 900, 30), "test networks only")
	}
	require.NoError(t, checkL2ToL2MessageExpiryPeriodOverride(state.IntentTypeCustom, 900, 30),
		"a custom intent on a local devnet may shorten the period")
}
