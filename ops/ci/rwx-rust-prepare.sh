#!/usr/bin/env bash
# Install op-reth's pinned build tools in a reusable RWX filesystem layer.
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")/../.."
: "${RWX_ENV:?RWX_ENV must name the exported-environment directory}"

bash .circleci/scripts/apt-install.sh clang llvm-dev libclang-dev libssl-dev cmake zstd
mise install mold protoc github:nextest-rs/nextest
PATH="$(mise bin-paths | paste -sd: -):${PATH}"
export PATH

# Separate from CircleCI's GCS cache: no cloud credentials or production writer.
SCCACHE_VERSION=0.18.0
SCCACHE_SHA256=45f1447fbe231e3037bde351ef70677dd212216c8d62ae7ca409fecc4d6acc89
download="$(mktemp -d)"
trap 'rm -rf "$download"' EXIT
curl -fsSL --retry 5 --retry-delay 2 \
  "https://github.com/mozilla/sccache/releases/download/v${SCCACHE_VERSION}/sccache-v${SCCACHE_VERSION}-x86_64-unknown-linux-musl.tar.gz" \
  -o "$download/sccache.tar.gz"
printf '%s  %s\n' "$SCCACHE_SHA256" "$download/sccache.tar.gz" | sha256sum -c -
tar -xzf "$download/sccache.tar.gz" -C "$download"
sudo install -m 0755 "$download/sccache-v${SCCACHE_VERSION}-x86_64-unknown-linux-musl/sccache" /usr/local/bin/sccache

# Installing the helpers in the tool layer lets the pinned develop checkout use
# the same implementation without overlaying PR source onto the baseline.
sudo mkdir -p /usr/local/lib/optimism-ci
sudo install -m 0755 ops/ci/op-reth-shadow.sh /usr/local/lib/optimism-ci/op-reth-shadow.sh
sudo install -m 0755 ops/ci/rust-target-cache.py /usr/local/lib/optimism-ci/rust-target-cache.py
sudo install -m 0755 ops/ci/op-reth-report.py /usr/local/lib/optimism-ci/op-reth-report.py
printf '%s\n' "$PATH" >"$RWX_ENV/PATH"
printf '%s\n' "${RUSTUP_HOME:?mise exec must export its Rust toolchain directory}" >"$RWX_ENV/RUSTUP_HOME"
python3 - <<'PY' >"$RWX_ENV/RUSTUP_TOOLCHAIN"
import tomllib
with open('mise.toml', 'rb') as source:
    print(tomllib.load(source)['tools']['rust'][0]['version'])
PY
sccache --version
protoc --version
mold --version
cargo nextest --version
