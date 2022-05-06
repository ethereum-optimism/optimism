package main

import (
	"fmt"
	"os"
	"time"

	"github.com/ethereum-optimism/optimism/gas-oracle/flags"
	ometrics "github.com/ethereum-optimism/optimism/gas-oracle/metrics"
	"github.com/ethereum-optimism/optimism/gas-oracle/oracle"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics/influxdb"
	"github.com/ethereum/go-ethereum/params"
	"github.com/urfave/cli"
)

var (
	GitVersion = ""
	GitCommit  = ""
	GitDate    = ""
)

func main() {
	app := cli.NewApp()
	app.Flags = flags.Flags

	app.Version = GitVersion + "-" + params.VersionWithCommit(GitCommit, GitDate)
	app.Name = "gas-oracle"
	app.Usage = "Remotely Control the Optimism Gas Price"
	app.Description = "Configure with a private key and an Optimism HTTP endpoint " +
		"to send transactions that update the L2 gas price."

	// Configure the logging
	app.Before = func(ctx *cli.Context) error {
		loglevel := ctx.GlobalUint64(flags.LogLevelFlag.Name)
		log.Root().SetHandler(log.LvlFilterHandler(log.Lvl(loglevel), log.StreamHandler(os.Stdout, log.TerminalFormat(true))))
		return nil
	}

	// Define the functionality of the application
	app.Action = func(ctx *cli.Context) error {
		if args := ctx.Args(); len(args) > 0 {
			return fmt.Errorf("invalid command: %q", args[0])
		}

		config := oracle.NewConfig(ctx)
		gpo, err := oracle.NewGasPriceOracle(config)
		if err != nil {
			return err
		}

		if err := gpo.Start(); err != nil {
			return err
		}

		if config.MetricsEnabled {
			address := fmt.Sprintf("%s:%d", config.MetricsHTTP, config.MetricsPort)
			log.Info("Enabling stand-alone metrics HTTP endpoint", "address", address)
			ometrics.Setup(address)
		}

		if config.MetricsEnableInfluxDB {
			endpoint := config.MetricsInfluxDBEndpoint
			database := config.MetricsInfluxDBDatabase
			username := config.MetricsInfluxDBUsername
			password := config.MetricsInfluxDBPassword
			log.Info("Enabling metrics export to InfluxDB", "endpoint", endpoint, "username", username, "database", database)
			go influxdb.InfluxDBWithTags(ometrics.DefaultRegistry, 10*time.Second, endpoint, database, username, password, "geth.", make(map[string]string))
		}

		gpo.Wait()

		return nil
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Crit("application failed", "message", err)
	}
}
