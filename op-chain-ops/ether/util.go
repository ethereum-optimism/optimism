package ether

import (
	"fmt"

	"github.com/ethereum/go-ethereum/log"
)

func wrapErr(err error, msg string, ctx ...any) error {
	return fmt.Errorf("%s: %w", fmt.Sprintf(msg, ctx...), err)
}

func ProgressLogger(n int, msg string) func(...any) {
	var i int

	return func(args ...any) {
		i++
		if i%n != 0 {
			return
		}
		log.Info(msg, append([]any{"count", i}, args...)...)
	}
}
