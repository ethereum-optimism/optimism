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

## Build the Execution Engine (op-reth)

The recommended execution client is op-reth. The upstream [op-reth](https://github.com/paradigmxyz/reth) includes Boba chains as built-in, so you can build directly from upstream.

```bash
git clone https://github.com/paradigmxyz/reth.git
cd reth
cargo build --bin op-reth --release
```

The binary will be at `target/release/op-reth`.

## Download Snapshots

Download the database snapshot for your network from the [snapshot downloads](snapshot-downloads) page. Always verify snapshots by comparing the sha256sum of the downloaded file to the sha256sum listed on that page.

```bash
sha256sum <filename>
```

## Create a JWT Secret

`op-reth` and `op-node` communicate over the engine API authrpc. This communication is secured using a shared secret. You will need to generate a shared secret and provide it to both `op-reth` and `op-node` when you start them. In this case, the secret takes the form of a 32 byte hex string.

Run the following command to generate a random 32 byte hex string:

```bash
openssl rand -hex 32 > jwt.txt
```

## Start `op-reth`

It's usually simpler to begin with `op-reth` before you start `op-node`. You can start `op-reth` even if `op-node` isn't running yet, but `op-reth` won't get any blocks until `op-node` starts.

Using the following command to start `op-reth` in a default configuration. The JSON-RPC API will become available on port 9545.

```bash
op-reth node \
  --chain=boba-sepolia \
  --datadir=./reth-data \
  --http \
  --http.addr=0.0.0.0 \
  --http.port=9545 \
  --http.corsdomain="*" \
  --http.api=eth,debug,net,web3 \
  --authrpc.addr=0.0.0.0 \
  --authrpc.port=8551 \
  --authrpc.jwtsecret=./jwt.txt \
  --rollup.sequencer-http=https://sepolia.boba.network \
  --rollup.disable-tx-pool-gossip
```

For mainnet, use `--chain=boba` and `--rollup.sequencer-http=https://mainnet.boba.network`.

## Start `op-node`

Once you've started `op-reth`, you can start `op-node`. `op-node` will connect to `op-reth` and begin synchronizing the BOBA network. `op-node` will begin sending block payloads to `op-reth` when it derives enough blocks from Ethereum.

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

## Synchronization

During the initial synchronization, you get log messages from `op-node`, and nothing else appears to happen.

```bash
INFO [08-04|16:36:07.150] Advancing bq origin                      origin=df76ff..48987e:8301316 originBehind=false
```

After a few minutes, `op-node` finds the right batch and then it starts synchronizing.

```bash
INFO [08-04|16:36:01.204] Found next batch                         epoch=44e203..fef9a5:8301309 batch_epoch=8301309                batch_timestamp=1,673,567,518
INFO [08-04|16:36:01.205] generated attributes in payload queue    txs=2  timestamp=1,673,567,518
INFO [08-04|16:36:01.265] inserted block                           hash=ee61ee..256300 number=4,069,725 state_root=a582ae..33a7c5 timestamp=1,673,567,518 parent=5b102e..13196c prev_randao=4758ca..11ff3a fee_recipient=0x4200000000000000000000000000000000000011 txs=2  update_safe=true
```
