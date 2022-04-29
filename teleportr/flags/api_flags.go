package flags

import (
	"fmt"
	"strings"

	"github.com/urfave/cli"
)

func prefixAPIEnvVar(name string) string {
	return fmt.Sprintf("TELEPORTR_API_%s", strings.ToUpper(name))
}

var (
	APIHostnameFlag = cli.StringFlag{
		Name:     "hostname",
		Usage:    "The hostname of the API server",
		Required: true,
		EnvVar:   prefixAPIEnvVar("HOSTNAME"),
	}
	APIPortFlag = cli.StringFlag{
		Name:     "port",
		Usage:    "The hostname of the API server",
		Required: true,
		EnvVar:   prefixAPIEnvVar("PORT"),
	}
)

var APIFlags = []cli.Flag{
	APIHostnameFlag,
	APIPortFlag,
	L1EthRpcFlag,
	DepositAddressFlag,
	NumDepositConfirmationsFlag,
	PostgresHostFlag,
	PostgresPortFlag,
	PostgresUserFlag,
	PostgresPasswordFlag,
	PostgresDBNameFlag,
	PostgresEnableSSLFlag,
}
