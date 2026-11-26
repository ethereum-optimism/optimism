#!/usr/bin/env bash
# Compare the OP Stack Go and Rust fee and DA-footprint implementations.
set -euo pipefail

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
work_dir=$(mktemp -d)
trap 'rm -rf "$work_dir"' EXIT

echo "==> generating the shared corpus and Go results"
(cd "$repo_root" && go run ./op-core/fees/cmd/fee-diff-dump "$work_dir/corpus.jsonl") > "$work_dir/go.out"

echo "==> computing Rust results"
(cd "$repo_root/rust" && cargo run --locked --quiet -p op-revm --example fee-diff-dump) \
  < "$work_dir/corpus.jsonl" > "$work_dir/rust.out"

echo "==> diffing (Bedrock, Regolith, Ecotone, Fjord, Isthmus operator fee, Jovian operator fee, DA footprint)"
test -s "$work_dir/go.out"
diff -u "$work_dir/go.out" "$work_dir/rust.out"
echo "OK: Go and Rust agree on $(wc -l < "$work_dir/go.out" | tr -d '[:space:]') cases"
