package main

import (
	"context"
	"os"

	"github.com/BurntSushi/toml"
	"github.com/ethereum-optimism/optimism/op-service/opio"
	"github.com/ethereum-optimism/optimism/proxyd"
	"github.com/ethereum/go-ethereum/log"
)

var (
	GitVersion = ""
	GitCommit  = ""
	GitDate    = ""
)

func main() {
	// Set up logger with a default INFO level in case we fail to parse flags.
	// Otherwise the final critical log won't show what the parsing error was.
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

	log.Info("starting proxyd", "version", GitVersion, "commit", GitCommit, "date", GitDate)

	if len(os.Args) < 2 {
		log.Crit("must specify a config file on the command line")
	}

	config := new(proxyd.Config)
	if _, err := toml.DecodeFile(os.Args[1], config); err != nil {
		log.Crit("error reading config file", "err", err)
	}

	// update log level from config
	logLevel, err := log.LvlFromString(config.Server.LogLevel)
	if err != nil {
		logLevel = log.LvlInfo
		if config.Server.LogLevel != "" {
			log.Warn("invalid server.log_level set: " + config.Server.LogLevel)
		}
	}
	log.Root().SetHandler(
		log.LvlFilterHandler(
			logLevel,
			log.StreamHandler(os.Stdout, log.JSONFormat()),
		),
	)

	_, shutdown, err := proxyd.Start(config)
	if err != nil {
		log.Crit("error starting proxyd", "err", err)
	}

	select {
	case <-ctx.Done():
		log.Info("shutting down proxyd")
	}

	shutdown()
}
