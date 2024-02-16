package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/exp/slog"

	"github.com/ethereum/go-ethereum/log"

	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-ufm/pkg/config"
	"github.com/ethereum-optimism/optimism/op-ufm/pkg/service"
)

var (
	GitVersion = ""
	GitCommit  = ""
	GitDate    = ""
)

func main() {
	oplog.SetGlobalLogHandler(slog.NewJSONHandler(
		os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	log.Info("initializing",
		"version", GitVersion,
		"commit", GitCommit,
		"date", GitDate)

	if len(os.Args) < 2 {
		log.Crit("must specify a config file on the command line")
	}
	cfg := initConfig(os.Args[1])

	ctx := context.Background()
	svc := service.New(cfg)
	svc.Start(ctx)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	recvSig := <-sig
	log.Info("caught signal, shutting down",
		"signal", recvSig)

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
	logLevel, err := oplog.LevelFromString(cfg.LogLevel)
	if err != nil {
		logLevel = log.LevelInfo
		if cfg.LogLevel != "" {
			log.Warn("invalid server.log_level",
				"log_level", cfg.LogLevel)
		}
	}
	oplog.SetGlobalLogHandler(slog.NewJSONHandler(
		os.Stdout, &slog.HandlerOptions{Level: logLevel}))

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
