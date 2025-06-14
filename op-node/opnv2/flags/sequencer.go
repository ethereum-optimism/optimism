package flags

import (
	"time"

	"github.com/urfave/cli/v2"
)

var (
	SequencerEnabledFlag = &cli.BoolFlag{
		Name:     "sequencer.enabled",
		Usage:    "Enable sequencing of new L2 blocks. A separate batch submitter has to be deployed to publish the data for verifiers.",
		EnvVars:  prefixEnvVars("SEQUENCER_ENABLED"),
		Category: SequencerCategory,
	}
	SequencerStoppedFlag = &cli.BoolFlag{
		Name:     "sequencer.stopped",
		Usage:    "Initialize the sequencer in a stopped state. The sequencer can be started using the admin_startSequencer RPC",
		EnvVars:  prefixEnvVars("SEQUENCER_STOPPED"),
		Category: SequencerCategory,
	}
	SequencerMaxSafeLagFlag = &cli.Uint64Flag{
		Name:     "sequencer.max-safe-lag",
		Usage:    "Maximum number of L2 blocks for restricting the distance between L2 safe and unsafe. Disabled if 0.",
		EnvVars:  prefixEnvVars("SEQUENCER_MAX_SAFE_LAG"),
		Value:    0,
		Category: SequencerCategory,
	}
	SequencerL1Confs = &cli.Uint64Flag{
		Name:     "sequencer.l1-confs",
		Usage:    "Number of L1 blocks to keep distance from the L1 head as a sequencer for picking an L1 origin.",
		EnvVars:  prefixEnvVars("SEQUENCER_L1_CONFS"),
		Value:    4,
		Category: SequencerCategory,
	}
	SequencerRecoverMode = &cli.BoolFlag{
		Name:     "sequencer.recover",
		Usage:    "Forces the sequencer to strictly prepare the next L1 origin and create empty L2 blocks",
		EnvVars:  prefixEnvVars("SEQUENCER_RECOVER"),
		Value:    false,
		Category: SequencerCategory,
	}
	ConductorEnabledFlag = &cli.BoolFlag{
		Name:     "conductor.enabled",
		Usage:    "Enable the conductor service",
		EnvVars:  prefixEnvVars("CONDUCTOR_ENABLED"),
		Value:    false,
		Category: SequencerCategory,
	}
	ConductorRpcFlag = &cli.StringFlag{
		Name:     "conductor.rpc",
		Usage:    "Conductor service rpc endpoint",
		EnvVars:  prefixEnvVars("CONDUCTOR_RPC"),
		Value:    "http://127.0.0.1:8547",
		Category: SequencerCategory,
	}
	ConductorRpcTimeoutFlag = &cli.DurationFlag{
		Name:     "conductor.rpc-timeout",
		Usage:    "Conductor service rpc timeout",
		EnvVars:  prefixEnvVars("CONDUCTOR_RPC_TIMEOUT"),
		Value:    time.Second * 1,
		Category: SequencerCategory,
	}
)

var SequencerFlags = []cli.Flag{
	SequencerEnabledFlag,
	SequencerStoppedFlag,
	SequencerMaxSafeLagFlag,
	SequencerL1Confs,
	SequencerRecoverMode,
	ConductorEnabledFlag,
	ConductorRpcFlag,
	ConductorRpcTimeoutFlag,
}
