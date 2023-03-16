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
| git      | OS default | |
| make     | 4.2.1-1.2  | `sudo apt install -y make`
| Go       | 1.20       | `sudo apt update` <br> `wget https://go.dev/dl/go1.20.linux-amd64.tar.gz` <br> `tar xvzf go1.20.linux-amd64.tar.gz` <br> `sudo cp go/bin/go /usr/bin/go` <br> `sudo mv go /usr/lib` <br> `echo export GOROOT=/usr/lib/go >> ~/.bashrc`
| Node     | 16.19.0    | `curl -fsSL https://deb.nodesource.com/setup_16.x | sudo -E bash -` <br> `sudo apt-get install -y nodejs`
| yarn     | 1.22.19    | `sudo npm install -g yarn`
| Foundry  | 0.2.0      | `curl -L https://foundry.paradigm.xyz | bash` <br> `sudo bash` <br> `foundryup`

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
    make build
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

- `Admin` — 0.2 ETH
- `Proposer` — 0.5 ETH
- `Batcher` — 1.0 ETH

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
    - Replace `"TIMESTAMP"` with the timestamp you got from the `cast` command. Note that although all the other fields are strings, this field is a number! Don’t include the quotation marks.

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
    npx hardhat deploy --network getting-started
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

## Run op-geth

Whew! We made it. It’s time to run `op-geth` and get our system started.

Run `op-geth` with the following command. Make sure to replace `<SEQUENCER>` with the address of the `Sequencer` account you generated earlier.

```bash
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
	--gcmode=full \
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
	--miner.etherbase=<SEQUENCER> \
	--unlock=<SEQUENCER>
```

And `op-geth` should be running! You should see some output, but you won’t see any blocks being created yet because `op-geth` is driven by the `op-node`. We’ll need to get that running next.

### Reinitializing op-geth

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


## Run op-node

Once we’ve got `op-geth` running we’ll need to run `op-node`. Like Ethereum, the OP Stack has a consensus client (the `op-node`) and an execution client (`op-geth`). The consensus client drives the execution client over the Engine API.

Head over to the `op-node` package and start the `op-node` using the following command. Replace `<SEQUENCERKEY>` with the private key for the `Sequencer` account, replace `<RPC>` with the URL for your L1 node, and replace `<RPCKIND>` with the kind of RPC you’re connected to. Although the `l1.rpckind` argument is optional, setting it will help the `op-node` optimize requests and reduce the overall load on your endpoint. Available options for the `l1.rpckind` argument are `"alchemy"`, `"quicknode"`, `"quicknode"`, `"parity"`, `"nethermind"`, `"debug_geth"`, `"erigon"`, `"basic"`, and `"any"`.

```bash
./bin/op-node \
	--l2=http://localhost:8551 \
	--l2.jwt-secret=./jwt.txt \
	--sequencer.enabled \
	--sequencer.l1-confs=3 \
	--verifier.l1-confs=3 \
	--rollup.config=./rollup.json \
	--rpc.addr=0.0.0.0 \
	--rpc.port=8547 \
	--p2p.listen.ip=0.0.0.0 \
	--p2p.listen.tcp=9003 \
	--p2p.listen.udp=9003 \
	--rpc.enable-admin \
	--p2p.sequencer.key=<SEQUENCERKEY> \
	--l1=<RPC> \
	--l1.rpckind=<RPCKIND>
```

Once you run this command, you should start seeing the `op-node` begin to process all of the L1 information after the starting block number that you picked earlier. Once the `op-node` has enough information, it’ll begin sending Engine API payloads to `op-geth`. At that point, you’ll start to see blocks being created inside of `op-geth`. We’re live!


## Run op-batcher

The final component necessary to put all the pieces together is the `op-batcher`. The `op-batcher` takes transactions from the Sequencer and publishes those transactions to L1. Once transactions are on L1, they’re officially part of the Rollup. Without the `op-batcher`, transactions sent to the Sequencer would never make it to L1 and wouldn’t become part of the canonical chain. The `op-batcher` is critical!

1. Head over to the `op-batcher` package inside the Optimism Monorepo:

    ```bash
    cd ~/optimism/op-batcher
    ```

1. And run the `op-batcher` using the following command. Replace `<RPC>` with your L1 node URL and replace `<BATCHERKEY>` with the private key for the `Batcher` account that you created and funded earlier. It’s best to give the `Batcher` at least 1 Goerli ETH to ensure that it can continue operating without running out of ETH for gas.

    ```bash
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
        --target-l1-tx-size-bytes=2048 \
        --l1-eth-rpc=<RPC> \
        --private-key=<BATCHERKEY>
    ```

## Get some ETH on your Rollup

Once you’ve connected your wallet, you’ll probably notice that you don’t have any ETH on your Rollup. You’ll need some ETH to pay for gas on your Rollup. The easiest way to deposit Goerli ETH into your chain is to send funds directly to the `OptimismPortalProxy` contract. You can find the address of the `OptimismPortalProxy` contract for your chain by looking inside the `deployments` folder in the `contracts-bedrock` package.

1. First, head over to the `contracts-bedrock` package:

    ```bash
    cd ~/optimism/packages/contracts-bedrock
    ```

1. Grab the address of the `OptimismPortalProxy` contract:

    ```bash
    cat deployments/getting-started/OptimismPortalProxy.json | grep \"address\":
    ```

    You should see a result like the following (**your address will be different**):

    ```
    "address": "0x264B5fde6B37fb6f1C92AaC17BA144cf9e3DcFE9",
            "address": "0x264B5fde6B37fb6f1C92AaC17BA144cf9e3DcFE9",
    ```

1. Grab the `OptimismPortalProxy` address and, using the wallet that you want to have ETH on your Rollup, send that address a small amount of ETH on Goerli (0.1 or less is fine). It may take up to 5 minutes for that ETH to appear in your wallet on L2.

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
    cast send --mnemonic-path mnem.delme $GREETER "setGreeting(string)" "New greeting"
    cast call $GREETER "greet()" | cast --to-ascii
    ```

To use any other development stack, see the getting started tutorial, just replace the Greeter address with the address of your rollup, and the Optimism Goerli URL with `http://localhost:8545`.

## Rollup operations

### Stopping your Rollup

To stop `op-geth` you should use Ctrl-C. 

If `op-geth` aborts (for example, because the computer it is running on crashes), you will get these errors on `op-node`: 

```
WARN [02-16|21:22:02.868] Derivation process temporary error       attempts=14 err="stage 0 failed resetting: temp: failed to find the L2 Heads to start from: failed to fetch L2 block by hash 0x0000000000000000000000000000000000000000000000000000000000000000: failed to determine block-hash of hash 0x0000000000000000000000000000000000000000000000000000000000000000, could not get payload: not found"
```

In that case, you need to remove `datadir`, reinitialize it:

```bash
cd ~/op-geth
rm -rf datadir
mkdir datadir
echo "pwd" > datadir/password
echo "<SEQUENCER KEY HERE>" > datadir/block-signer-key
./build/bin/geth account import --datadir=./datadir --password=./datadir/password ./datadir/block-signer-key
./build/bin/geth init --datadir=./datadir ./genesis.json
```

### Starting your Rollup

To restart the blockchain, use the same order of components you did when you initialized it.

1. `op-geth`
2. `op-node`
3. `op-batcher`

## Adding nodes

To add nodes to the rollup, you need to initialize `op-node` and `op-geth`, similar to what you did for the first node:

1. Configure the OS and prerequisites as you did for the first node.
1. Build the Optimism monorepo and `op-geth` as you did for the first node.
1. Copy from the first node these files:
    
    ```bash
    ~/op-geth/genesis.json
    ~/optimism/op-node/rollup.json
    ```
    
1. Create a new `jwt.txt` file as a shared secret:
    
    ```bash
    cd ~/op-geth
    openssl rand -hex 32 > jwt.txt
    cp jwt.txt ~/optimism/op-node
    ```
    
1. Initialize the new op-geth:
    
    ```bash
    cd ~/op-geth
    ./build/bin/geth init --datadir=./datadir ./genesis.json
    ```
    
1. Start `op-geth` (using the same command line you used on the initial node)
1. Start `op-node` (using the same command line you used on the initial node)
1. Wait while the node synchronizes

## What’s next?

You can use this rollup the same way you’d use any other test blockchain. Once the superchain is available, this blockchain should be able to join the test version. Alternatively, you could [modify the blockchain in various ways](./hacks.md). **Please note that OP Stack Hacks are unofficial and are not explicitly supported by the OP Stack.** You will not be able to receive significant developer support for any modifications you make to the OP Stack.