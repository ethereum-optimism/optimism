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

// TestCLIPCDJourney verifies that distinct dependency-set outputs use one genesis time and produce the same anchor for each chain.
func TestCLIPCDJourney(t *testing.T) {
	prestate := requirePCDPrestate(t, pcdPrestateArtifactPath(t))
	base := newPCDBootstrappedL1(t)

	// The OPCM created by this test enables SUPER_ROOT_GAMES_MIGRATION. The `prepare` command
	// rejects non-super game types for this OPCM, so this table contains only super-root cases.
	rows := []struct {
		name                       string
		chainIDs                   []common.Hash
		respectedGameType          embedded.GameType
		fallbackGameType           embedded.GameType
		expectedContinueNonceDelta uint64
	}{
		{
			name:                       "singleton-super-root",
			chainIDs:                   []common.Hash{uint256.NewInt(1).Bytes32()},
			respectedGameType:          embedded.GameTypeSuperCannonKona,
			fallbackGameType:           embedded.GameTypeSuperPermissioned,
			expectedContinueNonceDelta: 1,
		},
		{
			name:                       "two-chain-shared-super-root",
			chainIDs:                   []common.Hash{uint256.NewInt(1).Bytes32(), uint256.NewInt(2).Bytes32()},
			respectedGameType:          embedded.GameTypeSuperCannonKona,
			fallbackGameType:           embedded.GameTypeSuperPermissioned,
			expectedContinueNonceDelta: 2,
		},
	}

	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			journey := base.newJourney(t, row.chainIDs)
			journey.runInit(row.respectedGameType)

			require.FileExists(t, filepath.Join(journey.workdir, "intent.toml"))
			require.FileExists(t, filepath.Join(journey.workdir, "state.json"))
			prepared := journey.runPrepare()

			_, err := pipeline.ReadIntent(journey.workdir)
			require.NoError(t, err)
			require.NotNil(t, prepared.PreparedDeployment)
			require.Len(t, prepared.PreparedDeployment.Chains, len(row.chainIDs))

			predictedAddresses := pcdPredictedContractAddresses(prepared.PreparedDeployment)
			require.NotEmpty(t, predictedAddresses)
			probe := pcdL1Probe{client: journey.l1Client, deployer: journey.deployer}
			preparedL1State := probe.read(t, pcdDeploymentMarkerAddresses(predictedAddresses))
			requireNoPCDDeploymentMutation(t, journey.postBootstrapL1State, preparedL1State)

			artifacts := journey.runInspect()
			require.Len(t, artifacts, len(row.chainIDs))
			journey.writeDependencySet()
			expectedProposal, genesisTime, err := pcdSuperRootFromArtifacts(artifacts)
			require.NoErrorf(t, err, "compute proposal from rendered artifacts %v", pcdOracleArtifactPaths(artifacts))
			for _, artifact := range artifacts {
				rollupConfig, err := readPCDRollupConfig(artifact.rollupPath)
				require.NoErrorf(t, err, "read rollup config for row %s chain %s", row.name, artifact.chainID.Hex())
				require.Equalf(
					t,
					genesisTime,
					rollupConfig.Genesis.L2Time,
					"genesis time differs for row %s chain %s: expected %d, observed %d",
					row.name,
					artifact.chainID.Hex(),
					genesisTime,
					rollupConfig.Genesis.L2Time,
				)
			}
			if len(artifacts) == 2 {
				firstRoot := pcdArtifactOutputRoot(t, row.name, artifacts[0])
				secondRoot := pcdArtifactOutputRoot(t, row.name, artifacts[1])
				require.NotEqualf(
					t,
					firstRoot,
					secondRoot,
					"member output roots must differ for row %s chains %s and %s: both are %s",
					row.name,
					artifacts[0].chainID.Hex(),
					artifacts[1].chainID.Hex(),
					firstRoot,
				)
			}

			journey.runPrestate(prestate)
			journey.runContinue()

			liveChains := make([]pcdLiveChainExpectation, 0, len(row.chainIDs))
			for _, chainID := range row.chainIDs {
				preparedChain, err := prepared.PreparedDeployment.Chain(chainID)
				require.NoError(t, err)
				liveChains = append(liveChains, pcdLiveChainExpectation{
					chainID:                       chainID,
					portal:                        preparedChain.OptimismPortalProxy,
					disputeGameFactory:            preparedChain.DisputeGameFactoryProxy,
					anchorStateRegistry:           preparedChain.AnchorStateRegistryProxy,
					respectedGameType:             row.respectedGameType,
					fallbackGameType:              row.fallbackGameType,
					respectedGameImplementation:   preparedChain.FaultDisputeGameImpl,
					fallbackGameImplementation:    preparedChain.PermissionedDisputeGameImpl,
					respectedGameAbsolutePrestate: prestate,
					startingProposalRoot:          expectedProposal,
					startingProposalSequence:      genesisTime,
					proposalArtifactPaths:         pcdOracleArtifactPaths(artifacts),
				})
			}
			completedL1State := probe.requireCompletedDeployment(
				t,
				journey.postBootstrapL1State,
				row.expectedContinueNonceDelta,
				predictedAddresses,
				liveChains,
			)
			t.Logf(
				"PCD bootstrap-to-completion deployer nonce delta: %d at pinned L1 block %s (%s)",
				completedL1State.latestNonce-journey.postBootstrapL1State.latestNonce,
				completedL1State.blockNumber,
				completedL1State.blockHash,
			)
		})
	}
}

func pcdArtifactOutputRoot(t *testing.T, rowName string, artifact pcdChainArtifacts) common.Hash {
	t.Helper()
	genesis, err := readPCDGenesis(artifact.genesisPath)
	require.NoErrorf(t, err, "read genesis for row %s chain %s", rowName, artifact.chainID.Hex())
	header := genesis.ToBlock().Header()
	require.NotNilf(t, header.WithdrawalsHash, "genesis for row %s chain %s has no withdrawals hash", rowName, artifact.chainID.Hex())
	return common.Hash(pcdOutputRoot(header))
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
