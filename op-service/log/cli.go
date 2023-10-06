package log

import (
	"fmt"
	"io"
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
)

// CLIFlags creates flag definitions for the logging utils.
// Warning: flags are not safe to reuse due to an upstream urfave default-value mutation bug in GenericFlag.
// Use cliapp.ProtectFlags(flags) to create a copy before passing it into an App if the app runs more than once.
func CLIFlags(envPrefix string) []cli.Flag {
	return []cli.Flag{
		&cli.GenericFlag{
			Name:    LevelFlagName,
			Usage:   "The lowest log level that will be output",
			Value:   NewLvlFlagValue(log.LvlInfo),
			EnvVars: opservice.PrefixEnvVar(envPrefix, "LOG_LEVEL"),
		},
		&cli.GenericFlag{
			Name:    FormatFlagName,
			Usage:   "Format the log output. Supported formats: 'text', 'terminal', 'logfmt', 'json', 'json-pretty',",
			Value:   NewFormatFlagValue(FormatText),
			EnvVars: opservice.PrefixEnvVar(envPrefix, "LOG_FORMAT"),
		},
		&cli.BoolFlag{
			Name:    ColorFlagName,
			Usage:   "Color the log output if in terminal mode",
			EnvVars: opservice.PrefixEnvVar(envPrefix, "LOG_COLOR"),
		},
	}
}

// LvlFlagValue is a value type for cli.GenericFlag to parse and validate log-level values.
// Log level: trace, debug, info, warn, error, crit. Capitals are accepted too.
type LvlFlagValue log.Lvl

func NewLvlFlagValue(lvl log.Lvl) *LvlFlagValue {
	return (*LvlFlagValue)(&lvl)
}

func (fv *LvlFlagValue) Set(value string) error {
	value = strings.ToLower(value) // ignore case
	lvl, err := log.LvlFromString(value)
	if err != nil {
		return err
	}
	*fv = LvlFlagValue(lvl)
	return nil
}

func (fv LvlFlagValue) String() string {
	return log.Lvl(fv).String()
}

func (fv LvlFlagValue) LogLvl() log.Lvl {
	return log.Lvl(fv)
}

func (fv *LvlFlagValue) Clone() any {
	cpy := *fv
	return &cpy
}

var _ cliapp.CloneableGeneric = (*LvlFlagValue)(nil)

// FormatType defines a type of log format.
// Supported formats: 'text', 'terminal', 'logfmt', 'json', 'json-pretty'
type FormatType string

const (
	FormatText       FormatType = "text"
	FormatTerminal   FormatType = "terminal"
	FormatLogFmt     FormatType = "logfmt"
	FormatJSON       FormatType = "json"
	FormatJSONPretty FormatType = "json-pretty"
)

// Formatter turns a format type and color into a structured Format object
func (ft FormatType) Formatter(color bool) log.Format {
	switch ft {
	case FormatJSON:
		return log.JSONFormat()
	case FormatJSONPretty:
		return log.JSONFormatEx(true, true)
	case FormatText:
		if term.IsTerminal(int(os.Stdout.Fd())) {
			return log.TerminalFormat(color)
		} else {
			return log.LogfmtFormat()
		}
	case FormatTerminal:
		return log.TerminalFormat(color)
	case FormatLogFmt:
		return log.LogfmtFormat()
	default:
		panic(fmt.Errorf("failed to create `log.Format` for format-type=%q and color=%v", ft, color))
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
	case FormatText, FormatTerminal, FormatLogFmt, FormatJSON, FormatJSONPretty:
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
	Level  log.Lvl
	Color  bool
	Format FormatType
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
func NewLogHandler(wr io.Writer, cfg CLIConfig) log.Handler {
	handler := log.StreamHandler(wr, cfg.Format.Formatter(cfg.Color))
	handler = log.SyncHandler(handler)
	handler = NewDynamicLogHandler(cfg.Level, handler)
	return handler
}

// NewLogger creates a new configured logger.
// The log handler of the logger is a LvlSetter, i.e. the log level can be changed as needed.
func NewLogger(wr io.Writer, cfg CLIConfig) log.Logger {
	handler := NewLogHandler(wr, cfg)
	logger := log.New()
	logger.SetHandler(handler)
	return logger
}

// SetGlobalLogHandler sets the log handles as the handler of the global default logger.
// The usage of this logger is strongly discouraged,
// as it does makes it difficult to distinguish different services in the same process, e.g. during tests.
// Geth and other components may use the global logger however,
// and it is thus recommended to set the global log handler to catch these logs.
func SetGlobalLogHandler(h log.Handler) {
	log.Root().SetHandler(h)
}

// DefaultCLIConfig creates a default log configuration.
// Color defaults to true if terminal is detected.
func DefaultCLIConfig() CLIConfig {
	return CLIConfig{
		Level:  log.LvlInfo,
		Format: FormatText,
		Color:  term.IsTerminal(int(os.Stdout.Fd())),
	}
}

func ReadCLIConfig(ctx *cli.Context) CLIConfig {
	cfg := DefaultCLIConfig()
	cfg.Level = ctx.Generic(LevelFlagName).(*LvlFlagValue).LogLvl()
	cfg.Format = ctx.Generic(FormatFlagName).(*FormatFlagValue).FormatType()
	if ctx.IsSet(ColorFlagName) {
		cfg.Color = ctx.Bool(ColorFlagName)
	}
	return cfg
}
