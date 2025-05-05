package dsl

import (
	"github.com/stretchr/testify/require"
)

func RequireL2UnsafeNumberEquals(t require.TestingT, c *Chain, unsafeNumber uint64) {
	require.Equal(t, unsafeNumber, c.Sequencer.L2Unsafe().Number)
}

func RequireL2UnsafeHashNotZero(t require.TestingT, c *Chain) {
	require.NotZero(t, c.Sequencer.L2Unsafe().Hash)
}
