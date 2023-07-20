package op_e2e

import (
	"context"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/client/utils"
	"github.com/stretchr/testify/require"
)

func TestTimeTravel(t *testing.T) {
	InitParallel(t)

	cfg := DefaultSystemConfig(t)
	delete(cfg.Nodes, "verifier")
	cfg.SupportL1TimeTravel = true
	sys, err := cfg.Start()
	require.Nil(t, err, "Error starting up system")
	defer sys.Close()

	l1Client := sys.Clients["l1"]
	preTravel, err := l1Client.BlockByNumber(context.Background(), nil)
	require.NoError(t, err)

	sys.TimeTravelClock.AdvanceTime(24 * time.Hour)

	// Check that the L1 chain reaches the new time reasonably quickly (ie without taking a week)
	// It should be able to jump straight to the new time with just a single block
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	err = utils.WaitFor(ctx, time.Second, func() (bool, error) {
		postTravel, err := l1Client.BlockByNumber(context.Background(), nil)
		if err != nil {
			return false, err
		}
		diff := time.Duration(postTravel.Time()-preTravel.Time()) * time.Second
		return diff.Hours() > 23, nil
	})
	require.NoError(t, err)
}
