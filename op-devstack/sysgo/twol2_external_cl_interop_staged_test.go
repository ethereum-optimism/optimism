package sysgo

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/stretchr/testify/require"
)

var _ func(devtest.T, uint64, PresetConfig) *StagedTwoL2ExternalCLInteropRuntime = NewStagedTwoL2ExternalCLInteropRuntime

func TestStagedTwoL2ExternalCLInteropPreparesConfigurationWithoutServices(t *testing.T) {
	dt := devtest.SerialT(t)
	runtime := NewStagedTwoL2ExternalCLInteropRuntime(dt, 0, PresetConfig{EnableTimeTravel: true})
	chainA := eth.ChainIDFromUInt64(901)

	require.Nil(t, runtime.L1EL())
	require.Nil(t, runtime.L1CL())
	require.Empty(t, runtime.sequencerEL)
	require.Empty(t, runtime.verifierEL)
	require.Empty(t, runtime.sequencerCL)
	require.Empty(t, runtime.verifierCL)
	require.NotNil(t, runtime.TimeTravel())
	require.FileExists(t, runtime.jwtPath)
	require.Equal(t, []eth.ChainID{
		eth.ChainIDFromUInt64(901),
		eth.ChainIDFromUInt64(902),
	}, runtime.ChainIDs())
	require.True(t, runtime.DependencySet().HasChain(eth.ChainIDFromUInt64(901)))
	require.True(t, runtime.DependencySet().HasChain(eth.ChainIDFromUInt64(902)))
	require.NoError(t, runtime.checkStartL1())
	require.EqualError(t, runtime.checkStartEL(chainA, "sequencer", runtime.sequencerEL), "start the staged L1 first")
}

func TestStagedTwoL2ExternalCLInteropStartGuards(t *testing.T) {
	chainA := eth.ChainIDFromUInt64(901)
	unknown := eth.ChainIDFromUInt64(999)
	runtime := &StagedTwoL2ExternalCLInteropRuntime{
		l2Nets:      map[eth.ChainID]*L2Network{chainA: {}},
		sequencerEL: make(map[eth.ChainID]L2ELNode),
		verifierEL:  make(map[eth.ChainID]L2ELNode),
		sequencerCL: make(map[eth.ChainID]L2CLNode),
		verifierCL:  make(map[eth.ChainID]L2CLNode),
	}

	require.NoError(t, runtime.checkStartL1())
	require.EqualError(t, runtime.checkStartEL(chainA, "sequencer", runtime.sequencerEL), "start the staged L1 first")
	runtime.l1EL = fakeInteropL1EL{}
	require.EqualError(t, runtime.checkL1Ready(), "start the staged L1 first")
	runtime.l1CL = &L1CLNode{}
	require.EqualError(t, runtime.checkStartL1(), "staged L1 is already running")
	require.EqualError(t, runtime.checkStartEL(unknown, "sequencer", runtime.sequencerEL), "unknown staged L2 chain 999")
	require.NoError(t, runtime.checkStartEL(chainA, "sequencer", runtime.sequencerEL))

	sequencerEL := fakeInteropL2EL{}
	runtime.sequencerEL[chainA] = sequencerEL
	require.EqualError(t, runtime.checkStartEL(chainA, "sequencer", runtime.sequencerEL), "sequencer EL for chain 901 already exists")
	require.NoError(t, runtime.checkStartSequencerCL(chainA))
	require.EqualError(t, runtime.checkStartVerifierCL(chainA), "start chain 901 verifier EL before its CL")

	runtime.verifierEL[chainA] = fakeInteropL2EL{}
	require.EqualError(t, runtime.checkStartVerifierCL(chainA), "start chain 901 sequencer CL before its verifier CL")
	runtime.sequencerCL[chainA] = &fakeExternalL2CL{}
	require.EqualError(t, runtime.checkStartSequencerCL(chainA), "sequencer CL for chain 901 already exists")
	require.NoError(t, runtime.checkStartVerifierCL(chainA))
	runtime.verifierCL[chainA] = &fakeExternalL2CL{}
	require.EqualError(t, runtime.checkStartVerifierCL(chainA), "verifier CL for chain 901 already exists")
}
