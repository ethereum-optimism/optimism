package claimfollow

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"

	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-service/cliapp"
	"github.com/ethereum-optimism/optimism/op-supernode/flags"
)

func run(t *testing.T, args ...string) CLIConfig {
	t.Helper()
	var got CLIConfig
	app := cli.NewApp()
	app.Flags = cliapp.ProtectFlags(flags.FullDynamicFlags([]uint64{424243}))
	app.Action = func(ctx *cli.Context) error {
		got = ReadCLIConfig(ctx)
		return nil
	}
	require.NoError(t, app.Run(append([]string{"op-supernode"}, args...)))
	return got
}

func privateGenesisPath(t *testing.T) string {
	t.Helper()
	path, err := filepath.Abs("../../../../op-private-interop/genesis/testdata/private-chain-genesis.json")
	require.NoError(t, err)
	return path
}

func TestFlagsParseAndResolve(t *testing.T) {
	path := privateGenesisPath(t)
	cfg := run(t,
		"--chains=424243", "--l1=http://l1:8545",
		"--private-interop.chain-id=424243",
		"--private-interop.genesis="+path,
		"--private-interop.claim-scan-start-block=100",
	)
	require.True(t, cfg.Enabled())
	require.True(t, cfg.ChainIDSet)
	require.Equal(t, uint64(424243), cfg.ChainID)
	require.Equal(t, path, cfg.GenesisPath)
	require.Equal(t, uint64(100), cfg.ScanStartBlock)
	require.NoError(t, cfg.Check())

	modCfg, err := cfg.Resolve()
	require.NoError(t, err)
	genesis, err := cfg.LoadPrivateChainGenesis()
	require.NoError(t, err)
	require.Equal(t, genesis.ToBlock().Hash(), modCfg.GenesisHash)
	require.Equal(t, predeploys.ClaimRegistryAddr, modCfg.Registry)
	require.Equal(t, uint64(100), modCfg.StartBlock)
}

func TestCheckRequiresBothActivatingFlags(t *testing.T) {
	require.ErrorContains(t, (CLIConfig{}).Check(), ChainIDFlag.Name)
	require.ErrorContains(t, (CLIConfig{GenesisPath: "genesis.json"}).Check(), ChainIDFlag.Name)
	require.ErrorContains(t, (CLIConfig{ChainIDSet: true, ChainID: 1}).Check(), GenesisPathFlag.Name)
}

func TestNilContextIsDisabled(t *testing.T) {
	cfg := ReadCLIConfig(nil)
	require.Empty(t, cfg.GenesisPath)
	require.False(t, cfg.Enabled())
}
