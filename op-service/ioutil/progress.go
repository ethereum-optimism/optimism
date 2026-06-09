package ioutil

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"golang.org/x/term"
)

type Progressor func(curr, total int64)

func BarProgressor() Progressor {
	if !term.IsTerminal(int(os.Stderr.Fd())) {
		return NoopProgressor()
	}
	return (&terminalProgressor{
		out:      os.Stderr,
		interval: time.Second,
	}).Progressor
}

func NoopProgressor() Progressor {
	return func(curr, total int64) {}
}

type terminalProgressor struct {
	out      *os.File
	interval time.Duration

	lastLog time.Time
	done    bool
	mu      sync.Mutex
}

func (p *terminalProgressor) Progressor(curr, total int64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.done {
		return
	}

	now := time.Now()
	complete := total > 0 && curr >= total
	if !complete && !p.lastLog.IsZero() && now.Sub(p.lastLog) < p.interval {
		return
	}
	p.lastLog = now

	if total > 0 {
		percent := float64(curr) * 100 / float64(total)
		if percent > 100 {
			percent = 100
		}
		_, _ = fmt.Fprintf(p.out, "\rDownloading... %6.2f%% (%s/%s)", percent, formatBytes(curr), formatBytes(total))
	} else {
		_, _ = fmt.Fprintf(p.out, "\rDownloading... %s", formatBytes(curr))
	}

	if complete {
		_, _ = fmt.Fprintln(p.out)
		p.done = true
	}
}

func formatBytes(size int64) string {
	if size < 0 {
		size = 0
	}
	units := []string{"B", "KiB", "MiB", "GiB", "TiB"}
	value := float64(size)
	unit := 0
	for value >= 1024 && unit < len(units)-1 {
		value /= 1024
		unit++
	}
	if unit == 0 {
		return fmt.Sprintf("%d %s", size, units[unit])
	}
	return fmt.Sprintf("%.1f %s", value, units[unit])
}

type LogProgressor struct {
	L        log.Logger
	Msg      string
	Interval time.Duration

	lastLog time.Time
	mu      sync.Mutex
}

func NewLogProgressor(l log.Logger, msg string) *LogProgressor {
	return &LogProgressor{
		L:   l,
		Msg: msg,
	}
}

func (l *LogProgressor) Progressor(curr, total int64) {
	if !l.calcInterval() {
		return
	}

	msg := l.Msg
	if msg == "" {
		msg = "progress"
	}
	l.L.Info(msg, "current", curr, "total", total)
}

func (l *LogProgressor) calcInterval() bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	interval := l.Interval
	if interval == 0 {
		interval = time.Second
	}
	if time.Since(l.lastLog) < interval {
		return false
	}
	l.lastLog = time.Now()
	return true
}

type ProgressReader struct {
	R          io.Reader
	Progressor Progressor
	curr       int64
	Total      int64
}

func (pr *ProgressReader) Read(p []byte) (int, error) {
	n, err := pr.R.Read(p)
	pr.curr += int64(n)
	if pr.Progressor != nil {
		pr.Progressor(pr.curr, pr.Total)
	}
	return n, err
}
