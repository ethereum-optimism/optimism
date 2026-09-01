package supernode

import (
	"math/big"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"

	gethlog "github.com/ethereum/go-ethereum/log"

	opnodecfg "github.com/ethereum-optimism/optimism/op-node/config"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/cliapp"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-supernode/config"
	"github.com/ethereum-optimism/optimism/op-supernode/flags"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/resources"
)

const wiringChainID = 424243

func wiringSupernode(t *testing.T) *Supernode {
	return &Supernode{
		log:          testlog.Logger(t, gethlog.LevelInfo),
		metricsFanIn: resources.NewMetricsFanIn(1),
	}
}

func wiringVNCfgs(private bool) map[eth.ChainID]*opnodecfg.Config {
	cfg := &opnodecfg.Config{Rollup: rollup.Config{L2ChainID: big.NewInt(wiringChainID), BlockTime: 2}}
	if private {
		cfg.Rollup.PrivateInterop = &rollup.PrivateInteropConfig{}
	}
	return map[eth.ChainID]*opnodecfg.Config{eth.ChainIDFromUInt64(wiringChainID): cfg}
}

func wiringCfg(t *testing.T, args ...string) *config.CLIConfig {
	t.Helper()
	var cfg *config.CLIConfig
	app := cli.NewApp()
	app.Flags = cliapp.ProtectFlags(flags.FullDynamicFlags([]uint64{wiringChainID}))
	app.Action = func(ctx *cli.Context) error {
		cfg = config.NewConfig(ctx)
		return nil
	}
	require.NoError(t, app.Run(append([]string{"op-supernode",
		"--chains=424243", "--l1=http://l1:8545"}, args...)))
	return cfg
}

func wiringGenesisPath(t *testing.T) string {
	t.Helper()
	path, err := filepath.Abs("../../op-private-interop/genesis/testdata/private-chain-genesis.json")
	require.NoError(t, err)
	return path
}

func TestClaimFollowIsDormantWithoutRollupMarker(t *testing.T) {
	s := wiringSupernode(t)
	act, routes, err := s.initClaimFollow(wiringCfg(t), wiringVNCfgs(false))
	require.NoError(t, err)
	require.Nil(t, act)
	require.Empty(t, routes)
}

func TestClaimFollowAutoDetectsMarkedChain(t *testing.T) {
	s := wiringSupernode(t)
	act, routes, err := s.initClaimFollow(wiringCfg(t,
		"--private-interop.genesis="+wiringGenesisPath(t),
	), wiringVNCfgs(true))
	require.NoError(t, err)
	require.NotNil(t, act)
	require.Equal(t, eth.ChainIDFromUInt64(wiringChainID), act.ChainID())

	got := routes[eth.ChainIDFromUInt64(wiringChainID)]
	require.Len(t, got, 1)
	require.Equal(t, "/claimed", got[0].Route)
	require.Equal(t, "optimism", got[0].API.Namespace)
}

func TestClaimFollowRequiresGenesisForMarkedChain(t *testing.T) {
	s := wiringSupernode(t)
	_, _, err := s.initClaimFollow(wiringCfg(t), wiringVNCfgs(true))
	require.ErrorContains(t, err, "private-interop.genesis")
}

func TestClaimFollowRejectsMultipleMarkedChains(t *testing.T) {
	cfgs := wiringVNCfgs(true)
	second := eth.ChainIDFromUInt64(424244)
	cfgs[second] = &opnodecfg.Config{Rollup: rollup.Config{
		L2ChainID:      big.NewInt(424244),
		PrivateInterop: &rollup.PrivateInteropConfig{},
	}}
	s := wiringSupernode(t)
	_, _, err := s.initClaimFollow(wiringCfg(t,
		"--private-interop.genesis="+wiringGenesisPath(t),
	), cfgs)
	require.ErrorContains(t, err, "multiple rollup configs")
}
