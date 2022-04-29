# gas-oracle

This service is responsible for sending transactions to the Sequencer to update
the L2 gas price over time. It consists of a set of functions found in the
`gasprices` package that define the parameters of how the gas prices are updated
and then the `oracle` package is responsible for observing the Sequencer over
time and send transactions that actually do update the gas prices.

### Generating the Bindings

Note: this only needs to happen if the ABI of the `OVM_GasPriceOracle` is
updated.

This project uses `abigen` to automatically create smart contract bindings in
Go. To generate the bindings, be sure that the latest ABI and bytecode are
committed into the repository in the `abis` directory.

Use the following command to generate the bindings:

```bash
$ make binding
```

Be sure to use `abigen` built with the same version of `go-ethereum` as what is
in the `go.mod` file.

### Building the service

The service can be built with the `Makefile`. A binary will be produced
called the `gas-oracle`.

```bash
$ make gas-oracle
```

### Running the service

Use the `--help` flag when running the `gas-oracle` to see it's configuration
options.

```
NAME:
   gas-oracle - Remotely Control the Optimism Gas Price

USAGE:
   gas-oracle [global options] command [command options] [arguments...]

VERSION:
   0.0.0-1.10.4-stable

DESCRIPTION:
   Configure with a private key and an Optimism HTTP endpoint to send transactions that update the L2 gas price.

COMMANDS:
     help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --ethereum-http-url value                  Sequencer HTTP Endpoint (default: "http://127.0.0.1:8545") [$GAS_PRICE_ORACLE_ETHEREUM_HTTP_URL]
   --chain-id value                           L2 Chain ID (default: 0) [$GAS_PRICE_ORACLE_CHAIN_ID]
   --gas-price-oracle-address value           Address of OVM_GasPriceOracle (default: "0x420000000000000000000000000000000000000F") [$GAS_PRICE_ORACLE_GAS_PRICE_ORACLE_ADDRESS]
   --private-key value                        Private Key corresponding to OVM_GasPriceOracle Owner [$GAS_PRICE_ORACLE_PRIVATE_KEY]
   --transaction-gas-price value              Hardcoded tx.gasPrice, not setting it uses gas estimation (default: 0) [$GAS_PRICE_ORACLE_TRANSACTION_GAS_PRICE]
   --loglevel value                           log level to emit to the screen (default: 3) [$GAS_PRICE_ORACLE_LOG_LEVEL]
   --floor-price value                        gas price floor (default: 1) [$GAS_PRICE_ORACLE_FLOOR_PRICE]
   --target-gas-per-second value              target gas per second (default: 11000000) [$GAS_PRICE_ORACLE_TARGET_GAS_PER_SECOND]
   --max-percent-change-per-epoch value       max percent change of gas price per second (default: 0.1) [$GAS_PRICE_ORACLE_MAX_PERCENT_CHANGE_PER_EPOCH]
   --average-block-gas-limit-per-epoch value  average block gas limit per epoch (default: 1.1e+07) [$GAS_PRICE_ORACLE_AVERAGE_BLOCK_GAS_LIMIT_PER_EPOCH]
   --epoch-length-seconds value               length of epochs in seconds (default: 10) [$GAS_PRICE_ORACLE_EPOCH_LENGTH_SECONDS]
   --significant-factor value                 only update when the gas price changes by more than this factor (default: 0.05) [$GAS_PRICE_ORACLE_SIGNIFICANT_FACTOR]
   --wait-for-receipt                         wait for receipts when sending transactions [$GAS_PRICE_ORACLE_WAIT_FOR_RECEIPT]
   --metrics                                  Enable metrics collection and reporting [$GAS_PRICE_ORACLE_METRICS_ENABLE]
   --metrics.addr value                       Enable stand-alone metrics HTTP server listening interface (default: "127.0.0.1") [$GAS_PRICE_ORACLE_METRICS_HTTP]
   --metrics.port value                       Metrics HTTP server listening port (default: 6060) [$GAS_PRICE_ORACLE_METRICS_PORT]
   --metrics.influxdb                         Enable metrics export/push to an external InfluxDB database [$GAS_PRICE_ORACLE_METRICS_ENABLE_INFLUX_DB]
   --metrics.influxdb.endpoint value          InfluxDB API endpoint to report metrics to (default: "http://localhost:8086") [$GAS_PRICE_ORACLE_METRICS_INFLUX_DB_ENDPOINT]
   --metrics.influxdb.database value          InfluxDB database name to push reported metrics to (default: "gas-oracle") [$GAS_PRICE_ORACLE_METRICS_INFLUX_DB_DATABASE]
   --metrics.influxdb.username value          Username to authorize access to the database (default: "test") [$GAS_PRICE_ORACLE_METRICS_INFLUX_DB_USERNAME]
   --metrics.influxdb.password value          Password to authorize access to the database (default: "test") [$GAS_PRICE_ORACLE_METRICS_INFLUX_DB_PASSWORD]
   --help, -h                                 show help
   --version, -v                              print the version
```

### Testing the service

The service can be tested with the `Makefile`

```
$ make test
```
