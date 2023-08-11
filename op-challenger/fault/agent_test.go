package fault

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-node/testlog"
)

// TestAgent_ShouldResolve tests the [Agent] resolution logic.
func TestAgent_ShouldResolve(t *testing.T) {
	log := testlog.Logger(t, log.LvlCrit)

	t.Run("AgreeWithProposedOutput", func(t *testing.T) {
		agent := NewAgent(nil, 0, nil, nil, nil, true, log)
		require.False(t, agent.ShouldResolve(context.Background(), uint8(2)))
		require.True(t, agent.ShouldResolve(context.Background(), uint8(1)))
		require.False(t, agent.ShouldResolve(context.Background(), uint8(0)))
	})

	t.Run("DisagreeWithProposedOutput", func(t *testing.T) {
		agent := NewAgent(nil, 0, nil, nil, nil, false, log)
		require.True(t, agent.ShouldResolve(context.Background(), uint8(2)))
		require.False(t, agent.ShouldResolve(context.Background(), uint8(1)))
		require.False(t, agent.ShouldResolve(context.Background(), uint8(0)))
	})
}
