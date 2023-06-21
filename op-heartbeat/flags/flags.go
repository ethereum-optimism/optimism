package flags

import (
	opservice "github.com/ethereum-optimism/optimism/op-service"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/urfave/cli/v2"
)

const envPrefix = "OP_HEARTBEAT"

func prefixEnvVars(name string) []string {
	return opservice.PrefixEnvVar(envPrefix, name)
}

const (
	HTTPAddrFlagName = "http.addr"
	HTTPPortFlagName = "http.port"
)

var (
	HTTPAddrFlag = &cli.StringFlag{
		Name:    HTTPAddrFlagName,
		Usage:   "Address the server should listen on",
		Value:   "0.0.0.0",
		EnvVars: prefixEnvVars("HTTP_ADDR"),
	}
	HTTPPortFlag = &cli.IntFlag{
		Name:    HTTPPortFlagName,
		Usage:   "Port the server should listen on",
		Value:   8080,
		EnvVars: prefixEnvVars("HTTP_PORT"),
	}
)

var Flags []cli.Flag

func init() {
	Flags = []cli.Flag{
		HTTPAddrFlag,
		HTTPPortFlag,
	}

	Flags = append(Flags, oplog.CLIFlags(envPrefix)...)
	Flags = append(Flags, opmetrics.CLIFlags(envPrefix)...)
}
