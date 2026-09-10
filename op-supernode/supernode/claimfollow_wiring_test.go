package supernode

import (
	"math/big"
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

const (
	wiringChainID = 424243
	// The PRIVATE chain's genesis hash, which the module serves as its not-yet ref and which
	// nothing public carries.
	wiringGenesisHash = "0x00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff"
)

func wiringSupernode(t *testing.T) *Supernode {
	return &Supernode{
		log:          testlog.Logger(t, gethlog.LevelInfo),
		metricsFanIn: resources.NewMetricsFanIn(1),
	}
}

func wiringVNCfgs() map[eth.ChainID]*opnodecfg.Config {
	return map[eth.ChainID]*opnodecfg.Config{
		eth.ChainIDFromUInt64(wiringChainID): {
			Rollup: rollup.Config{L2ChainID: big.NewInt(wiringChainID), BlockTime: 2},
		},
	}
}

// wiringCfg builds a supernode CLIConfig whose RawCtx carries the given arguments, which is how the
// module's flag group reaches it without op-supernode/config importing the activity.
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
		"--chains=" + "424243", "--l1=http://l1:8545"}, args...)))
	return cfg
}

// THE DORMANCY GATE at the wiring level: with the flag group unset there is no module, no activity,
// and no route for ANY chain — so every chain container is constructed with an empty option list
// and behaves exactly as it did before this group existed.
func TestClaimFollowIsDormantWhenUnset(t *testing.T) {
	s := wiringSupernode(t)
	act, routes, err := s.initClaimFollow(wiringCfg(t), wiringVNCfgs())
	require.NoError(t, err)
	require.Nil(t, act)
	require.Empty(t, routes)
}

// A nil RawCtx — the shape a struct-literal Supernode in a test has — is dormant too, rather than a
// panic.
func TestClaimFollowIsDormantWithoutACLIContext(t *testing.T) {
	s := wiringSupernode(t)
	act, routes, err := s.initClaimFollow(&config.CLIConfig{}, wiringVNCfgs())
	require.NoError(t, err)
	require.Nil(t, act)
	require.Empty(t, routes)
}

func TestClaimFollowWiresOneChainsSiblingRoute(t *testing.T) {
	s := wiringSupernode(t)
	act, routes, err := s.initClaimFollow(wiringCfg(t,
		"--private-interop.enabled",
		"--private-interop.chain-id=424243",
		"--private-interop.claim-registry=0x4200000000000000000000000000000000000777",
		"--private-interop.genesis-hash="+wiringGenesisHash,
	), wiringVNCfgs())
	require.NoError(t, err)
	require.NotNil(t, act)
	require.Equal(t, eth.ChainIDFromUInt64(wiringChainID), act.ChainID())

	// Exactly one chain gets a route, and it is the sibling one carrying the standard namespace.
	require.Len(t, routes, 1)
	got := routes[eth.ChainIDFromUInt64(wiringChainID)]
	require.Len(t, got, 1)
	require.Equal(t, "/claimed", got[0].Route)
	require.Equal(t, "optimism", got[0].API.Namespace)

	// Every other chain the supernode runs is untouched: a nil slice, which registers nothing.
	require.Empty(t, routes[eth.ChainIDFromUInt64(10)])
}

// The one validation only the supernode can make: a module pointed at a chain this process does not
// run would scan nothing and serve an error forever, so it is a startup failure instead.
func TestClaimFollowRefusesAChainTheSupernodeDoesNotRun(t *testing.T) {
	s := wiringSupernode(t)
	_, _, err := s.initClaimFollow(wiringCfg(t,
		"--private-interop.enabled",
		"--private-interop.chain-id=999999",
		"--private-interop.claim-registry=0x4200000000000000000000000000000000000777",
		"--private-interop.genesis-hash="+wiringGenesisHash,
	), wiringVNCfgs())
	require.Error(t, err)
	require.Contains(t, err.Error(), "not one of this supernode's chains")
}

// A half-configured group fails at startup rather than at the first poll.
func TestClaimFollowRefusesAHalfConfiguredGroup(t *testing.T) {
	s := wiringSupernode(t)
	_, _, err := s.initClaimFollow(wiringCfg(t, "--private-interop.enabled"), wiringVNCfgs())
	require.Error(t, err)
	require.Contains(t, err.Error(), "private-interop.chain-id")
}
