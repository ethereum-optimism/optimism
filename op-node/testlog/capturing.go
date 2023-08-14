package testlog

import (
	"github.com/ethereum/go-ethereum/log"
)

// CapturingHandler provides a log handler that captures all log records and optionally forwards them to a delegate.
// Note that it is not thread safe.
type CapturingHandler struct {
	Delegate log.Handler
	Logs     []*log.Record
}

func Capture(l log.Logger) *CapturingHandler {
	handler := &CapturingHandler{
		Delegate: l.GetHandler(),
	}
	l.SetHandler(handler)
	return handler
}

func (c *CapturingHandler) Log(r *log.Record) error {
	c.Logs = append(c.Logs, r)
	if c.Delegate != nil {
		return c.Delegate.Log(r)
	}
	return nil
}

func (c *CapturingHandler) Clear() {
	c.Logs = nil
}

func (c *CapturingHandler) FindLog(lvl log.Lvl, msg string) *HelperRecord {
	for _, record := range c.Logs {
		if record.Lvl == lvl && record.Msg == msg {
			return &HelperRecord{record}
		}
	}
	return nil
}

type HelperRecord struct {
	*log.Record
}

func (h *HelperRecord) GetContextValue(name string) any {
	for i := 0; i < len(h.Ctx); i += 2 {
		if h.Ctx[i] == name {
			return h.Ctx[i+1]
		}
	}
	return nil
}

var _ log.Handler = (*CapturingHandler)(nil)
