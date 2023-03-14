package util

import (
	"github.com/ethereum/go-ethereum/log"
)

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
