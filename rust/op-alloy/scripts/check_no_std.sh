#!/usr/bin/env bash
set -eo pipefail

no_std_packages=(
  op-alloy
  op-alloy-consensus
  op-alloy-rpc-types
  op-alloy-rpc-types-engine
)

for package in "${no_std_packages[@]}"; do
  cmd="cargo +stable build -p $package --target riscv32imac-unknown-none-elf --no-default-features"
  if [ -n "$CI" ]; then
    echo "::group::$cmd"
  else
    printf "\n%s:\n  %s\n" "$package" "$cmd"
  fi

  $cmd

  if [ -n "$CI" ]; then
    echo "::endgroup::"
  fi
done
