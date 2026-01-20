#!/usr/bin/env bash
set +e  # Disable immediate exit on error

# Array of crates to compile
crates=($(cargo metadata --format-version=1 --no-deps | jq -r '.packages[].name' | grep '^reth' | sort))

# Array of crates to exclude
# Used with the `contains` function.
# shellcheck disable=SC2034
exclude_crates=(
  # The following require investigation if they can be fixed
  reth-basic-payload-builder
  reth-bench
  reth-bench-compare
  op-reth-proof-bench
  reth-cli
  reth-cli-commands
  reth-cli-runner
  reth-consensus-debug-client
  reth-db-common
  reth-discv4
  reth-discv5
  reth-dns-discovery
  reth-downloaders
  reth-e2e-test-utils
  reth-engine-service
  reth-engine-tree
  reth-engine-util
  reth-eth-wire
  reth-ethereum-cli
  reth-ethereum-payload-builder
  reth-etl
  reth-exex
  reth-exex-test-utils
  reth-ipc
  reth-net-nat
  reth-network
  reth-node-api
  reth-node-builder
  reth-node-core
  reth-node-ethereum
  reth-node-events
  reth-node-metrics
  reth-optimism-cli
  reth-optimism-flashblocks
  reth-optimism-node
  reth-optimism-payload-builder
  reth-optimism-rpc
  reth-optimism-storage
  reth-rpc
  reth-rpc-api
  reth-rpc-api-testing-util
  reth-rpc-builder
  reth-rpc-convert
  reth-rpc-e2e-tests
  reth-rpc-engine-api
  reth-rpc-eth-api
  reth-rpc-eth-types
  reth-rpc-layer
  reth-stages
  reth-engine-local
  reth-ress-protocol
  reth-ress-provider
  # The following are not supposed to be working
  reth # all of the crates below
  reth-storage-rpc-provider
  reth-invalid-block-hooks # reth-provider
  reth-libmdbx # mdbx
  reth-mdbx-sys # mdbx
  reth-payload-builder # reth-metrics
  reth-provider # tokio
  reth-prune # tokio
  reth-prune-static-files # reth-provider
  reth-stages-api # reth-provider, reth-prune
  reth-static-file # tokio
  reth-transaction-pool # c-kzg
  reth-payload-util # reth-transaction-pool
  reth-trie-parallel # tokio
  reth-trie-sparse-parallel # rayon
  reth-testing-utils
  reth-optimism-exex # reth-exex and reth-optimism-trie
  reth-optimism-trie # reth-trie
  reth-optimism-txpool # reth-transaction-pool
  reth-era-downloader # tokio
  reth-era-utils # tokio
  reth-tracing-otlp
  reth-node-ethstats
)

# Array to hold the results
results=()
# Flag to track if any command fails
any_failed=0

# Function to check if a value exists in an array
contains() {
  local array="$1[@]"
  local seeking=$2
  local in=1
  for element in "${!array}"; do
    if [[ "$element" == "$seeking" ]]; then
      in=0
      break
    fi
  done
  return $in
}

for crate in "${crates[@]}"; do
  if contains exclude_crates "$crate"; then
    results+=("3:⏭️:$crate")
    continue
  fi

  cmd="cargo +stable build -p $crate --target wasm32-wasip1 --no-default-features"

  if [ -n "$CI" ]; then
    echo "::group::$cmd"
  else
    printf "\n%s:\n  %s\n" "$crate" "$cmd"
  fi

  set +e  # Disable immediate exit on error
  # Run the command and capture the return code
  $cmd
  ret_code=$?
  set -e  # Re-enable immediate exit on error

  # Store the result in the dictionary
  if [ $ret_code -eq 0 ]; then
    results+=("1:✅:$crate")
  else
    results+=("2:❌:$crate")
    any_failed=1
  fi

  if [ -n "$CI" ]; then
    echo "::endgroup::"
  fi
done

# Sort the results by status and then by crate name
IFS=$'\n' sorted_results=($(sort <<<"${results[*]}"))
unset IFS

# Print summary
echo -e "\nSummary of build results:"
for result in "${sorted_results[@]}"; do
  status="${result#*:}"
  status="${status%%:*}"
  crate="${result##*:}"
  echo "$status $crate"
done

# Exit with a non-zero status if any command fails
exit $any_failed
