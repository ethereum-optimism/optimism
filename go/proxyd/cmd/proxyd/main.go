package main

import (
	"os"

	"github.com/BurntSushi/toml"
	"github.com/ethereum-optimism/optimism/go/proxyd"
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

	log.Info("starting proxyd", "version", GitVersion, "commit", GitCommit, "date", GitDate)

	if len(os.Args) < 2 {
		log.Crit("must specify a config file on the command line")
	}

	config := new(proxyd.Config)
	if _, err := toml.DecodeFile(os.Args[1], config); err != nil {
		log.Crit("error reading config file", "err", err)
	}

	if err := proxyd.Start(config); err != nil {
		log.Crit("error starting proxyd", "err", err)
	}
}
