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
	APIL1EthRpcFlag = cli.StringFlag{
		Name:     "l1-eth-rpc",
		Usage:    "The endpoint for the L1 ETH provider",
		Required: true,
		EnvVar:   prefixAPIEnvVar("L1_ETH_RPC"),
	}
	APIDepositAddressFlag = cli.StringFlag{
		Name:     "deposit-address",
		Usage:    "Address of the TeleportrDeposit contract",
		Required: true,
		EnvVar:   prefixAPIEnvVar("DEPOSIT_ADDRESS"),
	}
	APINumConfirmationsFlag = cli.StringFlag{
		Name: "num-confirmations",
		Usage: "Number of confirmations required until deposits are " +
			"considered confirmed",
		Required: true,
		EnvVar:   prefixAPIEnvVar("NUM_CONFIRMATIONS"),
	}
	APIPostgresHostFlag = cli.StringFlag{
		Name:     "postgres-host",
		Usage:    "Host of the teleportr postgres instance",
		Required: true,
		EnvVar:   prefixAPIEnvVar("POSTGRES_HOST"),
	}
	APIPostgresPortFlag = cli.Uint64Flag{
		Name:     "postgres-port",
		Usage:    "Port of the teleportr postgres instance",
		Required: true,
		EnvVar:   prefixAPIEnvVar("POSTGRES_PORT"),
	}
	APIPostgresUserFlag = cli.StringFlag{
		Name:     "postgres-user",
		Usage:    "Username of the teleportr postgres instance",
		Required: true,
		EnvVar:   prefixAPIEnvVar("POSTGRES_USER"),
	}
	APIPostgresPasswordFlag = cli.StringFlag{
		Name:     "postgres-password",
		Usage:    "Password of the teleportr postgres instance",
		Required: true,
		EnvVar:   prefixAPIEnvVar("POSTGRES_PASSWORD"),
	}
	APIPostgresDBNameFlag = cli.StringFlag{
		Name:     "postgres-db-name",
		Usage:    "Database name of the teleportr postgres instance",
		Required: true,
		EnvVar:   prefixAPIEnvVar("POSTGRES_DB_NAME"),
	}
	APIPostgresEnableSSLFlag = cli.BoolFlag{
		Name: "postgres-enable-ssl",
		Usage: "Whether or not to enable SSL on connections to " +
			"teleportr postgres instance",
		Required: true,
		EnvVar:   prefixAPIEnvVar("POSTGRES_ENABLE_SSL"),
	}
)

var APIFlags = []cli.Flag{
	APIHostnameFlag,
	APIPortFlag,
	APIL1EthRpcFlag,
	APIDepositAddressFlag,
	APINumConfirmationsFlag,
	APIPostgresHostFlag,
	APIPostgresPortFlag,
	APIPostgresUserFlag,
	APIPostgresPasswordFlag,
	APIPostgresDBNameFlag,
	APIPostgresEnableSSLFlag,
}
