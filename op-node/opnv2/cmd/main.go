package main

import (
	"context"
	"os"

	"github.com/ethereum/go-ethereum/log"

	service2 "github.com/ethereum-optimism/optimism/op-node/opnv2/service"
	"github.com/ethereum-optimism/optimism/op-node/version"
	"github.com/ethereum-optimism/optimism/op-service/ctxinterrupt"
)

var (
	GitCommit = ""
	GitDate   = ""
)

func main() {
	ctx := ctxinterrupt.WithSignalWaiterMain(context.Background())
	versionCfg := service2.VersionConfig{
		Version:   version.Version,
		GitCommit: GitCommit,
		GitDate:   GitDate,
	}
	err := service2.RunCmd(ctx, os.Args, versionCfg, service2.LifecycleFromConfig)
	if err != nil {
		log.Crit("Application failed", "message", err)
	}
}
