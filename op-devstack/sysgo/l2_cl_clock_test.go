package sysgo

import (
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/stretchr/testify/require"
)

func TestL2CLRuntimeClockOption(t *testing.T) {
	runtimeClock := clock.NewDeterministicClock(time.Unix(1_000, 0))
	config := DefaultL2CLConfig()

	L2CLRuntimeClock(runtimeClock).Apply(nil, ComponentTarget{}, config)

	require.Same(t, runtimeClock, config.RuntimeClock)
}
