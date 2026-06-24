package deployer

import (
	"flag"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

const testL1RPCUrl = "http://localhost:8545"

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
	require.NoError(t, flagSet.Set(L1RPCURLFlagName, testL1RPCUrl))

	return cli.NewContext(app, flagSet, nil)
}

func TestNewPrepareConfig_FlagsPassed(t *testing.T) {
	const addr = "0x1234567890123456789012345678901234567890"

	cfg, err := newPrepareConfig(newPrepareCtx(t, addr), log.NewLogger(log.DiscardHandler()))
	require.NoError(t, err)
	require.Equal(t, common.HexToAddress(addr), cfg.DeployerAddress)
	require.Equal(t, testL1RPCUrl, cfg.L1RPCUrl)
}

func TestNewPrepareConfig_InvalidDeployerAddress(t *testing.T) {
	for _, addr := range []string{"", "not-an-address", "0x123"} {
		_, err := newPrepareConfig(newPrepareCtx(t, addr), log.NewLogger(log.DiscardHandler()))
		require.ErrorContains(t, err, "invalid deployer address")
	}
}

func TestPrepareConfigCheck(t *testing.T) {
	valid := PrepareConfig{
		Workdir:         "/tmp",
		Logger:          log.NewLogger(log.DiscardHandler()),
		DeployerAddress: common.HexToAddress("0x1234567890123456789012345678901234567890"),
		L1RPCUrl:        testL1RPCUrl,
	}
	require.NoError(t, valid.Check())

	missingDeployer := valid
	missingDeployer.DeployerAddress = common.Address{}
	require.ErrorContains(t, missingDeployer.Check(), "deployer address must be specified")

	missingL1RPC := valid
	missingL1RPC.L1RPCUrl = ""
	require.ErrorContains(t, missingL1RPC.Check(), "l1 RPC URL must be specified")
}
