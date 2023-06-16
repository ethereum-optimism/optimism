package cmd

import (
	"io"

	"github.com/ethereum/go-ethereum/log"
)

func Logger(w io.Writer, lvl log.Lvl) log.Logger {
	h := log.StreamHandler(w, log.LogfmtFormat())
	h = log.SyncHandler(h)
	h = log.LvlFilterHandler(lvl, h)
	l := log.New()
	l.SetHandler(h)
	return l
}
