# gas-oracle


Generating the bindings:

```bash
cat abis/OVM_GasPriceOracle.json \
    | abigen --pkg gaspriceoracle --abi - --out bindings/gaspriceoracle.go
```

Running the service:

```
$ go run main.go
```
