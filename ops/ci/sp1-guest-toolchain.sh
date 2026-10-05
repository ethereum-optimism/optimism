#!/usr/bin/env bash
# Shared original Circle SP1 pin checks and native toolchain installation.
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")/../.."

install_sp1_toolchain() {
  (cd rust/kona/sp1 && just install-sp1-toolchain)
}

mise_sp1_version="$(yq -r '.tools."github:succinctlabs/sp1".version // "<missing>"' mise.toml)"
root_manifest_sp1_sdk="$(yq -r '.workspace.dependencies."sp1-sdk".version // "<missing>"' rust/Cargo.toml)"
guest_manifest_sp1_lib="$(yq -r '.workspace.dependencies."sp1-lib".version // "<missing>"' rust/kona/sp1/programs/Cargo.toml)"
guest_manifest_sp1_zkvm="$(yq -r '.workspace.dependencies."sp1-zkvm".version // "<missing>"' rust/kona/sp1/programs/Cargo.toml)"
sp1_tag="$(cd rust/kona/sp1 && just --evaluate SP1_TAG 2>/dev/null || printf '<missing>')"

lock_data_dir="$(mktemp -d)"
trap 'rm -rf "$lock_data_dir"' EXIT
root_lock_sp1_sdk="<lockfile parse failed>"
if yq -p=toml -o=json '.package // []' rust/Cargo.lock > "$lock_data_dir/root.json"; then
  if ! root_lock_sp1_sdk="$(jq -er '[.[] | select(.name == "sp1-sdk") | .version] | unique | if length == 1 then .[0] else error("expected exactly one resolved sp1-sdk version") end' "$lock_data_dir/root.json")"; then
    root_lock_sp1_sdk="<missing or multiple versions>"
  fi
fi

guest_lock_sp1_lib="<lockfile parse failed>"
guest_lock_sp1_zkvm="<lockfile parse failed>"
if yq -p=toml -o=json '.package // []' rust/kona/sp1/programs/Cargo.lock > "$lock_data_dir/guest.json"; then
  if ! guest_lock_sp1_lib="$(jq -er '[.[] | select(.name == "sp1-lib") | .version] | unique | if length == 1 then .[0] else error("expected exactly one resolved sp1-lib version") end' "$lock_data_dir/guest.json")"; then
    guest_lock_sp1_lib="<missing or multiple versions>"
  fi
  if ! guest_lock_sp1_zkvm="$(jq -er '[.[] | select(.name == "sp1-zkvm") | .version] | unique | if length == 1 then .[0] else error("expected exactly one resolved sp1-zkvm version") end' "$lock_data_dir/guest.json")"; then
    guest_lock_sp1_zkvm="<missing or multiple versions>"
  fi
fi

if [ "$root_manifest_sp1_sdk" != "$mise_sp1_version" ] || \
  [ "$guest_manifest_sp1_lib" != "$mise_sp1_version" ] || \
  [ "$guest_manifest_sp1_zkvm" != "$mise_sp1_version" ] || \
  [ "$root_lock_sp1_sdk" != "$mise_sp1_version" ] || \
  [ "$guest_lock_sp1_lib" != "$mise_sp1_version" ] || \
  [ "$guest_lock_sp1_zkvm" != "$mise_sp1_version" ] || \
  [ "$sp1_tag" != "v${mise_sp1_version}" ]; then
  printf '%s\n' \
    "ERROR: SP1 version drift; mise.toml cargo-prove is canonical." \
    "  mise.toml cargo-prove: ${mise_sp1_version}" \
    "  rust/Cargo.toml sp1-sdk: ${root_manifest_sp1_sdk}" \
    "  rust/Cargo.lock resolved sp1-sdk: ${root_lock_sp1_sdk}" \
    "  rust/kona/sp1/programs/Cargo.toml sp1-lib: ${guest_manifest_sp1_lib}" \
    "  rust/kona/sp1/programs/Cargo.toml sp1-zkvm: ${guest_manifest_sp1_zkvm}" \
    "  rust/kona/sp1/programs/Cargo.lock resolved sp1-lib: ${guest_lock_sp1_lib}" \
    "  rust/kona/sp1/programs/Cargo.lock resolved sp1-zkvm: ${guest_lock_sp1_zkvm}" \
    "  rust/kona/sp1/justfile SP1_TAG: ${sp1_tag} (expected v${mise_sp1_version})" >&2
  exit 1
fi

toolchain_dir="$(find "$HOME/.sp1/toolchains" -mindepth 1 -maxdepth 1 -type d -print -quit 2>/dev/null || true)"
if [ -n "$toolchain_dir" ]; then
  rustup toolchain remove succinct || true
  if rustup toolchain link succinct "$toolchain_dir" && rustc +succinct --version >/dev/null; then
    echo "Restored SP1 toolchain from $toolchain_dir"
  else
    echo "Cached SP1 toolchain is invalid; reinstalling"
    install_sp1_toolchain
  fi
else
  install_sp1_toolchain
fi
rustc +succinct --version

