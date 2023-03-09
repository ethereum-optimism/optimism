package log

import (
	"fmt"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli"
	"golang.org/x/term"

	opservice "github.com/ethereum-optimism/optimism/op-service"
)

const (
	LevelFlagName  = "log.level"
	FormatFlagName = "log.format"
	ColorFlagName  = "log.color"
)

func CLIFlags(envPrefix string) []cli.Flag {
	return []cli.Flag{
		cli.StringFlag{
			Name:   LevelFlagName,
			Usage:  "The lowest log level that will be output",
			Value:  "info",
			EnvVar: opservice.PrefixEnvVar(envPrefix, "LOG_LEVEL"),
		},
		cli.StringFlag{
			Name:   FormatFlagName,
			Usage:  "Format the log output. Supported formats: 'text', 'json'",
			Value:  "text",
			EnvVar: opservice.PrefixEnvVar(envPrefix, "LOG_FORMAT"),
		},
		cli.BoolFlag{
			Name:   ColorFlagName,
			Usage:  "Color the log output",
			EnvVar: opservice.PrefixEnvVar(envPrefix, "LOG_COLOR"),
		},
	}
}

type CLIConfig struct {
	Level  string // Log level: trace, debug, info, warn, error, crit. Capitals are accepted too.
	Color  bool   // Color the log output. Defaults to true if terminal is detected.
	Format string // Format the log output. Supported formats: 'text', 'json'
}

func (cfg CLIConfig) Check() error {
	switch cfg.Format {
	case "json", "json-pretty", "terminal", "text":
	default:
		return fmt.Errorf("unrecognized log format: %s", cfg.Format)
	}

	level := strings.ToLower(cfg.Level)
	_, err := log.LvlFromString(level)
	if err != nil {
		return fmt.Errorf("unrecognized log level: %w", err)
	}
	return nil
}

func NewLogger(cfg CLIConfig) log.Logger {
	handler := log.StreamHandler(os.Stdout, Format(cfg.Format, cfg.Color))
	handler = log.SyncHandler(handler)
	handler = log.LvlFilterHandler(Level(cfg.Level), handler)
	// Set the root handle to what we have configured. Some components like go-ethereum's RPC
	// server use log.Root() instead of being able to pass in a log.
	log.Root().SetHandler(handler)
	logger := log.New()
	logger.SetHandler(handler)
	return logger
}

func DefaultCLIConfig() CLIConfig {
	return CLIConfig{
		Level:  "info",
		Format: "text",
		Color:  term.IsTerminal(int(os.Stdout.Fd())),
	}
}

func ReadLocalCLIConfig(ctx *cli.Context) CLIConfig {
	cfg := DefaultCLIConfig()
	cfg.Level = ctx.String(LevelFlagName)
	cfg.Format = ctx.String(FormatFlagName)
	cfg.Color = ctx.Bool(ColorFlagName)
	return cfg
}

func ReadCLIConfig(ctx *cli.Context) CLIConfig {
	cfg := DefaultCLIConfig()
	cfg.Level = ctx.GlobalString(LevelFlagName)
	cfg.Format = ctx.GlobalString(FormatFlagName)
	cfg.Color = ctx.GlobalBool(ColorFlagName)
	return cfg
}

// Format turns a string and color into a structured Format object
func Format(lf string, color bool) log.Format {
	switch lf {
	case "json":
		return log.JSONFormat()
	case "json-pretty":
		return log.JSONFormatEx(true, true)
	case "text", "terminal":
		return log.TerminalFormat(color)
	default:
		panic("Failed to create `log.Format` from options")
	}
}

// Level parses the level string into an appropriate object
func Level(s string) log.Lvl {
	s = strings.ToLower(s) // ignore case
	l, err := log.LvlFromString(s)
	if err != nil {
		panic(fmt.Sprintf("Could not parse log level: %v", err))
	}
	return l
}
