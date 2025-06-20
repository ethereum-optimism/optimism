#!/usr/bin/env bash
set -euxo pipefail

source .env

ANVIL_PORT=8545
ANVIL_URL=http://localhost:$ANVIL_PORT

# All variables must be set

OP_ROOT="${1:-$(pwd)/..}"
OP_ROOT=$(realpath "${OP_ROOT}")

DEPLOYMENT_DIR="${OP_ROOT}/espresso/deployment"
DEPLOYER_DIR="${DEPLOYMENT_DIR}/deployer"
L1_CONFIG_DIR="${DEPLOYMENT_DIR}/l1-config"
OP_CONFIG_DIR="${DEPLOYMENT_DIR}/l2-config"
mkdir -p "${DEPLOYER_DIR}"
mkdir -p "${OP_CONFIG_DIR}"
mkdir -p "${L1_CONFIG_DIR}"

ANVIL_STATE_FILE="${DEPLOYMENT_DIR}/anvil_state.json"
ARTIFACTS_DIR="file:///${OP_ROOT}/packages/contracts-bedrock/forge-artifacts"

# Start anvil in dev mode and save PID to kill later
anvil --port $ANVIL_PORT --chain-id "${L1_CHAIN_ID}" --disable-gas-limit --disable-code-size-limit --dump-state "${ANVIL_STATE_FILE}" &
ANVIL_PID=$!
echo "Started anvil in dev mode with PID: $ANVIL_PID"

# Function to cleanup anvil process
cleanup() {
    if kill -0 $ANVIL_PID > /dev/null 2>&1; then
        echo "Stopping anvil (PID: $ANVIL_PID)"
        kill $ANVIL_PID
    fi
}
trap cleanup EXIT

# Give anvil a moment to start up
sleep 1

cast rpc anvil_setBalance "${OPERATOR_ADDRESS}" 0x100000000000000000000000000000000000

op-deployer bootstrap proxy \
                      --l1-rpc-url="${ANVIL_URL}" \
                      --private-key="${OPERATOR_PRIVATE_KEY}" \
                      --artifacts-locator="${ARTIFACTS_DIR}" \
                      --proxy-owner="${OPERATOR_ADDRESS}"

export LOG_LEVEL=debug

op-deployer bootstrap superchain \
                      --l1-rpc-url="${ANVIL_URL}" \
                      --private-key="${OPERATOR_PRIVATE_KEY}" \
                      --artifacts-locator="${ARTIFACTS_DIR}" \
                      --outfile="${DEPLOYER_DIR}/bootstrap_superchain.json" \
                      --superchain-proxy-admin-owner="${OPERATOR_ADDRESS}" \
                      --protocol-versions-owner="${OPERATOR_ADDRESS}" \
                      --guardian="${OPERATOR_ADDRESS}"

op-deployer bootstrap implementations \
                      --l1-rpc-url="${ANVIL_URL}" \
                      --private-key="${OPERATOR_PRIVATE_KEY}" \
                      --artifacts-locator="${ARTIFACTS_DIR}" \
                      --protocol-versions-proxy=`jq -r .protocolVersionsProxyAddress < ${DEPLOYER_DIR}/bootstrap_superchain.json` \
                      --superchain-config-proxy=`jq -r .superchainConfigProxyAddress < ${DEPLOYER_DIR}/bootstrap_superchain.json` \
                      --upgrade-controller="${OPERATOR_ADDRESS}" \
                      --outfile="${DEPLOYER_DIR}/bootstrap_implementations.json"

op-deployer init --l1-chain-id "${L1_CHAIN_ID}" \
                 --l2-chain-ids "${L2_CHAIN_ID}" \
                 --intent-type standard-overrides \
                 --outdir ${DEPLOYER_DIR}

dasel put -f "${DEPLOYER_DIR}/intent.toml" -s .l1ContractsLocator -v "${ARTIFACTS_DIR}"
dasel put -f "${DEPLOYER_DIR}/intent.toml" -s .l2ContractsLocator -v "${ARTIFACTS_DIR}"
dasel put -f "${DEPLOYER_DIR}/intent.toml" -s .fundDevAccounts -t bool -v true
dasel put -f "${DEPLOYER_DIR}/intent.toml" -s .chains.[0].baseFeeVaultRecipient -v "${OPERATOR_ADDRESS}"
dasel put -f "${DEPLOYER_DIR}/intent.toml" -s .chains.[0].l1FeeVaultRecipient -v "${OPERATOR_ADDRESS}"
dasel put -f "${DEPLOYER_DIR}/intent.toml" -s .chains.[0].sequencerFeeVaultRecipient -v "${OPERATOR_ADDRESS}"
dasel put -f "${DEPLOYER_DIR}/intent.toml" -s .chains.[0].roles.systemConfigOwner -v "${OPERATOR_ADDRESS}"
dasel put -f "${DEPLOYER_DIR}/intent.toml" -s .chains.[0].roles.unsafeBlockSigner -v "${OPERATOR_ADDRESS}"
dasel put -f "${DEPLOYER_DIR}/intent.toml" -s .chains.[0].roles.batcher -v "${OPERATOR_ADDRESS}"
dasel put -f "${DEPLOYER_DIR}/intent.toml" -s .chains.[0].roles.proposer -v "${OPERATOR_ADDRESS}"

op-deployer apply --l1-rpc-url "${ANVIL_URL}" \
                  --workdir "${DEPLOYER_DIR}" \
                  --private-key="${OPERATOR_PRIVATE_KEY}"

kill $ANVIL_PID

sleep 1

"${OP_ROOT}/espresso/scripts/reshape-allocs.jq" \
                  <(jq .accounts "${ANVIL_STATE_FILE}") \
                  | jq '{ "alloc": map_values(.state) }' \
                  > "${DEPLOYMENT_DIR}/deployer_allocs.json"

jq -s 'reduce .[] as $item ({}; . * $item)'        \
        <(jq '{ "alloc": map_values(.state) }' "${OP_ROOT}/espresso/environment/allocs.json") \
        "${DEPLOYMENT_DIR}/deployer_allocs.json"            \
        "${OP_ROOT}/espresso/docker/l1-geth/l1-genesis-devnet.json" \
        > "${L1_CONFIG_DIR}/genesis.json"

dasel put -f "${L1_CONFIG_DIR}/genesis.json" -s .timestamp -v $(printf '0x%x\n' $(date +%s))

eth-beacon-genesis devnet \
                  --eth1-config "${L1_CONFIG_DIR}/genesis.json" \
                  --config "${OP_ROOT}/espresso/docker/l1-geth/beacon-config.yaml" \
                  --mnemonics "${OP_ROOT}/espresso/docker/l1-geth/mnemonics.yaml" \
                  --state-output "${L1_CONFIG_DIR}/genesis.ssz"
cp -r ${OP_ROOT}/espresso/docker/l1-geth/beacon-config.yaml "${L1_CONFIG_DIR}/config.yaml"

rm -rf ${L1_CONFIG_DIR}/keystore && \
eth2-val-tools keystores --out-loc ${L1_CONFIG_DIR}/keystore \
                         --source-mnemonic "$(yq -r '.[0].mnemonic' "${OP_ROOT}/espresso/docker/l1-geth/mnemonics.yaml")" \
                         --source-min 0 \
                         --source-max 1

if [[ ! -f "${OP_CONFIG_DIR}/jwt.txt" ]]; then
    openssl rand -hex 32 > "${OP_CONFIG_DIR}/jwt.txt"
fi

if [[ ! -f "${L1_CONFIG_DIR}/jwt.txt" ]]; then
    openssl rand -hex 32 > "${L1_CONFIG_DIR}/jwt.txt"
fi

op-deployer inspect rollup --workdir "${DEPLOYER_DIR}" --outfile "${OP_CONFIG_DIR}/rollup.json" $L2_CHAIN_ID
op-deployer inspect genesis --workdir "${DEPLOYER_DIR}" --outfile "${OP_CONFIG_DIR}/genesis.json" $L2_CHAIN_ID
