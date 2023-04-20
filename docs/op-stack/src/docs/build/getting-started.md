---
title: Getting Started
lang: en-US
---

## Overview

Hello! This Getting Started guide is meant to help you kick off your OP Stack journey by taking you through the process of spinning up your very own OP Stack chain on the Ethereum Goerli testnet. You can use this chain to perform tests and prepare for the superchain, or you can modify it to adapt it to your own needs (which may make it incompatible with the superchain in the future). 

## Know before you go

Before we kick off, note that this is a relatively long tutorial! You should prepare to set aside an hour or two to get everything running. Here’s an itemized list of what we’re about to do:

1. Install dependencies
2. Build the source code
3. Generate and fund accounts and private keys
4. Configure your network
5. Deploy the L1 contracts
6. Initialize op-geth
7. Run op-geth
8. Run op-node
9. Get some Goerli ETH on your L2
10. Send some test transactions
11. Celebrate!

## Prerequisites

You’ll need the following software installed to follow this tutorial:

- [Git](https://git-scm.com/)
- [Go](https://go.dev/)
- [Node](https://nodejs.org/en/)
- [Yarn](https://classic.yarnpkg.com/lang/en/docs/install/)
- [Foundry](https://github.com/foundry-rs/foundry#installation)
- [Make](https://linux.die.net/man/1/make)

This tutorial was checked on:

| Software | Version    | Installation command(s) |
| -------- | ---------- | - |
| Ubuntu   | 20.04 LTS  | |
| git, curl, jq, and make | OS default | `sudo apt install -y git curl make jq` |
| Go       | 1.20       | `sudo apt update` <br> `wget https://go.dev/dl/go1.20.linux-amd64.tar.gz` <br> `tar xvzf go1.20.linux-amd64.tar.gz` <br> `sudo cp go/bin/go /usr/bin/go` <br> `sudo mv go /usr/lib` <br> `echo export GOROOT=/usr/lib/go >> ~/.bashrc`
| Node     | 16.19.0    | `curl -fsSL https://deb.nodesource.com/setup_16.x | sudo -E bash -` <br> `sudo apt-get install -y nodejs npm`
| yarn     | 1.22.19    | `sudo npm install -g yarn`
| Foundry  | 0.2.0      | `curl -L https://foundry.paradigm.xyz | bash` <br> `. ~/.bashrc` <br> `foundryup`

## Build the Source Code

We’re going to be spinning up an EVM Rollup from the OP Stack source code.  You could use docker images, but this way we keep the option to modify component behavior if you need to do so. The OP Stack source code is split between two repositories, the [Optimism Monorepo](https://github.com/ethereum-optimism/optimism) and the [`op-geth`](https://github.com/ethereum-optimism/op-geth) repository.

### Build the Optimism Monorepo

1. Clone the [Optimism Monorepo](https://github.com/ethereum-optimism/optimism).

    ```bash
    cd ~
    git clone https://github.com/ethereum-optimism/optimism.git
    ```

1. Enter the Optimism Monorepo.

    ```bash
    cd optimism
    ```

1. Install required modules. This is a slow process, while it is running you can already start building `op-geth`, as shown below.

    ```bash
    yarn install
    ```

1. Build the various packages inside of the Optimism Monorepo.

    ```bash
    make op-node op-batcher op-proposer
    yarn build
    ```

### Build op-geth

1. Clone [`op-geth`](https://github.com/ethereum-optimism/op-geth):

    ```bash
    cd ~
    git clone https://github.com/ethereum-optimism/op-geth.git
    ```

1. Enter `op-geth`:

    ```bash
    cd op-geth
    ```

1. Build `op-geth`:

    ```bash
    make geth
    ```

## Get access to a Goerli node

Since we’re deploying our OP Stack chain to Goerli, you’ll need to have access to a Goerli L1 node. You can either use a node provider like [Alchemy](https://www.alchemy.com/) (easier) or [run your own Goerli node](https://notes.ethereum.org/@launchpad/goerli) (harder).

## Generate some keys

You’ll need four accounts and their private keys when setting up the chain:

- The `Admin` account which has the ability to upgrade contracts.
- The `Batcher` account which publishes Sequencer transaction data to L1.
- The `Proposer` account which publishes L2 transaction results to L1.
- The `Sequencer` account which signs blocks on the p2p network.

You can generate all of these keys with the `rekey` tool in the `contracts-bedrock` package.


1. Enter the Optimism Monorepo:

    ```bash
    cd optimism
    ```

1. Move into the `contracts-bedrock` package:

    ```bash
    cd packages/contracts-bedrock
    ```

1. Run the `rekey` command:

    ```bash
    npx hardhat rekey
    ```

You should get an output like the following:

```
Mnemonic: barely tongue excite actor edge huge lion employ gauge despair this learn

Admin: 0x301c314ca0eedf88a5f7a44680d9dccceb8fcbea
Private Key: ef06ba0291b6e2fa336fd9c06de9c2f18f72ed17cd4fcbda7b376f10592b43d8

Proposer: 0x54355b7d195fcdea96696a522c444c185afaf1a8
Private Key: 8bf67a8cd20087472db00fd869a0ffd7574a4481fb2a07a5f5c6bfb46dcb09ca

Batcher: 0x9a686086e3c74ddd5b59b710b26a73407d9c7e97
Private Key: 1533b607f668cce9553cafbfdfe9529eb31d67f1958d4b16fbdf857a8c50dd56

Sequencer: 0x0324a4c8c1955cb8364e8f07558238b3d2aa5f55
Private Key: fba31658f320bb8ce1ce39fab3c7c2acea6b4dd69cc8483fd85388a461d8426b
```

Save these accounts and their respective private keys somewhere, you’ll need them later. Fund the `Admin` address with a small amount of Goerli ETH as we’ll use that account to deploy our smart contracts. You’ll also need to fund the `Proposer` and `Batcher` address — note that the `Batcher` burns through the most ETH because it publishes transaction data to L1.

Recommended funding amounts are as follows:

- `Admin` — 2 ETH
- `Proposer` — 5 ETH
- `Batcher` — 10 ETH

::: danger Not for production deployments 

The `rekey` tool is *not* designed for production deployments. If you are deploying an OP Stack based chain into production, you should likely be using a combination of hardware security modules and hardware wallets.

:::

## Configure your network

Once you’ve built both repositories, you’ll need head back to the Optimism Monorepo to set up the configuration for your chain. Currently, chain configuration lives inside of the [`contracts-bedrock`](https://github.com/ethereum-optimism/optimism/tree/develop/packages/contracts-bedrock) package.

1. Enter the Optimism Monorepo:

    ```bash
    cd ~/optimism
    ```

1. Move into the `contracts-bedrock` package:

    ```bash
    cd packages/contracts-bedrock
    ```

1. Before we can create our configuration file, we’ll need to pick an L1 block to serve as the starting point for our Rollup. It’s best to use a finalized L1 block as our starting block. You can use the `cast` command provided by Foundry to grab all of the necessary information (replace `<RPC>` with the URL for your L1 Goerli node):

    ```bash
    cast block finalized --rpc-url <RPC> | grep -E "(timestamp|hash|number)"
    ```

    You’ll get back something that looks like the following:

    ```
    hash                 0x784d8e7f0e90969e375c7d12dac7a3df6879450d41b4cb04d4f8f209ff0c4cd9
    number               8482289
    timestamp            1676253324
    ```

1. Fill out the remainder of the pre-populated config file found at [`deploy-config/getting-started.json`](https://github.com/ethereum-optimism/optimism/blob/develop/packages/contracts-bedrock/deploy-config/getting-started.json). Use the default values in the config file and make following modifications:

    - Replace `"ADMIN"` with the address of the Admin account you generated earlier.
    - Replace `"PROPOSER"` with the address of the Proposer account you generated earlier.
    - Replace `"BATCHER"` with the address of the Batcher account you generated earlier.
    - Replace `"SEQUENCER"` with the address of the Sequencer account you generated earlier.
    - Replace `"BLOCKHASH"` with the blockhash you got from the `cast` command.
    - Replace `TIMESTAMP` with the timestamp you got from the `cast` command. Note that although all the other fields are strings, this field is a number! Don’t include the quotation marks.

## Deploy the L1 contracts

Once you’ve configured your network, it’s time to deploy the L1 smart contracts necessary for the functionality of the chain. 

1. Inside of `contracts-bedrock`, copy `.env.example` to `.env`.

    ```sh
    cp .env.example .env
    ```

1. Fill out the two environment variables inside of that file:

    - `L1_RPC` — URL for your L1 node.
    - `PRIVATE_KEY_DEPLOYER` — Private key of the `Admin` account.

1. Once you’re ready, deploy the L1 smart contracts:

    ```bash
    npx hardhat deploy --network getting-started --tags l1
    ```

Contract deployment can take up to 15 minutes. Please wait for all smart contracts to be fully deployed before continuing to the next step.

## Generate the L2 config files

We’ve set up the L1 side of things, but now we need to set up the L2 side of things. We do this by generating three important files, a `genesis.json` file, a `rollup.json` configuration file, and a `jwt.txt` [JSON Web Token](https://jwt.io/introduction) that allows the `op-node` and `op-geth` to communicate securely. 

1. Head over to the `op-node` package:

    ```bash
    cd ~/optimism/op-node
    ```

1. Run the following command, and make sure to replace `<RPC>` with your L1 RPC URL:

    ```bash
    go run cmd/main.go genesis l2 \
        --deploy-config ../packages/contracts-bedrock/deploy-config/getting-started.json \
        --deployment-dir ../packages/contracts-bedrock/deployments/getting-started/ \
        --outfile.l2 genesis.json \
        --outfile.rollup rollup.json \
        --l1-rpc <RPC>
    ```

    You should then see the `genesis.json` and `rollup.json` files inside the `op-node` package.

1. Next, generate the `jwt.txt` file with the following command:

    ```bash
    openssl rand -hex 32 > jwt.txt
    ```

1. Finally, we’ll need to copy the `genesis.json` file and `jwt.txt` file into `op-geth` so we can use it to initialize and run `op-geth` in just a minute:

    ```bash
    cp genesis.json ~/op-geth
    cp jwt.txt ~/op-geth
    ```

## Initialize op-geth

We’re almost ready to run our chain! Now we just need to run a few commands to initialize `op-geth`. We’re going to be running a Sequencer node, so we’ll need to import the `Sequencer` private key that we generated earlier. This private key is what our Sequencer will use to sign new blocks.

1. Head over to the `op-geth` repository:

    ```bash
    cd ~/op-geth
    ```

1. Create a data directory folder:

    ```bash
    mkdir datadir
    ```

1. Put a password file into the data directory folder:

    ```bash
    echo "pwd" > datadir/password
    ```

1. Put the `Sequencer` private key into the data directory folder (don’t include a “0x” prefix):

    ```bash
    echo "<SEQUENCER KEY HERE>" > datadir/block-signer-key
    ```

1. Import the key into `op-geth`:

    ```bash
    ./build/bin/geth account import --datadir=datadir --password=datadir/password datadir/block-signer-key
    ```

1. Next we need to initialize `op-geth` with the genesis file we generated and copied earlier:

    ```bash
    build/bin/geth init --datadir=datadir genesis.json
    ```

Everything is now initialized and ready to go!


## Run the node software

There are four components that need to run for a rollup.
The first two, `op-geth` and `op-node`, have to run on every node.
The other two, `op-batcher` and `op-proposer`, run only in one place, the sequencer that accepts transactions.

Set these environment variables for the configuration

| Variable       | Value |
| -------------- | - 
| `SEQ_ADDR`     | Address of the `Sequencer` account
| `SEQ_KEY`      | Private key of the `Sequencer` account
| `BATCHER_KEY`  | Private key of the `Batcher` accounts, which should have at least 1 ETH
| `PROPOSER_KEY` | Private key of the `Proposer` account
| `L1_RPC`       | URL for the L1 (such as Goerli) you're using
| `RPC_KIND`     | The type of L1 server to which you connect, which can optimize requests. Available options are `alchemy`, `quicknode`, `parity`, `nethermind`, `debug_geth`, `erigon`, `basic`, and `any`
| `L2OO_ADDR`    | The address of the `L2OutputOracleProxy`, available at `~/optimism/packages/contracts-bedrock/deployments/getting-started/L2OutputOracleProxy.json

### `op-geth`

Run `op-geth` with the following commands.

```bash
cd ~/op-geth

./build/bin/geth \
	--datadir ./datadir \
	--http \
	--http.corsdomain="*" \
	--http.vhosts="*" \
	--http.addr=0.0.0.0 \
	--http.api=web3,debug,eth,txpool,net,engine \
	--ws \
	--ws.addr=0.0.0.0 \
	--ws.port=8546 \
	--ws.origins="*" \
	--ws.api=debug,eth,txpool,net,engine \
	--syncmode=full \
	--gcmode=archive \
	--nodiscover \
	--maxpeers=0 \
	--networkid=42069 \
	--authrpc.vhosts="*" \
	--authrpc.addr=0.0.0.0 \
	--authrpc.port=8551 \
	--authrpc.jwtsecret=./jwt.txt \
	--rollup.disabletxpoolgossip=true \
	--password=./datadir/password \
	--allow-insecure-unlock \
	--mine \
	--miner.etherbase=$SEQ_ADDR \
	--unlock=$SEQ_ADDR
```

And `op-geth` should be running! You should see some output, but you won’t see any blocks being created yet because `op-geth` is driven by the `op-node`. We’ll need to get that running next.

::: tip Why archive mode?

Archive mode takes more disk storage than full mode.
However, using it is important for two reasons:

- The `op-proposer` requires access to the full state.
  If at some point `op-proposer` needs to look beyond 256 blocks in the past (8.5 minutes in the default configuration), for example because it was down for that long, we need archive mode.

- The [explorer](./explorer.md) requires archive mode.

:::

#### Reinitializing op-geth

There are several situations are indicate database corruption and require you to reset the `op-geth` component:

- When `op-node` errors out when first started and exits.
- When `op-node` emits this error:

  ```
  stage 0 failed resetting: temp: failed to find the L2 Heads to start from: failed to fetch L2 block by hash 0x0000000000000000000000000000000000000000000000000000000000000000
  ```

This is the reinitialization procedure:

1. Stop the `op-geth` process.
1. Delete the geth data.

    ```bash
    cd ~/op-geth
    rm -rf datadir/geth
    ```

1. Rerun init.

    ```bash
    build/bin/geth init --datadir=datadir genesis.json
    ```

1. Start `op-geth`

1. Start `op-node`


### `op-node`

Once we’ve got `op-geth` running we’ll need to run `op-node`. Like Ethereum, the OP Stack has a consensus client (the `op-node`) and an execution client (`op-geth`). The consensus client drives the execution client over the Engine API.

```bash
cd ~/optimism/op-node

./bin/op-node \
	--l2=http://localhost:8551 \
	--l2.jwt-secret=./jwt.txt \
	--sequencer.enabled \
	--sequencer.l1-confs=3 \
	--verifier.l1-confs=3 \
	--rollup.config=./rollup.json \
	--rpc.addr=0.0.0.0 \
	--rpc.port=8547 \
	--p2p.disable \
	--rpc.enable-admin \
	--p2p.sequencer.key=$SEQ_KEY \
	--l1=$L1_RPC \
	--l1.rpckind=$RPC_KIND
```

Once you run this command, you should start seeing the `op-node` begin to process all of the L1 information after the starting block number that you picked earlier. Once the `op-node` has enough information, it’ll begin sending Engine API payloads to `op-geth`. At that point, you’ll start to see blocks being created inside of `op-geth`. We’re live!


::: tip Peer to peer synchronization

If you use a chain ID that is also used by others, for example the default (42069), your `op-node` will try to use peer to peer to speed up synchronization.
These attempts will fail, because they will be signed with the wrong key, but they will waste time and network resources.

To avoid this , we start with peer to peer synchronization disabled (`--p2p.disable`).
Once you have multiple nodes, it makes sense to use these command line parameters to synchronize between them without getting confused by other blockchains.

```
	--p2p.static=<nodes> \
	--p2p.listen.ip=0.0.0.0 \
	--p2p.listen.tcp=9003 \
	--p2p.listen.udp=9003 \
```

:::




### `op-batcher`

The `op-batcher` takes transactions from the Sequencer and publishes those transactions to L1. Once transactions are on L1, they’re officially part of the Rollup. Without the `op-batcher`, transactions sent to the Sequencer would never make it to L1 and wouldn’t become part of the canonical chain. The `op-batcher` is critical!

It is best to give the `Batcher` at least 1 Goerli ETH to ensure that it can continue operating without running out of ETH for gas.


```bash
cd ~/optimism/op-batcher

./bin/op-batcher \
    --l2-eth-rpc=http://localhost:8545 \
    --rollup-rpc=http://localhost:8547 \
    --poll-interval=1s \
    --sub-safety-margin=6 \
    --num-confirmations=1 \
    --safe-abort-nonce-too-low-count=3 \
    --resubmission-timeout=30s \
    --rpc.addr=0.0.0.0 \
    --rpc.port=8548 \
    --rpc.enable-admin \
    --max-channel-duration=1 \
    --l1-eth-rpc=$L1_RPC \
    --private-key=$BATCHER_KEY
```

::: tip Controlling batcher costs

The `--max-channel-duration=n` setting tells the batcher to write all the data to L1 every `n` L1 blocks. 
When it is low, transactions are written to L1 frequently, withdrawals are quick, and other nodes can synchronize from L1 fast.
When it is high, transactions are written to L1 less frequently, and the batcher spends less ETH.

:::

### `op-proposer`

Now start `op-proposer`, which proposes new state roots.

```bash
cd ~/optimism/op-proposer

./bin/op-proposer \
    --poll-interval 12s \
    --rpc.port 8560 \
    --rollup-rpc http://localhost:8547 \
    --l2oo-address $L2OO_ADDR \
    --private-key $PROPOSER_KEY \
    --l1-eth-rpc $L1_RPC
```

<!--
::: warning Change before moving to production

The `--allow-non-finalized` flag allows for faster tests on a test network. 
However, in production you would probably want to only submit proposals on properly finalized blocks.

:::
-->

## Get some ETH on your Rollup

Once you’ve connected your wallet, you’ll probably notice that you don’t have any ETH on your Rollup. You’ll need some ETH to pay for gas on your Rollup. The easiest way to deposit Goerli ETH into your chain is to send funds directly to the `L1StandardBridge` contract. You can find the address of the `L1StandardBridge` contract for your chain by looking inside the `deployments` folder in the `contracts-bedrock` package.

1. First, head over to the `contracts-bedrock` package:

    ```bash
    cd ~/optimism/packages/contracts-bedrock
    ```

1. Grab the address of the proxy to the L1 standard bridge contract:

    ```bash
    cat deployments/getting-started/Proxy__OVM_L1StandardBridge.json |  jq -r .address
    ```

1. Grab the L1 bridge proxy contract address and, using the wallet that you want to have ETH on your Rollup, send that address a small amount of ETH on Goerli (0.1 or less is fine). It may take up to 5 minutes for that ETH to appear in your wallet on L2.

## Use your Rollup

Congratulations, you made it! You now have a complete OP Stack based EVM Rollup. 

To see your rollup in action, you can use the [Optimism Mainnet Getting Started tutorial](https://github.com/ethereum-optimism/optimism-tutorial/blob/main/getting-started). Follow these steps:

1. Clone the tutorials repository.

    ```bash
    cd ~
    git clone https://github.com/ethereum-optimism/optimism-tutorial.git
    ```

1. Change to the Foundry directory of the Getting Started tutorial.

    ```bash
    cd optimism-tutorial/getting-started/foundry
    ```

1. Put your mnemonic (for the address where you have ETH, the one that sent ETH to `OptimismPortalProxy` on Goerli) in a file `mnem.delme`.
1. Provide the URL to your blockchain:

    ```bash
    export ETH_RPC_URL=http://localhost:8545
    ```

1. Compile and deploy the `Greeter` contract:

    ```bash
    forge create --mnemonic-path ./mnem.delme Greeter --constructor-args "hello" \
        | tee deployment
    ```

1. Set the greeter to the deployed to address:

    ```bash
    export GREETER=`cat deployment | awk '/Deployed to:/ {print $3}'`
    echo $GREETER
    ```

1. See and modify the greeting

    ```bash
    cast call $GREETER "greet()" | cast --to-ascii
    cast send --mnemonic-path ./mnem.delme $GREETER "setGreeting(string)" "New greeting"
    cast call $GREETER "greet()" | cast --to-ascii
    ```

To use any other development stack, see the getting started tutorial, just replace the Greeter address with the address of your rollup, and the Optimism Goerli URL with `http://localhost:8545`.


### Errors

#### Corrupt data directory

If `op-geth` aborts (for example, because the computer it is running on crashes), you might get these errors on `op-node`: 

```
WARN [02-16|21:22:02.868] Derivation process temporary error       attempts=14 err="stage 0 failed resetting: temp: failed to find the L2 Heads to start from: failed to fetch L2 block by hash 0x0000000000000000000000000000000000000000000000000000000000000000: failed to determine block-hash of hash 0x0000000000000000000000000000000000000000000000000000000000000000, could not get payload: not found"
```

This means that the data directory is corrupt and you need to reinitialize it:

```bash
cd ~/op-geth
rm -rf datadir
mkdir datadir
echo "pwd" > datadir/password
echo "<SEQUENCER KEY HERE>" > datadir/block-signer-key
./build/bin/geth account import --datadir=./datadir --password=./datadir/password ./datadir/block-signer-key
./build/bin/geth init --datadir=./datadir ./genesis.json
```


#### Batcher out of ETH

If `op-batcher` runs out of ETH, it cannot submit write new transaction batches to L1.
You will get error messages similar to this one:

```
INFO [03-21|14:22:32.754] publishing transaction                   service=batcher txHash=2ace6d..7eb248 nonce=2516 gasTipCap=2,340,741 gasFeeCap=172,028,434,515
ERROR[03-21|14:22:32.844] unable to publish transaction            service=batcher txHash=2ace6d..7eb248 nonce=2516 gasTipCap=2,340,741 gasFeeCap=172,028,434,515 err="insufficient funds for gas * price + value"
```

Just send more ETH and to the batcher, and the problem will be resolved.



## What’s next?

You can use this rollup the same way you’d use any other test blockchain. Once the superchain is available, this blockchain should be able to join the test version. Alternatively, you could [modify the blockchain in various ways](./hacks.md). **Please note that OP Stack Hacks are unofficial and are not explicitly supported by the OP Stack.** You will not be able to receive significant developer support for any modifications you make to the OP Stack.
