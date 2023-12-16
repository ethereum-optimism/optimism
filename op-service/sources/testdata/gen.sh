#!/bin/bash
set -euo pipefail

SOURCE_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
cd "$SOURCE_DIR"
export ETH_RPC_URL=https://ethereum-goerli-rpc.allthatnode.com

jq_mutate() {
    local name="$1"
    jq -c "$2" "$name" > "$name.tmp" && mv "$name.tmp" "$name"
}

success_case() {
  # just format the files
  jq_mutate "$1" .
  jq_mutate "$2" .
}

bad_receipts_root() {
    local data_file="$1"
    local metadata_file="$2"
    jq_mutate "$data_file" '. + {"receiptsRoot": "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}'
    jq_mutate "$metadata_file" '. + {"fail": true}'
}

bad_withdrawals_root() {
    local data_file="$1"
    local metadata_file="$2"
    jq_mutate "$data_file" '. + {"withdrawalsRoot": "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}'
    jq_mutate "$metadata_file" '. + {"fail": true}'
}

bad_transactions_root() {
    local data_file="$1"
    local metadata_file="$2"
    jq_mutate "$data_file" '. + {"transactionsRoot": "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}'
    jq_mutate "$metadata_file" '. + {"fail": true}'
}

generate_test_vector() {
    local name="$1"
    local blockhash="$2"
    local fulltxs="$3"
    local mutation_func="$4"

    local metadata_file=""
    local data_file=""

    if [[ "$fulltxs" == true ]]; then
        metadata_file="data/blocks/${name}_metadata.json"
        data_file="data/blocks/${name}_data.json"
    else
        metadata_file="data/headers/${name}_metadata.json"
        data_file="data/headers/${name}_data.json"
    fi

    echo "{\"name\": \"$name\"}" > "$metadata_file"

    cast rpc eth_getBlockByHash "$blockhash" "$fulltxs" > "$data_file"

    # Mutate data using the provided function
    $mutation_func "$data_file" "$metadata_file"
}

mkdir -p data/headers

# Headers
generate_test_vector "pre-shanghai-success" "0x9ef7cd2241202b919a0e51240818a8666c73f7ce4b908931e3ae6d26d30f7663" false success_case
generate_test_vector "pre-shanghai-bad-transactions" "0x9ef7cd2241202b919a0e51240818a8666c73f7ce4b908931e3ae6d26d30f7663" false bad_transactions_root
generate_test_vector "pre-shanghai-bad-receipts" "0x9ef7cd2241202b919a0e51240818a8666c73f7ce4b908931e3ae6d26d30f7663" false bad_receipts_root
generate_test_vector "post-shanghai-success" "0xa16c6bcda4fdca88b5761965c4d724f7afc6a6900d9051a204e544870adb3452" false success_case
generate_test_vector "post-shanghai-bad-withdrawals" "0xa16c6bcda4fdca88b5761965c4d724f7afc6a6900d9051a204e544870adb3452" false bad_withdrawals_root
generate_test_vector "post-shanghai-bad-transactions" "0xa16c6bcda4fdca88b5761965c4d724f7afc6a6900d9051a204e544870adb3452" false  bad_transactions_root
generate_test_vector "post-shanghai-bad-receipts" "0xa16c6bcda4fdca88b5761965c4d724f7afc6a6900d9051a204e544870adb3452" false bad_receipts_root
