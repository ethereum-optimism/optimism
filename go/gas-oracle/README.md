# gas-oracle

Generating the bindings:

```bash
cat abis/OVM_GasPriceOracle.json \
    | abigen --pkg gaspriceoracle \
    --abi - \
    --out bindings/gaspriceoracle.go \
    --type GasPriceOracle
```

Be sure to use `abigen` built with the same version of `go-ethereum` as what is
in the `go.mod` file.

Running the service:

```
EXPORT GAS_PRICE_ORACLE_KEY=0x..
$ go run main.go --ethereum-http-url https://kovan.optimism.io --transaction-gas-price 0
```
