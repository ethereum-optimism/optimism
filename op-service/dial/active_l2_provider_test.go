package dial

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

// TestActiveSequencerFailoverBehavior_RollupProvider tests that the ActiveL2RollupProvider
// will failover to the next provider if the current one is not active.
func TestActiveSequencerFailoverBehavior_RollupProviders(t *testing.T) {
	// Create two mock rollup clients, one of which will declare itself inactive after first check.
	primarySequencer := testutils.MockRollupClient{}
	primarySequencer.ExpectSequencerActive(true, nil)
	primarySequencer.ExpectSequencerActive(false, nil)
	primarySequencer.ExpectClose()
	secondarySequencer := testutils.MockRollupClient{}
	secondarySequencer.ExpectSequencerActive(true, nil)

	mockRollupDialer := func(ctx context.Context, timeout time.Duration, log log.Logger, url string) (RollupClientInterface, error) {
		if url == "primary" {
			return &primarySequencer, nil
		} else if url == "secondary" {
			return &secondarySequencer, nil
		} else {
			return nil, fmt.Errorf("unknown test url: %s", url)
		}
	}

	endpointProvider, err := NewActiveL2RollupProvider(
		context.Background(),
		[]string{"primary", "secondary"},
		1*time.Microsecond,
		1*time.Minute,
		testlog.Logger(t, log.LvlDebug),
		mockRollupDialer,
	)
	require.NoError(t, err)
	// Check that the first client is used, then the second once the first declares itself inactive.
	firstSequencerUsed, err := endpointProvider.RollupClient(context.Background())
	require.NoError(t, err)
	require.True(t, &primarySequencer == firstSequencerUsed) // avoids copying the struct (and its mutex, etc.)
	secondSequencerUsed, err := endpointProvider.RollupClient(context.Background())
	require.NoError(t, err)
	require.True(t, &secondarySequencer == secondSequencerUsed)
}

// TestActiveSequencerFailoverBehavior_L2Providers tests that the ActiveL2EndpointProvider
// will failover to the next provider if the current one is not active.
func TestActiveSequencerFailoverBehavior_L2Providers(t *testing.T) {
	// as TestActiveSequencerFailoverBehavior_RollupProviders,
	// but ensure the added `EthClient()` method also triggers the failover.
	primarySequencer := testutils.MockRollupClient{}
	primarySequencer.ExpectSequencerActive(true, nil)
	primarySequencer.ExpectSequencerActive(false, nil)
	primarySequencer.ExpectClose()
	secondarySequencer := testutils.MockRollupClient{}
	secondarySequencer.ExpectSequencerActive(true, nil)

	mockRollupDialer := func(ctx context.Context, timeout time.Duration, log log.Logger, url string) (RollupClientInterface, error) {
		if url == "primary" {
			return &primarySequencer, nil
		} else if url == "secondary" {
			return &secondarySequencer, nil
		} else {
			return nil, fmt.Errorf("unknown test url: %s", url)
		}
	}
	primaryEthClient := testutils.MockEthClient{}
	primaryEthClient.ExpectClose()
	secondaryEthClient := testutils.MockEthClient{}
	mockEthDialer := func(ctx context.Context, timeout time.Duration, log log.Logger, url string) (EthClientInterface, error) {
		if url == "primary" {
			return &primaryEthClient, nil
		} else if url == "secondary" {
			return &secondaryEthClient, nil
		} else {
			return nil, fmt.Errorf("unknown test url: %s", url)
		}
	}
	endpointProvider, err := NewActiveL2EndpointProvider(
		context.Background(),
		[]string{"primary", "secondary"},
		[]string{"primary", "secondary"},
		1*time.Microsecond,
		1*time.Minute,
		testlog.Logger(t, log.LvlDebug),
		mockEthDialer,
		mockRollupDialer,
	)
	require.NoError(t, err)
	firstClientUsed, err := endpointProvider.EthClient(context.Background())
	require.NoError(t, err)
	require.True(t, &primaryEthClient == firstClientUsed) // avoids copying the struct (and its mutex, etc.)
	secondClientUsed, err := endpointProvider.EthClient(context.Background())
	require.NoError(t, err)
	require.True(t, &secondaryEthClient == secondClientUsed)
}
