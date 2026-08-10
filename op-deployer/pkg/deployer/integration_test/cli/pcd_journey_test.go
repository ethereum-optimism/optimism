package cli

import (
	"path/filepath"
	"testing"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/pipeline"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/upgrade/embedded"
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

func TestCLIPCDJourney(t *testing.T) {
	rows := []struct {
		name              string
		chainIDs          []common.Hash
		respectedGameType embedded.GameType
	}{
		{
			name:              "singleton-super-root",
			chainIDs:          []common.Hash{uint256.NewInt(1).Bytes32()},
			respectedGameType: embedded.GameTypeSuperCannonKona,
		},
	}

	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			journey := newPCDJourneyFixture(t, row.chainIDs)
			journey.bootstrapOPCM()
			journey.runInit(row.respectedGameType)

			require.FileExists(t, filepath.Join(journey.workdir, "intent.toml"))
			require.FileExists(t, filepath.Join(journey.workdir, "state.json"))
			prepared := journey.runPrepare()

			_, err := pipeline.ReadIntent(journey.workdir)
			require.NoError(t, err)
			require.NotNil(t, prepared.PreparedDeployment)
			require.NotEmpty(t, prepared.PreparedDeployment.Chains)

			predictedAddresses := pcdPreparedContractAddresses(prepared.PreparedDeployment)
			require.NotEmpty(t, predictedAddresses)
			probe := pcdL1Probe{client: journey.l1Client, deployer: journey.deployer}
			preparedL1State := probe.read(t, predictedAddresses)
			requireNoPCDDeploymentMutation(t, journey.postBootstrapL1State, preparedL1State)
		})
	}
}
