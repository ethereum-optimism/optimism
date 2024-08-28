package main

import (
	"os"
	"runtime"
)

func main() {
	if len(os.Args) != 1 {
		panic("expected 1 arg")
	}
	if os.Args[0] != "op-program" {
		panic("unexpected arg0")
	}

	var memProfileRate bool
	env := os.Environ()
	for _, env := range env {
		if env != "GODEBUG=memprofilerate=0" {
			panic("invalid envar")
		}
		memProfileRate = true
	}
	if !memProfileRate {
		panic("memProfileRate env is not set")
	}
	if runtime.MemProfileRate != 0 {
		panic("runtime.MemProfileRate is non-zero")
	}
}
