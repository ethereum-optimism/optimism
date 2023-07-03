package main

import (
	"fmt"
	"os"

	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli/v2"

	endpointMonitor "github.com/ethereum-optimism/optimism/endpoint-monitor"
)

var (
	Version   = ""
	GitCommit = ""
	GitDate   = ""
)

func main() {
	oplog.SetupDefaults()

	app := cli.NewApp()
	app.Flags = endpointMonitor.CLIFlags("ENDPOINT_MONITOR")
	app.Version = fmt.Sprintf("%s-%s-%s", Version, GitCommit, GitDate)
	app.Name = "endpoint-monitor"
	app.Usage = "Endpoint Monitoring Service"
	app.Description = ""

	app.Action = endpointMonitor.Main(Version)
	err := app.Run(os.Args)
	if err != nil {
		log.Crit("Application failed", "message", err)
	}
}
