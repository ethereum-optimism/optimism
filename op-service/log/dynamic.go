package log

import "github.com/ethereum/go-ethereum/log"

type LvlSetter interface {
	SetLogLevel(lvl log.Lvl)
}

// DynamicLogHandler allow runtime-configuration of the log handler.
type DynamicLogHandler struct {
	log.Handler // embedded, to expose any extra methods the underlying handler might provide
	maxLvl      log.Lvl
}

func NewDynamicLogHandler(lvl log.Lvl, h log.Handler) *DynamicLogHandler {
	return &DynamicLogHandler{
		Handler: h,
		maxLvl:  lvl,
	}
}

func (d *DynamicLogHandler) SetLogLevel(lvl log.Lvl) {
	d.maxLvl = lvl
}

func (d *DynamicLogHandler) Log(r *log.Record) error {
	if r.Lvl > d.maxLvl { // lower log level values are more critical
		return nil
	}
	return d.Handler.Log(r) // process the log
}
