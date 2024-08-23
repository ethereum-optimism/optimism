package mipsevm

import (
	"bytes"
	"strings"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
)

// LoggingWriter is a simple util to wrap a logger,
// and expose an io Writer interface,
// for the program running within the VM to write to.
// logs are line-buffered
type LoggingWriter struct {
	Log log.Logger
	buf strings.Builder
}

func logAsText(b string) bool {
	for _, c := range b {
		if (c < 0x20 || c >= 0x7F) && (c != '\n' && c != '\t') {
			return false
		}
	}
	return true
}

func (lw *LoggingWriter) Write(b []byte) (int, error) {
	const maxBufLen = 1000

	lw.buf.Write(b)
	flush := bytes.IndexByte(b, '\n') != -1 || lw.buf.Len() > maxBufLen
	if flush {
		lw.Flush()
	}
	return len(b), nil
}

func (lw *LoggingWriter) Flush() {
	if lw.buf.Len() == 0 {
		return
	}
	t := lw.buf.String()
	if t[len(t)-1] == '\n' {
		t = t[:len(t)-1]
	}
	lw.buf.Reset()
	if logAsText(t) {
		lw.Log.Info("", "text", t)
	} else {
		lw.Log.Info("", "data", hexutil.Bytes([]byte(t)))
	}
}
