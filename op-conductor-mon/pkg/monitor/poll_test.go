package monitor

import (
	"context"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-conductor-mon/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestNetwork_cleanupState(t *testing.T) {
	t.Run("should remove expired state", func(t *testing.T) {
		n := &Poller{
			config: &config.Config{NodeStateExpiration: 10 * time.Hour},
			state: map[string]*NodeState{
				"clean_me": {updatedAt: time.Now().Add(-11 * time.Hour)},
				"keep_me":  {updatedAt: time.Now()},
			},
		}
		ctx := context.Background()
		require.Equal(t, 2, len(n.state))
		n.cleanup(ctx)
		require.Equal(t, 1, len(n.state))
		require.NotNil(t, n.state["keep_me"])
	})
}
