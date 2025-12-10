#!/usr/bin/env bash
set -eo pipefail

no_std_packages=(
  # proof crates
  kona-executor
  kona-mpt
  kona-preimage
  kona-proof
  kona-proof-interop

  # protocol crates
  kona-genesis
  kona-hardforks
  kona-registry
  kona-protocol
  kona-derive
  kona-driver
  kona-interop

  # utilities
  kona-serde
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
