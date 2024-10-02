package anvil

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/log"
)

func Test(t *testing.T) {
	if os.Getenv("ENABLE_ANVIL") == "" {
		t.Skip("skipping Anvil test")
	}
}

const AnvilPort = 31967

type Runner struct {
	proc      *exec.Cmd
	stdout    io.ReadCloser
	stderr    io.ReadCloser
	logger    log.Logger
	startedCh chan struct{}
	wg        sync.WaitGroup
}

func New(l1RPCURL string, logger log.Logger) (*Runner, error) {
	proc := exec.Command(
		"anvil",
		"--fork-url", l1RPCURL,
		"--port",
		strconv.Itoa(AnvilPort),
	)
	stdout, err := proc.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr, err := proc.StderrPipe()
	if err != nil {
		return nil, err
	}

	return &Runner{
		proc:      proc,
		stdout:    stdout,
		stderr:    stderr,
		logger:    logger,
		startedCh: make(chan struct{}, 1),
	}, nil
}

func (r *Runner) Start(ctx context.Context) error {
	if err := r.proc.Start(); err != nil {
		return err
	}

	r.wg.Add(2)
	go r.outputStream(r.stdout)
	go r.outputStream(r.stderr)

	select {
	case <-r.startedCh:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (r *Runner) Stop() error {
	err := r.proc.Process.Signal(os.Interrupt)
	if err != nil {
		return err
	}

	// make sure the output streams close
	defer r.wg.Wait()
	return r.proc.Wait()
}

func (r *Runner) outputStream(stream io.ReadCloser) {
	defer r.wg.Done()
	scanner := bufio.NewScanner(stream)
	listenLine := fmt.Sprintf("Listening on 127.0.0.1:%d", AnvilPort)
	started := sync.OnceFunc(func() {
		r.startedCh <- struct{}{}
	})

	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, listenLine) {
			started()
		}

		r.logger.Debug("[ANVIL] " + scanner.Text())
	}
}

func (r *Runner) RPCUrl() string {
	return fmt.Sprintf("http://localhost:%d", AnvilPort)
}
