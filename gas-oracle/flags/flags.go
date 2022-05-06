package flags

import (
	"github.com/urfave/cli"
)

var (
	EthereumHttpUrlFlag = cli.StringFlag{
		Name:   "ethereum-http-url",
		Value:  "http://127.0.0.1:8545",
		Usage:  "L1 HTTP Endpoint",
		EnvVar: "GAS_PRICE_ORACLE_ETHEREUM_HTTP_URL",
	}
	LayerTwoHttpUrlFlag = cli.StringFlag{
		Name:   "layer-two-http-url",
		Value:  "http://127.0.0.1:9545",
		Usage:  "Sequencer HTTP Endpoint",
		EnvVar: "GAS_PRICE_ORACLE_LAYER_TWO_HTTP_URL",
	}
	L1ChainIDFlag = cli.Uint64Flag{
		Name:   "l1-chain-id",
		Usage:  "L1 Chain ID",
		EnvVar: "GAS_PRICE_ORACLE_L1_CHAIN_ID",
	}
	L2ChainIDFlag = cli.Uint64Flag{
		Name:   "l2-chain-id",
		Usage:  "L2 Chain ID",
		EnvVar: "GAS_PRICE_ORACLE_L2_CHAIN_ID",
	}
	GasPriceOracleAddressFlag = cli.StringFlag{
		Name:   "gas-price-oracle-address",
		Usage:  "Address of OVM_GasPriceOracle",
		Value:  "0x420000000000000000000000000000000000000F",
		EnvVar: "GAS_PRICE_ORACLE_GAS_PRICE_ORACLE_ADDRESS",
	}
	PrivateKeyFlag = cli.StringFlag{
		Name:   "private-key",
		Usage:  "Private Key corresponding to OVM_GasPriceOracle Owner",
		EnvVar: "GAS_PRICE_ORACLE_PRIVATE_KEY",
	}
	TransactionGasPriceFlag = cli.Uint64Flag{
		Name:   "transaction-gas-price",
		Usage:  "Hardcoded tx.gasPrice, not setting it uses gas estimation",
		EnvVar: "GAS_PRICE_ORACLE_TRANSACTION_GAS_PRICE",
	}
	EnableL1BaseFeeFlag = cli.BoolFlag{
		Name:   "enable-l1-base-fee",
		Usage:  "Enable updating the L1 base fee",
		EnvVar: "GAS_PRICE_ORACLE_ENABLE_L1_BASE_FEE",
	}
	EnableL2GasPriceFlag = cli.BoolFlag{
		Name:   "enable-l2-gas-price",
		Usage:  "Enable updating the L2 gas price",
		EnvVar: "GAS_PRICE_ORACLE_ENABLE_L2_GAS_PRICE",
	}
	LogLevelFlag = cli.IntFlag{
		Name:   "loglevel",
		Value:  3,
		Usage:  "log level to emit to the screen",
		EnvVar: "GAS_PRICE_ORACLE_LOG_LEVEL",
	}
	FloorPriceFlag = cli.Uint64Flag{
		Name:   "floor-price",
		Value:  1,
		Usage:  "gas price floor",
		EnvVar: "GAS_PRICE_ORACLE_FLOOR_PRICE",
	}
	TargetGasPerSecondFlag = cli.Uint64Flag{
		Name:   "target-gas-per-second",
		Value:  11_000_000,
		Usage:  "target gas per second",
		EnvVar: "GAS_PRICE_ORACLE_TARGET_GAS_PER_SECOND",
	}
	MaxPercentChangePerEpochFlag = cli.Float64Flag{
		Name:   "max-percent-change-per-epoch",
		Value:  0.1,
		Usage:  "max percent change of gas price per second",
		EnvVar: "GAS_PRICE_ORACLE_MAX_PERCENT_CHANGE_PER_EPOCH",
	}
	AverageBlockGasLimitPerEpochFlag = cli.Uint64Flag{
		Name:   "average-block-gas-limit-per-epoch",
		Value:  11_000_000,
		Usage:  "average block gas limit per epoch",
		EnvVar: "GAS_PRICE_ORACLE_AVERAGE_BLOCK_GAS_LIMIT_PER_EPOCH",
	}
	EpochLengthSecondsFlag = cli.Uint64Flag{
		Name:   "epoch-length-seconds",
		Value:  10,
		Usage:  "length of epochs in seconds",
		EnvVar: "GAS_PRICE_ORACLE_EPOCH_LENGTH_SECONDS",
	}
	L1BaseFeeEpochLengthSecondsFlag = cli.Uint64Flag{
		Name:   "l1-base-fee-epoch-length-seconds",
		Value:  15,
		Usage:  "polling time for updating the L1 base fee",
		EnvVar: "GAS_PRICE_ORACLE_L1_BASE_FEE_EPOCH_LENGTH_SECONDS",
	}
	L1BaseFeeSignificanceFactorFlag = cli.Float64Flag{
		Name:   "l1-base-fee-significant-factor",
		Value:  0.10,
		Usage:  "only update when the L1 base fee changes by more than this factor",
		EnvVar: "GAS_PRICE_ORACLE_L1_BASE_FEE_SIGNIFICANT_FACTOR",
	}
	L2GasPriceSignificanceFactorFlag = cli.Float64Flag{
		Name:   "significant-factor",
		Value:  0.05,
		Usage:  "only update when the gas price changes by more than this factor",
		EnvVar: "GAS_PRICE_ORACLE_SIGNIFICANT_FACTOR",
	}
	WaitForReceiptFlag = cli.BoolFlag{
		Name:   "wait-for-receipt",
		Usage:  "wait for receipts when sending transactions",
		EnvVar: "GAS_PRICE_ORACLE_WAIT_FOR_RECEIPT",
	}
	MetricsEnabledFlag = cli.BoolFlag{
		Name:   "metrics",
		Usage:  "Enable metrics collection and reporting",
		EnvVar: "GAS_PRICE_ORACLE_METRICS_ENABLE",
	}
	MetricsHTTPFlag = cli.StringFlag{
		Name:   "metrics.addr",
		Usage:  "Enable stand-alone metrics HTTP server listening interface",
		Value:  "127.0.0.1",
		EnvVar: "GAS_PRICE_ORACLE_METRICS_HTTP",
	}
	MetricsPortFlag = cli.IntFlag{
		Name:   "metrics.port",
		Usage:  "Metrics HTTP server listening port",
		Value:  6060,
		EnvVar: "GAS_PRICE_ORACLE_METRICS_PORT",
	}
	MetricsEnableInfluxDBFlag = cli.BoolFlag{
		Name:   "metrics.influxdb",
		Usage:  "Enable metrics export/push to an external InfluxDB database",
		EnvVar: "GAS_PRICE_ORACLE_METRICS_ENABLE_INFLUX_DB",
	}
	MetricsInfluxDBEndpointFlag = cli.StringFlag{
		Name:   "metrics.influxdb.endpoint",
		Usage:  "InfluxDB API endpoint to report metrics to",
		Value:  "http://localhost:8086",
		EnvVar: "GAS_PRICE_ORACLE_METRICS_INFLUX_DB_ENDPOINT",
	}
	MetricsInfluxDBDatabaseFlag = cli.StringFlag{
		Name:   "metrics.influxdb.database",
		Usage:  "InfluxDB database name to push reported metrics to",
		Value:  "gas-oracle",
		EnvVar: "GAS_PRICE_ORACLE_METRICS_INFLUX_DB_DATABASE",
	}
	MetricsInfluxDBUsernameFlag = cli.StringFlag{
		Name:   "metrics.influxdb.username",
		Usage:  "Username to authorize access to the database",
		Value:  "test",
		EnvVar: "GAS_PRICE_ORACLE_METRICS_INFLUX_DB_USERNAME",
	}
	MetricsInfluxDBPasswordFlag = cli.StringFlag{
		Name:   "metrics.influxdb.password",
		Usage:  "Password to authorize access to the database",
		Value:  "test",
		EnvVar: "GAS_PRICE_ORACLE_METRICS_INFLUX_DB_PASSWORD",
	}
)

var Flags = []cli.Flag{
	EthereumHttpUrlFlag,
	LayerTwoHttpUrlFlag,
	L1ChainIDFlag,
	L2ChainIDFlag,
	L1BaseFeeSignificanceFactorFlag,
	GasPriceOracleAddressFlag,
	PrivateKeyFlag,
	TransactionGasPriceFlag,
	LogLevelFlag,
	FloorPriceFlag,
	TargetGasPerSecondFlag,
	MaxPercentChangePerEpochFlag,
	AverageBlockGasLimitPerEpochFlag,
	EpochLengthSecondsFlag,
	L1BaseFeeEpochLengthSecondsFlag,
	L2GasPriceSignificanceFactorFlag,
	WaitForReceiptFlag,
	EnableL1BaseFeeFlag,
	EnableL2GasPriceFlag,
	MetricsEnabledFlag,
	MetricsHTTPFlag,
	MetricsPortFlag,
	MetricsEnableInfluxDBFlag,
	MetricsInfluxDBEndpointFlag,
	MetricsInfluxDBDatabaseFlag,
	MetricsInfluxDBUsernameFlag,
	MetricsInfluxDBPasswordFlag,
}
