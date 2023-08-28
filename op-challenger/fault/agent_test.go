package fault

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-challenger/fault/types"
	"github.com/ethereum-optimism/optimism/op-node/testlog"
)

// TestShouldResolve tests the resolution logic.
func TestShouldResolve(t *testing.T) {
	log := testlog.Logger(t, log.LvlCrit)

	t.Run("AgreeWithProposedOutput", func(t *testing.T) {
		agent := NewAgent(nil, 0, nil, nil, nil, true, log)
		require.False(t, agent.shouldResolve(context.Background(), types.GameStatusDefenderWon))
		require.True(t, agent.shouldResolve(context.Background(), types.GameStatusChallengerWon))
		require.False(t, agent.shouldResolve(context.Background(), types.GameStatusInProgress))
	})

	t.Run("DisagreeWithProposedOutput", func(t *testing.T) {
		agent := NewAgent(nil, 0, nil, nil, nil, false, log)
		require.True(t, agent.shouldResolve(context.Background(), types.GameStatusDefenderWon))
		require.False(t, agent.shouldResolve(context.Background(), types.GameStatusChallengerWon))
		require.False(t, agent.shouldResolve(context.Background(), types.GameStatusInProgress))
	})
}
