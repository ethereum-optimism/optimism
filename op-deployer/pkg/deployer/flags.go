package deployer

import (
	"fmt"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/flags"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"

	op_service "github.com/ethereum-optimism/optimism/op-service"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/urfave/cli/v2"
)

const (
	EnvVarPrefix             = flags.EnvVarPrefix
	L1RPCURLFlagName         = flags.L1RPCURLFlagName
	CacheDirFlagName         = flags.CacheDirFlagName
	L1ChainIDFlagName        = flags.L1ChainIDFlagName
	ArtifactsLocatorFlagName = flags.ArtifactsLocatorFlagName
	L2ChainIDsFlagName       = flags.L2ChainIDsFlagName
	WorkdirFlagName          = flags.WorkdirFlagName
	OutdirFlagName           = flags.OutdirFlagName
	PrivateKeyFlagName       = flags.PrivateKeyFlagName
	IntentTypeFlagName       = flags.IntentTypeFlagName
	VerifierAPIKeyFlagName   = flags.VerifierAPIKeyFlagName
	EtherscanAPIKeyFlagName  = flags.EtherscanAPIKeyFlagName // Deprecated: use VerifierAPIKeyFlagName
	InputFileFlagName        = flags.InputFileFlagName
	ContractNameFlagName     = flags.ContractNameFlagName
	VerifierTypeFlagName     = flags.VerifierTypeFlagName
	VerifierUrlFlagName      = flags.VerifierUrlFlagName
)

var (
	L1RPCURLFlag = &cli.StringFlag{
		Name: L1RPCURLFlagName,
		Usage: "RPC URL for the L1 chain. Must be set for live chains. " +
			"Must be blank for chains deploying to local allocs files.",
		EnvVars: []string{
			"L1_RPC_URL",
		},
	}
	ArtifactsLocatorFlag = &cli.StringFlag{
		Name:    ArtifactsLocatorFlagName,
		Usage:   "Locator for artifacts.",
		EnvVars: PrefixEnvVar("ARTIFACTS_LOCATOR"),
		Value:   artifacts.EmbeddedLocatorString,
	}
	CacheDirFlag = &cli.StringFlag{
		Name: CacheDirFlagName,
		Usage: "Cache directory. " +
			"If set, the deployer will attempt to cache downloaded artifacts in the specified directory.",
		EnvVars: PrefixEnvVar("CACHE_DIR"),
		Value:   flags.DefaultCacheDir(),
	}
	L1ChainIDFlag = &cli.Uint64Flag{
		Name:    L1ChainIDFlagName,
		Usage:   "Chain ID of the L1 chain.",
		EnvVars: PrefixEnvVar("L1_CHAIN_ID"),
		Value:   11155111,
	}
	L2ChainIDsFlag = &cli.StringFlag{
		Name:    L2ChainIDsFlagName,
		Usage:   "Comma-separated list of L2 chain IDs to deploy.",
		EnvVars: PrefixEnvVar("L2_CHAIN_IDS"),
	}
	WorkdirFlag = &cli.StringFlag{
		Name:    WorkdirFlagName,
		Usage:   "Directory storing intent and stage. Defaults to the current directory.",
		EnvVars: PrefixEnvVar("WORKDIR"),
		Value:   cwd(),
		Aliases: []string{
			OutdirFlagName,
		},
	}
	PrivateKeyFlag = &cli.StringFlag{
		Name:    PrivateKeyFlagName,
		Usage:   "Private key of the deployer account.",
		EnvVars: PrefixEnvVar("PRIVATE_KEY"),
	}
	DeploymentTargetFlag = &cli.StringFlag{
		Name:    "deployment-target",
		Usage:   fmt.Sprintf("Where to deploy L1 contracts. Options: %s, %s, %s, %s", DeploymentTargetLive, DeploymentTargetGenesis, DeploymentTargetCalldata, DeploymentTargetNoop),
		EnvVars: PrefixEnvVar("DEPLOYMENT_TARGET"),
		Value:   string(DeploymentTargetLive),
	}
	OpProgramSvcUrlFlag = &cli.StringFlag{
		Name:    "op-program-svc-url",
		Usage:   "URL of the OP Program SVC",
		EnvVars: PrefixEnvVar("OP_PROGRAM_SVC_URL"),
	}
	IntentTypeFlag = &cli.StringFlag{
		Name: IntentTypeFlagName,
		Usage: fmt.Sprintf("Intent config type to use. Options: %s (default), %s, %s",
			state.IntentTypeStandard,
			state.IntentTypeCustom,
			state.IntentTypeStandardOverrides),
		EnvVars: PrefixEnvVar("INTENT_TYPE"),
		Value:   string(state.IntentTypeStandard),
		Aliases: []string{
			"intent-config-type",
		},
	}
	VerifierAPIKeyFlag = &cli.StringFlag{
		Name:    VerifierAPIKeyFlagName,
		Usage:   "API key for contract verifier (etherscan, blockscout, etc.)",
		EnvVars: append(PrefixEnvVar("VERIFIER_API_KEY"), PrefixEnvVar("ETHERSCAN_API_KEY")...),
		Aliases: []string{EtherscanAPIKeyFlagName},
	}
	InputFileFlag = &cli.StringFlag{
		Name:    InputFileFlagName,
		Usage:   "filepath of input file for command",
		EnvVars: PrefixEnvVar("INPUT_FILE"),
	}
	ContractNameFlag = &cli.StringFlag{
		Name:    ContractNameFlagName,
		Usage:   "(optional) contract name matching a field within the input file",
		EnvVars: PrefixEnvVar("CONTRACT_NAME"),
	}
	VerifierFlag = &cli.StringFlag{
		Name:    VerifierTypeFlagName,
		Usage:   "contract verifier type(s) to use. Comma-separated for multiple verifiers. Options: etherscan (default), blockscout, custom. Example: etherscan,blockscout",
		EnvVars: PrefixEnvVar("VERIFIER_TYPE"),
		Value:   "etherscan",
	}
	VerifierUrlFlag = &cli.StringFlag{
		Name:    VerifierUrlFlagName,
		Usage:   "verifier URL (optional for blockscout, required for custom, ignored for etherscan)",
		EnvVars: PrefixEnvVar("VERIFIER_URL"),
	}
	AutoVerifyFlag = &cli.BoolFlag{
		Name:    "verify",
		Usage:   "automatically verify contracts after deployment",
		EnvVars: PrefixEnvVar("VERIFY"),
		Value:   false,
	}
)

var GlobalFlags = append([]cli.Flag{CacheDirFlag}, oplog.CLIFlags(EnvVarPrefix)...)

var InitFlags = []cli.Flag{
	L1ChainIDFlag,
	L2ChainIDsFlag,
	WorkdirFlag,
	IntentTypeFlag,
}

var ApplyFlags = []cli.Flag{
	L1RPCURLFlag,
	WorkdirFlag,
	PrivateKeyFlag,
	DeploymentTargetFlag,
	OpProgramSvcUrlFlag,
	AutoVerifyFlag,
	VerifierAPIKeyFlag,
	VerifierFlag,
	VerifierUrlFlag,
}

var UpgradeFlags = []cli.Flag{
	L1RPCURLFlag,
	PrivateKeyFlag,
	DeploymentTargetFlag,
}

var VerifyFlags = []cli.Flag{
	L1RPCURLFlag,
	ArtifactsLocatorFlag,
	VerifierAPIKeyFlag,
	InputFileFlag,
	ContractNameFlag,
	VerifierFlag,
	VerifierUrlFlag,
}

func PrefixEnvVar(name string) []string {
	return op_service.PrefixEnvVar(EnvVarPrefix, name)
}
