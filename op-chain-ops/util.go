package state_surgery

import (
	"fmt"

	"github.com/ethereum/go-ethereum/log"
)

func wrapErr(err error, msg string, ctx ...any) error {
	return fmt.Errorf("%s: %w", fmt.Sprintf(msg, ctx...), err)
}

func ProgressLogger(n int, msg string) func() {
	var i int

	return func() {
		i++
		if i%n != 0 {
			return
		}
		log.Info(msg, "count", i)
	}
}
