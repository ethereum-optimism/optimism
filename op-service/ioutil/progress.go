package ioutil

import (
	"io"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/schollz/progressbar/v3"
)

type Progressor func(curr, total int64)

func BarProgressor() Progressor {
	var bar *progressbar.ProgressBar
	var init sync.Once
	return func(curr, total int64) {
		init.Do(func() {
			bar = progressbar.DefaultBytes(total)
		})
		_ = bar.Set64(curr)
	}
}

func NoopProgressor() Progressor {
	return func(curr, total int64) {}
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
