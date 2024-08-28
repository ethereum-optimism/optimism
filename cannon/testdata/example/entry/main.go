package main

import (
	"os"
	"runtime"
	"strings"
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
		toks := strings.Split(env, "=")
		if len(toks) != 2 {
			panic("invalid envar")
		}
		if toks[0] == "memprofilerate" && toks[1] == "0" {
			memProfileRate = true
		}
	}
	if !memProfileRate {
		panic("memProfileRate env is not set")
	}
	if runtime.MemProfileRate != 0 {
		panic("runtime.MemProfileRate is non-zero")
	}
}
