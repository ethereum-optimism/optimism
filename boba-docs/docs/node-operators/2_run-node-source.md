# Running a Node from Source

DDocker images make it very simple to run a BOBA node, but you can also create your own node using the source code. You might choose to do this if you need the node to work on a  specific architecture or if you want to look closely at the node's code. This guide will show you how to build and run a node from scratch.

## Software Dependencies

| Dependency                                                   | Version  | Version Check Command |
| ------------------------------------------------------------ | -------- | --------------------- |
| [git](https://git-scm.com)                                   | `^2`     | `git --version`       |
| [go](https://go.dev)                                         | `^1.21`  | `go --version`        |
| [node](https://nodejs.org/en/)                               | `^20`    | `node --version`      |
| [pnpm](https://pnpm.io/installation)                         | `^8`     | `pnpm --version`      |
| [foundry](https://github.com/foundry-rs/foundry#installation) | `^0.2.0` | `forge --version`     |
| [make](https://linux.die.net/man/1/make)                     | `^4`     | `make --version`      |

## Build the Rollup Node

### Clone the Boba Monorepo

```bash
git clone https://github.com/bobanetwork/boba.git
cd boba
```

### Check out the required release branch

Release branches are created when new versions of the `op-node` are created. Read through the [Releases page](https://github.com/bobanetwork/boba/tags) to determine the correct branch to check out.

```
git checkout <name of release branch>
```

### Install dependencies

Install the Node.js dependencies for the Boba Monorepo.

```bash
pnpm install
```

### Build packages

Builde the Node.js packages for the Boba Monorepo.

```bash
pnpm build
```

### Build op-node

Build the `op-node`.

```bash
make op-node
```

## Build the Execution Engine

### Clone the Erigon repo

```bash
git clone https://github.com/bobanetwork/op-erigon.git
cd op-erigon
```

### Check out the required release branch

Release branches are created when new versions of the `erigon` are created. Read through the [Releases page](https://github.com/bobanetwork/op-erigon/releases) to determine the correct branch to check out.

```
git checkout <name of release branch>
```

### Build erigon

Build the `erigon`.

```bash
make erigon
```

## Download Snapshots

You can download the database snapshot for the client and network you wish to run.

Always verify snapshots by comparing the sha256sum of the downloaded file to the sha256sum listed on this [page](snapshot-downloads). Check the sha256sum of the downloaded file by running `sha256sum <filename>`in a terminal.

* BOBA Mainnet

  The **erigon** db can be downloaded from the [boba mainnet erigon db](https://boba-db.s3.us-east-2.amazonaws.com/mainnet/boba-mainnet-erigon-db-1149019.tgz).

  ```bash
  curl -o boba-mainnet-erigon-db-1149019.tgz -sL https://boba-db.s3.us-east-2.amazonaws.com/mainnet/boba-mainnet-erigon-db-1149019.tgz
  ```

  The **geth** db can be downloaded from [boba mainnet geth db](https://boba-db.s3.us-east-2.amazonaws.com/mainnet/boba-mainnet-geth-db-114909.tgz).

  ```bash
  curl -o boba-mainnet-geth-db-114909.tgz -sL https://boba-db.s3.us-east-2.amazonaws.com/mainnet/boba-mainnet-geth-db-114909.tgz
  ```

- BOBA Sepolia

  The **erigon** db can be downloaded from the [boba sepolia erigon db](https://boba-db.s3.us-east-2.amazonaws.com/sepolia/boba-sepolia-erigon-db.tgz).

  ```bash
  curl -o boba-sepolia-erigon-db.tgz -sL https://boba-db.s3.us-east-2.amazonaws.com/sepolia/boba-sepolia-erigon-db.tgz
  ```

  The **geth** db can be downloaded from [boba sepolia geth db](https://boba-db.s3.us-east-2.amazonaws.com/sepolia/boba-sepolia-geth-db.tgz).

  ```bash
  curl -o boba-sepolia-geth-db.tgz -sL https://boba-db.s3.us-east-2.amazonaws.com/sepolia/boba-sepolia-geth-db.tgz
  ```

- OP Mainnet

  The **erigon** db can be downloaded from [Test in Prod OP Mainnet](https://op-erigon-backup.mainnet.testinprod.io).

- OP Sepolia

  The **erigon** db can be downloaded from [optimism sepolia erigon db](https://boba-db.s3.us-east-2.amazonaws.com/sepolia/optimism-sepolia-erigon-db.tgz).

  Or you can download the genesis file from [Optimsim](https://networks.optimism.io/op-sepolia/genesis.json) and initialize the data directory with it.

  ```bash
  curl -o op-sepolia-genesis.json -sL https://networks.optimism.io/op-sepolia/genesis.json
  erigon init --datadir=/db genesis.json
  ```

  The erigon can be built from the [source](https://github.com/bobanetwork/v3-erigon) using `make erigon` .

## Create a JWT Secret

`op-erigon` and `op-node` communicate over the engine API authrpc. This communication is secured using a shared secret. You will need to generate a shared secret and provide it to both `op-erigon` and `op-node` when you start them. In this case, the secret takes the form of a 32 byte hex string.

Run the following command to generate a random 32 byte hex string:

```bash
openssl rand -hex 32 > jwt.txt
```

## Start `op-erigon`

It's usually simpler to begin with `op-erigon` before you start `op-node`. You can start `op-erigon` even if `op-node` isn't running yet, but `op-erigon` won't get any blocks until `op-node` starts.

### Navigate to your op-erigon directory

```bash
cd op-erigon
```

### Copy in the JWT secret

Copy the JWT secret you generated in the previous step into the `v3-erigon` directory.

```bash
cp /path/to/jwt.txt .
```

### Start op-erigon

Using the following command to start `op-erigon` in a default configuration. The JSON-RPC API will become available on port 9545.

```bash
./build/bin/erigon \
	--datadir=$DBDIR_PATH \
	--private.api.addr=localhost:9090 \
	--http.addr=0.0.0.0 \
	--http.port=9545 \
	--http.corsdomain="*" \
	--http.vhosts="*" \
	--authrpc.addr=0.0.0.0 \
	--authrpc.port=8551 \
	--authrpc.vhosts="*" \
	--authrpc.jwtsecret=./jwt.txt \
	--chain=boba-sepolia \
	--http.api=eth,debug,net,engine,web3 \
	--rollup.sequencerhttp=https://mainnet.boba.network \
	--db.size.limit=8TB
```

## Start `op-node`

Once you've started `op-erigon`, you can start `op-node`. `op-node` will connect to `op-erigon` and begin synchronizing the BOBA network. `op-node` will begin sending block payloads to `op-erigon` when it derives enough blocks from Ethereum.

### Navigate to your op-node directory

```bash
cd op-node
```

### Copy in the JWT secret

Copy the JWT secret you generated in the previous step into the `v3-erigon` directory.

```bash
cp /path/to/jwt.txt .
```

### Set environment variables

Set the following environment variable:

```bash
export L1_RPC_URL=... # URL for the L1 node to sync from
```

### Start op-node

Using the following command to start `op-node` in a default configuration. The JSON-RPC API will become available on port 8545.

```bash
./bin/op-node \
  --l1=$L1_RPC_URL \
  --l2=http://localhost:8551 \
  --l2.jwt-secret=./jwt.txt \
  --network=boba-sepolia \
  --rpc.addr=0.0.0.0 \
  --rpc.port=8545
```

## Synchornization

During the initial synchonization, you get log messages from `op-node`, and nothing else appears to happen.

```bash
INFO [08-04|16:36:07.150] Advancing bq origin                      origin=df76ff..48987e:8301316 originBehind=false
```

After a few minutes, `op-node` finds the right batch and then it starts synchronizing. During this synchonization process, you get log messags from `op-node`.

```bash
INFO [08-04|16:36:01.204] Found next batch                         epoch=44e203..fef9a5:8301309 batch_epoch=8301309                batch_timestamp=1,673,567,518
INFO [08-04|16:36:01.205] generated attributes in payload queue    txs=2  timestamp=1,673,567,518
INFO [08-04|16:36:01.265] inserted block                           hash=ee61ee..256300 number=4,069,725 state_root=a582ae..33a7c5 timestamp=1,673,567,518 parent=5b102e..13196c prev_randao=4758ca..11ff3a fee_recipient=0x4200000000000000000000000000000000000011 txs=2  update_safe=true
```
