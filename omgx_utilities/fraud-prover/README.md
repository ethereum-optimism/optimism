# Fraud Prover

Contains an executable fraud prover. This repo allows you to:

1. Generate a fraud prover Docker
2. Run that docker and the associated L2 in Verifier mode, pointed at a local OMGX.
3. Inject fraudulant state roots to debug the currently non-operational fraud-prover.

## 1. FIRST TERMINAL WINDOW: Building & Running a local system with Verifier and Fraud Prover

Make sure dependencies are installed and everything is built - just run `yarn` and `yarn build` in the top directory  (/optimsm). Then spin up the entire system with the L2, the Verifier, and the Fraud Prover:

```bash

./up_local.sh #to spin up and verify a local OMGX

```

At this point you will have the *L1*, *L2*, the *Verifier*, the *data_transport_layer*, the *batch_submitter*, *the *message_relayer*, the *deployer*, and the *fraud_prover* all running and talking to one another. So far so good. 

## 2. SECOND TERMINAL WINDOW: Injecting Fraudulant State Roots

The `docker-compose-local.env.yaml` sets:

```bash

#this is the address that will trigger the batch_submitter to inject fake state roots
x-var: &FRAUD_SUBMISSION_ADDRESS
  FRAUD_SUBMISSION_ADDRESS=0xb0dd88dfcc929a78fec13daa1bd77843e267c729

```

Any transactions fron this wallet will cause the `batch_submitter` to submit a bad state root (`0xbad1bad1......`) instead of the correct state root. Normnally, the batch submitter only does this once, but we patched the batch submitter to allow many fraudulant transaction to be submitted. **Open a second terminal** and then:

```bash

yarn build:fraud
yarn deploy 

```

You will see a few contracts get deployed, and you will see *Mr. Fraud* transferring some funds to Alice. Running `yarn deploy` **does** trigger the batch submitter to inject a bad state root, but then, the system reverts, proably due to indexing and other issues. Note that hardhat needs a `.env` for the deploy - see the `example.env` for working settings.

```bash
#expected terminal output

  System setup
L1ERC20 deployed to: 0x196A1b2750D9d39ECcC7667455DdF1b8d5D65696
L2DepositedERC20 deployed to: 0x3e4CFaa8730092552d9425575E49bB542e329981
L1ERC20Gateway deployed to: 0x60ba97F7cE0CD19Df71bAd0C3BE9400541f6548E
L2 ERC20 initialized: 0x2b4793dfe3a8241d776cacd604904c30601f7a895debc59ff66bef1b187d3899
 Bob Depositing L1 ERC20 to L2...
 On L1, Bob has: BigNumber { _hex: '0x204fce5e3e25026110000000', _isBigNumber: true }
 On L2, Bob has: BigNumber { _hex: '0x00', _isBigNumber: true }
 On L1, Bob now has: BigNumber { _hex: '0x204fce561c79f51cfb680000', _isBigNumber: true }
 On L2, Bob now has: BigNumber { _hex: '0x0821ab0d4414980000', _isBigNumber: true }
    ✓ Bob Approve and Deposit ERC20 from L1 to L2 (5233ms)
    ✓ should transfer ERC20 token to Alice and Fraud (4166ms)
 On L2, Alice has: 10000000000000000000
 On L2, Fraud has: 10000000000000000000
 On L2, Alice now has: 7000000000000000000
 On L2, Fraud now has: 13000000000000000000
    ✓ should transfer ERC20 token from Alice to Fraud (4119ms)
 On L2, Bob has: 130000000000000000000
 On L2, Fraud has: 13000000000000000000
 On L2, Bob now has: 131000000000000000000
 On L2, Fraud now has: 12000000000000000000
    ✓ should transfer ERC20 token from Fraud to Bob and commit fraud (4123ms)

```

## 3. THIRD TERMINAL WINDOW: Running a local Fraud Prover for rapid debugging purposes

First, *terminate the dockerized fraud-prover service* and then `yarn build` and `yarn start` a fraud-prover from your command line - this makes it much easier and faster to debug, since you get better debug and console.log output, and its easier to make code changes and see what happens. This fraud-prover will also use whatever you set in the .env (which should match of couse with what all the dockerized services are getting from the `docker-compose-local.env.yaml`)

```bash

yarn build
yarn start

```

## CURRENTLY BROKEN AT One of three places - Lib_MerkleTree.sol, OVM_FraudVerifier.sol, or "makeStateTrie for this proof"

If you do all of the above, while using the standard contracts, you will get stuck at *EITHER*:

```bash
# this is in Lib_MerkleTree.sol

require(
  _siblings.length == _ceilLog2(_totalLeaves),
  "Lib_MerkleTree: Total siblings does not correctly correspond to total leaves."
);

```

*OR* you will get through that but then revert at `VM Exception while processing transaction: revert Pre-state root global index must equal to the transaction root global index`:

```bash
# this is in OVM_FraudVerifier.sol

line 134

require (
    _preStateRootBatchHeader.prevTotalElements + _preStateRootProof.index + 1 == _transactionBatchHeader.prevTotalElements + _transactionProof.index,
    "Pre-state root global index must equal to the transaction root global index."
);

```

*OR* you will get stuck at `_makeStateTrie for this proof`. There are also assorted and sundy other ways for this to fail, mostly relating to out of bounds access to various arrays, most notably, in the transactions.

## 4. Generating the Fraud Prover Docker 

To build the Docker:

```bash

docker build . --file Dockerfile.fraud-prover --tag omgx/fraud-prover:latest
docker push omgx/fraud-prover:latest

```

## NOT UPDATED 5. Configuration

All configuration is done via environment variables. See below for more information.

## NOT UPDATED 6. Testing & linting

- See lint errors with `yarn lint`; auto-fix with `yarn lint --fix`

## NOT UPDATED 7. Envs (Need to reconcile with the below - ToDo)

| Environment Variable   | Required? | Default Value         | Description            |
| -----------            | --------- | -------------         | -----------           |
| `FP_WALLET_KEY`        | Yes       | N/A                   | Private key for an account on Layer 1 (Ethereum) to be used to submit fraud proof transactions. |
| `L2_NODE_WEB3_URL`     | No        | http://localhost:9545 | HTTP endpoint for a Layer 2 (Optimism) Verifier node.  |
| `L1_NODE_WEB3_URL`     | No        | http://localhost:8545 | HTTP endpoint for a Layer 1 (Ethereum) node.      |
| `RELAY_GAS_LIMIT`      | No        | 9,000,000             | Maximum amount of gas to provide to fraud proof transactions (except for the "transaction execution" step). |
| `RUN_GAS_LIMIT`        | No        | 9,000,000             | Maximum amount of gas to provide to the "transaction execution" step. |
| `POLLING_INTERVAL`     | No        | 5,000                 | Time (in milliseconds) to wait while polling for new transactions. |
| `L2_BLOCK_OFFSET`      | No        | 1                     | Offset between the `CanonicalTransactionChain` contract on Layer 1 and the blocks on Layer 2. Currently defaults to 1, but will likely be removed as soon as possible. |
| `L1_BLOCK_FINALITY`    | No        | 0                     | Number of Layer 1 blocks to wait before considering a given event. |
| `L1_START_OFFSET`      | No        | 0                     | Layer 1 block number to start scanning for transactions from. |
| `FROM_L2_TRANSACTION_INDEX` | No        | 0                     | Layer 2 block number to start scanning for transactions from. |

## NOT UPDATED 8. Local testing

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
      - FP_WALLET_KEY=0xYOUR_FP_WALLET_KEY_HERE
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
