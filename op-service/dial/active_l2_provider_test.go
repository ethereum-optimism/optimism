package dial

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

// do a test with just rollupclients
// then do a test with rollupclients and ethclients
// TestActiveSequencerFailoverBehavior tests the behavior of the ActiveSequencerProvider when the active sequencer fails
func TestActiveSequencerFailoverBehavior_RollupProviders(t *testing.T) {
	primarySequencer := testutils.MockRollupClient{}
	primarySequencer.ExpectSequencerActive(true, nil)
	primarySequencer.ExpectSequencerActive(false, nil)
	secondarySequencer := testutils.MockRollupClient{}
	secondarySequencer.ExpectSequencerActive(true, nil)

	rollupClients := []RollupClientInterface{}
	rollupClients = append(rollupClients, &primarySequencer)
	rollupClients = append(rollupClients, &secondarySequencer)

	endpointProvider := ActiveL2RollupProvider{
		rollupClients:  rollupClients,
		checkDuration:  1 * time.Duration(time.Microsecond),
		networkTimeout: 1 * time.Duration(time.Second),
		log:            testlog.Logger(t, log.LvlDebug),
		activeTimeout:  time.Now(),
		currentIdx:     0,
		clientLock:     &sync.Mutex{},
	}
	_, err := endpointProvider.RollupClient(context.Background())
	require.NoError(t, err)
	require.Equal(t, 0, endpointProvider.currentIdx)
	_, err = endpointProvider.RollupClient(context.Background())
	require.NoError(t, err)
	require.Equal(t, 1, endpointProvider.currentIdx)
}

func TestActiveSequencerFailoverBehavior_L2Providers(t *testing.T) {
	primarySequencer := testutils.MockRollupClient{}
	primarySequencer.ExpectSequencerActive(true, nil)
	primarySequencer.ExpectSequencerActive(false, nil)
	secondarySequencer := testutils.MockRollupClient{}
	secondarySequencer.ExpectSequencerActive(true, nil)

	rollupClients := []RollupClientInterface{}
	rollupClients = append(rollupClients, &primarySequencer)
	rollupClients = append(rollupClients, &secondarySequencer)

	rollupProvider := ActiveL2RollupProvider{
		rollupClients:  rollupClients,
		checkDuration:  1 * time.Duration(time.Microsecond),
		networkTimeout: 1 * time.Duration(time.Second),
		log:            testlog.Logger(t, log.LvlDebug),
		activeTimeout:  time.Now(),
		currentIdx:     0,
		clientLock:     &sync.Mutex{},
	}
	ethClients := []EthClientInterface{}
	primaryEthClient := testutils.MockEthClient{}
	ethClients = append(ethClients, &primaryEthClient)
	secondaryEthClient := testutils.MockEthClient{}
	ethClients = append(ethClients, &secondaryEthClient)
	endpointProvider := ActiveL2EndpointProvider{
		ActiveL2RollupProvider: rollupProvider,
		ethClients:             ethClients,
	}
	_, err := endpointProvider.EthClient(context.Background())
	require.NoError(t, err)
	require.Equal(t, 0, endpointProvider.currentIdx)
	_, err = endpointProvider.EthClient(context.Background())
	require.NoError(t, err)
	require.Equal(t, 1, endpointProvider.currentIdx)
}
