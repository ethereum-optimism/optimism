package main

import (
	"context"
	"os"

	"github.com/ethereum/go-ethereum/log"

	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/opio"
)

func main() {
	oplog.SetupDefaults()

	cli := newCli()
	ctx := opio.WithInterruptBlocker(context.Background())
	if err := cli.RunContext(ctx, os.Args); err != nil {
		log.Error("application failed", "err", err)
		os.Exit(1)
	}
}
