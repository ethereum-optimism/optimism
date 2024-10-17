package bootstrap

import (
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer"
	"github.com/ethereum-optimism/optimism/op-service/cliapp"
	"github.com/urfave/cli/v2"
)

const (
	ArtifactsURLFlagName     = "artifacts-url"
	ContractsReleaseFlagName = "contracts-release"
)

var (
	ArtifactsURLFlag = &cli.StringFlag{
		Name:    ArtifactsURLFlagName,
		Usage:   "URL to the artifacts directory.",
		EnvVars: deployer.PrefixEnvVar("ARTIFACTS_URL"),
	}
	ContractsReleaseFlag = &cli.StringFlag{
		Name:    ContractsReleaseFlagName,
		Usage:   "Release of the contracts to deploy.",
		EnvVars: deployer.PrefixEnvVar("CONTRACTS_RELEASE"),
	}
)

var OPCMFlags = []cli.Flag{
	deployer.L1RPCURLFlag,
	deployer.PrivateKeyFlag,
	ArtifactsURLFlag,
	ContractsReleaseFlag,
}

var Commands = []*cli.Command{
	{
		Name:   "opcm",
		Usage:  "Bootstrap an instance of OPCM.",
		Flags:  cliapp.ProtectFlags(OPCMFlags),
		Action: OPCMCLI,
	},
}
