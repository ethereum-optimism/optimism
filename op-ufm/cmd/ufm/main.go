package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/ethereum-optimism/optimism/op-service/opio"
	"github.com/ethereum-optimism/optimism/op-ufm/pkg/config"
	"github.com/ethereum-optimism/optimism/op-ufm/pkg/service"

	"github.com/ethereum/go-ethereum/log"
)

var (
	GitVersion = ""
	GitCommit  = ""
	GitDate    = ""
)

func main() {
	log.Root().SetHandler(
		log.LvlFilterHandler(
			log.LvlInfo,
			log.StreamHandler(os.Stdout, log.JSONFormat()),
		),
	)

	// Invoke cancel when an interrupt is received.
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		opio.BlockOnInterrupts()
		cancel()
	}()

	log.Info("initializing",
		"version", GitVersion,
		"commit", GitCommit,
		"date", GitDate)

	if len(os.Args) < 2 {
		log.Crit("must specify a config file on the command line")
	}
	cfg := initConfig(os.Args[1])

	svc := service.New(cfg)
	svc.Start(ctx)

	select {
	case <-ctx.Done():
		log.Info("shutting down op-ufm")
	}

	svc.Shutdown()
}

func initConfig(cfgFile string) *config.Config {
	cfg, err := config.New(cfgFile)
	if err != nil {
		log.Crit("error reading config file",
			"file", cfgFile,
			"err", err)
	}

	// update log level from config
	logLevel, err := log.LvlFromString(cfg.LogLevel)
	if err != nil {
		logLevel = log.LvlInfo
		if cfg.LogLevel != "" {
			log.Warn("invalid server.log_level",
				"log_level", cfg.LogLevel)
		}
	}
	log.Root().SetHandler(
		log.LvlFilterHandler(
			logLevel,
			log.StreamHandler(os.Stdout, log.JSONFormat()),
		),
	)

	// readable parsed config
	jsonCfg, _ := json.MarshalIndent(cfg, "", "  ")
	fmt.Printf("%s", string(jsonCfg))

	err = cfg.Validate()
	if err != nil {
		log.Crit("invalid config",
			"err", err)
	}

	return cfg
}
