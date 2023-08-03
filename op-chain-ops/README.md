# op-chain-ops

This package contains utilities for working with chain state.

## check-l2

The `check-l2` binary is used for verifying that an OP Stack L2
has been configured correctly. It iterates over all 2048 predeployed
proxies to make sure they are configured correctly with the correct
proxy admin address. After that, it checks that all [predeploys](../op-bindings/predeploys/addresses.go)
are configured and aliased correctly. Additional contract-specific
checks ensure configuration like ownership, version, and storage
is set correctly for the predeploys.

#### Usage

It can be built and run using the [Makefile](./Makefile) `check-l2` target.
Run `make check-l2` to create a binary in [./bin/check-l2](./bin/check-l2)
that can be executed by providing the `--l1-rpc-url` and `--l2-rpc-url` flags.

```sh
./bin/check-l2 \
  --l2-rpc-url http://localhost:9545 \
  --l1-rpc-url http://localhost:8545
```
