// Package engineipc spawns a subprocess execution engine that serves JSON-RPC over a Unix domain
// socket (go-ethereum rpc.DialIPC-compatible, reth-ipc newline framing) and manages its lifecycle.
//
// It is the shared transport layer for driving out-of-process Rust engines from Go tests: it owns
// the temp socket path, starts the child with a death-signal so a hard-killed parent leaks nothing,
// drains the child's stderr, waits for readiness, and dials the socket. Argument construction and
// the typed RPC method set stay with each consumer — this package is transport only, with no
// op-node, op-e2e, or op-chain-ops dependencies.
package engineipc

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/ethereum/go-ethereum/rpc"
)

// DefaultSocketFlag is the CLI flag under which the engine binary expects its Unix socket path.
const DefaultSocketFlag = "--socket"

// DefaultReadyTimeout bounds how long Spawn waits for the child to create and serve its socket.
const DefaultReadyTimeout = 30 * time.Second

// Opts tunes Spawn. The zero value is valid and uses the defaults above.
type Opts struct {
	// SocketFlag overrides the flag under which the socket path is passed (default "--socket").
	SocketFlag string
	// ReadyTimeout overrides how long to wait for the socket (default 30s).
	ReadyTimeout time.Duration
	// SocketName overrides the socket file name inside the temp dir (default "engine.sock").
	SocketName string
}

func (o *Opts) socketFlag() string {
	if o != nil && o.SocketFlag != "" {
		return o.SocketFlag
	}
	return DefaultSocketFlag
}

func (o *Opts) readyTimeout() time.Duration {
	if o != nil && o.ReadyTimeout > 0 {
		return o.ReadyTimeout
	}
	return DefaultReadyTimeout
}

func (o *Opts) socketName() string {
	if o != nil && o.SocketName != "" {
		return o.SocketName
	}
	return "engine.sock"
}

// Proc is a handle to a spawned engine subprocess and its dialed IPC client.
type Proc struct {
	cmd    *exec.Cmd
	client *rpc.Client
	tmpDir string
	sock   string
}

// Spawn launches binPath with args plus the socket flag (whose value it owns), waits for the
// socket, and dials it. The child's stderr is forwarded to logw. The returned Proc must be closed.
//
// Spawn appends the socket flag itself — callers pass every other argument (e.g. "--genesis <path>")
// in args. This keeps socket lifecycle (temp dir, unique path, cleanup) entirely inside the package.
func Spawn(binPath string, args []string, logw io.Writer, opts *Opts) (*Proc, error) {
	tmpDir, err := os.MkdirTemp("", "engineipc")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}
	sock := filepath.Join(tmpDir, opts.socketName())

	fullArgs := make([]string, 0, len(args)+2)
	fullArgs = append(fullArgs, args...)
	fullArgs = append(fullArgs, opts.socketFlag(), sock)

	cmd := exec.Command(binPath, fullArgs...)
	setDeathSignal(cmd) // SIGKILL the engine if this process dies (no leaked engines)
	stderr, err := cmd.StderrPipe()
	if err != nil {
		_ = os.RemoveAll(tmpDir)
		return nil, fmt.Errorf("pipe stderr: %w", err)
	}
	if err := cmd.Start(); err != nil {
		_ = os.RemoveAll(tmpDir)
		return nil, fmt.Errorf("start %s: %w", binPath, err)
	}

	go drainStderr(stderr, logw)

	p := &Proc{cmd: cmd, tmpDir: tmpDir, sock: sock}
	deadline := time.Now().Add(opts.readyTimeout())
	for {
		if _, statErr := os.Stat(sock); statErr == nil {
			if cl, dialErr := rpc.DialIPC(context.Background(), sock); dialErr == nil {
				p.client = cl
				return p, nil
			}
		}
		if time.Now().After(deadline) {
			p.Close()
			return nil, fmt.Errorf("engine never became ready on %s within %s", sock, opts.readyTimeout())
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// drainStderr forwards the child's stderr to logw. It uses a bufio.Reader rather than a Scanner so
// a single long log line can't hit Scanner's 64KiB token cap, stop the drain, fill the pipe, and
// hang the engine.
func drainStderr(stderr io.Reader, logw io.Writer) {
	br := bufio.NewReader(stderr)
	for {
		line, err := br.ReadString('\n')
		if len(line) > 0 && logw != nil {
			fmt.Fprintf(logw, "[engine] %s", line)
		}
		if err != nil {
			return
		}
	}
}

// Client returns the dialed go-ethereum IPC client. Valid until Close.
func (p *Proc) Client() *rpc.Client {
	return p.client
}

// SocketPath returns the Unix socket path the engine listens on.
func (p *Proc) SocketPath() string {
	return p.sock
}

// Close terminates the engine and removes its temp dir. It is idempotent.
func (p *Proc) Close() {
	if p == nil {
		return
	}
	if p.client != nil {
		p.client.Close()
		p.client = nil
	}
	if p.cmd != nil && p.cmd.Process != nil {
		_ = p.cmd.Process.Kill()
		_ = p.cmd.Wait()
		p.cmd = nil
	}
	if p.tmpDir != "" {
		_ = os.RemoveAll(p.tmpDir)
		p.tmpDir = ""
	}
}
