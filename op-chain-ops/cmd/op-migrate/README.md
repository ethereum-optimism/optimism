# State migrator

This tool allows migrating the state of a Celo chain to a genesis block for a CeL2 chain.

## Usage

```sh

```

## Test Setup

Creating a local chain

```sh
build/bin/mycelo genesis --buildpath compiled-system-contracts --dev.accounts 2 --newenv tmp/testenv --mnemonic "miss fire behind decide egg buyer honey seven advance uniform profit renew"
build/bin/mycelo validator-init tmp/testenv/
build/bin/mycelo validator-run tmp/testenv/
```

Create some data

```sh
build/bin/mycelo load-bot tmp/testenv
```