package types

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMaxValue(t *testing.T) {
	require.Equal(t, ResolvedBondAmount.String(), "340282366920938463463374607431768211455")
}
