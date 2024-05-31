package main

import (
	"fmt"

	"github.com/urfave/cli/v2"

	plasma "github.com/ethereum-optimism/optimism/op-plasma"
	"github.com/ethereum-optimism/optimism/op-plasma/cmd/adapters"
	availService "github.com/ethereum-optimism/optimism/op-plasma/cmd/avail/service"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/opio"
)

func StartDAServer(cliCtx *cli.Context) error {
	if err := CheckRequired(cliCtx); err != nil {
		return err
	}

	cfg := ReadCLIConfig(cliCtx)

	if err := cfg.Check(); err != nil {
		return err
	}

	logCfg := oplog.ReadCLIConfig(cliCtx)

	l := oplog.NewLogger(oplog.AppOut(cliCtx), logCfg)
	oplog.SetGlobalLogHandler(l.Handler())

	l.Info("Initializing Plasma DA server...")

	availService := availService.NewAvailService(cfg.RPC, cfg.Seed, cfg.AppId, cfg.Timeout)

	server := plasma.NewDAServer(cliCtx.String(ListenAddrFlagName), cliCtx.Int(PortFlagName), adapters.DAServiceAdapter{DAService: availService}, l, true)

	if err := server.Start(); err != nil {
		return fmt.Errorf("failed to start the DA server")
	} else {
		l.Info("Started DA Server")
	}

	defer func() {
		if err := server.Stop(); err != nil {
			l.Error("failed to stop DA server", "err", err)
		}
	}()

	opio.BlockOnInterrupts()

	return nil
}
