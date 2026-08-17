package broadcaster

import (
	"context"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestCaptureBroadcaster(t *testing.T) {
	caster := new(CaptureBroadcaster)
	want := []script.Broadcast{
		{Type: script.BroadcastCall, From: common.Address{0x01}, To: common.Address{0x02}, Nonce: 3},
		{Type: script.BroadcastCreate, From: common.Address{0x04}, Nonce: 5},
	}
	for _, bcast := range want {
		caster.Hook(bcast)
	}

	got := caster.Drain()
	require.Equal(t, want, got)
	require.Empty(t, caster.Drain())

	results, err := caster.Broadcast(context.Background())
	require.ErrorContains(t, err, "cannot broadcast")
	require.Nil(t, results)
}
