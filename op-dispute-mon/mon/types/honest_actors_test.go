package types

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestHonestActorsNeverIncludesZeroAddress(t *testing.T) {
	actor := common.Address{0x11}
	honest := NewHonestActors([]common.Address{{}, actor})
	require.False(t, honest.Contains(common.Address{}))
	require.True(t, honest.Contains(actor))
}
