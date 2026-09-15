package inspect

import (
	"flag"
	"log/slog"
	"math/big"
	"path/filepath"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/addresses"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/integration_test/shared"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/pipeline"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/standard"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/testutil"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

// buildPreparedWorkdir builds a fully "prepared" intent+state pair (predicted L1 addresses,
// pinned anchor block/genesis time, generated L2 genesis allocs, computed output root) and
// writes intent.toml + state.json to a fresh temp workdir, mirroring what op-deployer prepare
// produces before continue ever runs. Returns the workdir and the chain ID.
func buildPreparedWorkdir(t *testing.T) (string, common.Hash) {
	t.Helper()

	lgr := testlog.Logger(t, slog.LevelWarn)
	_, pk, dk := shared.DefaultPrivkey(t)
	deployerAddr := crypto.PubkeyToAddress(pk.PublicKey)

	l1ChainID := big.NewInt(900)
	l2ChainID := uint256.NewInt(1)
	loc, afacts := testutil.LocalArtifacts(t)

	intent, st := shared.NewIntent(t, l1ChainID, dk, l2ChainID, loc, loc, standard.GasLimit)
	superchainConfig := common.HexToAddress("0x5ea4")
	intent.SuperchainConfigProxy = &superchainConfig
	chain := intent.Chains[0]

	// RollupConfig requires the predicted L1 addresses prepare's predictChains would have set.
	st.SetChainContracts(chain.ID, addresses.OpChainContracts{
		OpChainCoreContracts: addresses.OpChainCoreContracts{
			OptimismPortalProxy: common.HexToAddress("0x1111111111111111111111111111111111111111"),
			SystemConfigProxy:   common.HexToAddress("0x2222222222222222222222222222222222222222"),
		},
	}, false)

	genesisTime := hexutil.Uint64(1_700_000_000)
	st.PinChainAnchor(chain.ID, &state.L1BlockRefJSON{
		Hash:   common.HexToHash("0xaaaa"),
		Number: 100,
		Time:   hexutil.Uint64(uint64(genesisTime) - 100),
	}, genesisTime)

	pEnv := &pipeline.Env{Logger: lgr, Deployer: deployerAddr}
	bundle := artifacts.Bundle{L1: afacts, L2: afacts}
	require.NoError(t, pipeline.GenerateL2Genesis(pEnv, intent, bundle, st, chain.ID))
	depSet, err := pipeline.BuildInteropDepSet(intent.Chains)
	require.NoError(t, err)
	st.InteropDepSet = depSet
	require.NoError(t, pipeline.ComputeGenesisOutputRoots(pEnv, intent, st))

	prepared, err := pipeline.NewPreparedDeployment(intent, st, deployerAddr, common.HexToAddress("0x1234"), bundle)
	require.NoError(t, err)
	st.PreparedDeployment = prepared

	dir := t.TempDir()
	require.NoError(t, intent.WriteToFile(filepath.Join(dir, "intent.toml")))
	require.NoError(t, pipeline.WriteState(dir, st))

	return dir, chain.ID
}

// newInspectCtx builds a CLI context for one of the inspect subcommands (genesis/rollup/etc),
// mirroring the flag.NewFlagSet + cli.NewContext pattern used by prepare_test.go's newPrepareCtx.
func newInspectCtx(t *testing.T, workdir, outfile, chainIDArg string) *cli.Context {
	t.Helper()

	app := cli.NewApp()
	flagSet := flag.NewFlagSet("test-inspect", flag.ContinueOnError)
	for _, f := range Flags {
		require.NoError(t, f.Apply(flagSet))
	}
	require.NoError(t, flagSet.Parse([]string{
		"--" + deployer.WorkdirFlagName, workdir,
		"--" + OutfileFlagName, outfile,
		chainIDArg,
	}))

	return cli.NewContext(app, flagSet, nil)
}
