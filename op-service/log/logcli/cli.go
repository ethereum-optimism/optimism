// Package logcli wires the logging package into urfave/cli applications: the log
// flags, the CLIConfig they parse into, the logger constructors that consume it,
// and the process-level logger setup a service performs at startup.
//
// It is separate from op-service/log so that op-service/log stays free of
// in-repo dependencies and can be imported by any package in the monorepo.
package logcli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/urfave/cli/v2"
	"golang.org/x/term"

	opservice "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum-optimism/optimism/op-service/cliapp"
	"github.com/ethereum-optimism/optimism/op-service/cliiface"
	"github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/log/logfilter"
)

const (
	LevelFlagName  = "log.level"
	FormatFlagName = "log.format"
	ColorFlagName  = "log.color"
	PidFlagName    = "log.pid"
)

// These flag configurations are used during testing, where level is set to trace.
var (
	flLevel  = flag.String(LevelFlagName, "trace", "Lowest log level that will be output")
	flFormat = flag.String(FormatFlagName, "text", "Log format: text|terminal|logfmt|logfmtms|json|jsonms|json-pretty")
	flColor  = flag.Bool(ColorFlagName, false, "Color the log output if in terminal mode: true|false")
	flPID    = flag.Bool(PidFlagName, false, "Show pid in the log")
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
			Usage:    fmt.Sprintf("Format the log output. Supported formats: %s", log.SupportedFormatsString()),
			Value:    NewFormatFlagValue(log.FormatText),
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
	lvl, err := log.LevelFromString(value)
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

var _ cliapp.CloneableGeneric = (*LevelFlagValue)(nil)

// FormatFlagValue is a value type for cli.GenericFlag to parse and validate log-formatting-type values
type FormatFlagValue log.FormatType

func NewFormatFlagValue(fmtType log.FormatType) *FormatFlagValue {
	return (*FormatFlagValue)(&fmtType)
}

func (fv *FormatFlagValue) Set(value string) error {
	switch log.FormatType(value) {
	case log.FormatText, log.FormatTerminal, log.FormatLogFmt, log.FormatLogFmtMs, log.FormatJSON, log.FormatJSONMs:
		*fv = FormatFlagValue(value)
		return nil
	default:
		return fmt.Errorf("unrecognized log-format: %q", value)
	}
}

func (fv FormatFlagValue) String() string {
	return log.FormatType(fv).String()
}

func (fv FormatFlagValue) FormatType() log.FormatType {
	return log.FormatType(fv)
}

func (fv *FormatFlagValue) Clone() any {
	cpy := *fv
	return &cpy
}

var _ cliapp.CloneableGeneric = (*FormatFlagValue)(nil)

type CLIConfig struct {
	Level  slog.Level
	Color  bool
	Format log.FormatType
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
	handler := log.FormatHandler(cfg.Format, cfg.Color)(wr)
	return log.NewDynamicLogHandler(cfg.Level, handler)
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
//
// This cannot move into op-service/log: it tags the global logger through
// logfilter, an in-repo package, and op-service/log must stay free of in-repo
// imports.
func SetGlobalLogHandler(h slog.Handler) {
	l := log.NewLogger(h)
	ctx := logfilter.AddLogAttrToContext(context.Background(), "global", true)
	l.SetContext(ctx)
	log.SetDefault(l)
}

// DefaultCLIConfig creates a default log configuration.
// Color defaults to true if terminal is detected.
func DefaultCLIConfig() CLIConfig {
	return CLIConfig{
		Level:  log.LevelInfo,
		Format: log.FormatText,
		Color:  term.IsTerminal(int(os.Stdout.Fd())),
	}
}

func ReadCLIConfig(ctx cliiface.Context) CLIConfig {
	cfg := DefaultCLIConfig()
	cfg.Level = ctx.Generic(LevelFlagName).(*LevelFlagValue).Level()
	cfg.Format = ctx.Generic(FormatFlagName).(*FormatFlagValue).FormatType()
	if ctx.IsSet(ColorFlagName) {
		cfg.Color = ctx.Bool(ColorFlagName)
	}
	cfg.Pid = ctx.Bool(PidFlagName)
	return cfg
}

// ReadTestCLIConfig reads the CLI config from flags and environment variables into a CLIConfig.
// flag.Parse() must be called before calling this function.
func ReadTestCLIConfig() CLIConfig {
	*flFormat = "logfmtms" // override the default cli format of text
	if v := os.Getenv("LOG_LEVEL"); v != "" {
		*flLevel = v
	}
	if v := os.Getenv("LOG_FORMAT"); v != "" {
		*flFormat = v
	}
	if v := os.Getenv("LOG_COLOR"); v != "" {
		*flColor = v == "true"
	}
	if v := os.Getenv("LOG_PID"); v != "" {
		*flPID = v == "true"
	}

	lvl, err := log.LevelFromString(*flLevel)
	if err != nil {
		panic(fmt.Errorf("failed to parse log level: %w", err))
	}

	return CLIConfig{
		Level:  lvl,
		Format: log.FormatType(*flFormat),
		Color:  term.IsTerminal(int(os.Stdout.Fd())) || *flColor,
		Pid:    *flPID,
	}
}
