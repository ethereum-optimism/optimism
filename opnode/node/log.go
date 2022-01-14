package node

import (
	"fmt"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/log"
	"golang.org/x/term"
)

type LogLvl string

func (lvl LogLvl) String() string {
	return string(lvl)
}

func (lvl *LogLvl) Set(v string) error {
	v = strings.ToLower(v) // ignore case
	_, err := log.LvlFromString(v)
	if err != nil {
		return err
	}
	*lvl = LogLvl(v)
	return nil
}

func (lvl *LogLvl) Type() string {
	return "log-level"
}

func (lvl LogLvl) Lvl() log.Lvl {
	out, err := log.LvlFromString(string(lvl))
	if err != nil {
		panic("lvl.Set failed")
	}
	return out
}

type LogFormat string

func (lf LogFormat) String() string {
	return string(lf)
}

func (lf *LogFormat) Set(v string) error {
	switch v {
	case "json", "json-pretty", "terminal", "text":
		*lf = LogFormat(v)
		return nil
	default:
		return fmt.Errorf("unrecognized log format: %s", v)
	}
}

func (lf *LogFormat) Type() string {
	return "log-format"
}

func (lf LogFormat) Format(color bool) log.Format {
	switch lf {
	case "json":
		return log.JSONFormat()
	case "json-pretty":
		return log.JSONFormatEx(true, true)
	case "text", "terminal":
		return log.TerminalFormat(color)
	default:
		panic("lf.Set failed")
	}
}

type LogCmd struct {
	LogLvl LogLvl    `ask:"--level" help:"Log level: trace, debug, info, warn, error, crit. Capitals are accepted too."`
	Color  bool      `ask:"--color" help:"Color the log output. Defaults to true if terminal is detected."`
	Format LogFormat `ask:"--format" help:"Format the log output. Supported formats: 'text', 'json'"`
}

func (c *LogCmd) Default() {
	c.LogLvl = "info"
	c.Color = term.IsTerminal(int(os.Stdout.Fd()))
	c.Format = "text"
}

func (c *LogCmd) Create() log.Logger {
	handler := log.StreamHandler(os.Stdout, c.Format.Format(c.Color))
	handler = log.SyncHandler(handler)
	log.LvlFilterHandler(c.LogLvl.Lvl(), handler)
	logger := log.New()
	logger.SetHandler(handler)
	return logger
}
