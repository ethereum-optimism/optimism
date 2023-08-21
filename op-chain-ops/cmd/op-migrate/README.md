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


## Current error

```
1 % make && ./test.sh
go build -o ./bin/op-migrate ./cmd/op-migrate/main.go
+ echo 'Starting Migration'
Starting Migration
+ ./bin/op-migrate --l1-rpc-url=http://127.0.0.1:8546 --db-path=/Users/paul/Projects/celo-blockchain/tmp/testenv/validator-00/ --rollup-config-out=rollup.json --dry-run
INFO [08-02|15:42:24.994] L1 ChainID                               chainId=9,266,000
INFO [08-02|15:42:24.994] Using L1 Starting Block Tag              tag=1
CRIT [08-02|15:42:24.996] error in migration                       err="cannot fetch L1 starting block tag: missing required field 'sha3Uncles' for Header"
```


## Tasks
- [ ] Load DB with op geth, check some state
- [ ] Make a nice testing script