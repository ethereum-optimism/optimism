package flags

import "github.com/urfave/cli"

// TODO: this should be sequencer http url
var EthereumHttpUrlFlag = cli.StringFlag{
	Name:   "ethereum-http-url",
	Value:  "http://127.0.0.1:8545",
	Usage:  "HTTP Endpoint to ",
	EnvVar: "GAS_PRICE_ORACLE_ETHEREUM_HTTP_URL",
}

var ChainIDFlag = cli.Uint64Flag{
	Name:   "chain-id",
	Usage:  "Chain id",
	EnvVar: "GAS_PRICE_ORACLE_CHAIN_ID",
}

var GasPriceOracleAddressFlag = cli.StringFlag{
	Name:   "gas-price-oracle-address",
	Usage:  "",
	Value:  "0x420000000000000000000000000000000000000F",
	EnvVar: "GAS_PRICE_ORACLE_GAS_PRICE_ORACLE_ADDRESS",
}

var PrivateKeyFlag = cli.StringFlag{
	Name:   "private-key",
	Usage:  "",
	Value:  "0x",
	EnvVar: "GAS_PRICE_ORACLE_PRIVATE_KEY",
}

var TransactionGasPriceFlag = cli.Uint64Flag{
	Name:   "transaction-gas-price",
	Usage:  "",
	EnvVar: "GAS_PRICE_ORACLE_TRANSACTION_GAS_PRICE",
}

var LogLevelFlag = cli.IntFlag{
	Name:  "loglevel",
	Value: 3,
	Usage: "log level to emit to the screen",
}

var Flags = []cli.Flag{
	EthereumHttpUrlFlag,
	ChainIDFlag,
	GasPriceOracleAddressFlag,
	PrivateKeyFlag,
	TransactionGasPriceFlag,
	LogLevelFlag,
}
