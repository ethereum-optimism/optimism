package log

import (
	"fmt"
	"log/slog"
	"strings"
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
		return LevelTrace, nil
	case "debug", "dbug":
		return LevelDebug, nil
	case "info":
		return LevelInfo, nil
	case "warn":
		return LevelWarn, nil
	case "error", "eror":
		return LevelError, nil
	case "crit":
		return LevelCrit, nil
	default:
		return LevelDebug, fmt.Errorf("unknown level: %v", lvlString)
	}
}
