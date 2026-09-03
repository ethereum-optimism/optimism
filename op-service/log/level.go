package log

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/ethereum/go-ethereum/log"
)

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
