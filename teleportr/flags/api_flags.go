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
	DisburserWalletAddressFlag = cli.StringFlag{
		Name:     "disburser-wallet-address",
		Usage:    "The address of the disburser wallet",
		Required: true,
		EnvVar:   prefixAPIEnvVar("DISBURSER_WALLET_ADDRESS"),
	}
)

var APIFlags = []cli.Flag{
	APIHostnameFlag,
	APIPortFlag,
	DisburserWalletAddressFlag,
	DisburserAddressFlag,
	L1EthRpcFlag,
	L2EthRpcFlag,
	DepositAddressFlag,
	NumDepositConfirmationsFlag,
	PostgresHostFlag,
	PostgresPortFlag,
	PostgresUserFlag,
	PostgresPasswordFlag,
	PostgresDBNameFlag,
	PostgresEnableSSLFlag,
	MetricsServerEnableFlag,
	MetricsHostnameFlag,
	MetricsPortFlag,
	HTTP2DisableFlag,
}
