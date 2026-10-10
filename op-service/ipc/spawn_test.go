package ipc_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/big"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/ipc"
)

// waitBound is a generous upper bound for anything a test waits on; reaching it is a failure, never
// a synchronisation step.
const waitBound = 30 * time.Second

// spawnHelper spawns the helper child with args and closes it at the end of the test.
func spawnHelper(t *testing.T, logw io.Writer, args ...string) *ipc.Proc {
	t.Helper()
	p, err := ipc.Spawn(os.Args[0], "--socket", append([]string{helperArg}, args...), logw)
	require.NoError(t, err)
	t.Cleanup(p.Close)
	return p
}

func callCtx(t *testing.T) context.Context {
	ctx, cancel := context.WithTimeout(t.Context(), waitBound)
	t.Cleanup(cancel)
	return ctx
}

// closeWithin closes p, failing the test if that does not return within waitBound.
func closeWithin(t *testing.T, p *ipc.Proc) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		p.Close()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(waitBound):
		t.Fatal("Close did not return")
	}
}

func TestEchoRoundTrip(t *testing.T) {
	p := spawnHelper(t, nil)

	type inner struct {
		Name  string          `json:"name"`
		Flags map[string]bool `json:"flags"`
		List  []int64         `json:"list"`
		Next  *inner          `json:"next"`
	}
	type payload struct {
		Quantity hexutil.Uint64 `json:"quantity"`
		Big      *hexutil.Big   `json:"big"`
		// A plain JSON number beyond float64 precision, which only survives if nothing in between
		// decodes it into an interface.
		MaxUint uint64        `json:"maxUint"`
		Data    hexutil.Bytes `json:"data"`
		Inner   inner         `json:"inner"`
	}
	in := payload{
		Quantity: hexutil.Uint64(0x1234),
		Big:      (*hexutil.Big)(new(big.Int).Lsh(big.NewInt(1), 200)),
		MaxUint:  math.MaxUint64,
		Data:     hexutil.Bytes{0x00, 0xde, 0xad, 0xbe, 0xef},
		Inner: inner{
			Name:  "outer",
			Flags: map[string]bool{"a": true, "b": false},
			List:  []int64{-1, 0, 1 << 40},
			Next:  &inner{Name: "nested", List: []int64{}},
		},
	}
	var out payload
	require.NoError(t, p.Client().CallContext(callCtx(t), &out, "test_echo", in))
	require.Equal(t, in, out)
}

func TestErrorPassthrough(t *testing.T) {
	p := spawnHelper(t, nil)

	const code, message = -38002, "invalid forkchoice state"
	data := json.RawMessage(`{"reason":"unknown head","detail":{"number":"0x2a","hashes":["0x01","0x02"]}}`)
	err := p.Client().CallContext(callCtx(t), nil, "test_fail", code, message, data)

	var rpcErr rpc.Error
	require.ErrorAs(t, err, &rpcErr)
	require.Equal(t, code, rpcErr.ErrorCode())
	require.Equal(t, message, rpcErr.Error())
	var dataErr rpc.DataError
	require.ErrorAs(t, err, &dataErr)
	gotData, mErr := json.Marshal(dataErr.ErrorData())
	require.NoError(t, mErr)
	require.JSONEq(t, string(data), string(gotData))
}

// TestStderrForwarding checks that the child's stderr reaches logw one prefixed line per line,
// however the writes split it, including a line much longer than any pipe or copy buffer.
func TestStderrForwarding(t *testing.T) {
	logs := newSyncLog(t)
	p := spawnHelper(t, logs)

	long := strings.Repeat("x", 100<<10)
	for _, s := range []string{"first\nsecond\n", "par", "tial\n", long + "\n", "unterminated"} {
		require.NoError(t, p.Client().CallContext(callCtx(t), nil, "test_log", s))
	}
	// Each test_log write completed before its reply, and Close reaps the child only once its
	// stderr is fully forwarded, so nothing is in flight afterwards.
	closeWithin(t, p)

	prefix := "[" + filepath.Base(os.Args[0]) + "] "
	want := prefix + "first\n" + prefix + "second\n" + prefix + "partial\n" +
		prefix + abbreviate(long) + "\n" + prefix + "unterminated\n"
	require.Equal(t, want, abbreviate(logs.String()))
}

var xRun = regexp.MustCompile(`x{65,}`)

// abbreviate shortens long runs of 'x' to their length, keeping a failure diff readable.
func abbreviate(s string) string {
	return xRun.ReplaceAllStringFunc(s, func(run string) string {
		return fmt.Sprintf("<%d x>", len(run))
	})
}

// TestSpawnReportsEarlyExit spawns a child that exits before listening. Spawn must fail on the exit
// itself, not on the ready timeout, and report the exit status and stderr. The child's stderr shows
// it received the socket flag and path after the caller's arguments.
func TestSpawnReportsEarlyExit(t *testing.T) {
	_, err := ipc.Spawn(os.Args[0], "--socket", []string{helperArg, "--exit-early", "3"}, nil)
	require.ErrorContains(t, err, "exited before serving its socket (exit status 3)")
	require.Regexp(t, `helper args: --exit-early 3 --socket /\S+\nexiting with status 3\n`, err.Error())
}

// TestSpawnErrorQuotesStderrTail checks that a Spawn error quotes the last lines of a chatty
// child's stderr, not the first.
func TestSpawnErrorQuotesStderrTail(t *testing.T) {
	const prelude = 30
	_, err := ipc.Spawn(os.Args[0], "--socket",
		[]string{helperArg, "--stderr-prelude", strconv.Itoa(prelude), "--exit-early", "4"}, nil)
	require.ErrorContains(t, err, "exit status 4")
	_, tail, found := strings.Cut(err.Error(), "stderr tail:\n")
	require.True(t, found, "error lacks the stderr tail: %v", err)

	got := strings.Split(strings.TrimSuffix(tail, "\n"), "\n")
	require.Len(t, got, ipc.StderrTailLines)
	// The helper ends with two lines of its own after the prelude.
	require.Equal(t, "exiting with status 4", got[len(got)-1])
	var want []string
	for i := prelude - (ipc.StderrTailLines - 2); i < prelude; i++ {
		want = append(want, fmt.Sprintf("prelude %02d", i))
	}
	require.Equal(t, want, got[:len(got)-2])
}

func TestSpawnReadyTimeout(t *testing.T) {
	defer ipc.SetReadyTimeout(500 * time.Millisecond)()
	_, err := ipc.Spawn(os.Args[0], "--socket", []string{helperArg, "--no-listen"}, nil)
	require.ErrorContains(t, err, "never became ready")
}

// TestSpawnPollsUntilReady spawns a child that creates its socket only after a delay, so Spawn's
// first readiness checks find nothing.
func TestSpawnPollsUntilReady(t *testing.T) {
	p := spawnHelper(t, nil, "--ready-delay", "500ms")
	var out string
	require.NoError(t, p.Client().CallContext(callCtx(t), &out, "test_echo", "ready"))
	require.Equal(t, "ready", out)
}

// TestSpawnWaitsForItsOwnSocket spawns a child that serves, but on a path other than the one Spawn
// passed. Spawn must not become ready on it; once the child is proven to serve, it is made to exit,
// which ends Spawn without depending on the ready timeout.
func TestSpawnWaitsForItsOwnSocket(t *testing.T) {
	// A short dir of its own, as a socket path is length-limited.
	dir, err := os.MkdirTemp("", "ipc-alt")
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	alt := filepath.Join(dir, "alt.sock")

	spawnErr := make(chan error, 1)
	go func() {
		p, err := ipc.Spawn(os.Args[0], "--socket", []string{helperArg, "--ignore-socket-arg", alt}, nil)
		p.Close()
		spawnErr <- err
	}()

	var cl *rpc.Client
	require.Eventually(t, func() bool {
		c, err := rpc.DialIPC(t.Context(), alt)
		if err != nil {
			return false
		}
		cl = c
		return true
	}, waitBound, 20*time.Millisecond, "child never served on %s", alt)
	t.Cleanup(cl.Close)
	var out string
	require.NoError(t, cl.CallContext(callCtx(t), &out, "test_echo", "elsewhere"))
	require.Equal(t, "elsewhere", out)
	require.Error(t, cl.CallContext(callCtx(t), nil, "test_exit", 5))

	select {
	case err := <-spawnErr:
		require.ErrorContains(t, err, "exited before serving its socket (exit status 5)")
	case <-time.After(waitBound):
		t.Fatal("Spawn did not return after the child exited")
	}
}

// TestDeathAfterReady kills the child from inside after Spawn succeeded: calls fail from then on,
// and Close returns.
func TestDeathAfterReady(t *testing.T) {
	p := spawnHelper(t, nil)
	cl := p.Client()

	require.Error(t, cl.CallContext(callCtx(t), nil, "test_exit", 7), "test_exit never replies")
	require.Error(t, cl.CallContext(callCtx(t), nil, "test_echo", "after death"))
	closeWithin(t, p)
}

func TestCloseTwice(t *testing.T) {
	p := spawnHelper(t, nil)
	closeWithin(t, p)
	require.Nil(t, p.Client())
	closeWithin(t, p)
}

// TestCloseWithCallInFlight closes the Proc while a call is blocked in the child. The call must
// fail with the client's closed error and Close must not wait for the call.
func TestCloseWithCallInFlight(t *testing.T) {
	logs := newSyncLog(t)
	p := spawnHelper(t, logs)
	cl := p.Client()

	callErr := make(chan error, 1)
	go func() { callErr <- cl.Call(nil, "test_block", time.Hour) }()
	logs.waitFor(t, blockingMarker)
	closeWithin(t, p)

	select {
	case err := <-callErr:
		require.ErrorIs(t, err, rpc.ErrClientQuit)
	case <-time.After(waitBound):
		t.Fatal("in-flight call did not return after Close")
	}
}

// TestCallCancellation cancels the context of a call blocked in the child. The call must return
// the context's error, and the connection must stay usable.
func TestCallCancellation(t *testing.T) {
	logs := newSyncLog(t)
	p := spawnHelper(t, logs)
	cl := p.Client()

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	callErr := make(chan error, 1)
	go func() { callErr <- cl.CallContext(ctx, nil, "test_block", time.Hour) }()
	logs.waitFor(t, blockingMarker)
	cancel()

	select {
	case err := <-callErr:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(waitBound):
		t.Fatal("blocked call did not return after its context was cancelled")
	}
	var out string
	require.NoError(t, cl.CallContext(callCtx(t), &out, "test_echo", "still here"))
	require.Equal(t, "still here", out)
}

// TestSpawnFailureDrainsStderr reads the forwarded stderr right after a failed Spawn whose child was
// still writing when the ready timeout hit. Spawn (and Close, which it calls) must not return before
// the child's stderr is fully forwarded, or the read races the forwarder (run with -race).
func TestSpawnFailureDrainsStderr(t *testing.T) {
	defer ipc.SetReadyTimeout(2 * time.Second)()

	// Deliberately unsynchronised: Spawn must be done writing to it when it returns.
	var logs bytes.Buffer
	_, err := ipc.Spawn(os.Args[0], "--socket",
		[]string{helperArg, "--stderr-prelude", strconv.Itoa(math.MaxInt32), "--no-listen"}, &logs)
	require.ErrorContains(t, err, "never became ready")
	require.Contains(t, logs.String(), "] prelude 00\n")
}

// TestSpawnFailureWithStderrHeldByDescendant spawns a child that leaves behind a descendant holding
// its stderr open, then exits. Reaping the child must not wait for that descendant to close the
// pipe.
func TestSpawnFailureWithStderrHeldByDescendant(t *testing.T) {
	pidFile := filepath.Join(t.TempDir(), "descendant.pid")
	t.Cleanup(func() {
		raw, err := os.ReadFile(pidFile)
		if err != nil {
			return
		}
		if pid, err := strconv.Atoi(strings.TrimSpace(string(raw))); err == nil {
			if proc, err := os.FindProcess(pid); err == nil {
				_ = proc.Kill()
			}
		}
	})

	done := make(chan error, 1)
	go func() {
		p, err := ipc.Spawn(os.Args[0], "--socket",
			[]string{helperArg, "--orphan-stderr", pidFile, "--exit-early", "3"}, nil)
		p.Close()
		done <- err
	}()
	select {
	case err := <-done:
		require.ErrorContains(t, err, "exited before serving its socket (exit status 3)")
	case <-time.After(waitBound):
		t.Fatal("Spawn did not return while a descendant of the child held its stderr open")
	}
}
