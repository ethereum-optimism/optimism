package cmd

import (
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli/v2"

	"cannon/mipsevm"
)

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
		Usage:     "path of output JSON state. Stdout if left empty.",
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
		Usage:    "format for proof data output file names. Proof data is written to stdout if empty.",
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
)

type Proof struct {
	Step uint64 `json:"step"`

	Pre  common.Hash `json:"pre"`
	Post common.Hash `json:"post"`

	StepInput   hexutil.Bytes `json:"step-input"`
	OracleInput hexutil.Bytes `json:"oracle-input"`
}

func Run(ctx *cli.Context) error {
	state, err := loadJSON[mipsevm.State](ctx.Path(RunInputFlag.Name))
	if err != nil {
		return err
	}
	mu, err := mipsevm.NewUnicorn()
	if err != nil {
		return fmt.Errorf("failed to create unicorn emulator: %w", err)
	}
	if err := mipsevm.LoadUnicorn(state, mu); err != nil {
		return fmt.Errorf("failed to load state into unicorn emulator: %w", err)
	}
	l := Logger(os.Stderr, log.LvlInfo)
	outLog := &mipsevm.LoggingWriter{Name: "program std-out", Log: l}
	errLog := &mipsevm.LoggingWriter{Name: "program std-err", Log: l}

	var po mipsevm.PreimageOracle // TODO need to set this up

	stopAt := ctx.Generic(RunStopAtFlag.Name).(*StepMatcherFlag).Matcher()
	proofAt := ctx.Generic(RunProofAtFlag.Name).(*StepMatcherFlag).Matcher()
	snapshotAt := ctx.Generic(RunSnapshotAtFlag.Name).(*StepMatcherFlag).Matcher()

	us, err := mipsevm.NewUnicornState(mu, state, po, outLog, errLog)
	if err != nil {
		return fmt.Errorf("failed to setup instrumented VM state: %w", err)
	}
	proofFmt := ctx.String(RunProofFmtFlag.Name)
	snapshotFmt := ctx.String(RunSnapshotFmtFlag.Name)

	for !state.Exited {
		step := state.Step

		if stopAt(state) {
			break
		}

		if snapshotAt(state) {
			if err := writeJSON[*mipsevm.State](fmt.Sprintf(snapshotFmt, step), state, false); err != nil {
				return fmt.Errorf("failed to write state snapshot: %w", err)
			}
		}

		if proofAt(state) {
			preStateHash := crypto.Keccak256Hash(state.EncodeWitness())
			witness := us.Step(true)
			postStateHash := crypto.Keccak256Hash(state.EncodeWitness())
			proof := &Proof{
				Step:      step,
				Pre:       preStateHash,
				Post:      postStateHash,
				StepInput: witness.EncodeStepInput(),
			}
			if witness.HasPreimage() {
				proof.OracleInput, err = witness.EncodePreimageOracleInput()
				if err != nil {
					return fmt.Errorf("failed to encode pre-image oracle input: %w", err)
				}
			}
			if err := writeJSON[*Proof](fmt.Sprintf(proofFmt, step), proof, true); err != nil {
				return fmt.Errorf("failed to write proof data: %w", err)
			}
		} else {
			_ = us.Step(false)
		}
	}

	if err := writeJSON[*mipsevm.State](ctx.Path(RunOutputFlag.Name), state, true); err != nil {
		return fmt.Errorf("failed to write state output: %w", err)
	}
	return nil
}

var RunCommand = &cli.Command{
	Name:        "run",
	Usage:       "Run VM step(s) and generate proof data to replicate onchain.",
	Description: "Run VM step(s) and generate proof data to replicate onchain. See flags to match when to output a proof, a snapshot, or to stop early.",
	Action:      Run,
	Flags: []cli.Flag{
		RunInputFlag,
		RunOutputFlag,
		RunProofAtFlag,
		RunProofFmtFlag,
		RunSnapshotAtFlag,
		RunSnapshotFmtFlag,
		RunStopAtFlag,
	},
}
