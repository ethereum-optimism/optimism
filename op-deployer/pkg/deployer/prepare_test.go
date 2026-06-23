package deployer

import (
	"flag"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

// newPrepareCtx builds a CLI context with the prepare flags applied and the
// deployer address set to addr.
func newPrepareCtx(t *testing.T, addr string) *cli.Context {
	t.Helper()

	app := cli.NewApp()
	flagSet := flag.NewFlagSet("test-prepare", flag.ContinueOnError)
	for _, f := range PrepareFlags {
		require.NoError(t, f.Apply(flagSet))
	}
	require.NoError(t, flagSet.Set(DeployerAddressFlagName, addr))

	return cli.NewContext(app, flagSet, nil)
}

func TestNewPrepareConfig_DeployerAddressPassed(t *testing.T) {
	const addr = "0x1234567890123456789012345678901234567890"

	cfg, err := newPrepareConfig(newPrepareCtx(t, addr), log.NewLogger(log.DiscardHandler()))
	require.NoError(t, err)
	require.Equal(t, common.HexToAddress(addr), cfg.DeployerAddress)
}

func TestNewPrepareConfig_InvalidDeployerAddress(t *testing.T) {
	for _, addr := range []string{"", "not-an-address", "0x123"} {
		_, err := newPrepareConfig(newPrepareCtx(t, addr), log.NewLogger(log.DiscardHandler()))
		require.ErrorContains(t, err, "invalid deployer address")
	}
}

func TestPrepareConfigCheck_RequiresDeployerAddress(t *testing.T) {
	cfg := PrepareConfig{
		Workdir: "/tmp",
		Logger:  log.NewLogger(log.DiscardHandler()),
	}
	require.ErrorContains(t, cfg.Check(), "deployer address must be specified")

	cfg.DeployerAddress = common.HexToAddress("0x1234567890123456789012345678901234567890")
	require.NoError(t, cfg.Check())
}
