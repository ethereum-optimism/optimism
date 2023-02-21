#!/usr/bin/env bash

# This script starts a local devnet using Docker Compose. We have to use
# this more complicated Bash script rather than Compose's native orchestration
# tooling because we need to start each service in a specific order, and specify
# their configuration along the way. The order is:
#
# 1. Start L1.
# 2. Compile contracts.
# 3. Deploy the contracts to L1 if necessary.
# 4. Start L2, inserting the compiled contract artifacts into the genesis.
# 5. Get the genesis hashes and timestamps from L1/L2.
# 6. Generate the rollup driver's config using the genesis hashes and the
#    timestamps recovered in step 4 as well as the address of the OptimismPortal
#    contract deployed in step 3.
# 7. Start the rollup driver.
# 8. Start the L2 output submitter.
#
# The timestamps are critically important here, since the rollup driver will fill in
# empty blocks if the tip of L1 lags behind the current timestamp. This can lead to
# a perceived infinite loop. To get around this, we set the timestamp to the current
# time in this script.
#
# This script is safe to run multiple times. It stores state in `.devnet`, and
# contracts-bedrock/deployments/devnetL1.
#
# Don't run this script directly. Run it using the makefile, e.g. `make devnet-up`.
# To clean up your devnet, run `make devnet-clean`.

set -eu

L1_URL="http://localhost:8545"
L2_URL="http://localhost:9545"

OP_NODE="$PWD/op-node"
CONTRACTS_BEDROCK="$PWD/packages/contracts-bedrock"
NETWORK=devnetL1
DEVNET="$PWD/.devnet"

# Helper method that waits for a given URL to be up. Can't use
# cURL's built-in retry logic because connection reset errors
# are ignored unless you're using a very recent version of cURL
function wait_up {
  echo -n "Waiting for $1 to come up..."
  i=0
  until curl -s -f -o /dev/null "$1"
  do
    echo -n .
    sleep 0.25

    ((i=i+1))
    if [ "$i" -eq 300 ]; then
      echo " Timeout!" >&2
      exit 1
    fi
  done
  echo "Done!"
}

mkdir -p ./.devnet

# Regenerate the L1 genesis file if necessary. The existence of the genesis
# file is used to determine if we need to recreate the devnet's state folder.
if [ ! -f "$DEVNET/done" ]; then
  echo "Regenerating genesis files"

  TIMESTAMP=$(date +%s | xargs printf '0x%x')
  cat "$CONTRACTS_BEDROCK/deploy-config/devnetL1.json" | jq -r ".l1GenesisBlockTimestamp = \"$TIMESTAMP\"" > /tmp/bedrock-devnet-deploy-config.json

  (
    cd "$OP_NODE"
    go run cmd/main.go genesis devnet \
        --deploy-config /tmp/bedrock-devnet-deploy-config.json \
        --outfile.l1 $DEVNET/genesis-l1.json \
        --outfile.l2 $DEVNET/genesis-l2.json \
        --outfile.rollup $DEVNET/rollup.json
    touch "$DEVNET/done"
  )
fi

# Bring up L1.
(
  cd ops-bedrock
  echo "Bringing up L1..."
  DOCKER_BUILDKIT=1 docker-compose build --progress plain
  docker-compose up -d l1
  wait_up $L1_URL
)

# Bring up L2.
(
  cd ops-bedrock
  echo "Bringing up L2..."
  docker-compose up -d l2
  wait_up $L2_URL
)

L2OO_ADDRESS="0x6900000000000000000000000000000000000000"

# Bring up everything else.
(
  cd ops-bedrock
  echo "Bringing up devnet..."
  L2OO_ADDRESS="$L2OO_ADDRESS" \
      docker-compose up -d op-proposer op-batcher

  echo "Bringing up stateviz webserver..."
  docker-compose up -d stateviz
)

echo "Devnet ready."
