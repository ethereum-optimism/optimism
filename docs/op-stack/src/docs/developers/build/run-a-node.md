---
title: Running a testnet or mainnet node
lang: en-US
---

If you're looking to build an app on Optimism you'll need access to an Optimism node. You have two options - use a hosted node from providers like Alchemy or run your own. 

## Hosted node providers

You can get a free, hosted one from [any of these providers](../../useful-tools/providers.md) to get up and building quickly. Of them, [Alchemy](https://www.alchemy.com/optimism) is our preferred node provider, and is used to power our [public endpoint](../../useful-tools/networks.md). 

However, you might be interested in running your very own Optimism node.
Here we'll go over the process of running a testnet or mainnet Optimism node for yourself.

## Upgrades

If you run a node you need to subscribe to [an update feed](../releases.md) (either [the mailing list](https://groups.google.com/a/optimism.io/g/optimism-announce) or [the RSS feed](https://changelog.optimism.io/feed.xml)) to know when to upgrade. 
Otherwise, your node will eventually stop working.

## Configuration choices

### Hardware requirements

Replicas need to store the transaction history of Optimism and to run Geth. 
They need to be relatively powerful machines (real or virtual). 
We recommend at least 16 GB RAM, and an SSD drive with at least 100 GB free.

### Source of synchronization

<details>
<summary><b>Pre-Bedrock (current version)</b></summary>

Prior to Bedrock you choose one of two configurations.

- **Replicas** replicate from L2 (Optimism).
  Replicas gives you the most up to date information, at the cost of having to trust Optimism's updates.

- **Verifiers** replicate from L1 (Ethereum).
  Verifiers read and execute transactions from the cannonical block chain. 
  As a result, the only way for them to have inaccurate information is an [Ethereum reorg](https://www.paradigm.xyz/2021/07/ethereum-reorgs-after-the-merge#post-merge-ethereum-with-proof-of-stake), an extremely rare event. 

</details>

<details>
<summary><b>Bedrock (coming late 2022)</b></summary>

In Bedrock the [op-geth](https://community.optimism.io/docs/developers/bedrock-temp/infra/#bedrock-geth) typically synchronizes from other Optimism nodes (https://github.com/ethereum-optimism/optimism/blob/develop/specs/exec-engine.md#happy-path-sync), meaning L2, but it can [synchronize from L1](https://github.com/ethereum-optimism/optimism/blob/develop/specs/exec-engine.md#worst-case-sync) if necessary.

To synchronize only from L1, you edit the [op-node configuration](https://github.com/ethereum-optimism/optimism/blob/develop/specs/rollup-node.md) to set `OP_NODE_P2P_DISABLE` to `true`.

When you use RPC to get block information (https://github.com/ethereum-optimism/optimism/blob/develop/specs/rollup-node.md#l2-output-rpc-method), you can specify one of four options for `blockNumber`:

- an actual block number
- **pending**: Latest L2 block
- **latest**: Latest block written to L1
- **finalized**: Latest block fully finalized on L1 (a process that takes 12 minutes with Proof of Stake)


</details>

## Docker configuration

The recommended method to create a replica is to use [Docker](https://www.docker.com/) and the [Docker images we provide](https://hub.docker.com/u/ethereumoptimism). 
They include all the configuration settings.
This is the recommended method because it is what we for our own systems.
As such, the docker images go through a lot more tests than any other configuration.

### Configuring and running the node

Follow [these instructions](https://github.com/smartcontracts/simple-optimism-node) to build and run the node.


## Non-docker configuration

Here are the instructions if you want to build you own replica without relying on our images.
These instructions were generated with a [GCP e2-standard-4](https://cloud.google.com/compute/docs/general-purpose-machines#e2-standard) virtual machine running [Debian 10](https://www.debian.org/News/2021/2021100902) with a 100 GB SSD drive. 
They should work on different operating systems with minor changes, but there are no guarantees.

Note that these directions are for a replica of the main network. 
You need to modify some of them if you want to create a replica of the test network.

**Note:** This is *not* the recommended configuration.
While we did QA on these instructions and they work, the QA that the docker images undergo is much more extensive.


### Install packages

1. These packages are all required either to compile the software or to run it. 
    We need `libusb-1.0` because geth requires it to check for hardware wallets.

    ```sh
    sudo apt install -y git make wget gcc pkg-config libusb-1.0 jq
    ```

1. Install [the node.js package](https://nodejs.org/).
   These instructions were written using the 12.x version.

1. Install [yarn](https://classic.yarnpkg.com/): 
   ```sh
    sudo npm install -g yarn 
    ```    

1. Install [the Go programming language](https://go.dev/doc/install).
   These instructions were written using Go version 1.17.6


<!--
    ```sh
    curl -sL https://deb.nodesource.com/setup_12.x -o nodesource_setup.sh
    sudo bash nodesource_setup.sh
    sudo apt install -y nodejs
    ```

    ```sh
    wget https://go.dev/dl/go1.17.6.linux-amd64.tar.gz
    sudo tar -C /usr/local -xzf go1.17.6.linux-amd64.tar.gz
    cp /etc/profile /tmp
    echo "export PATH=$PATH:/usr/local/go/bin" >> /tmp/profile
    sudo mv /tmp/profile /etc
    . /etc/profile
    . ~/.profile
    ```
--->

### The Data Transport Layer (DTL)

This TypeScript program reads data from an Optimism endpoint and passes it over to the local instance of l2geth ([geth](https://geth.ethereum.org/) with minor changes for layer 2 support).

1. Download [the source code](https://github.com/ethereum-optimism/optimism).
    Then, compile the DTL:

    ```sh
    git clone -b master https://github.com/ethereum-optimism/optimism.git     
    cd optimism
    yarn
    yarn build
    cd ~/optimism/packages/data-transport-layer
    cp .env.example .env
    ```

1. Edit `.env` to specify your own configuration.
   Modify these parameters:

   | Parameter | Value |
   | --------- | ----- |
   | DATA_TRANSPORT_LAYER__NODE_ENV         | production |
   | DATA_TRANSPORT_LAYER__ETH_NETWORK_NAME | mainnet |    
   | DATA_TRANSPORT_LAYER__ADDRESS_MANAGER  | 0xdE1FCfB0851916CA5101820A69b13a4E276bd81F 
   | DATA_TRANSPORT_LAYER__SERVER_HOSTNAME  | localhost
   | DATA_TRANSPORT_LAYER__SERVER_PORT      | 7878
   | DATA_TRANSPORT_LAYER__SYNC_FROM_L1     | false |    
   | DATA_TRANSPORT_LAYER__L1_RPC_ENDPOINT  | Get an endpoint from [a service provider](https://ethereum.org/en/developers/docs/nodes-and-clients/nodes-as-a-service/) unless you run a node yourself |
   | DATA_TRANSPORT_LAYER__SYNC_FROM_L2     | true |
   | DATA_TRANSPORT_LAYER__L2_RPC_ENDPOINT  | [See here](../../useful-tools/networks/) |
   | DATA_TRANSPORT_LAYER__L2_CHAIN_ID      | 10 (for a mainnet replica) |

   These directions are written with the assumption that you sync from L2, which is faster.
   If you prefer, you can syncronize from L1, which is more secure but slower.
   To use L1, keep the value of `DATA_TRANSPORT_LAYER__SYNC_FROM_L1` as `true` and 
   `DATA_TRANSPORT_LAYER__SYNC_FROM_L2` as `false`. Also, add this line:

   ```
   DATA_TRANSPORT_LAYER__L1_START_HEIGHT=13596466
   ```

1. Start the DTL (as a daemon, logging to `~/dtl.log`):

    ```sh
    nohup yarn start > ~/dtl.log &
    ```

    Note that you cannot just close the window if you want DTL to continue running, you have to exit the shell gracefully.

1. To verify the DTL is running correctly you can run a command.

   - If synchronizing from L2:
     ```sh
     curl -s http://localhost:7878/eth/syncing?backend=l2  | jq .currentTransactionIndex
     ```

   - If synchronizing from L1:
     ```sh
     curl -s http://localhost:7878/eth/syncing?backend=l1  | jq .currentTransactionIndex
     ```

   It gives you the current transaction index, which should increase with time.

   
1. For debugging purposes, it is sometimes useful to get a transaction's information from the DTL:

   ```
   curl -s http://localhost:7878/transaction/index/<transaction number>?backend=l2 | jq .transaction
   ```

   Note that the transaction indexes are one below the number on etherscan, so for example

   ```
   curl -s http://localhost:7878/transaction/index/31337?backend=l2 | jq .transaction
   ```

   Corresponds to [Etherscan transaction 31338](https://explorer.optimism.io/tx/31338).


The DTL now needs to download the entire transaction history since regenesis, a process that takes hours.
While it is running, we can get started on the client software.

### The Optimism client software

The client software, called l2geth, is a minimally modified version of [`geth`](https://geth.ethereum.org/). 
Because `geth` supports hardware wallets you might get USB errors. If you do, ignore them.

These directions use `~/gethData` as the data directory. 
You can replace it with you own directory as long as you are consistent.

1. To compile l2geth, run:

    ```sh
    cd ~/optimism/l2geth
    make geth
    ```

1. Download and verify the genesis state, the state of the Optimism blockchain during the final regenesis, 11 November 2021. 

   ```sh
   wget -O /tmp/genesis.json https://storage.googleapis.com/optimism/mainnet/genesis-berlin.json
   sha256sum /tmp/genesis.json
   ```

   The output of the `sha256sum` command should be:
   ```
   0x106b0a3247ca54714381b1109e82cc6b7e32fd79ae56fbcc2e7b1541122f84ea  /tmp/genesis.json
   ```

1. Create a file called `env.sh` (in whatever directory is convenient) with this content:

    ```sh
    export CHAIN_ID=10
    export DATADIR=~/gethData
    export NETWORK_ID=10
    export NO_DISCOVER=true
    export NO_USB=true
    export GASPRICE=0
    export GCMODE=archive
    export BLOCK_SIGNER_ADDRESS=0x00000398232E2064F896018496b4b44b3D62751F
    export BLOCK_SIGNER_PRIVATE_KEY=6587ae678cf4fc9a33000cdbf9f35226b71dcc6a4684a31203241f9bcfd55d27
    export ETH1_CTC_DEPLOYMENT_HEIGHT=13596466
    export ETH1_SYNC_SERVICE_ENABLE=true
    export ROLLUP_ADDRESS_MANAGER_OWNER_ADDRESS=0x9BA6e03D8B90dE867373Db8cF1A58d2F7F006b3A
    export ROLLUP_CLIENT_HTTP=http://localhost:7878
    export ROLLUP_DISABLE_TRANSFERS=false
    export ROLLUP_ENABLE_L2_GAS_POLLING=false
    export ROLLUP_GAS_PRICE_ORACLE_OWNER_ADDRESS=0x648E3e8101BFaB7bf5997Bd007Fb473786019159
    export ROLLUP_MAX_CALLDATA_SIZE=40000
    export ROLLUP_POLL_INTERVAL_FLAG=1s
    export ROLLUP_SYNC_SERVICE_ENABLE=true
    export ROLLUP_TIMESTAMP_REFRESH=5m
    export ROLLUP_VERIFIER_ENABLE=true
    export RPC_ADDR=0.0.0.0
    export RPC_API=eth,rollup,net,web3,debug
    export RPC_CORS_DOMAIN=*
    export RPC_ENABLE=true
    export RPC_PORT=8545
    export RPC_VHOSTS=*
    export TARGET_GAS_LIMIT=15000000
    export USING_OVM=true
    export WS_ADDR=0.0.0.0
    export WS_API=eth,rollup,net,web3,debug
    export WS_ORIGINS=*
    export WS=true
    export ROLLUP_BACKEND=l2    
    ```

    **Note**: If synchronizing from L1, replace the last line with 
    ```sh
    export ROLLUP_BACKEND=l1    
    ```

1. Run the new file. 
   This syntax (dot, space, and then the name of the script) runs the script in the context of the current shell, rather than in a new shell.
   The reason for doing this is that we want to modify the current shell's environment variables, not start a new shell, set up the environment in it, and then exit.

   ```sh
   . env.sh
   ```


1. Initialize l2geth with the genesis state.
   This process takes about nine minutes on my system.

    ```sh
    mkdir ~/gethData
    ./build/bin/geth init --datadir=$DATADIR /tmp/genesis.json --nousb
    ```



1. Create the geth account. 
   The private key needs to be the one specified in the configuration, otherwise the consensus algorithm fails and the node does not synchronize.

    ```sh
    touch $DATADIR/password
    echo $BLOCK_SIGNER_PRIVATE_KEY > $DATADIR/block-signer-key
    ./build/bin/geth account import --datadir=$DATADIR --password $DATADIR/password $DATADIR/block-signer-key
    ```

1. Start geth (logging to `~/geth.log`). 

    ```sh
    nohup build/bin/geth \
       --datadir=$DATADIR \
       --password=$DATADIR/password \
       --allow-insecure-unlock \
       --unlock=$BLOCK_SIGNER_ADDRESS \
       --mine \
       --miner.etherbase=$BLOCK_SIGNER_ADDRESS > ~/geth.log &
    ```

    It is possible that `geth` won't listen to IPC or the TCP port (8545) until it finishes the initial synchronization.

1. To check if l2geth is running correctly, open another command line window and run these commands:

   ```sh
   cd ~/optimism/l2geth
   build/bin/geth attach --datadir=~/gethData 
   eth.blockNumber
   ```

   Wait a few seconds and then look at the blocknumber again and exit:

   ```sh
   eth.blockNumber
   exit
   ```
 
   If l2geth is synchronizing, the second block number is higher than the first.

1. Wait a few hours until the entire history is downloaded by dtl and then propagated to l2geth.
   If you have any problems, [contact us on our Discord](https://discord-gateway.optimism.io/).



<!--
latest.sh:

#! /bin/sh

echo dtl:
curl -s http://localhost:7878/eth/syncing?backend=l2  | jq .currentTransactionIndex

echo
echo l2geth:
tail -1 ~/geth.log | awk '{print $5}'
-->


