package flags

import (
	"github.com/urfave/cli/v2"

	opservice "github.com/ethereum-optimism/optimism/op-service"
)

const EnvVarPrefix = "OP_TEST"

func prefixEnvVars(name string) []string {
	return opservice.PrefixEnvVar(EnvVarPrefix, name)
}

var (
	ServerFlag = &cli.StringFlag{
		Name:    "server",
		Usage:   "RPC server endpoint for remote op-test backend execution. Disabled if empty.",
		EnvVars: prefixEnvVars("SERVER"),
		Value:   "http://localhost:5000/experimental",
	}
	ModeFlag = &cli.StringFlag{
		Name:    "mode",
		Usage:   "Operation mode. 'plan' or 'server'",
		EnvVars: prefixEnvVars("MODE"),
	}
	PlanOutputPath = &cli.StringFlag{
		Name:    "plan-output",
		Usage:   "Path to write test-plan JSON to. Plan is written to out/<dots-import-path> if left empty.",
		EnvVars: prefixEnvVars("PLAN_OUTPUT"),
	}
	PlanInputPath = &cli.StringFlag{
		Name:    "plan-input",
		Usage:   "When running as server, a pre-existing plan is consumed to navigate all parameter choices",
		EnvVars: prefixEnvVars("PLAN_INPUT"),
	}
)

// parameters
var (
	L1ForksFlag = &cli.StringSliceFlag{
		Name:     "params.l1-forks",
		Category: "params",
		Usage:    "L1 forks to run test with",
		EnvVars:  prefixEnvVars("PARAMS_L1_FORKS"),
		Value:    cli.NewStringSlice("dencun", "shapella"),
	}
	L2ForksFlag = &cli.StringSliceFlag{
		Name:     "params.l2-forks",
		Category: "params",
		Usage:    "L2 forks to run test with",
		EnvVars:  prefixEnvVars("PARAMS_L2_FORKS"),
		Value:    cli.NewStringSlice("ecotone", "delta", "canyon", "bedrock"),
	}
)

var Flags []cli.Flag

func init() {
	// op-test framework flags
	Flags = append(Flags,
		ServerFlag,
		ModeFlag,
		PlanOutputPath,
		PlanInputPath,
	)
	// test parameters
	Flags = append(Flags,
		L1ForksFlag,
		L2ForksFlag,
	)
}
