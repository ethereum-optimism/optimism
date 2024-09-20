package cmd

import (
	"os"
	"strings"

	"github.com/urfave/cli/v2"
)

// Common Run Flags used by all build tags
var (
	RunInputFlag = &cli.PathFlag{
		Name:      "input",
		Usage:     "path of input JSON state. Stdin if left empty.",
		TakesFile: true,
		Value:     "state.json",
		Required:  true,
	}
	RunOutputFlag = &cli.PathFlag{
		Name:      "output",
		Usage:     "path of output JSON state. Not written if empty, use - to write to Stdout.",
		TakesFile: true,
		Value:     "out.json",
		Required:  false,
	}
	patternHelp    = "'never' (default), 'always', '=123' at exactly step 123, '%123' for every 123 steps"
	RunProofAtFlag = &cli.GenericFlag{
		Name:     "proof-at",
		Usage:    "step pattern to output proof at: " + patternHelp,
		Value:    new(StepMatcherFlag),
		Required: false,
	}
	RunProofFmtFlag = &cli.StringFlag{
		Name:     "proof-fmt",
		Usage:    "format for proof data output file names. Proof data is written to stdout if -.",
		Value:    "proof-%d.json",
		Required: false,
	}
	RunSnapshotAtFlag = &cli.GenericFlag{
		Name:     "snapshot-at",
		Usage:    "step pattern to output snapshots at: " + patternHelp,
		Value:    new(StepMatcherFlag),
		Required: false,
	}
	RunSnapshotFmtFlag = &cli.StringFlag{
		Name:     "snapshot-fmt",
		Usage:    "format for snapshot output file names.",
		Value:    "state-%d.json",
		Required: false,
	}
	RunStopAtFlag = &cli.GenericFlag{
		Name:     "stop-at",
		Usage:    "step pattern to stop at: " + patternHelp,
		Value:    new(StepMatcherFlag),
		Required: false,
	}
	RunStopAtPreimageFlag = &cli.StringFlag{
		Name:     "stop-at-preimage",
		Usage:    "stop at the first preimage request matching this key",
		Required: false,
	}
	RunStopAtPreimageTypeFlag = &cli.StringFlag{
		Name:     "stop-at-preimage-type",
		Usage:    "stop at the first preimage request matching this type",
		Required: false,
	}
	RunStopAtPreimageLargerThanFlag = &cli.StringFlag{
		Name:     "stop-at-preimage-larger-than",
		Usage:    "stop at the first step that requests a preimage larger than the specified size (in bytes)",
		Required: false,
	}
	RunMetaFlag = &cli.PathFlag{
		Name:     "meta",
		Usage:    "path to metadata file for symbol lookup for enhanced debugging info during execution.",
		Value:    "meta.json",
		Required: false,
	}
	RunInfoAtFlag = &cli.GenericFlag{
		Name:     "info-at",
		Usage:    "step pattern to print info at: " + patternHelp,
		Value:    MustStepMatcherFlag("%100000"),
		Required: false,
	}
	RunPProfCPU = &cli.BoolFlag{
		Name:  "pprof.cpu",
		Usage: "enable pprof cpu profiling",
	}
	RunDebugFlag = &cli.BoolFlag{
		Name:  "debug",
		Usage: "enable debug mode, which includes stack traces and other debug info in the output. Requires --meta.",
	}
	RunDebugInfoFlag = &cli.PathFlag{
		Name:      "debug-info",
		Usage:     "path to write debug info to",
		TakesFile: true,
		Required:  false,
	}

	OutFilePerm = os.FileMode(0o755)
)

func CreateRunCommand(action cli.ActionFunc) *cli.Command {
	return &cli.Command{
		Name:        "run",
		Usage:       "Run VM step(s) and generate proof data to replicate onchain.",
		Description: "Run VM step(s) and generate proof data to replicate onchain. See flags to match when to output a proof, a snapshot, or to stop early.",
		Action:      action,
		Flags: []cli.Flag{
			RunInputFlag,
			RunOutputFlag,
			RunProofAtFlag,
			RunProofFmtFlag,
			RunSnapshotAtFlag,
			RunSnapshotFmtFlag,
			RunStopAtFlag,
			RunStopAtPreimageFlag,
			RunStopAtPreimageTypeFlag,
			RunStopAtPreimageLargerThanFlag,
			RunMetaFlag,
			RunInfoAtFlag,
			RunPProfCPU,
			RunDebugFlag,
			RunDebugInfoFlag,
		},
	}
}

func removeArg(args []string, remove string) []string {
	filter := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		a := strings.Split(args[i], "=")
		if a[0] == remove {
			// don't advance if the arg value is in the same arg
			if !strings.Contains(args[i], "=") && len(a) > 0 {
				i++
			}
		} else {
			filter = append(filter, args[i])
		}
	}
	return filter
}
