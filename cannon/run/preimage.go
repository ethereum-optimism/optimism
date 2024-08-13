package run

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/ethereum-optimism/optimism/cannon/logutil"
	preimage "github.com/ethereum-optimism/optimism/op-preimage"
	"github.com/ethereum/go-ethereum/log"
)

type rawHint string

func (rh rawHint) Hint() string {
	return string(rh)
}

type rawKey [32]byte

func (rk rawKey) PreimageKey() [32]byte {
	return rk
}

type ProcessPreimageOracle struct {
	pCl      *preimage.OracleClient
	hCl      *preimage.HintWriter
	cmd      *exec.Cmd
	waitErr  chan error
	cancelIO context.CancelCauseFunc
}

const clientPollTimeout = time.Second * 15

func NewProcessPreimageOracle(name string, args []string, stdout log.Logger, stderr log.Logger) (*ProcessPreimageOracle, error) {
	if name == "" {
		return &ProcessPreimageOracle{}, nil
	}

	pClientRW, pOracleRW, err := preimage.CreateBidirectionalChannel()
	if err != nil {
		return nil, err
	}
	hClientRW, hOracleRW, err := preimage.CreateBidirectionalChannel()
	if err != nil {
		return nil, err
	}

	cmd := exec.Command(name, args...) // nosemgrep
	cmd.Stdout = &logutil.LoggingWriter{Log: stdout}
	cmd.Stderr = &logutil.LoggingWriter{Log: stderr}
	cmd.ExtraFiles = []*os.File{
		hOracleRW.Reader(),
		hOracleRW.Writer(),
		pOracleRW.Reader(),
		pOracleRW.Writer(),
	}

	// Note that the client file descriptors are not closed when the pre-image server exits.
	// So we use the FilePoller to ensure that we don't get stuck in a blocking read/write.
	ctx, cancelIO := context.WithCancelCause(context.Background())
	preimageClientIO := preimage.NewFilePoller(ctx, pClientRW, clientPollTimeout)
	hostClientIO := preimage.NewFilePoller(ctx, hClientRW, clientPollTimeout)
	out := &ProcessPreimageOracle{
		pCl:      preimage.NewOracleClient(preimageClientIO),
		hCl:      preimage.NewHintWriter(hostClientIO),
		cmd:      cmd,
		waitErr:  make(chan error),
		cancelIO: cancelIO,
	}
	return out, nil
}

func (p *ProcessPreimageOracle) GuardStep(fn StepFn) StepFn {
	if p.cmd == nil {
		return fn
	}
	proc := p.cmd.ProcessState
	return func(proof bool) (*StepWitness, error) {
		wit, err := fn(proof)
		if err != nil {
			if proc.Exited() {
				return nil, fmt.Errorf("pre-image server exited with code %d, resulting in err %w", proc.ExitCode(), err)
			} else {
				return nil, err
			}
		}
		return wit, nil
	}
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
	err := p.cmd.Start()
	go p.wait()
	return err
}

func (p *ProcessPreimageOracle) Close() error {
	if p.cmd == nil {
		return nil
	}
	// Give the pre-image server time to exit cleanly before killing it.
	time.Sleep(time.Second * 1)
	_ = p.cmd.Process.Signal(os.Interrupt)
	return <-p.waitErr
}

func (p *ProcessPreimageOracle) wait() {
	err := p.cmd.Wait()
	var waitErr error
	if err, ok := err.(*exec.ExitError); !ok || !err.Success() {
		waitErr = err
	}
	p.cancelIO(fmt.Errorf("%w: pre-image server has exited", waitErr))
	p.waitErr <- waitErr
	close(p.waitErr)
}
