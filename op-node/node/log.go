package node

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"

	"github.com/ethereum/go-ethereum/log"
)

type LogConfig struct {
	Level  string // Log level: trace, debug, info, warn, error, crit. Capitals are accepted too.
	Color  bool   // Color the log output. Defaults to true if terminal is detected.
	Format string // Format the log output. Supported formats: 'text', 'json'
}

func DefaultLogConfig() LogConfig {
	return LogConfig{
		Level:  "info",
		Format: "text",
		Color:  term.IsTerminal(int(os.Stdout.Fd())),
	}
}

// Check verifes that the LogConfig is valid. Calls to `NewLogger` may fail if this
// returns an error.
func (cfg *LogConfig) Check() error {
	switch cfg.Format {
	case "json", "json-pretty", "terminal", "text":
	default:
		return fmt.Errorf("unrecognized log format: %s", cfg.Format)
	}
	level := strings.ToLower(cfg.Level) // ignore case
	_, err := log.LvlFromString(level)
	if err != nil {
		return fmt.Errorf("unrecognized log level: %w", err)
	}
	return nil
}

// NewLogger creates a logger based on the supplied configuration
func (cfg *LogConfig) NewLogger() log.Logger {
	handler := log.StreamHandler(os.Stdout, format(cfg.Format, cfg.Color))
	handler = log.SyncHandler(handler)
	log.LvlFilterHandler(level(cfg.Level), handler)
	logger := log.New()
	logger.SetHandler(handler)
	return logger

}

// format turns a string and color into a structured Format object
func format(lf string, color bool) log.Format {
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

// level parses the level string into an appropriate object
func level(s string) log.Lvl {
	s = strings.ToLower(s) // ignore case
	l, err := log.LvlFromString(s)
	if err != nil {
		panic(fmt.Sprintf("Could not parse log level: %v", err))
	}
	return l
}
