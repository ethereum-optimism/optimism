package main

import (
	"context"
	"os"

	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/opio"
	"github.com/ethereum/go-ethereum/log"
)

var (
	GitCommit = ""
	GitDate   = ""
)

func main() {
	// This is the most root context, used to propagate
	// cancellations to all spawned application-level goroutines
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		opio.BlockOnInterrupts()
		cancel()
	}()

	oplog.SetupDefaults()
	app := newCli(GitCommit, GitDate)
	if err := app.RunContext(ctx, os.Args); err != nil {
		log.Error("application failed", "err", err)
	}
}
