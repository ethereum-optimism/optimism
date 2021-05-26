- [Fraud Prover](#fraud-prover)
  * [1. Building & Running from the command line](#1-building---running-from-the-command-line)
  * [2. Generating the Fraud Prover Docker](#2-generating-the-fraud-prover-docker)
  * [3. Running the Fraud Prover Service](#3-running-the-fraud-prover-service)
  * [4. Injecting Fraudulant State Roots](#4-injecting-fraudulant-state-roots)
  * [5. Configuration](#5-configuration)
  * [6. Testing & linting](#6-testing---linting)
  * [7. Envs (Need to reconcile with the below - ToDo)](#7-envs--need-to-reconcile-with-the-below---todo-)
  * [8. Local testing](#8-local-testing)
    + [local.yaml Settings](#localyaml-settings)
    + [local.env.yaml Settings](#localenvyaml-settings)
    + [Fraud Prover spinup wait-for-l1-and-l2.sh](#fraud-prover-spinup-wait-for-l1-and-l2sh)
    + [Fraud Prover Dockerfile.fraud_prover](#fraud-prover-dockerfilefraud-prover)

# Fraud Prover

Contains an executable fraud prover. This repo allows you to:

1. Generate a fraud prover Docker
2. Run that docker and the associated L2 in Verifier mode, pointed at a local OMGX.
3. Run that docker and the associated L2 in Verifier mode, pointed at the Rinkeby OMGX.

## 1. Building & Running a local System with Fraud Prover

1. Make sure dependencies are installed - just run `yarn` in the base directory
2. Build `yarn build`

Then spin up the entire system with the L2, the Verifier, and the Fraud Prover:

```bash

./up_local.sh #to spin up and verify a local OMGX

#or...

./up_rinkeby.sh #to verify OMGX Rinkeby

```

Or, you can run just the Fraud Prover from the command line. 

```bash

yarn start

```

## 2. Generating the Fraud Prover Docker 

To build the Docker:

```bash

docker build . --file Dockerfile.fraud_prover --tag omgx/fraud-prover:latest
docker push omgx/fraud-prover:latest

```

## 3. Injecting Fraudulant State Roots

This will set up some basic smart contracts and transfer some funds. Critically, the Fraud wallet address will be used in a transaction, triggering the Batch_submitter to submit a garbage state root to L1. Unfortunately, right now, if you do that, the system seems completely oblivious to that. Open a second terminal, and then:

```bash

yarn build:fraud
yarn deploy 

```

Running `yarn deploy` **does** trigger the batch submitter to inject a bad state root, but then, nothing seems to happen in either the Verifier or Fraud_prover.

## 4. TBD

## 5. Configuration

All configuration is done via environment variables. See below for more information.

## 6. Testing & linting

- See lint errors with `yarn lint`; auto-fix with `yarn lint --fix`

## 7. Envs (Need to reconcile with the below - ToDo)

| Environment Variable   | Required? | Default Value         | Description            |
| -----------            | --------- | -------------         | -----------           |
| `L1_WALLET_KEY`        | Yes       | N/A                   | Private key for an account on Layer 1 (Ethereum) to be used to submit fraud proof transactions. |
| `L2_NODE_WEB3_URL`     | No        | http://localhost:9545 | HTTP endpoint for a Layer 2 (Optimism) Verifier node.  |
| `L1_NODE_WEB3_URL`     | No        | http://localhost:8545 | HTTP endpoint for a Layer 1 (Ethereum) node.      |
| `RELAY_GAS_LIMIT`      | No        | 9,000,000             | Maximum amount of gas to provide to fraud proof transactions (except for the "transaction execution" step). |
| `RUN_GAS_LIMIT`        | No        | 9,000,000             | Maximum amount of gas to provide to the "transaction execution" step. |
| `POLLING_INTERVAL`     | No        | 5,000                 | Time (in milliseconds) to wait while polling for new transactions. |
| `L2_BLOCK_OFFSET`      | No        | 1                     | Offset between the `CanonicalTransactionChain` contract on Layer 1 and the blocks on Layer 2. Currently defaults to 1, but will likely be removed as soon as possible. |
| `L1_BLOCK_FINALITY`    | No        | 0                     | Number of Layer 1 blocks to wait before considering a given event. |
| `L1_START_OFFSET`      | No        | 0                     | Layer 1 block number to start scanning for transactions from. |
| `FROM_L2_TRANSACTION_INDEX` | No        | 0                     | Layer 2 block number to start scanning for transactions from. |

## 8. Local testing

The fraud prover will first connect to the relevant chains and then look for mismatched state roots. Note that the *Fraud Prover* does not connect to the *Sequencer*, rather, it connects to the *Verifier*, and the Verifier in turn is looking at the L1. Assuming _your sequencer is not fraudulant_, the standard Fraud Prover output looks like this:

```

{"level":30,"time":1619122304289,"msg":"Looking for mismatched state roots..."}
{"level":30,"time":1619122304295,"nextAttemptInS":5,"msg":"Did not find any mismatched state roots"}
{"level":30,"time":1619122309301,"msg":"Looking for mismatched state roots..."}
{"level":30,"time":1619122309306,"nextAttemptInS":5,"msg":"Did not find any mismatched state roots"}
{"level":30,"time":1619122314311,"msg":"Looking for mismatched state roots..."}

```

When you spin up your local test system some small changes to the generic `local.env.yaml` and `local.yaml` may be needed. Also, you will have to provide two extra files, `wait-for-l1-and-l2.sh`. For your testing conveniance, there is also a `Dockerfile.fraud_prover`.

### local.yaml Settings

Add to your `local.yaml`
```bash
#all the usual things here (L2, Batch submitter, Message Relay, Hardhat, Deployer), but then...

  verifier:
    image: omgx/go-ethereum
    volumes:
      - verifier:/root/.ethereum:rw
    ports:
      - 8045:8045
      - 8046:8046

  fraud_prover:
    image: omgx/fraud-prover:latest

volumes:

  geth:

  verifier:
```

### local.env.yaml Settings

Add to your `local.env.yaml`

```bash

x-var: &L1_NODE_WEB3_URL
  L1_NODE_WEB3_URL=http://l1_chain:9545

x-var: &DEPLOYER_HTTP
  DEPLOYER_HTTP=http://deployer:8080

x-var: &ADDRESS_MANAGER_ADDRESS
  ADDRESS_MANAGER_ADDRESS=0xYOUR_ADDRESS_MANAGER_HERE

services:

#all the usual things here (L2, Batch submitter, Message Relay, Hardhat, Deployer), but then...

  verifier:
    environment:
      - *DEPLOYER_HTTP
      - *L1_NODE_WEB3_URL
      - ROLLUP_VERIFIER_ENABLE=true
      - ETH1_SYNC_SERVICE_ENABLE=true
      - ETH1_CTC_DEPLOYMENT_HEIGHT=8
      - ETH1_CONFIRMATION_DEPTH=0
      - ROLLUP_CLIENT_HTTP=http://data_transport_layer:7878
      - ROLLUP_POLL_INTERVAL_FLAG=3s
      - USING_OVM=true
      - CHAIN_ID=420
      - NETWORK_ID=420
      - DEV=true
      - DATADIR=/root/.ethereum
      - RPC_ENABLE=true
      - RPC_ADDR=verifier
      - RPC_CORS_DOMAIN=*
      - RPC_VHOSTS=*
      - RPC_PORT=8045
      - WS=true
      - WS_ADDR=0.0.0.0
      - IPC_DISABLE=true
      - TARGET_GAS_LIMIT=9000000
      - RPC_API=eth,net,rollup,web3
      - WS_API=eth,net,rollup,web3
      - WS_ORIGINS=*
      - GASPRICE=0
      - NO_USB=true
      - GCMODE=archive
      - NO_DISCOVER=true
      - ROLLUP_STATE_DUMP_PATH=http://deployer:8080/state-dump.latest.json
      - RETRIES=60

  fraud_prover:
    environment:
      - NO_TIMEOUT=true
      - *L1_NODE_WEB3_URL
      - *ADDRESS_MANAGER_ADDRESS
      - L2_NODE_WEB3_URL=http://verifier:8045
      - L1_WALLET_KEY=0xYOUR_FP_WALLET_KEY_HERE
      - POLLING_INTERVAL=5000
      - RUN_GAS_LIMIT=8999999
      - RELAY_GAS_LIMIT=8999999
      - FROM_L2_TRANSACTION_INDEX=0
      - L2_BLOCK_OFFSET=1
      - L1_START_OFFSET=8
      - RETRIES=60

```

### Fraud Prover spinup wait-for-l1-and-l2.sh

```bash

#!/bin/bash

# Copyright Optimism PBC 2020
# MIT License
# github.com/ethereum-optimism

cmd="$@"
JSON='{"jsonrpc":"2.0","id":0,"method":"net_version","params":[]}'

RETRIES=${RETRIES:-50}
until $(curl --silent --fail \
    --output /dev/null \
    -H "Content-Type: application/json" \
    --data "$JSON" "$L1_NODE_WEB3_URL"); do
  sleep 1
  echo "Will wait $((RETRIES--)) more times for $L1_NODE_WEB3_URL to be up..."

  if [ "$RETRIES" -lt 0 ]; then
    echo "Timeout waiting for layer one node at $L1_NODE_WEB3_URL"
    exit 1
  fi
done
echo "Connected to L1 Node at $L1_NODE_WEB3_URL"

RETRIES=${RETRIES:-50}
until $(curl --silent --fail \
    --output /dev/null \
    -H "Content-Type: application/json" \
    --data "$JSON" "$L2_NODE_WEB3_URL"); do
  sleep 1
  echo "Will wait $((RETRIES--)) more times for $L2_NODE_WEB3_URL to be up..."

  if [ "$RETRIES" -lt 0 ]; then
    echo "Timeout waiting for layer two node at $L2_NODE_WEB3_URL"
    exit 1
  fi
done
echo "Connected to L2 Verifier Node at $L2_NODE_WEB3_URL"

if [ ! -z "$DEPLOYER_HTTP" ]; then
    RETRIES=${RETRIES:-50}
    until $(curl --silent --fail \
        --output /dev/null \
        "$DEPLOYER_HTTP/addresses.json"); do
      sleep 1
      echo "Will wait $((RETRIES--)) more times for $DEPLOYER_HTTP to be up..."

      if [ "$RETRIES" -lt 0 ]; then
        echo "Timeout waiting for contract deployment"
        exit 1
      fi
    done
    echo "Contracts are deployed"
    ADDRESS_MANAGER_ADDRESS=$(curl --silent $DEPLOYER_HTTP/addresses.json | jq -r .AddressManager)
    exec env \
        ADDRESS_MANAGER_ADDRESS=$ADDRESS_MANAGER_ADDRESS \
        L1_BLOCK_OFFSET=$L1_BLOCK_OFFSET \
        $cmd
else
    exec $cmd
fi

```

### Fraud Prover Dockerfile.fraud_prover

```

FROM node:14-buster as base

RUN apt-get update && apt-get install -y bash curl jq

FROM base as build

RUN apt-get update && apt-get install -y bash git python build-essential

ADD . /opt/fraud-prover

RUN cd /opt/fraud-prover yarn install yarn build

FROM base

RUN apt-get update && apt-get install -y bash curl jq

COPY --from=build /opt/fraud-prover /opt/fraud-prover

COPY wait-for-l1-and-l2.sh /opt/
RUN chmod +x /opt/wait-for-l1-and-l2.sh
RUN chmod +x /opt/fraud-prover/exec/run.js
RUN ln -s /opt/fraud-prover/exec/run.js /usr/local/bin/

ENTRYPOINT ["/opt/wait-for-l1-and-l2.sh", "run.js"]

```
