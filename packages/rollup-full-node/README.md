# Dependencies
The `/exec/` scripts depend on [parity](https://github.com/paritytech/parity-ethereum/releases/tag/v2.5.13) being installed.

If you'd like to use a containerized version, you'll need to [install docker](https://docs.docker.com/install/).

For other dependencies, please refer to the root README of this repo.

# Setup
Run `yarn install` to install necessary dependencies.

# Configuration
Config is handled entirely through environment variables. Below are some config variable names, whether or not they're optional, and what they do:

**Rollup Server Data**:
* `CLEAR_DATA_KEY` - (optional) Set to clear all persisted data in the full node. Data is only cleared on startup when this variable is set _and_ is different from last startup (e.g. last start up it wasn't set, this time it is or last start up it was set to a different value than it is this start up)

**L1**:
* `L1_NODE_WEB3_URL` - (optional) The Web3 node url (including port) to be used to connect to L1 Ethereum. If this is not present, a local L1 node will be created at runtime using Ganache.
* `LOCAL_L1_NODE_PERSISTENT_DB_PATH` - (optional) Only applicable if `L1_NODE_WEB3_URL` is not set. Path to store local L1 ganache instance's data in.
* `LOCAL_L1_NODE_PORT` - (optional) If the L1 node is going to be simulated by running a Ganache instance locally, this is the port it listens on. If not set, this defaults to 7545.
* `L1_SEQUENCER_PRIVATE_KEY` - (optional) Set to provide a PK to use for L1 contract deployment (if not already deployed) and for signing and sending rollup blocks. This takes priority over mnemonic if both are set.
* `L1_SEQUENCER_MNEMONIC` - (optional) Set to provide a mnemonic to use for L1 contract deployment (if not already deployed) and for signing and sending rollup blocks. If not set, this will default to the dev mnemonic `rebel talent argue catalog maple duty file taxi dust hire funny steak`.
* `L1_TO_L2_TRANSACTION_PASSER_ADDRESS` - (optional) Set to point to the deployed L1 to L2 transaction passer contract address. If not set, this will be the second contract deployed from the sequencer wallet on startup.
* `L2_TO_L1_MESSAGE_RECEIVER_ADDRESS` - (optional) Set to point to the deployed L2 to L1 transaction receiver contract address on L1. If not set, this will be the first contract deployed from the sequencer wallet on startup.
* `L2_TO_L1_MESSAGE_FINALITY_DELAY_IN_BLOCKS` - (optional) The number of additional L1 block confirmations after which a message passed from L2 to L1 will be considered final. If not set, this defaults to `0`.

**L2**:
* `L2_NODE_WEB3_URL` - (optional) The Web3 node url (including port) to be used to connect to the L2 node. If this is not present, a local L2 node will be created at runtime using Ganache.
* `L2_RPC_SERVER_HOST` - (optional) The hostname / IP address of the RPC server exposed by this process. If not provided, this defaults to `0.0.0.0`.
* `L2_RPC_SERVER_PORT` - (optional) The port to expose the L2 RPC server on. If not provided, this defaults to 8545.
* `L2_RPC_SERVER_PERSISTENT_DB_PATH` - (required) The path to store persistent data procesed by this RPC server. Note: This server is ephemeral in unit tests through the [Truffle Provider Wrapper package](https://github.com/ethereum-optimism/optimism-monorepo/tree/master/packages/ovm-truffle-provider-wrapper) and [passing a test provider into the Web3 RPC Handler](https://github.com/ethereum-optimism/optimism-monorepo/blob/master/packages/rollup-full-node/src/app/test-web3-rpc-handler.ts#L43)
* `L2_WALLET_PRIVATE_KEY` - (optional) Set to provide a PK to use for L2 contract deployment (if not already deployed) and for signing and sending L2 transactions. This takes priority over PK path and mnemonic if multiple are set.
* `L2_WALLET_PRIVATE_KEY_PATH` - (optional) The path to the private key file from which the L2 wallet private key can be read. This file is assumed to only contain the private key in UTF-8 hexadecimal characters.
* `L2_WALLET_MNEMONIC` - (optional) Set to provide a mnemonic to use for L2 contract deployment (if not already deployed) and for signing and sending rollup blocks. If not set and `L2_NODE_WEB3_URL` is not set, the default Ganache wallet will be used with the Ganache local node created at runtime.
* `LOCAL_L2_NODE_PERSISTENT_DB_PATH` - (optional) If a local L2 node is to be run, this may be set to persist the state of the local node so as to be able to stop the node and restart it with the same state.

# Building
Run `yarn build` to build the code. Note: `yarn all` may be used to build and run tests.

## Building Docker Image
_Make sure you're in the base directory_ (`cd ../..`)

Run `docker build -t optimism/rollup-full-node .` to build and tag the fullnode.
You may also use `docker-compose up --build` to build and run the docker containers with default settings listed in the `docker-compose.yml` file.

### Pushing Image to AWS Registry:
Note: Image registration and deployment to our internal dev environment is done automatically upon merge to the `master` branch.

Make sure the [AWS CLI](https://docs.aws.amazon.com/cli/latest/userguide/cli-chap-install.html) is installed and [configured](https://docs.aws.amazon.com/cli/latest/userguide/cli-chap-configure.html#cli-quick-configuration)

1. Make sure you're authenticated: 
    ```
    aws ecr get-login-password --region us-east-2 | docker login --username AWS --password-stdin <aws_account_id>.dkr.ecr.us-east-2.amazonaws.com/optimism/rollup-full-node
    ```
2. Build and tag latest: 
    ```
    docker build -t optimism/rollup-full-node .
    ```
3. Tag the build: 
    ```
    optimism/rollup-full-node:latest <aws_account_id>.dkr.ecr.us-east-2.amazonaws.com/optimism/rollup-full-node:latest
    ```
4. Push tag to ECR:
    ```
    docker push <aws_account_id>.dkr.ecr.us-east-2.amazonaws.com/optimism/rollup-full-node:latest
    ``` 

## Running in Docker
_Make sure you're in the base directory_ (`cd ../..`)

Run `docker-compose up --build` to build and run. If you don't need to build the full node or geth, omit the `--build`

When the containers are up, you should see the following output:
```
rollup-full-node_1  | <timestamp> info:rollup-fullnode Listening at http://0.0.0.0:8545
```

You can run a simple connectivity test against the rollup node by running:
```
curl -H "Content-Type: application/json" -d '{"jsonrpc": "2.0", "id": 9999999, "method": "net_version"}' http://0.0.0.0:8545
```
which should yield the response:
```
{"id":9999999,"jsonrpc":"2.0","result":"108"}
```

# Testing
Run `yarn test` to run the unit tests.

# Running the Fullnode Server (outside of docker)
Run `yarn server:fullnode` to run the fullnode server.

