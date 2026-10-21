package inspect

import (
	"encoding/json"
	"maps"
	"path/filepath"
	"slices"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/addresses"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/pipeline"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum-optimism/optimism/op-service/jsonutil"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

// TestL1CLI_KeepsChainGameImpls checks that the chain's own dispute game implementations
// survive the JSON output next to the OPCM implementations of the same name.
func TestL1CLI_KeepsChainGameImpls(t *testing.T) {
	chainID := common.HexToHash("0x01")
	superchain := addresses.SuperchainContracts{
		SuperchainConfigProxy: common.HexToAddress("0x5c"),
	}
	impls := addresses.ImplementationsContracts{
		OpcmV2Impl:                       common.HexToAddress("0x0c"),
		FaultDisputeGameImpl:             common.HexToAddress("0x1a"),
		PermissionedDisputeGameImpl:      common.HexToAddress("0x1b"),
		SuperFaultDisputeGameImpl:        common.HexToAddress("0x1c"),
		SuperPermissionedDisputeGameImpl: common.HexToAddress("0x1d"),
	}
	chain := addresses.OpChainContracts{
		OpChainCoreContracts: addresses.OpChainCoreContracts{
			SystemConfigProxy: common.HexToAddress("0x5f"),
		},
		OpChainFaultProofsContracts: addresses.OpChainFaultProofsContracts{
			DisputeGameFactoryProxy:          common.HexToAddress("0xdf"),
			FaultDisputeGameImpl:             common.HexToAddress("0x2a"),
			PermissionedDisputeGameImpl:      common.HexToAddress("0x2b"),
			SuperFaultDisputeGameImpl:        common.HexToAddress("0x2c"),
			SuperPermissionedDisputeGameImpl: common.HexToAddress("0x2d"),
		},
	}

	workdir := t.TempDir()
	require.NoError(t, pipeline.WriteState(workdir, &state.State{
		SuperchainDeployment:      &superchain,
		ImplementationsDeployment: &impls,
		Chains:                    []*state.ChainState{{ID: chainID, OpChainContracts: chain}},
	}))
	outfile := filepath.Join(t.TempDir(), "l1.json")
	require.NoError(t, L1CLI(newInspectCtx(t, workdir, outfile, chainID.Hex())))

	got, err := jsonutil.LoadJSON[addresses.L1Contracts](outfile)
	require.NoError(t, err)
	require.Equal(t, &addresses.L1Contracts{
		Superchain:      superchain,
		Implementations: impls,
		OpChain:         chain,
	}, got)

	raw, err := jsonutil.LoadJSON[map[string]json.RawMessage](outfile)
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"superchain", "implementations", "opChain"}, slices.Collect(maps.Keys(*raw)))
}
