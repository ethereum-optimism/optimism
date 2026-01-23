#!/usr/bin/env bash
set +e  # Disable immediate exit on error

# Array of crates to check
crates_to_check=(
    #reth-codecs-derive
    #reth-primitives
    #reth-primitives-traits
    #reth-network-peers
    #reth-trie-common
    #reth-trie-sparse
    #reth-chainspec
    #reth-consensus
    #reth-consensus-common
    #reth-prune-types
    #reth-static-file-types
    #reth-storage-errors
    #reth-execution-errors
    #reth-errors
    #reth-execution-types
    #reth-db-models
    #reth-evm
    #reth-revm
    #reth-storage-api

    ## ethereum
    #reth-evm-ethereum
    #reth-ethereum-forks
    #reth-ethereum-primitives
    #reth-ethereum-consensus
    #reth-stateless

    ## optimism
    #reth-optimism-chainspec
    #reth-optimism-forks
    #reth-optimism-consensus
    #reth-optimism-primitives
    #reth-optimism-evm
)

# Array to hold the results
results=()
# Flag to track if any command fails
any_failed=0

for crate in "${crates_to_check[@]}"; do
  cmd="cargo +stable build -p $crate --target riscv32imac-unknown-none-elf --no-default-features"

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
