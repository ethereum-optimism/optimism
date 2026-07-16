#!/usr/bin/env bash
# Final-binary cache helper for the rust-binary-cached-build command in
# .circleci/continue/main.yml. CircleCI's restore_cache/save_cache must stay
# YAML steps; this provides the shell halves around them:
#
#   compute-key  Before restore_cache: writes the cache-key checksum inputs
#                (git tree hash of BINCACHE_HASH_PATHS + build params, and
#                the sorted binary list) under .circleci-cache/.
#   build        After restore_cache: on a hit (all named binaries cached)
#                skips cargo, re-seeding the shared cargo registry via
#                `cargo fetch` if its restore missed; on a miss builds and
#                copies the binaries into the cache dir.
#   stage        Before persist_to_workspace: copies the cached binaries to
#                .circleci-cache/workspace-persist/<dir>/target/<profile>/.
#
# Inputs are BINCACHE_* env vars set from the command's parameters:
#   BINCACHE_DIRECTORY          Cargo workspace directory, e.g. "rust"
#   BINCACHE_BINARIES           space-separated final binary names
#   BINCACHE_PROFILE            profile as spelled in the target path
#   BINCACHE_FEATURES           "default", "all", or a comma list
#   BINCACHE_PACKAGE            single package; "" = --workspace
#   BINCACHE_BINARY             single --bin; "" = all targets
#   BINCACHE_HASH_PATHS         git paths hashed into the key; must cover
#                               everything that changes the binaries
#   BINCACHE_SAVE_SHARED_CACHE  "true" iff the job saves the shared dep cache
#   BINCACHE_NEEDS_CLANG        "true" to assert clang before a miss build
set -euo pipefail

BIN_CACHE_RELDIR=".circleci-cache/rust-workspace-binaries"
SHARED_CARGO_REGISTRY="/data/mise-data/.cargo/registry"

usage() {
  echo "usage: $0 {compute-key|build|stage}" >&2
  echo "Inputs are BINCACHE_* environment variables; see the header of $0." >&2
}

require_env() {
  local var
  for var in "$@"; do
    if [ -z "${!var:-}" ]; then
      echo "ERROR: required environment variable $var is not set" >&2
      exit 1
    fi
  done
}

cmd_compute_key() {
  require_env BINCACHE_HASH_PATHS BINCACHE_BINARIES BINCACHE_PROFILE BINCACHE_FEATURES
  mkdir -p .circleci-cache

  # Key on everything that changes the binaries: the workspace tree
  # (includes Cargo.lock and rust-toolchain.toml), the mise-pinned toolchain
  # versions, the superchain-registry submodule pin (op-reth chainspec's
  # build.rs embeds it), and the build parameters. Guard against a
  # stale/typo'd path first: `git ls-tree` emits nothing (exit 0) for a
  # non-existent path, which would silently shrink the hash and reintroduce
  # stale binaries.
  local -a hash_paths
  read -r -a hash_paths <<<"$BINCACHE_HASH_PATHS"
  local p
  for p in "${hash_paths[@]}"; do
    if [ -z "$(git ls-tree HEAD "$p")" ]; then
      echo "ERROR: hash path '$p' does not exist at HEAD" >&2
      exit 1
    fi
  done
  {
    git ls-tree -r HEAD "${hash_paths[@]}"
    echo "profile=$BINCACHE_PROFILE features=$BINCACHE_FEATURES package=${BINCACHE_PACKAGE:-} binary=${BINCACHE_BINARY:-}"
  } | sha256sum | awk '{print $1}' >.circleci-cache/expected-workspace-sha.txt
  echo "Content HASH: $(cat .circleci-cache/expected-workspace-sha.txt)"

  echo "$BINCACHE_BINARIES" | tr ' ' '\n' | awk 'NF' | sort -u \
    >.circleci-cache/expected-workspace-binaries.txt
  echo "Expected binaries: $(tr '\n' ' ' <.circleci-cache/expected-workspace-binaries.txt)"
}

cmd_build() {
  require_env BINCACHE_DIRECTORY BINCACHE_BINARIES BINCACHE_PROFILE \
    BINCACHE_FEATURES BINCACHE_SAVE_SHARED_CACHE BINCACHE_NEEDS_CLANG
  local root_dir bin_cache_dir bin
  root_dir="$(pwd)"
  bin_cache_dir="$root_dir/$BIN_CACHE_RELDIR"

  local hit=true
  for bin in $BINCACHE_BINARIES; do
    if [ ! -f "$bin_cache_dir/$bin" ]; then
      hit=false
      break
    fi
  done
  if [ "$hit" = "true" ]; then
    echo "Cache hit - binaries exist"
    # The shared dep-cache save that follows refuses to save an empty cargo
    # registry. The dep restore normally populates it (a Cargo.lock change
    # also changes the binary key, so a fresh lockfile lands on the build
    # path), but the older dep key can be evicted while the binary key is
    # still live. Re-seed it with a fetch instead of tripping that guard.
    if [ "$BINCACHE_SAVE_SHARED_CACHE" = "true" ] &&
      { [ ! -d "$SHARED_CARGO_REGISTRY" ] || [ -z "$(ls -A "$SHARED_CARGO_REGISTRY" 2>/dev/null)" ]; }; then
      echo "Shared dep cache restore missed - running cargo fetch to re-seed it"
      (cd "$BINCACHE_DIRECTORY" && cargo fetch)
    fi
    return 0
  fi

  echo "Cache miss - will build"

  if [ "$BINCACHE_NEEDS_CLANG" = "true" ] && ! command -v clang >/dev/null; then
    echo "ERROR: clang is required but was not found in PATH" >&2
    exit 1
  fi

  export CARGO_TARGET_DIR="$root_dir/$BINCACHE_DIRECTORY/target"
  echo "CARGO_TARGET_DIR: $CARGO_TARGET_DIR"

  # The profile is "debug" in the config and target path, but cargo build
  # expects "dev".
  local -a profile_args=(--profile "$BINCACHE_PROFILE")
  if [ "$BINCACHE_PROFILE" = "debug" ]; then
    profile_args=(--profile dev)
  fi

  local -a package_args=(--workspace)
  if [ -n "${BINCACHE_PACKAGE:-}" ]; then
    package_args=(--package "$BINCACHE_PACKAGE")
  fi

  local -a binary_args=()
  if [ -n "${BINCACHE_BINARY:-}" ]; then
    binary_args=(--bin "$BINCACHE_BINARY")
  fi

  local -a features_args=(--features "$BINCACHE_FEATURES")
  if [ "$BINCACHE_FEATURES" = "all" ]; then
    features_args=(--all-features)
  fi

  (
    cd "$BINCACHE_DIRECTORY" &&
      cargo build "${profile_args[@]}" "${package_args[@]}" \
        "${features_args[@]}" ${binary_args[@]+"${binary_args[@]}"}
  )

  mkdir -p "$bin_cache_dir"
  local src
  for bin in $BINCACHE_BINARIES; do
    src="$root_dir/$BINCACHE_DIRECTORY/target/$BINCACHE_PROFILE/$bin"
    if [ ! -f "$src" ]; then
      echo "ERROR: expected built binary not found at $src" >&2
      exit 1
    fi
    cp "$src" "$bin_cache_dir/$bin"
    chmod +x "$bin_cache_dir/$bin" || true
  done
}

cmd_stage() {
  require_env BINCACHE_DIRECTORY BINCACHE_BINARIES BINCACHE_PROFILE
  local root_dir bin_cache_dir stage_dir bin
  root_dir="$(pwd)"
  bin_cache_dir="$root_dir/$BIN_CACHE_RELDIR"
  stage_dir="$root_dir/.circleci-cache/workspace-persist/$BINCACHE_DIRECTORY/target/$BINCACHE_PROFILE"
  mkdir -p "$stage_dir"
  for bin in $BINCACHE_BINARIES; do
    cp "$bin_cache_dir/$bin" "$stage_dir/$bin"
  done
  ls -la "$stage_dir"
}

case "${1:-}" in
compute-key) cmd_compute_key ;;
build) cmd_build ;;
stage) cmd_stage ;;
*)
  usage
  exit 1
  ;;
esac
