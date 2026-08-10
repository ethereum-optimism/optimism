package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/pipeline"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/upgrade/embedded"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

type pcdResumeBoundary uint8

const (
	pcdAfterInit pcdResumeBoundary = iota + 1
	pcdAfterPrepare
	pcdAfterInspect
	pcdAfterPrestate
	pcdAfterContinue
)

func (b pcdResumeBoundary) String() string {
	switch b {
	case pcdAfterInit:
		return "after-init"
	case pcdAfterPrepare:
		return "after-prepare"
	case pcdAfterInspect:
		return "after-inspect"
	case pcdAfterPrestate:
		return "after-prestate"
	case pcdAfterContinue:
		return "after-continue"
	default:
		return "unknown"
	}
}

type pcdResumeIdentity struct {
	intent             []byte
	preparedDeployment *state.PreparedDeployment
	predictedAddresses []pcdNamedAddress
	startBlock         *state.L1BlockRefJSON
	genesisTime        *hexutil.Uint64
	startingAnchor     *state.StartingAnchorProposal
}

func TestCLIPCDResume(t *testing.T) {
	boundaries := []pcdResumeBoundary{
		pcdAfterInit,
		pcdAfterPrepare,
		pcdAfterInspect,
		pcdAfterPrestate,
		pcdAfterContinue,
	}

	for _, boundary := range boundaries {
		t.Run(boundary.String(), func(t *testing.T) {
			prestate := requirePCDPrestate(t, pcdPrestateArtifactPath(t))
			journey := newPCDResumeJourney(t, boundary, prestate)
			probe := pcdL1Probe{client: journey.l1Client, deployer: journey.deployer}
			l1BeforeRestart := probe.read(t, nil)
			if boundary < pcdAfterContinue {
				requirePCDNoncePair(
					t,
					boundary.String(),
					journey.postBootstrapL1State.latestNonce,
					journey.postBootstrapL1State.pendingNonce,
					l1BeforeRestart,
				)
			}

			committedWorkdir := journey.cloneCommittedWorkdir()
			frozen := readPCDResumeIdentity(t, committedWorkdir, journey.chainIDs[0], boundary >= pcdAfterPrepare)
			journey.restartCold(committedWorkdir)
			resumePCDJourney(t, journey, boundary, prestate)

			finalIdentity := readPCDResumeIdentity(t, journey.workdir, journey.chainIDs[0], boundary >= pcdAfterPrepare)
			requirePCDResumeIdentity(t, boundary.String(), frozen, finalIdentity)
			completed := requirePCDSingletonCompletion(t, boundary.String(), journey, prestate)
			if boundary == pcdAfterContinue {
				requirePCDNoncePair(
					t,
					boundary.String(),
					l1BeforeRestart.latestNonce,
					l1BeforeRestart.pendingNonce,
					completed,
				)
			}
		})
	}

	t.Run("elapsed-genesis", func(t *testing.T) {
		prestate := requirePCDPrestate(t, pcdPrestateArtifactPath(t))
		journey := newPCDResumeJourney(t, pcdAfterPrestate, prestate)
		committedWorkdir := journey.cloneCommittedWorkdir()
		frozen := readPCDResumeIdentity(t, committedWorkdir, journey.chainIDs[0], true)
		require.NotNil(t, frozen.genesisTime)
		journey.restartCold(committedWorkdir)

		elapsedTimestamp := uint64(*frozen.genesisTime) + 1
		require.NoError(t, journey.l1Client.Client().Call(nil, "anvil_setNextBlockTimestamp", elapsedTimestamp))
		require.NoError(t, journey.l1Client.Client().Call(nil, "evm_mine"))
		head, err := journey.l1Client.HeaderByNumber(t.Context(), nil)
		require.NoError(t, err)
		require.Greater(t, head.Time, uint64(*frozen.genesisTime))

		_, output := journey.runContinueWithOutput()
		require.Contains(t, output, "committed genesis time has elapsed")
		finalIdentity := readPCDResumeIdentity(t, journey.workdir, journey.chainIDs[0], true)
		requirePCDResumeIdentity(t, "elapsed-genesis", frozen, finalIdentity)
		requirePCDSingletonCompletion(t, "elapsed-genesis", journey, prestate)
	})

	t.Run("reprepare-invalidates-prestate", func(t *testing.T) {
		prestate := requirePCDPrestate(t, pcdPrestateArtifactPath(t))
		journey := newPCDResumeJourney(t, pcdAfterPrestate, prestate)
		committedWorkdir := journey.cloneCommittedWorkdir()
		frozen := readPCDResumeIdentity(t, committedWorkdir, journey.chainIDs[0], true)
		journey.restartCold(committedWorkdir)

		// Prepare calls state.ChainState.ClearDerivedArtifacts in state/state.go after every
		// prediction, even when it reuses the pinned anchor and predicts the same addresses.
		// This clear removes the committed prestate. A resume after prestate must start with the next stage.
		journey.runPrepare()
		wipedIdentity := readPCDResumeIdentity(t, journey.workdir, journey.chainIDs[0], true)
		requirePCDResumeIdentity(t, "reprepare-invalidates-prestate/wipe", frozen, wipedIdentity)
		wipedState, err := pipeline.ReadState(journey.workdir)
		require.NoError(t, err)
		wipedChain, err := wipedState.Chain(journey.chainIDs[0])
		require.NoError(t, err)
		require.Zero(t, wipedChain.Prestate, "re-prepare did not clear the committed prestate")

		probe := pcdL1Probe{client: journey.l1Client, deployer: journey.deployer}
		beforeGate := probe.read(t, nil)
		journey.runner.ExpectErrorContainsWithNetwork(t, []string{
			"continue",
			"--workdir", journey.workdir,
		}, nil, "has no prestate committed. Run op-deployer prestate")
		afterGate := probe.read(t, nil)
		requirePCDNoncePair(
			t,
			"reprepare-invalidates-prestate/gate",
			beforeGate.latestNonce,
			beforeGate.pendingNonce,
			afterGate,
		)

		journey.runPrestate(prestate)
		journey.runContinue()
		finalIdentity := readPCDResumeIdentity(t, journey.workdir, journey.chainIDs[0], true)
		requirePCDResumeIdentity(t, "reprepare-invalidates-prestate/unblock", frozen, finalIdentity)
		requirePCDSingletonCompletion(t, "reprepare-invalidates-prestate/unblock", journey, prestate)
	})
}

func newPCDResumeJourney(t *testing.T, boundary pcdResumeBoundary, prestate common.Hash) *pcdJourneyFixture {
	t.Helper()
	journey := newPCDJourneyFixture(t, []common.Hash{uint256.NewInt(1).Bytes32()})
	journey.bootstrapOPCM()
	journey.runInit(embedded.GameTypeSuperCannonKona)
	if boundary >= pcdAfterPrepare {
		journey.runPrepare()
	}
	if boundary >= pcdAfterInspect {
		journey.runInspect()
		journey.writeDependencySet()
	}
	if boundary >= pcdAfterPrestate {
		journey.runPrestate(prestate)
	}
	if boundary >= pcdAfterContinue {
		journey.runContinue()
	}
	return journey
}

func resumePCDJourney(t *testing.T, journey *pcdJourneyFixture, boundary pcdResumeBoundary, prestate common.Hash) {
	t.Helper()
	if boundary < pcdAfterPrepare {
		journey.runPrepare()
	}
	if boundary < pcdAfterInspect {
		journey.runInspect()
		journey.writeDependencySet()
	}
	if boundary < pcdAfterPrestate {
		journey.runPrestate(prestate)
	}
	journey.runContinue()
}

func readPCDResumeIdentity(
	t *testing.T,
	workdir string,
	chainID common.Hash,
	expectPrepared bool,
) pcdResumeIdentity {
	t.Helper()
	intentBytes, err := os.ReadFile(filepath.Join(workdir, "intent.toml"))
	require.NoError(t, err)
	identity := pcdResumeIdentity{intent: intentBytes}
	if !expectPrepared {
		return identity
	}

	st, err := pipeline.ReadState(workdir)
	require.NoError(t, err)
	require.NotNil(t, st.PreparedDeployment)
	identity.preparedDeployment, err = st.PreparedDeployment.Clone()
	require.NoError(t, err)
	identity.predictedAddresses = pcdPredictedContractAddresses(st.PreparedDeployment)
	chain, err := st.Chain(chainID)
	require.NoError(t, err)
	if chain.StartBlock != nil {
		startBlock := *chain.StartBlock
		identity.startBlock = &startBlock
	}
	if chain.GenesisTime != nil {
		genesisTime := *chain.GenesisTime
		identity.genesisTime = &genesisTime
	}
	if chain.StartingAnchorRoot != nil {
		startingAnchor := *chain.StartingAnchorRoot
		identity.startingAnchor = &startingAnchor
	}
	return identity
}

func requirePCDResumeIdentity(t *testing.T, point string, expected, observed pcdResumeIdentity) {
	t.Helper()
	require.Equalf(t, expected.intent, observed.intent, "%s changed intent.toml", point)
	if expected.preparedDeployment == nil {
		return
	}
	require.Equalf(t, expected.preparedDeployment, observed.preparedDeployment, "%s changed the prepared deployment", point)
	require.Equalf(
		t,
		expected.predictedAddresses,
		observed.predictedAddresses,
		"%s changed predicted addresses: expected %v, observed %v",
		point,
		expected.predictedAddresses,
		observed.predictedAddresses,
	)
	require.Equalf(t, expected.startBlock, observed.startBlock, "%s changed the pinned L1 anchor", point)
	require.Equalf(t, expected.genesisTime, observed.genesisTime, "%s changed the genesis time", point)
	require.Equalf(t, expected.startingAnchor, observed.startingAnchor, "%s changed the starting anchor proposal", point)
}

func requirePCDNoncePair(t *testing.T, point string, expectedLatest, expectedPending uint64, observed pcdL1State) {
	t.Helper()
	require.Equalf(
		t,
		expectedLatest,
		observed.latestNonce,
		"%s latest nonce differs: expected %d, observed %d at L1 block %s (%s)",
		point,
		expectedLatest,
		observed.latestNonce,
		observed.blockNumber,
		observed.blockHash,
	)
	require.Equalf(
		t,
		expectedPending,
		observed.pendingNonce,
		"%s pending nonce differs: expected %d, observed %d at L1 block %s (%s)",
		point,
		expectedPending,
		observed.pendingNonce,
		observed.blockNumber,
		observed.blockHash,
	)
}

func requirePCDSingletonCompletion(
	t *testing.T,
	point string,
	journey *pcdJourneyFixture,
	prestate common.Hash,
) pcdL1State {
	t.Helper()
	require.Len(t, journey.chainIDs, 1)
	chainID := journey.chainIDs[0]
	st, err := pipeline.ReadState(journey.workdir)
	require.NoError(t, err)
	require.NotNil(t, st.PreparedDeployment)
	preparedChain, err := st.PreparedDeployment.Chain(chainID)
	require.NoError(t, err)

	artifacts := pcdArtifactPaths(journey.workdir, journey.chainIDs)
	expectedProposal, genesisTime, err := pcdSuperRootFromArtifacts(artifacts)
	require.NoErrorf(t, err, "%s compute proposal from rendered artifacts %v", point, pcdOracleArtifactPaths(artifacts))
	predictedAddresses := pcdPredictedContractAddresses(st.PreparedDeployment)
	probe := pcdL1Probe{client: journey.l1Client, deployer: journey.deployer}
	completed := probe.requireCompletedDeployment(
		t,
		journey.postBootstrapL1State,
		1,
		predictedAddresses,
		[]pcdLiveChainExpectation{{
			chainID:                       chainID,
			portal:                        preparedChain.OptimismPortalProxy,
			disputeGameFactory:            preparedChain.DisputeGameFactoryProxy,
			anchorStateRegistry:           preparedChain.AnchorStateRegistryProxy,
			respectedGameType:             embedded.GameTypeSuperCannonKona,
			fallbackGameType:              embedded.GameTypeSuperPermissioned,
			respectedGameImplementation:   preparedChain.FaultDisputeGameImpl,
			fallbackGameImplementation:    preparedChain.PermissionedDisputeGameImpl,
			respectedGameAbsolutePrestate: prestate,
			startingProposalRoot:          expectedProposal,
			startingProposalSequence:      genesisTime,
			proposalArtifactPaths:         pcdOracleArtifactPaths(artifacts),
		}},
	)
	t.Logf(
		"PCD resume point %s completed with deployer nonces latest=%d pending=%d at L1 block %s (%s)",
		point,
		completed.latestNonce,
		completed.pendingNonce,
		completed.blockNumber,
		completed.blockHash,
	)
	return completed
}
