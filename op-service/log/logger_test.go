package log_test

import (
	"bytes"
	"context"
	"errors"
	stdlog "log"
	"log/slog"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	oplog "github.com/ethereum-optimism/optimism/op-service/log"
)

// recordingHandler keeps every record it handles together with the context it
// was handled with, and the contexts Enabled was asked about. Derived handlers
// share the recording.
type recordingHandler struct {
	mu          *sync.Mutex
	records     *[]slog.Record
	ctxs        *[]context.Context
	enabledCtxs *[]context.Context
}

func newRecordingHandler() *recordingHandler {
	return &recordingHandler{
		mu:          new(sync.Mutex),
		records:     new([]slog.Record),
		ctxs:        new([]context.Context),
		enabledCtxs: new([]context.Context),
	}
}

func (h *recordingHandler) Enabled(ctx context.Context, level slog.Level) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	*h.enabledCtxs = append(*h.enabledCtxs, ctx)
	return true
}

func (h *recordingHandler) Handle(ctx context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	*h.records = append(*h.records, r)
	*h.ctxs = append(*h.ctxs, ctx)
	return nil
}

func (h *recordingHandler) WithAttrs([]slog.Attr) slog.Handler { return h }

func (h *recordingHandler) WithGroup(string) slog.Handler { return h }

func (h *recordingHandler) all() []slog.Record {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]slog.Record(nil), *h.records...)
}

func (h *recordingHandler) contexts() []context.Context {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]context.Context(nil), *h.ctxs...)
}

func (h *recordingHandler) enabledContexts() []context.Context {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]context.Context(nil), *h.enabledCtxs...)
}

// requireAttributedTo asserts that every record's source is the function fn.
func requireAttributedTo(t *testing.T, records []slog.Record, fn string) {
	t.Helper()
	for _, r := range records {
		frame, _ := runtime.CallersFrames([]uintptr{r.PC}).Next()
		require.Equal(t, fn, frame.Function,
			"%q was attributed to %s (%s:%d), not to its call site", r.Message, frame.Function, frame.File, frame.Line)
	}
}

// thisFunction returns the name of its caller.
func thisFunction() string {
	pc, _, _, _ := runtime.Caller(1)
	return runtime.FuncForPC(pc).Name()
}

type ctxKey struct{}

func withValue(v string) context.Context {
	return context.WithValue(context.Background(), ctxKey{}, v)
}

func ctxValue(ctx context.Context) any { return ctx.Value(ctxKey{}) }

// saveGlobals restores, on cleanup, every global logger a test may replace:
// the package's default logger (and with it go-ethereum's), the log/slog
// default, and the standard library logger's output and flags, which
// slog.SetDefault redirects.
func saveGlobals(t *testing.T) {
	root, slogDefault := oplog.Root(), slog.Default()
	writer, flags := stdlog.Writer(), stdlog.Flags()
	t.Cleanup(func() {
		oplog.SetDefault(root)
		slog.SetDefault(slogDefault)
		stdlog.SetOutput(writer)
		stdlog.SetFlags(flags)
	})
}

// logEveryMethod logs one record through each logging method of l.
func logEveryMethod(l oplog.Logger) int {
	ctx := context.Background()
	l.Trace("trace")
	l.Debug("debug")
	l.Info("info")
	l.Warn("warn")
	l.Error("error")
	l.Log(oplog.LevelInfo, "log")
	l.LogAttrs(ctx, oplog.LevelInfo, "log-attrs", slog.Int("a", 1))
	l.TraceContext(ctx, "trace-ctx")
	l.DebugContext(ctx, "debug-ctx")
	l.InfoContext(ctx, "info-ctx")
	l.WarnContext(ctx, "warn-ctx")
	l.ErrorContext(ctx, "error-ctx")
	return 12
}

// TestMethodSourceAttribution checks that every logging method attributes its
// record to the method's caller, also on loggers derived with With and New.
func TestMethodSourceAttribution(t *testing.T) {
	for name, derive := range map[string]func(oplog.Logger) oplog.Logger{
		"root": func(l oplog.Logger) oplog.Logger { return l },
		"with": func(l oplog.Logger) oplog.Logger { return l.With("k", "v") },
		"new":  func(l oplog.Logger) oplog.Logger { return l.New("k", "v").With("k2", "v2") },
	} {
		t.Run(name, func(t *testing.T) {
			h := newRecordingHandler()
			n := logEveryMethod(derive(oplog.NewLogger(h)))
			require.Len(t, h.all(), n)
			requireAttributedTo(t, h.all(), "github.com/ethereum-optimism/optimism/op-service/log_test.logEveryMethod")
		})
	}
}

// TestPackageLevelSourceAttribution checks that the package-level log functions
// attribute their record to their caller.
func TestPackageLevelSourceAttribution(t *testing.T) {
	saveGlobals(t)
	h := newRecordingHandler()
	oplog.SetDefault(oplog.NewLogger(h))

	oplog.Info("info")
	oplog.Warn("warn")
	oplog.Error("error")
	oplog.Debug("debug")
	oplog.Trace("trace")
	oplog.New("k", "v").Info("new")

	require.Len(t, h.all(), 6)
	requireAttributedTo(t, h.all(), thisFunction())
}

// TestWithContext checks that WithContext sets the context of records logged
// without an explicit one, for both Enabled and Handle, that it leaves the
// receiver unchanged, that loggers derived from it keep the context, and that
// an explicit context takes precedence.
func TestWithContext(t *testing.T) {
	h := newRecordingHandler()
	l := oplog.NewLogger(h)
	ctxLogger := l.WithContext(withValue("default"))

	l.Info("parent")
	ctxLogger.Info("derived")
	ctxLogger.With("k", "v").New("k2", "v2").Info("derived further")
	ctxLogger.WithContext(withValue("replaced")).Info("replaced")
	ctxLogger.Warn("unchanged by derivation")
	ctxLogger.InfoContext(withValue("explicit"), "explicit")
	ctxLogger.LogAttrs(withValue("explicit-attrs"), oplog.LevelInfo, "explicit attrs")
	l.Info("parent still")

	want := []any{nil, "default", "default", "replaced", "default", "explicit", "explicit-attrs", nil}
	values := func(ctxs []context.Context) []any {
		var out []any
		for _, ctx := range ctxs {
			out = append(out, ctxValue(ctx))
		}
		return out
	}
	require.Equal(t, want, values(h.contexts()), "contexts passed to Handle")
	require.Equal(t, want, values(h.enabledContexts()), "contexts passed to Enabled")
}

// TestNilContext checks that a nil context reaches the handler as
// context.Background(), as it does with log/slog.
func TestNilContext(t *testing.T) {
	h := newRecordingHandler()
	l := oplog.NewLogger(h)
	//nolint:staticcheck // SA1012: a nil context is what this test is about.
	l.InfoContext(nil, "explicit nil")
	//nolint:staticcheck // SA1012: a nil context is what this test is about.
	l.LogAttrs(nil, oplog.LevelInfo, "explicit nil attrs")
	//nolint:staticcheck // SA1012: a nil context is what this test is about.
	l.WithContext(nil).Info("nil default")
	for _, ctx := range append(h.contexts(), h.enabledContexts()...) {
		require.Equal(t, context.Background(), ctx)
	}
	require.Len(t, h.contexts(), 3)
}

// TestLevelFiltering checks that a record is only built and handled when the
// handler is enabled for its level.
func TestLevelFiltering(t *testing.T) {
	var buf bytes.Buffer
	l := oplog.NewLogger(oplog.LogfmtHandlerWithLevel(&buf, oplog.LevelWarn))
	l.Debug("dropped")
	l.Info("dropped")
	l.LogAttrs(context.Background(), oplog.LevelInfo, "dropped")
	l.Warn("kept")
	l.ErrorContext(context.Background(), "kept")
	require.Equal(t, 2, strings.Count(buf.String(), "kept"))
	require.NotContains(t, buf.String(), "dropped")
	require.False(t, l.Enabled(context.Background(), oplog.LevelInfo))
	require.True(t, l.Enabled(context.Background(), oplog.LevelWarn))
}

// TestAttributes checks that attributes from With, key/value pairs, slog.Attr
// arguments and LogAttrs all reach the handler, and that an odd trailing
// argument is kept under slog's !BADKEY key, also on the go-ethereum view.
func TestAttributes(t *testing.T) {
	var buf bytes.Buffer
	base := oplog.NewLogger(oplog.LogfmtHandlerWithLevel(&buf, oplog.LevelTrace))
	l := base.With("a", 1).New("b", 2)
	l.Info("kv", "c", 3, slog.String("d", "x"), "e", true)
	l.LogAttrs(context.Background(), oplog.LevelInfo, "attrs", slog.Int("f", 4))
	l.Info("odd", "lonely")
	base.With("lonely").Info("odd-with")
	oplog.ToGeth(base).Info("odd-geth", "lonely")

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	require.Len(t, lines, 5)
	require.Contains(t, lines[0], "msg=kv a=1 b=2 c=3 d=x e=true")
	require.Contains(t, lines[1], "msg=attrs a=1 b=2 f=4")
	require.Contains(t, lines[2], "msg=odd a=1 b=2 !BADKEY=lonely")
	require.Contains(t, lines[3], "msg=odd-with !BADKEY=lonely")
	require.Contains(t, lines[4], "msg=odd-geth !BADKEY=lonely")
}

// TestHandler checks that Handler returns the handler the logger was built with.
func TestHandler(t *testing.T) {
	h := newRecordingHandler()
	require.Same(t, h, oplog.NewLogger(h).Handler())
}

// TestEnabledForwardsContext checks that Enabled asks the handler about the
// given context, not the default one, also through the go-ethereum view.
func TestEnabledForwardsContext(t *testing.T) {
	h := newRecordingHandler()
	l := oplog.NewLogger(h).WithContext(withValue("default"))
	l.Enabled(withValue("explicit"), oplog.LevelInfo)
	oplog.ToGeth(l).Enabled(withValue("explicit-geth"), oplog.LevelInfo)
	var got []any
	for _, ctx := range h.enabledContexts() {
		got = append(got, ctxValue(ctx))
	}
	require.Equal(t, []any{"explicit", "explicit-geth"}, got)
}

const critEnv = "OPLOG_TEST_CRIT"

// TestCritExits checks that Crit logs its message at LevelCrit and exits with
// status 1, for the method, the package-level function and the go-ethereum
// view.
func TestCritExits(t *testing.T) {
	switch os.Getenv(critEnv) {
	case "method":
		oplog.NewLogger(oplog.NewTerminalHandler(os.Stderr, false)).Crit("crit-method")
		return
	case "package":
		oplog.SetDefault(oplog.NewLogger(oplog.NewTerminalHandler(os.Stderr, false)))
		oplog.Crit("crit-package")
		return
	case "geth":
		oplog.ToGeth(oplog.NewLogger(oplog.NewTerminalHandler(os.Stderr, false))).Crit("crit-geth")
		return
	}
	for _, mode := range []string{"method", "package", "geth"} {
		t.Run(mode, func(t *testing.T) {
			cmd := exec.Command(os.Args[0], "-test.run=^TestCritExits$")
			cmd.Env = append(os.Environ(), critEnv+"="+mode)
			out, err := cmd.CombinedOutput()
			var exitErr *exec.ExitError
			require.True(t, errors.As(err, &exitErr), "want exit error, got %v; output:\n%s", err, out)
			require.Equal(t, 1, exitErr.ExitCode())
			require.Contains(t, string(out), "CRIT")
			require.Contains(t, string(out), "crit-"+mode)
		})
	}
}
