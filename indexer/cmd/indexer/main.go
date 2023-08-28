package main

import (
	"os"

	"github.com/ethereum-optimism/optimism/op-service/opio"
	"github.com/ethereum/go-ethereum/log"
	"golang.org/x/net/context"
)

var (
	GitCommit = ""
	GitDate   = ""
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	// Spinup a goroutine to catch interrupts and cancel the application context.
	go func() {
		opio.BlockOnInterrupts()
		log.Crit("caught interrupt, shutting down...")
		cancel()
	}()

	app := newCli(GitCommit, GitDate)
	if err := app.RunContext(ctx, os.Args); err != nil {
		log.Crit("application failed", "err", err)
	}
}
