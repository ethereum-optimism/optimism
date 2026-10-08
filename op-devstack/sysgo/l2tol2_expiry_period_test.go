package sysgo

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCheckL2ToL2MessageExpiryPeriod(t *testing.T) {
	window := uint64(12)
	zero := uint64(0)
	withPeriod := func(period uint64, window *uint64) PresetConfig {
		return PresetConfig{L2ToL2MessageExpiryPeriod: period, MessageExpiryWindow: window}
	}

	require.NoError(t, checkL2ToL2MessageExpiryPeriod(PresetConfig{}, false, 10), "no period, no constraint")
	require.NoError(t, checkL2ToL2MessageExpiryPeriod(withPeriod(30, &window), true, 0))

	require.ErrorContains(t, checkL2ToL2MessageExpiryPeriod(withPeriod(12, &window), true, 0), "must exceed")
	require.ErrorContains(t, checkL2ToL2MessageExpiryPeriod(withPeriod(5, &window), true, 0), "must exceed")
	require.ErrorContains(t, checkL2ToL2MessageExpiryPeriod(withPeriod(30, nil), true, 0), "must exceed",
		"an unset window is the protocol's 7 days")
	require.ErrorContains(t, checkL2ToL2MessageExpiryPeriod(withPeriod(30, &zero), true, 0), "must exceed",
		"a zero window override is the protocol's 7 days")
	require.ErrorContains(t, checkL2ToL2MessageExpiryPeriod(withPeriod(30, &window), false, 0), "interop at genesis")
	require.ErrorContains(t, checkL2ToL2MessageExpiryPeriod(withPeriod(30, &window), true, 6), "interop at genesis")
}
