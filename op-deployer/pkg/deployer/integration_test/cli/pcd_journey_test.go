package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
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

			artifacts := journey.runInspect()
			require.Len(t, artifacts, 1)
			journey.writeDependencySet()

			prestatePath := pcdPrestateArtifactPath(t)
			prestate := requirePCDPrestate(t, prestatePath)
			journey.runPrestate(prestate)
		})
	}
}

func TestPCDPrestateReader(t *testing.T) {
	want := common.HexToHash("0x1111111111111111111111111111111111111111111111111111111111111111")
	tests := []struct {
		name    string
		content []byte
		missing bool
		want    common.Hash
	}{
		{
			name:    "missing directory",
			missing: true,
		},
		{
			name:    "malformed JSON",
			content: []byte("{\"pre\":"),
		},
		{
			name:    "reserved hash",
			content: marshalPCDPrestateFile(t, opcm.PermissionedCannonFallbackPrestatePlaceholder),
		},
		{
			name:    "valid hash",
			content: marshalPCDPrestateFile(t, want),
			want:    want,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "artifact", "prestate-proof.json")
			if !test.missing {
				require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
				require.NoError(t, os.WriteFile(path, test.content, 0o644))
			}

			got, err := readPCDPrestate(path)
			if test.want == (common.Hash{}) {
				require.Error(t, err)
				require.ErrorContains(t, err, path)
				require.ErrorContains(t, err, "run just reproducible-prestate-kona")
				return
			}
			require.NoError(t, err)
			require.Equal(t, test.want, got)
		})
	}
}

func marshalPCDPrestateFile(t *testing.T, prestate common.Hash) []byte {
	t.Helper()
	data, err := json.Marshal(pcdPrestateFile{Pre: prestate})
	require.NoError(t, err)
	return data
}
