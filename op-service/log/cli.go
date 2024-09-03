package log

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/urfave/cli/v2"
	"golang.org/x/term"

	"github.com/ethereum/go-ethereum/log"

	opservice "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum-optimism/optimism/op-service/cliapp"
)

const (
	LevelFlagName  = "log.level"
	FormatFlagName = "log.format"
	ColorFlagName  = "log.color"
	PidFlagName    = "log.pid"
)

func CLIFlags(envPrefix string) []cli.Flag {
	return CLIFlagsWithCategory(envPrefix, "")
}

// CLIFlagsWithCategory creates flag definitions for the logging utils.
// Warning: flags are not safe to reuse due to an upstream urfave default-value mutation bug in GenericFlag.
// Use cliapp.ProtectFlags(flags) to create a copy before passing it into an App if the app runs more than once.
func CLIFlagsWithCategory(envPrefix string, category string) []cli.Flag {
	return []cli.Flag{
		&cli.GenericFlag{
			Name:     LevelFlagName,
			Usage:    "The lowest log level that will be output",
			Value:    NewLevelFlagValue(log.LevelInfo),
			EnvVars:  opservice.PrefixEnvVar(envPrefix, "LOG_LEVEL"),
			Category: category,
		},
		&cli.GenericFlag{
			Name:     FormatFlagName,
			Usage:    "Format the log output. Supported formats: 'text', 'terminal', 'logfmt', 'json', 'json-pretty',",
			Value:    NewFormatFlagValue(FormatText),
			EnvVars:  opservice.PrefixEnvVar(envPrefix, "LOG_FORMAT"),
			Category: category,
		},
		&cli.BoolFlag{
			Name:     ColorFlagName,
			Usage:    "Color the log output if in terminal mode",
			EnvVars:  opservice.PrefixEnvVar(envPrefix, "LOG_COLOR"),
			Category: category,
		},
		&cli.BoolFlag{
			Name:     PidFlagName,
			Usage:    "Show pid in the log",
			EnvVars:  opservice.PrefixEnvVar(envPrefix, "LOG_PID"),
			Category: category,
		},
	}
}

// LevelFlagValue is a value type for cli.GenericFlag to parse and validate log-level values.
// Log level: trace, debug, info, warn, error, crit. Capitals are accepted too.
type LevelFlagValue slog.Level

func NewLevelFlagValue(lvl slog.Level) *LevelFlagValue {
	return (*LevelFlagValue)(&lvl)
}

func (fv *LevelFlagValue) Set(value string) error {
	value = strings.ToLower(value) // ignore case
	lvl, err := LevelFromString(value)
	if err != nil {
		return err
	}
	*fv = LevelFlagValue(lvl)
	return nil
}

func (fv LevelFlagValue) String() string {
	return slog.Level(fv).String()
}

func (fv LevelFlagValue) Level() slog.Level {
	return slog.Level(fv).Level()
}

func (fv *LevelFlagValue) Clone() any {
	cpy := *fv
	return &cpy
}

// LevelFromString returns the appropriate Level from a string name.
// Useful for parsing command line args and configuration files.
// It also converts strings to lowercase.
// If the string is unknown, LevelDebug is returned as a default, together with
// a non-nil error.
func LevelFromString(lvlString string) (slog.Level, error) {
	lvlString = strings.ToLower(lvlString) // ignore case
	switch lvlString {
	case "trace", "trce":
		return log.LevelTrace, nil
	case "debug", "dbug":
		return log.LevelDebug, nil
	case "info":
		return log.LevelInfo, nil
	case "warn":
		return log.LevelWarn, nil
	case "error", "eror":
		return log.LevelError, nil
	case "crit":
		return log.LevelCrit, nil
	default:
		return log.LevelDebug, fmt.Errorf("unknown level: %v", lvlString)
	}
}

var _ cliapp.CloneableGeneric = (*LevelFlagValue)(nil)

// FormatType defines a type of log format.
// Supported formats: 'text', 'terminal', 'logfmt', 'json'
type FormatType string

const (
	FormatText     FormatType = "text"
	FormatTerminal FormatType = "terminal"
	FormatLogFmt   FormatType = "logfmt"
	FormatJSON     FormatType = "json"
)

// FormatHandler returns the correct slog handler factory for the provided format.
func FormatHandler(ft FormatType, color bool) func(io.Writer) slog.Handler {
	termColorHandler := func(w io.Writer) slog.Handler {
		return log.NewTerminalHandler(w, color)
	}
	logfmtHandler := func(w io.Writer) slog.Handler { return log.LogfmtHandlerWithLevel(w, log.LevelTrace) }
	switch ft {
	case FormatJSON:
		return log.JSONHandler
	case FormatText:
		if color {
			return termColorHandler
		} else {
			return logfmtHandler
		}
	case FormatTerminal:
		return termColorHandler
	case FormatLogFmt:
		return logfmtHandler
	default:
		panic(fmt.Errorf("failed to create slog.Handler factory for format-type=%q and color=%v", ft, color))
	}
}

func (ft FormatType) String() string {
	return string(ft)
}

// FormatFlagValue is a value type for cli.GenericFlag to parse and validate log-formatting-type values
type FormatFlagValue FormatType

func NewFormatFlagValue(fmtType FormatType) *FormatFlagValue {
	return (*FormatFlagValue)(&fmtType)
}

func (fv *FormatFlagValue) Set(value string) error {
	switch FormatType(value) {
	case FormatText, FormatTerminal, FormatLogFmt, FormatJSON:
		*fv = FormatFlagValue(value)
		return nil
	default:
		return fmt.Errorf("unrecognized log-format: %q", value)
	}
}

func (fv FormatFlagValue) String() string {
	return FormatType(fv).String()
}

func (fv FormatFlagValue) FormatType() FormatType {
	return FormatType(fv)
}

func (fv *FormatFlagValue) Clone() any {
	cpy := *fv
	return &cpy
}

var _ cliapp.CloneableGeneric = (*FormatFlagValue)(nil)

type CLIConfig struct {
	Level  slog.Level
	Color  bool
	Format FormatType
	Pid    bool
}

// AppOut returns an io.Writer to write app output to, like logs.
// This falls back to os.Stdout if the ctx, ctx.App or ctx.App.Writer are nil.
func AppOut(ctx *cli.Context) io.Writer {
	if ctx == nil || ctx.App == nil || ctx.App.Writer == nil {
		return os.Stdout
	}
	return ctx.App.Writer
}

// NewLogHandler creates a new configured handler, compatible as LvlSetter for log-level changes during runtime.
func NewLogHandler(wr io.Writer, cfg CLIConfig) slog.Handler {
	handler := FormatHandler(cfg.Format, cfg.Color)(wr)
	return NewDynamicLogHandler(cfg.Level, handler)
}

// NewLogger creates a new configured logger.
// The log handler of the logger is a LvlSetter, i.e. the log level can be changed as needed.
func NewLogger(wr io.Writer, cfg CLIConfig) log.Logger {
	h := NewLogHandler(wr, cfg)
	l := log.NewLogger(h)
	if cfg.Pid {
		l = l.With("pid", os.Getpid())
	}
	return l
}

// SetGlobalLogHandler sets the log handles as the handler of the global default logger.
// The usage of this logger is strongly discouraged,
// as it does makes it difficult to distinguish different services in the same process, e.g. during tests.
// Geth and other components may use the global logger however,
// and it is thus recommended to set the global log handler to catch these logs.
func SetGlobalLogHandler(h slog.Handler) {
	log.SetDefault(log.NewLogger(h))
}

// DefaultCLIConfig creates a default log configuration.
// Color defaults to true if terminal is detected.
func DefaultCLIConfig() CLIConfig {
	return CLIConfig{
		Level:  log.LevelInfo,
		Format: FormatText,
		Color:  term.IsTerminal(int(os.Stdout.Fd())),
	}
}

func ReadCLIConfig(ctx *cli.Context) CLIConfig {
	cfg := DefaultCLIConfig()
	cfg.Level = ctx.Generic(LevelFlagName).(*LevelFlagValue).Level()
	cfg.Format = ctx.Generic(FormatFlagName).(*FormatFlagValue).FormatType()
	if ctx.IsSet(ColorFlagName) {
		cfg.Color = ctx.Bool(ColorFlagName)
	}
	cfg.Pid = ctx.Bool(PidFlagName)
	return cfg
}
