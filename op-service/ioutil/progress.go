package ioutil

import (
	"io"
	"sync"

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

func LogProgressor(msg string, lgr log.Logger) Progressor {
	return func(curr, total int64) {
		lgr.Info(msg, "current", curr, "total", total)
	}
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
