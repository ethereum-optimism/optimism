package ipc_test

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/rpc"
)

// helperArg, as the first argument, makes the test binary run as the helper child instead of
// running tests. Tests spawn os.Args[0] with it, so the package needs no external binary.
const helperArg = "--ipc-helper"

// helperLingerBound is how long a helper that never serves stays alive on its own, so one orphaned
// by a test cannot outlive the test run by much.
const helperLingerBound = time.Minute

func TestMain(m *testing.M) {
	if len(os.Args) > 1 && os.Args[1] == helperArg {
		os.Exit(runHelper(os.Args[2:]))
	}
	os.Exit(m.Run())
}

// runHelper is the helper child: a JSON-RPC server under the "test" namespace on the socket passed
// via --socket, with flags that make it misbehave at startup. It returns the exit status.
func runHelper(args []string) int {
	fs := flag.NewFlagSet(helperArg, flag.ContinueOnError)
	socket := fs.String("socket", "", "path to serve JSON-RPC on")
	exitEarly := fs.Int("exit-early", -1, "exit with this status before listening, after a few stderr lines")
	noListen := fs.Bool("no-listen", false, "never create the socket")
	readyDelay := fs.Duration("ready-delay", 0, "wait this long before creating the socket")
	listenInstead := fs.String("ignore-socket-arg", "", "serve on this path instead of --socket")
	prelude := fs.Int("stderr-prelude", 0, "write this many stderr lines before listening")
	orphanPidFile := fs.String("orphan-stderr", "", "start a descendant holding stderr open, write its pid here")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	for i := range *prelude {
		fmt.Fprintf(os.Stderr, "prelude %02d\n", i)
	}
	if *orphanPidFile != "" {
		// The descendant inherits stderr and outlives this process, like a daemon a wrapper script
		// left behind.
		cmd := exec.Command(os.Args[0], helperArg, "--no-listen")
		cmd.Stderr = os.Stderr
		if err := cmd.Start(); err != nil {
			fmt.Fprintf(os.Stderr, "start descendant: %v\n", err)
			return 1
		}
		if err := os.WriteFile(*orphanPidFile, []byte(strconv.Itoa(cmd.Process.Pid)), 0o600); err != nil {
			fmt.Fprintf(os.Stderr, "write pid file: %v\n", err)
			return 1
		}
	}
	if *exitEarly >= 0 {
		fmt.Fprintf(os.Stderr, "helper args: %s\n", strings.Join(args, " "))
		fmt.Fprintf(os.Stderr, "exiting with status %d\n", *exitEarly)
		return *exitEarly
	}
	if *noListen {
		time.Sleep(helperLingerBound)
		return 0
	}
	time.Sleep(*readyDelay)

	path := *socket
	if *listenInstead != "" {
		path = *listenInstead
	}
	l, err := net.Listen("unix", path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "listen: %v\n", err)
		return 1
	}
	srv := rpc.NewServer()
	if err := srv.RegisterName("test", new(helperService)); err != nil {
		fmt.Fprintf(os.Stderr, "register: %v\n", err)
		return 1
	}
	_ = srv.ServeListener(l)
	return 0
}

// helperService is the helper's "test" RPC namespace, one method per client-side path.
type helperService struct{}

// Echo returns v unchanged, byte for byte. go-ethereum's server has no variadic methods, so it
// takes a single value; callers wrap several in an array or object.
func (helperService) Echo(v json.RawMessage) json.RawMessage {
	return v
}

// Fail returns a JSON-RPC error with exactly the given code, message and data.
func (helperService) Fail(code int, message string, data json.RawMessage) error {
	return &helperError{code: code, message: message, data: data}
}

// Log writes s to stderr verbatim, without adding a newline.
func (helperService) Log(s string) error {
	_, err := os.Stderr.WriteString(s)
	return err
}

// Exit terminates the helper with the given status without replying, so the call always fails
// rather than racing the reply against the exit.
func (helperService) Exit(code int) {
	os.Exit(code)
}

// Block answers after d, or once the connection closes. It announces on stderr that the call has
// arrived, so a test can act while it is in flight.
func (helperService) Block(ctx context.Context, d time.Duration) error {
	fmt.Fprintf(os.Stderr, "%s %s\n", blockingMarker, d)
	select {
	case <-time.After(d):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

const blockingMarker = "test_block: blocking for"

type helperError struct {
	code    int
	message string
	data    json.RawMessage
}

func (e *helperError) Error() string  { return e.message }
func (e *helperError) ErrorCode() int { return e.code }
func (e *helperError) ErrorData() any { return e.data }

// syncLog collects a child's forwarded stderr. Spawn writes to it from its own goroutine while the
// test reads, so access is locked, and every write wakes waitFor.
type syncLog struct {
	mu   sync.Mutex
	buf  bytes.Buffer
	grew chan struct{}
}

// newSyncLog returns a syncLog that is printed to the test log if the test fails.
func newSyncLog(t *testing.T) *syncLog {
	l := &syncLog{grew: make(chan struct{})}
	t.Cleanup(func() {
		if t.Failed() {
			t.Logf("child stderr:\n%s", l.String())
		}
	})
	return l
}

func (l *syncLog) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.buf.Write(p)
	close(l.grew)
	l.grew = make(chan struct{})
	return len(p), nil
}

func (l *syncLog) String() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.buf.String()
}

// waitFor blocks until the log contains substr, failing the test after a generous bound.
func (l *syncLog) waitFor(t *testing.T, substr string) {
	t.Helper()
	deadline := time.NewTimer(waitBound)
	defer deadline.Stop()
	for {
		l.mu.Lock()
		found := strings.Contains(l.buf.String(), substr)
		grew := l.grew
		l.mu.Unlock()
		if found {
			return
		}
		select {
		case <-grew:
		case <-deadline.C:
			t.Fatalf("child stderr never contained %q", substr)
		}
	}
}
