#!/usr/bin/env bash
#
# Cross-language differential test for Interop activation deposits.
#
# Runs both the Go and Rust dumpers and asserts byte-identical stdout.
# Replaces a Go unit test that triggered a Rust build on every `derive`
# package test run.
#
# Exit status:
#   0  outputs match
#   1  outputs differ (diff is printed to stdout)
#   2  one of the dumpers failed to run (its stderr is printed)

set -euo pipefail

# Resolve the monorepo root: walk up from this script until a directory
# containing both `go.mod` and `rust/` is found.
script_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
repo_root=$script_dir
while [[ "$repo_root" != "/" ]]; do
  if [[ -f "$repo_root/go.mod" && -d "$repo_root/rust" ]]; then
    break
  fi
  repo_root=$(dirname "$repo_root")
done
if [[ "$repo_root" == "/" ]]; then
  echo "error: could not locate repo root from $script_dir" >&2
  exit 2
fi

go_out=$(mktemp)
go_err=$(mktemp)
rust_out=$(mktemp)
rust_err=$(mktemp)
trap 'rm -f "$go_out" "$go_err" "$rust_out" "$rust_err"' EXIT

# Capture stdout into the comparison file and stderr separately. `go run` and
# `cargo run` both print build progress to stderr (e.g. `go: downloading ...`
# on a cold module cache), which must not pollute the diff input.

echo "==> running Go dumper"
if ! (cd "$repo_root" && go run ./op-node/cmd/interop-deposits-dump) >"$go_out" 2>"$go_err"; then
  echo "error: Go dumper failed" >&2
  cat "$go_err" >&2
  exit 2
fi

echo "==> running Rust dumper"
if ! (cd "$repo_root/rust/kona" && cargo run --quiet -p kona-hardforks --example interop-deposits-dump) >"$rust_out" 2>"$rust_err"; then
  echo "error: Rust dumper failed" >&2
  cat "$rust_err" >&2
  exit 2
fi

echo "==> diffing"
if diff -u "$go_out" "$rust_out"; then
  echo "OK: Go and Rust Interop activation dumps are byte-identical"
  exit 0
fi
echo "error: Go and Rust Interop activation dumps differ (see diff above)" >&2
exit 1
