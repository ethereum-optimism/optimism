package log_test

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"

	oplog "github.com/ethereum-optimism/optimism/op-service/log"
)

// methodCase is one logging method, called through a Logger or its go-ethereum
// view. own and geth log one record and return the function the record should
// be attributed to, which is themselves, or "" for the go-ethereum view's Write,
// which attributes to the caller of the function calling it.
type methodCase struct {
	name     string
	level    slog.Level
	explicit bool // whether the method takes the context explicitly
	own      func(l oplog.Logger, ctx context.Context) string
	geth     func(l log.Logger) string
}

func methodCases() []methodCase {
	return []methodCase{
		{name: "Trace", level: oplog.LevelTrace,
			own:  func(l oplog.Logger, _ context.Context) string { l.Trace("m"); return thisFunction() },
			geth: func(l log.Logger) string { l.Trace("m"); return thisFunction() }},
		{name: "Debug", level: oplog.LevelDebug,
			own:  func(l oplog.Logger, _ context.Context) string { l.Debug("m"); return thisFunction() },
			geth: func(l log.Logger) string { l.Debug("m"); return thisFunction() }},
		{name: "Info", level: oplog.LevelInfo,
			own:  func(l oplog.Logger, _ context.Context) string { l.Info("m"); return thisFunction() },
			geth: func(l log.Logger) string { l.Info("m"); return thisFunction() }},
		{name: "Warn", level: oplog.LevelWarn,
			own:  func(l oplog.Logger, _ context.Context) string { l.Warn("m"); return thisFunction() },
			geth: func(l log.Logger) string { l.Warn("m"); return thisFunction() }},
		{name: "Error", level: oplog.LevelError,
			own:  func(l oplog.Logger, _ context.Context) string { l.Error("m"); return thisFunction() },
			geth: func(l log.Logger) string { l.Error("m"); return thisFunction() }},
		{name: "Log", level: oplog.LevelWarn,
			own:  func(l oplog.Logger, _ context.Context) string { l.Log(oplog.LevelWarn, "m"); return thisFunction() },
			geth: func(l log.Logger) string { l.Log(oplog.LevelWarn, "m"); return thisFunction() }},
		{name: "Write", level: oplog.LevelError,
			geth: func(l log.Logger) string { l.Write(oplog.LevelError, "m"); return "" }},
		{name: "LogAttrs", level: oplog.LevelWarn, explicit: true,
			own: func(l oplog.Logger, ctx context.Context) string {
				l.LogAttrs(ctx, oplog.LevelWarn, "m", slog.Int("a", 1))
				return thisFunction()
			}},
		{name: "TraceContext", level: oplog.LevelTrace, explicit: true,
			own: func(l oplog.Logger, ctx context.Context) string { l.TraceContext(ctx, "m"); return thisFunction() }},
		{name: "DebugContext", level: oplog.LevelDebug, explicit: true,
			own: func(l oplog.Logger, ctx context.Context) string { l.DebugContext(ctx, "m"); return thisFunction() }},
		{name: "InfoContext", level: oplog.LevelInfo, explicit: true,
			own: func(l oplog.Logger, ctx context.Context) string { l.InfoContext(ctx, "m"); return thisFunction() }},
		{name: "WarnContext", level: oplog.LevelWarn, explicit: true,
			own: func(l oplog.Logger, ctx context.Context) string { l.WarnContext(ctx, "m"); return thisFunction() }},
		{name: "ErrorContext", level: oplog.LevelError, explicit: true,
			own: func(l oplog.Logger, ctx context.Context) string { l.ErrorContext(ctx, "m"); return thisFunction() }},
	}
}

// TestEveryMethodContract checks, for every logging method on a Logger, a
// derived Logger, and the go-ethereum views of both, the level of the record,
// the context passed to the handler's Enabled and Handle, and the record's
// source line.
func TestEveryMethodContract(t *testing.T) {
	type view struct {
		own  func(oplog.Logger) oplog.Logger
		geth func(oplog.Logger) log.Logger
	}
	views := map[string]view{
		"logger":       {own: func(l oplog.Logger) oplog.Logger { return l }},
		"with":         {own: func(l oplog.Logger) oplog.Logger { return l.With("k", "v") }},
		"geth":         {geth: func(l oplog.Logger) log.Logger { return oplog.ToGeth(l) }},
		"geth-of-with": {geth: func(l oplog.Logger) log.Logger { return oplog.ToGeth(l.With("k", "v")).New("k2", "v2") }},
	}
	defaultCtx, explicitCtx := withValue("default"), withValue("explicit")
	for viewName, v := range views {
		for _, c := range methodCases() {
			if (v.geth != nil && c.geth == nil) || (v.own != nil && c.own == nil) {
				continue // the go-ethereum view needs only upstream's method set
			}
			t.Run(viewName+"/"+c.name, func(t *testing.T) {
				h := newRecordingHandler()
				base := oplog.NewLogger(h).WithContext(defaultCtx)

				var wantFn string
				if v.geth != nil {
					wantFn = c.geth(v.geth(base))
				} else {
					wantFn = c.own(v.own(base), explicitCtx)
				}
				if wantFn == "" {
					wantFn = thisFunction()
				}

				records := h.all()
				require.Len(t, records, 1)
				r := records[0]
				require.Equal(t, c.level, r.Level)
				wantCtx := "default"
				if c.explicit {
					wantCtx = "explicit"
				}
				require.Equal(t, wantCtx, ctxValue(h.contexts()[0]), "context passed to Handle")
				require.Equal(t, wantCtx, ctxValue(h.enabledContexts()[0]), "context passed to Enabled")
				requireAttributedTo(t, records, wantFn)
			})
		}
	}

	t.Run("global", func(t *testing.T) {
		saveGlobals(t)
		h := newRecordingHandler()
		oplog.SetDefault(oplog.NewLogger(h).WithContext(defaultCtx))

		oplog.Info("package-level")
		oplog.New("k", "v").Info("derived from root")
		log.Info("geth package-level")

		require.Len(t, h.all(), 3)
		requireAttributedTo(t, h.all(), thisFunction())
		for _, ctx := range h.contexts() {
			require.Equal(t, "default", ctxValue(ctx))
		}
	})
}

func logViaGethWrite(g log.Logger) {
	g.Write(oplog.LevelInfo, "write")
	// WriteCtx is only part of op-geth's log.Logger.
	g.(interface {
		WriteCtx(ctx context.Context, level slog.Level, msg string, args ...any)
	}).WriteCtx(withValue("explicit"), oplog.LevelInfo, "write-ctx")
}

// TestToGethWriteSourceAttribution checks that Write and WriteCtx on the
// go-ethereum view attribute their record to the caller of the function that
// calls them, which is what go-ethereum's package-level functions rely on.
func TestToGethWriteSourceAttribution(t *testing.T) {
	h := newRecordingHandler()
	logViaGethWrite(oplog.ToGeth(oplog.NewLogger(h).With("k", "v").WithContext(withValue("default"))))
	require.Len(t, h.all(), 2)
	requireAttributedTo(t, h.all(), thisFunction())
	require.Equal(t, "default", ctxValue(h.contexts()[0]))
	require.Equal(t, "explicit", ctxValue(h.contexts()[1]))
}

// TestToGethWithContext checks that the go-ethereum view of a logger with a
// default context, and loggers derived from the view, log with that context,
// while the view of the original logger does not.
func TestToGethWithContext(t *testing.T) {
	h := newRecordingHandler()
	l := oplog.NewLogger(h)
	g := oplog.ToGeth(l.WithContext(withValue("A")))
	g.Info("view")
	g.With("k", "v").New("k2", "v2").Info("derived from view")
	oplog.ToGeth(l).Info("view of original")

	var got []any
	for _, ctx := range h.contexts() {
		got = append(got, ctxValue(ctx))
	}
	require.Equal(t, []any{"A", "A", nil}, got)
}

// TestToGethSharesAttributes checks that the go-ethereum view writes to the same
// handler with the same attributes.
func TestToGethSharesAttributes(t *testing.T) {
	var buf bytes.Buffer
	l := oplog.NewLogger(oplog.LogfmtHandlerWithLevel(&buf, oplog.LevelTrace)).With("a", 1)
	oplog.ToGeth(l).With("b", 2).Info("via geth", "c", 3)
	require.Contains(t, buf.String(), "msg=\"via geth\" a=1 b=2 c=3")
}

// TestSetDefaultSetsGethRoot checks that SetDefault also installs the logger as
// go-ethereum's global logger, and as the log/slog default.
func TestSetDefaultSetsGethRoot(t *testing.T) {
	saveGlobals(t)
	h := newRecordingHandler()
	oplog.SetDefault(oplog.NewLogger(h).WithContext(withValue("global")))

	log.Info("geth package-level")
	log.Root().With("k", "v").Warn("geth root")
	slog.Info("slog default")

	records := h.all()
	require.Len(t, records, 3)
	requireAttributedTo(t, records, thisFunction())
	for _, ctx := range h.contexts()[:2] {
		require.Equal(t, "global", ctxValue(ctx))
	}
}

// otherLogger is a Logger implementation other than the one NewLogger returns.
type otherLogger struct {
	oplog.Logger
	infos *[]string
}

func (o otherLogger) Info(msg string, args ...any) { *o.infos = append(*o.infos, msg) }

func (o otherLogger) With(args ...any) oplog.Logger {
	return otherLogger{Logger: o.Logger.With(args...), infos: o.infos}
}

// TestToGethForwardsOtherLoggers checks that converting any other Logger
// implementation forwards to it, also after With.
func TestToGethForwardsOtherLoggers(t *testing.T) {
	infos := new([]string)
	g := oplog.ToGeth(otherLogger{Logger: oplog.NewLogger(oplog.DiscardHandler()), infos: infos})
	g.Info("direct")
	g.With("k", "v").Info("derived")
	require.Equal(t, []string{"direct", "derived"}, *infos)
}

// TestOutputMatchesGeth checks that the owned logger, and its go-ethereum view,
// write the same output as go-ethereum's logger for the methods they share.
// Arguments are plain key/value pairs: upstream go-ethereum pads an odd-length
// argument list, which a list mixing pairs and slog.Attr values can be, so
// TestAttributes covers those.
func TestOutputMatchesGeth(t *testing.T) {
	emit := func(l interface {
		Trace(string, ...any)
		Debug(string, ...any)
		Info(string, ...any)
		Warn(string, ...any)
		Error(string, ...any)
		Log(slog.Level, string, ...any)
	}) {
		l.Trace("trace", "a", 1)
		l.Debug("debug", "b", "two", "c", 3)
		l.Info("info", "err", nil, "bytes", []byte{1, 2})
		l.Warn("warn")
		l.Error("error", "x", 1.5)
		l.Log(oplog.LevelCrit, "crit via Log", "k", "v")
	}
	handlers := map[string]func(*bytes.Buffer, slog.Level) slog.Handler{
		"terminal": func(b *bytes.Buffer, lvl slog.Level) slog.Handler {
			return oplog.NewTerminalHandlerWithLevel(b, lvl, false)
		},
		"logfmt": func(b *bytes.Buffer, lvl slog.Level) slog.Handler { return oplog.LogfmtHandlerWithLevel(b, lvl) },
		"json":   func(b *bytes.Buffer, lvl slog.Level) slog.Handler { return oplog.JSONHandlerWithLevel(b, lvl) },
	}
	for name, mk := range handlers {
		for _, lvl := range []slog.Level{oplog.LevelTrace, oplog.LevelDebug} {
			t.Run(name+"/"+oplog.LevelString(lvl), func(t *testing.T) {
				var gethBuf, ownBuf, viewBuf bytes.Buffer
				emit(log.NewLogger(mk(&gethBuf, lvl)).With("w", "x"))
				emit(oplog.NewLogger(mk(&ownBuf, lvl)).With("w", "x"))
				emit(oplog.ToGeth(oplog.NewLogger(mk(&viewBuf, lvl))).With("w", "x"))
				want := stripTimes(gethBuf.String())
				require.Equal(t, want, stripTimes(ownBuf.String()), "owned logger")
				require.Equal(t, want, stripTimes(viewBuf.String()), "go-ethereum view")
				require.Equal(t, lvl == oplog.LevelTrace, strings.Contains(ownBuf.String(), "trace"))
			})
		}
	}
}

// stripTimes drops the timestamp from each line of terminal, logfmt or JSON
// output, which is the only part expected to differ between two loggers.
func stripTimes(s string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		switch {
		case strings.HasPrefix(line, "{"):
			if j := strings.Index(line, `"t":"`); j >= 0 {
				k := strings.Index(line[j+5:], `"`)
				line = line[:j] + line[j+5+k+1:]
			}
		case strings.HasPrefix(line, "t="):
			line = line[strings.Index(line, " "):]
		default:
			// Terminal output: "INFO [01-02|15:04:05.000] msg ...".
			if j, k := strings.Index(line, "["), strings.Index(line, "]"); j >= 0 && k > j {
				line = line[:j+1] + line[k:]
			}
		}
		lines[i] = line
	}
	return strings.Join(lines, "\n")
}
