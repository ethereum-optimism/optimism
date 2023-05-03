package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli/v2"

	"cannon/mipsevm"
	"github.com/ethereum-optimism/cannon/preimage"
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
	RunMetaFlag = &cli.PathFlag{
		Name:     "meta",
		Usage:    "path to metadata file for symbol lookup for enhanced debugging info durign execution.",
		Value:    "meta.json",
		Required: false,
	}
	RunInfoAtFlag = &cli.GenericFlag{
		Name:     "info-at",
		Usage:    "step pattern to print info at: " + patternHelp,
		Value:    MustStepMatcherFlag("%1000"),
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

type rawHint string

func (rh rawHint) Hint() string {
	return string(rh)
}

type rawKey [32]byte

func (rk rawKey) PreimageKey() [32]byte {
	return rk
}

type ProcessPreimageOracle struct {
	pCl *preimage.OracleClient
	hCl *preimage.HintWriter
	cmd *exec.Cmd
}

func NewProcessPreimageOracle(name string, args []string) *ProcessPreimageOracle {
	if name == "" {
		return &ProcessPreimageOracle{}
	}

	pCh := preimage.ClientPreimageChannel()
	hCh := preimage.ClientHinterChannel()

	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.ExtraFiles = []*os.File{
		hCh.Reader(),
		hCh.Writer(),
		pCh.Reader(),
		pCh.Writer(),
	}
	out := &ProcessPreimageOracle{
		pCl: preimage.NewOracleClient(pCh),
		hCl: preimage.NewHintWriter(hCh),
		cmd: cmd,
	}
	return out
}

func (p *ProcessPreimageOracle) Hint(v []byte) {
	if p.hCl == nil { // no hint processor
		return
	}
	p.hCl.Hint(rawHint(v))
}

func (p *ProcessPreimageOracle) GetPreimage(k [32]byte) []byte {
	if p.pCl == nil {
		panic("no pre-image retriever available")
	}
	return p.pCl.Get(rawKey(k))
}

func (p *ProcessPreimageOracle) Start() error {
	if p.cmd == nil {
		return nil
	}
	return p.cmd.Start()
}

func (p *ProcessPreimageOracle) Close() error {
	if p.cmd == nil {
		return nil
	}
	_ = p.cmd.Process.Signal(os.Interrupt)
	p.cmd.WaitDelay = time.Second * 10
	return p.cmd.Wait()
}

type StepFn func(proof bool) (*mipsevm.StepWitness, error)

func Guard(proc *os.ProcessState, fn StepFn) StepFn {
	return func(proof bool) (*mipsevm.StepWitness, error) {
		wit, err := fn(proof)
		if err != nil {
			if proc.Exited() {
				return nil, fmt.Errorf("pre-image server exited with code %d, resulting in err %v", proc.ExitCode(), err)
			} else {
				return nil, err
			}
		}
		return wit, nil
	}
}

var _ mipsevm.PreimageOracle = (*ProcessPreimageOracle)(nil)

func Run(ctx *cli.Context) error {
	state, err := loadJSON[mipsevm.State](ctx.Path(RunInputFlag.Name))
	if err != nil {
		return err
	}
	//mu, err := mipsevm.NewUnicorn()
	//if err != nil {
	//	return fmt.Errorf("failed to create unicorn emulator: %w", err)
	//}
	//if err := mipsevm.LoadUnicorn(state, mu); err != nil {
	//	return fmt.Errorf("failed to load state into unicorn emulator: %w", err)
	//}
	l := Logger(os.Stderr, log.LvlInfo)
	outLog := &mipsevm.LoggingWriter{Name: "program std-out", Log: l}
	errLog := &mipsevm.LoggingWriter{Name: "program std-err", Log: l}

	// split CLI args after first '--'
	args := ctx.Args().Slice()
	for i, arg := range args {
		if arg == "--" {
			args = args[i+1:]
			break
		}
	}
	if len(args) == 0 {
		args = []string{""}
	}

	po := NewProcessPreimageOracle(args[0], args[1:])
	if err := po.Start(); err != nil {
		return fmt.Errorf("failed to start pre-image oracle server: %w", err)
	}
	defer func() {
		if err := po.Close(); err != nil {
			l.Error("failed to close pre-image server", "err", err)
		}
	}()

	stopAt := ctx.Generic(RunStopAtFlag.Name).(*StepMatcherFlag).Matcher()
	proofAt := ctx.Generic(RunProofAtFlag.Name).(*StepMatcherFlag).Matcher()
	snapshotAt := ctx.Generic(RunSnapshotAtFlag.Name).(*StepMatcherFlag).Matcher()
	infoAt := ctx.Generic(RunInfoAtFlag.Name).(*StepMatcherFlag).Matcher()

	var meta *mipsevm.Metadata
	if metaPath := ctx.Path(RunMetaFlag.Name); metaPath == "" {
		l.Info("no metadata file specified, defaulting to empty metadata")
		meta = &mipsevm.Metadata{Symbols: nil} // provide empty metadata by default
	} else {
		if m, err := loadJSON[mipsevm.Metadata](metaPath); err != nil {
			return fmt.Errorf("failed to load metadata: %w", err)
		} else {
			meta = m
		}
	}

	//us, err := mipsevm.NewUnicornState(mu, state, po, outLog, errLog)
	//if err != nil {
	//	return fmt.Errorf("failed to setup instrumented VM state: %w", err)
	//}
	us := mipsevm.NewNonUnicornState(state, po, outLog, errLog)
	proofFmt := ctx.String(RunProofFmtFlag.Name)
	snapshotFmt := ctx.String(RunSnapshotFmtFlag.Name)

	stepFn := us.NonUnicornStep
	if po.cmd != nil {
		stepFn = Guard(po.cmd.ProcessState, stepFn)
	}

	for !state.Exited {
		step := state.Step

		name := meta.LookupSymbol(state.PC)
		if infoAt(state) {
			l.Info("processing",
				"step", step,
				"pc", mipsevm.HexU32(state.PC),
				"insn", mipsevm.HexU32(state.Memory.GetMemory(state.PC)),
				"name", name,
			)
		}
		if name == "runtime.notesleep" { // don't loop forever when we get stuck because of an unexpected bad program
			return fmt.Errorf("got stuck in Go sleep at step %d", step)
		}

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
			witness, err := stepFn(true)
			if err != nil {
				return fmt.Errorf("failed at proof-gen step %d (PC: %08x): %w", step, state.PC, err)
			}
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
			_, err = stepFn(false)
			if err != nil {
				return fmt.Errorf("failed at step %d (PC: %08x): %w", step, state.PC, err)
			}
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
		RunMetaFlag,
		RunInfoAtFlag,
	},
}
