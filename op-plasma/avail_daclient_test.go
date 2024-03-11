package plasma

import (
	"context"
	"math/rand"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/stretchr/testify/require"
)

func TestAvailDAClient(t *testing.T) {
	ctx := context.Background()
	client := getClient(t)

	rng := rand.New(rand.NewSource(1234))
	input := testutils.RandomData(rng, 2000)

	comm, err := client.SetInput(ctx, input)
	require.NoError(t, err)

	returnData, err := client.GetInput(ctx, comm)

	require.NoError(t, err)
	require.Equal(t, returnData, input)
}

func getClient(t *testing.T) DataClient {

	cfg := getCliConfig()

	client := cfg.NewDAClient()
	require.NoError(t, cfg.Check())
	return client
}

func getCliConfig() CLIConfig {
	return CLIConfig{
		Enabled:      true,
		DAServerURL:  "wss://goldberg.avail.tools/ws",
		VerifyOnRead: true,
		UseAvailDA:   true,
	}
}
