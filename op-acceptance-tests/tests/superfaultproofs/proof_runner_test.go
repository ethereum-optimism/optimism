package superfaultproofs

import (
	"encoding/json"
	"math/big"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/vm"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/ptr"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

type proofRunnerTestNode struct {
	stack.L1ELNode
	t devtest.T
}

func (n proofRunnerTestNode) T() devtest.T                  { return n.t }
func (proofRunnerTestNode) UserRPC() string                 { return "http://127.0.0.1:1" }
func (proofRunnerTestNode) BeaconHTTPAddr() string          { return "http://127.0.0.1:2" }
func (proofRunnerTestNode) BeaconClient() apis.BeaconClient { return nil }

type proofRunnerTestSuperRoots struct {
	dsl.SuperRootSource
}

func (proofRunnerTestSuperRoots) UserRPC() string { return "http://127.0.0.1:3" }

func TestSuperRangeExecutorArgsPassL1ChainConfig(gt *testing.T) {
	t := devtest.SerialT(gt)
	config := &params.ChainConfig{
		ChainID:      big.NewInt(900),
		ShanghaiTime: ptr.New(uint64(10)),
		CancunTime:   ptr.New(uint64(20)),
		PragueTime:   ptr.New(uint64(30)),
		OsakaTime:    ptr.New(uint64(40)),
	}
	genesisBytes, err := json.Marshal(&core.Genesis{Config: config, Difficulty: big.NewInt(0), Alloc: types.GenesisAlloc{}})
	require.NoError(gt, err)
	genesisPath := filepath.Join(gt.TempDir(), "l1-genesis-900.json")
	require.NoError(gt, os.WriteFile(genesisPath, genesisBytes, 0o600))
	node := proofRunnerTestNode{t: t}
	sys := &presets.SingleChainInterop{
		SuperRoots: proofRunnerTestSuperRoots{},
		L1EL:       dsl.NewL1ELNode(node),
		L1CL:       dsl.NewL1CLNode(node),
	}
	args := superRangeExecutorArgs(t, sys,
		[]proofChain{{id: eth.ChainIDFromUInt64(901), l2NodeAddress: "http://127.0.0.1:4"}},
		vm.Config{
			RollupConfigPaths: []string{"rollup-901.json"},
			DepsetConfigPath:  "depset.json",
			L1GenesisPath:     genesisPath,
		}, eth.BlockID{}, 100)
	flagIndex := slices.Index(args, "--l1-config-path")
	require.GreaterOrEqual(gt, flagIndex, 0)
	encodedConfig, err := os.ReadFile(args[flagIndex+1])
	require.NoError(gt, err)
	var actual params.ChainConfig
	require.NoError(gt, json.Unmarshal(encodedConfig, &actual))
	require.Equal(gt, config.ChainID, actual.ChainID)
	require.Equal(gt, config.ShanghaiTime, actual.ShanghaiTime)
	require.Equal(gt, config.CancunTime, actual.CancunTime)
	require.Equal(gt, config.PragueTime, actual.PragueTime)
	require.Equal(gt, config.OsakaTime, actual.OsakaTime)
	unchangedGenesis, err := os.ReadFile(genesisPath)
	require.NoError(gt, err)
	require.Equal(gt, genesisBytes, unchangedGenesis)
}
