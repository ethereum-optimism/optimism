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

// TestActiveSequencerFailoverBehavior_RollupProviders_Inactive tests that the ActiveL2RollupProvider
// will failover to the next provider if the current one is not active.
func TestActiveSequencerFailoverBehavior_RollupProviders_Inactive(t *testing.T) {
	// Create two mock rollup clients, one of which will declare itself inactive after first check.
	primarySequencer := new(testutils.MockRollupClient)
	primarySequencer.ExpectSequencerActive(true, nil)
	primarySequencer.ExpectSequencerActive(false, nil)
	primarySequencer.ExpectClose()
	secondarySequencer := new(testutils.MockRollupClient)
	secondarySequencer.ExpectSequencerActive(true, nil)

	mockRollupDialer := func(ctx context.Context, timeout time.Duration, log log.Logger, url string) (RollupClientInterface, error) {
		if url == "primaryRollup" {
			return primarySequencer, nil
		} else if url == "secondaryRollup" {
			return secondarySequencer, nil
		} else {
			return nil, fmt.Errorf("unknown test url: %s", url)
		}
	}

	endpointProvider, err := newActiveL2RollupProvider(
		context.Background(),
		[]string{"primaryRollup", "secondaryRollup"},
		1*time.Microsecond,
		1*time.Minute,
		testlog.Logger(t, log.LvlDebug),
		mockRollupDialer,
	)
	require.NoError(t, err)
	// Check that the first client is used, then the second once the first declares itself inactive.
	firstSequencerUsed, err := endpointProvider.RollupClient(context.Background())
	require.NoError(t, err)
	require.Same(t, primarySequencer, firstSequencerUsed)
	secondSequencerUsed, err := endpointProvider.RollupClient(context.Background())
	require.NoError(t, err)
	require.Same(t, secondarySequencer, secondSequencerUsed)
}

// TestActiveSequencerFailoverBehavior_L2Providers_Inactive tests that the ActiveL2EndpointProvider
// will failover to the next provider if the current one is not active.
func TestActiveSequencerFailoverBehavior_L2Providers_Inactive(t *testing.T) {
	// as TestActiveSequencerFailoverBehavior_RollupProviders,
	// but ensure the added `EthClient()` method also triggers the failover.
	primarySequencer := new(testutils.MockRollupClient)
	primarySequencer.ExpectSequencerActive(true, nil)
	primarySequencer.ExpectSequencerActive(false, nil)
	primarySequencer.ExpectClose()
	secondarySequencer := new(testutils.MockRollupClient)
	secondarySequencer.ExpectSequencerActive(true, nil)

	mockRollupDialer := func(ctx context.Context, timeout time.Duration, log log.Logger, url string) (RollupClientInterface, error) {
		if url == "primaryRollup" {
			return primarySequencer, nil
		} else if url == "secondaryRollup" {
			return secondarySequencer, nil
		} else {
			return nil, fmt.Errorf("unknown test url: %s", url)
		}
	}
	primaryEthClient := new(testutils.MockEthClient)
	primaryEthClient.ExpectClose()
	secondaryEthClient := new(testutils.MockEthClient)
	mockEthDialer := func(ctx context.Context, timeout time.Duration, log log.Logger, url string) (EthClientInterface, error) {
		if url == "primaryEth" {
			return primaryEthClient, nil
		} else if url == "secondaryEth" {
			return secondaryEthClient, nil
		} else {
			return nil, fmt.Errorf("unknown test url: %s", url)
		}
	}
	endpointProvider, err := newActiveL2EndpointProvider(
		context.Background(),
		[]string{"primaryEth", "secondaryEth"},
		[]string{"primaryRollup", "secondaryRollup"},
		1*time.Microsecond,
		1*time.Minute,
		testlog.Logger(t, log.LvlDebug),
		mockEthDialer,
		mockRollupDialer,
	)
	require.NoError(t, err)
	firstClientUsed, err := endpointProvider.EthClient(context.Background())
	require.NoError(t, err)
	require.Same(t, primaryEthClient, firstClientUsed)
	secondClientUsed, err := endpointProvider.EthClient(context.Background())
	require.NoError(t, err)
	require.Same(t, secondaryEthClient, secondClientUsed)
}

// TestActiveSequencerFailoverBehavior_RollupProviders_Errored is as the _Inactive test,
// but with the first provider returning an error instead of declaring itself inactive.
func TestActiveSequencerFailoverBehavior_RollupProviders_Errored(t *testing.T) {
	// Create two mock rollup clients, one of which will error out after first check
	primarySequencer := new(testutils.MockRollupClient)
	primarySequencer.ExpectSequencerActive(true, nil)
	primarySequencer.ExpectSequencerActive(true, fmt.Errorf("a test error"))
	primarySequencer.ExpectClose()
	secondarySequencer := new(testutils.MockRollupClient)
	secondarySequencer.ExpectSequencerActive(true, nil)

	mockRollupDialer := func(ctx context.Context, timeout time.Duration, log log.Logger, url string) (RollupClientInterface, error) {
		if url == "primaryRollup" {
			return primarySequencer, nil
		} else if url == "secondaryRollup" {
			return secondarySequencer, nil
		} else {
			return nil, fmt.Errorf("unknown test url: %s", url)
		}
	}

	endpointProvider, err := newActiveL2RollupProvider(
		context.Background(),
		[]string{"primaryRollup", "secondaryRollup"},
		1*time.Microsecond,
		1*time.Minute,
		testlog.Logger(t, log.LvlDebug),
		mockRollupDialer,
	)
	require.NoError(t, err)
	// Check that the first client is used, then the second once the first declares itself inactive.
	firstSequencerUsed, err := endpointProvider.RollupClient(context.Background())
	require.NoError(t, err)
	require.Same(t, primarySequencer, firstSequencerUsed)
	secondSequencerUsed, err := endpointProvider.RollupClient(context.Background())
	require.NoError(t, err)
	require.Same(t, secondarySequencer, secondSequencerUsed)
}

// TestActiveSequencerFailoverBehavior_L2Providers_Errored is as the _Inactive test,
// but with the first provider returning an error instead of declaring itself inactive.
func TestActiveSequencerFailoverBehavior_L2Providers_Errored(t *testing.T) {
	// as TestActiveSequencerFailoverBehavior_RollupProviders,
	// but ensure the added `EthClient()` method also triggers the failover.
	primarySequencer := new(testutils.MockRollupClient)
	primarySequencer.ExpectSequencerActive(true, nil)
	primarySequencer.ExpectSequencerActive(false, fmt.Errorf("a test error"))
	primarySequencer.ExpectClose()
	secondarySequencer := new(testutils.MockRollupClient)
	secondarySequencer.ExpectSequencerActive(true, nil)

	mockRollupDialer := func(ctx context.Context, timeout time.Duration, log log.Logger, url string) (RollupClientInterface, error) {
		if url == "primaryRollup" {
			return primarySequencer, nil
		} else if url == "secondaryRollup" {
			return secondarySequencer, nil
		} else {
			return nil, fmt.Errorf("unknown test url: %s", url)
		}
	}
	primaryEthClient := new(testutils.MockEthClient)
	primaryEthClient.ExpectClose()
	secondaryEthClient := new(testutils.MockEthClient)
	mockEthDialer := func(ctx context.Context, timeout time.Duration, log log.Logger, url string) (EthClientInterface, error) {
		if url == "primaryEth" {
			return primaryEthClient, nil
		} else if url == "secondaryEth" {
			return secondaryEthClient, nil
		} else {
			return nil, fmt.Errorf("unknown test url: %s", url)
		}
	}
	endpointProvider, err := newActiveL2EndpointProvider(
		context.Background(),
		[]string{"primaryEth", "secondaryEth"},
		[]string{"primaryRollup", "secondaryRollup"},
		1*time.Microsecond,
		1*time.Minute,
		testlog.Logger(t, log.LvlDebug),
		mockEthDialer,
		mockRollupDialer,
	)
	require.NoError(t, err)
	firstClientUsed, err := endpointProvider.EthClient(context.Background())
	require.NoError(t, err)
	require.Same(t, primaryEthClient, firstClientUsed)
	secondClientUsed, err := endpointProvider.EthClient(context.Background())
	require.NoError(t, err)
	require.Same(t, secondaryEthClient, secondClientUsed)
}
