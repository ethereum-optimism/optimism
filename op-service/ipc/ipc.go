// Package ipc runs a child process that serves JSON-RPC on a Unix domain socket and dials it with
// go-ethereum's IPC client (newline-delimited JSON).
//
// Spawn owns the socket path, passes it to the child under a caller-chosen flag, starts the child
// with a parent-death signal so a hard-killed parent leaks nothing, forwards the child's stderr,
// waits for the socket to serve, and fails as soon as the child exits. What the child serves, and
// its other arguments, are up to the caller.
package ipc

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/rpc"
)

// readyTimeout bounds how long Spawn waits for the child to create and serve its socket.
var readyTimeout = 30 * time.Second

const (
	// stderrTailLines is how many of the child's last stderr lines a Spawn error quotes.
	stderrTailLines = 20
	// waitDelay bounds how long reaping the child waits for its stderr to reach EOF after the
	// child exits. A descendant that inherited the pipe (e.g. from a wrapper script) can hold it
	// open indefinitely, which would otherwise block Close forever.
	waitDelay = 2 * time.Second
)

// Proc is a handle to a spawned child process and its dialed IPC client.
type Proc struct {
	client *rpc.Client
	tmpDir string
	cmd    *exec.Cmd
	stderr *lineWriter
	// exited is closed once the child has exited and its stderr is fully forwarded.
	exited chan struct{}
	// waitErr is the child's exit result; read it only after exited is closed.
	waitErr error
}

// Spawn launches binPath with args followed by socketArg and a socket path it owns (e.g.
// "--socket <path>"), waits for the child to serve JSON-RPC on that socket, and dials it. Each line
// of the child's stderr is forwarded to logw, prefixed with the binary's name. The returned Proc
// must be closed.
//
// If the child exits or does not serve the socket within the ready timeout, Spawn fails with an
// error carrying the child's exit status and the tail of its stderr.
func Spawn(binPath string, socketArg string, args []string, logw io.Writer) (*Proc, error) {
	tmpDir, err := os.MkdirTemp("", "ipc")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}
	sock := filepath.Join(tmpDir, "rpc.sock")

	fullArgs := make([]string, 0, len(args)+2)
	fullArgs = append(fullArgs, args...)
	fullArgs = append(fullArgs, socketArg, sock)

	cmd := exec.Command(binPath, fullArgs...)
	setDeathSignal(cmd)
	stderr := &lineWriter{w: logw, prefix: "[" + filepath.Base(binPath) + "] "}
	cmd.Stderr = stderr
	cmd.WaitDelay = waitDelay
	if err := cmd.Start(); err != nil {
		_ = os.RemoveAll(tmpDir)
		return nil, fmt.Errorf("start %s: %w", binPath, err)
	}

	p := &Proc{cmd: cmd, tmpDir: tmpDir, stderr: stderr, exited: make(chan struct{})}
	go func() {
		p.waitErr = cmd.Wait()
		stderr.flush()
		close(p.exited)
	}()

	timeout := time.NewTimer(readyTimeout)
	defer timeout.Stop()
	poll := time.NewTicker(20 * time.Millisecond)
	defer poll.Stop()
	for {
		if _, statErr := os.Stat(sock); statErr == nil {
			if cl, dialErr := rpc.DialIPC(context.Background(), sock); dialErr == nil {
				p.client = cl
				return p, nil
			}
		}
		select {
		case <-p.exited:
			p.Close()
			return nil, fmt.Errorf("%s exited before serving its socket (%s)%s",
				binPath, exitStatus(p.waitErr), stderr.tailReport())
		case <-timeout.C:
			p.Close()
			return nil, fmt.Errorf("%s never became ready on %s within %s%s",
				binPath, sock, readyTimeout, stderr.tailReport())
		case <-poll.C:
		}
	}
}

func exitStatus(waitErr error) string {
	if waitErr == nil {
		return "exit status 0"
	}
	return waitErr.Error()
}

// lineWriter forwards the child's stderr to w, one prefixed line per write, holding
// back a trailing partial line until its newline arrives or flush is called, and keeps the last
// stderrTailLines lines. exec copies the child's stderr into it from its own goroutine, and
// Cmd.Wait returns only once that copy is done, so after Wait nothing writes to it any more.
type lineWriter struct {
	w       io.Writer
	prefix  string
	partial []byte
	tail    []string
}

func (l *lineWriter) Write(p []byte) (int, error) {
	l.partial = append(l.partial, p...)
	start := 0
	for {
		i := bytes.IndexByte(l.partial[start:], '\n')
		if i < 0 {
			break
		}
		l.emit(l.partial[start : start+i+1])
		start += i + 1
	}
	l.partial = append(l.partial[:0], l.partial[start:]...)
	return len(p), nil
}

// flush forwards a final line that lacks its newline.
func (l *lineWriter) flush() {
	if len(l.partial) > 0 {
		l.emit(append(l.partial, '\n'))
		l.partial = nil
	}
}

func (l *lineWriter) emit(line []byte) {
	if l.w != nil {
		fmt.Fprintf(l.w, "%s%s", l.prefix, line)
	}
	l.tail = append(l.tail, string(line))
	if len(l.tail) > stderrTailLines {
		l.tail = l.tail[1:]
	}
}

// tailReport formats the kept stderr lines for an error message. Call it only after the child has
// exited.
func (l *lineWriter) tailReport() string {
	if len(l.tail) == 0 {
		return "; no stderr output"
	}
	return "; stderr tail:\n" + strings.Join(l.tail, "")
}

// Client returns the dialed go-ethereum IPC client. Valid until Close.
func (p *Proc) Client() *rpc.Client {
	return p.client
}

// Close kills the child, waits for it to exit, and removes the socket's temp dir. It is
// idempotent.
func (p *Proc) Close() {
	if p == nil {
		return
	}
	if p.client != nil {
		p.client.Close()
		p.client = nil
	}
	if p.cmd != nil {
		_ = p.cmd.Process.Kill()
		<-p.exited
		p.cmd = nil
	}
	if p.tmpDir != "" {
		_ = os.RemoveAll(p.tmpDir)
		p.tmpDir = ""
	}
}
