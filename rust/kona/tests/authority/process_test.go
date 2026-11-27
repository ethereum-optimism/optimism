package authority_test

import (
	"bufio"
	"context"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/shared/rustbin"
	"github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
	"github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/logpipe"
	"github.com/stretchr/testify/require"
)

// process is a Rust binary run for the length of a test, its JSON logs written to a file and,
// from warnings up, to the test log.
type process struct {
	name string
	cmd  *exec.Cmd
	done chan struct{}
}

// resolveBinary returns the path of a Rust binary, from RUST_BINARY_PATH_<NAME> as devstack reads
// it, or else built.
func resolveBinary(t helpers.Testing, logger log.Logger, srcDir, pkg, bin string) string {
	path, err := rustbin.Spec{SrcDir: srcDir, Package: pkg, Binary: bin}.EnsureExists(t.Ctx(), logger)
	require.NoError(t, err, "resolve %s", bin)
	return path
}

// startProcess starts bin with args and env and returns once it logged every message in ready,
// with the fields of each, or fails the test if it exits first.
func startProcess(t helpers.Testing, logger log.Logger, logDir, name, bin string, args, env []string, ready ...string) (*process, map[string]logpipe.LogEntry) {
	cmd := exec.Command(bin, args...)
	cmd.Env = append(os.Environ(), env...)
	stdout, err := cmd.StdoutPipe()
	require.NoError(t, err)
	stderr, err := cmd.StderrPipe()
	require.NoError(t, err)
	logFile, err := os.OpenFile(filepath.Join(logDir, name+".log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	require.NoError(t, err)

	forward := logpipe.ToLoggerWithMinLevel(logger.New("process", name), slog.LevelWarn)
	var mu sync.Mutex
	seen := make(map[string]logpipe.LogEntry)
	allSeen := make(chan struct{})
	pipe := func(r io.Reader, wg *sync.WaitGroup) {
		defer wg.Done()
		scanner := bufio.NewScanner(r)
		scanner.Buffer(make([]byte, 1<<20), 16<<20)
		for scanner.Scan() {
			line := scanner.Bytes()
			mu.Lock()
			_, _ = logFile.Write(append(append([]byte{}, line...), '\n'))
			e := logpipe.ParseRustStructuredLogs(line)
			forward(e)
			for _, msg := range ready {
				if _, ok := seen[msg]; !ok && e.LogMessage() == msg {
					seen[msg] = e
					if len(seen) == len(ready) {
						close(allSeen)
					}
				}
			}
			mu.Unlock()
		}
	}
	require.NoError(t, cmd.Start(), "start %s", name)
	p := &process{name: name, cmd: cmd, done: make(chan struct{})}
	var wg sync.WaitGroup
	wg.Add(2)
	go pipe(stdout, &wg)
	go pipe(stderr, &wg)
	go func() {
		wg.Wait()
		_ = cmd.Wait()
		_ = logFile.Close()
		close(p.done)
	}()
	t.Cleanup(p.stop)

	if len(ready) == 0 {
		close(allSeen)
	}
	select {
	case <-allSeen:
	case <-p.done:
		t.Fatalf("%s exited before it was ready; see %s", name, filepath.Join(logDir, name+".log"))
	case <-time.After(2 * time.Minute):
		t.Fatalf("%s did not log %v within two minutes", name, ready)
	}
	mu.Lock()
	defer mu.Unlock()
	return p, seen
}

// stop interrupts the process and kills it if it has not exited after ten seconds.
func (p *process) stop() {
	select {
	case <-p.done:
		return
	default:
	}
	_ = p.cmd.Process.Signal(syscall.SIGINT)
	select {
	case <-p.done:
	case <-time.After(10 * time.Second):
		_ = p.cmd.Process.Kill()
		<-p.done
	}
}

// run runs bin to completion, for a one-shot command such as op-reth's init.
func run(t helpers.Testing, bin string, args ...string) {
	ctx, cancel := context.WithTimeout(t.Ctx(), time.Minute)
	defer cancel()
	out, err := exec.CommandContext(ctx, bin, args...).CombinedOutput()
	require.NoError(t, err, "%s %v: %s", bin, args, out)
}
