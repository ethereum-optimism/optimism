#!/usr/bin/env bash
# Fail if the Rust compiler that builds the shipped images (melange) or the
# op-stack-go Dockerfile drifts from rust/rust-toolchain.toml, which CI tests.
# Compares major.minor only: rust-toolchain.toml pins a channel such as "1.99"
# and melange pins the matching Wolfi stream, such as rust-1.99.
set -euo pipefail

cd "$(dirname "$0")/../.."

toml=rust/rust-toolchain.toml
dockerfile=ops/docker/op-stack-go/Dockerfile
cargo_pipeline=melange/pipelines/cargo/build.yaml
for f in "$toml" "$dockerfile" "$cargo_pipeline"; do
  [[ -f "$f" ]] || {
    echo "$f: not found" >&2
    exit 1
  }
done

toolchain=$(sed -nE 's/^[[:space:]]*channel[[:space:]]*=[[:space:]]*"([0-9]+\.[0-9]+)(\.[0-9]+)?"[[:space:]]*(#.*)?$/\1/p' "$toml")
if [[ ! "$toolchain" =~ ^[0-9]+\.[0-9]+$ ]]; then
  echo "$toml: expected one channel = \"X.Y\" or \"X.Y.Z\"" >&2
  exit 1
fi

fail=0
want_melange="rust-$toolchain"

# Every melange file that builds with cargo, and the cargo pipeline itself,
# must name the Rust compiler, and every Rust entry must be the pinned stream.
for f in melange/*.yaml melange/pipelines/*/*.yaml; do
  entries=$(grep -E "^[[:space:]]*-[[:space:]]*[\"']?rust(-[0-9][^[:space:]\"'#]*)?([=<>~][^[:space:]\"'#]*)?[\"']?[[:space:]]*(#.*)?$" "$f" || true)
  if [[ -z "$entries" ]]; then
    if [[ "$f" == "$cargo_pipeline" ]] || grep -qE '^[[:space:]]*-?[[:space:]]*uses:[[:space:]]*cargo/build' "$f"; then
      echo "$f: builds with cargo but lists no Rust package, want \"$want_melange\"" >&2
      fail=1
    fi
    continue
  fi
  while IFS= read -r line; do
    # Drop quotes and any version constraint such as "=1.99.1-r2".
    pkg=$(sed -E "s/^[[:space:]]*-[[:space:]]*[\"']?([^[:space:]\"'#=<>~]+).*/\1/" <<< "$line")
    if [[ "$pkg" != "$want_melange" ]]; then
      echo "$f: Rust package is \"$pkg\", want \"$want_melange\" to match $toml" >&2
      fail=1
    fi
  done <<< "$entries"
done

images=$(sed -nE 's/^FROM[[:space:]].*rust:([^[:space:]]+).*/\1/p' "$dockerfile")
if [[ -z "$images" ]]; then
  echo "$dockerfile: no FROM rust:<version> line found" >&2
  fail=1
fi
while IFS= read -r ver; do
  [[ -n "$ver" ]] || continue
  if [[ ! "$ver" =~ ^${toolchain//./\\.}(\.[0-9]+)?$ ]]; then
    echo "$dockerfile: rust:$ver, want rust:$toolchain or rust:$toolchain.<patch> to match $toml" >&2
    fail=1
  fi
done <<< "$images"

if [[ "$fail" -ne 0 ]]; then
  exit 1
fi
echo "Rust toolchain $toolchain matches melange and $dockerfile"
