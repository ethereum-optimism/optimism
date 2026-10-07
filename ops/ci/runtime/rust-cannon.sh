#!/usr/bin/env bash
# Preserve Circle's full Cannon recipes with shared artifacts and guest evidence.
set -euo pipefail
HELPERS="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(git rev-parse --show-toplevel)"
cd "$ROOT"
job="${1:?Pass env, go, witness, lint, build or offline}"
case "$job" in env|go|witness|lint|build|offline) ;;
  *) echo "Unknown Cannon job: $job" >&2; exit 1;;
esac
report="$ROOT/.ci/rust-workspace/cannon-$job"
rm -rf "$report"
mkdir -p "$report" rust/target
python3 "$HELPERS/rust-cannon-report.py" begin "$report" "$job"
finish() {
  local status=$? diagnostics=0
  trap - EXIT
  if [[ "${CI_RUST_PROVIDER:-circleci}" == rwx && "$job" == offline ]]; then
    sccache --show-stats --stats-format json >"$report/sccache.json" || diagnostics=$?
    sccache --stop-server >"$report/sccache-stop.log" 2>&1 || diagnostics=$?
  fi
  if [[ "${CI_RUST_PROVIDER:-circleci}" == rwx && ( "$job" == env || "$job" == lint || "$job" == build || "$job" == offline ) && "$status" == 0 ]]; then
    python3 "$HELPERS/rust-target-cache.py" commit || diagnostics=$?
  fi
  if [[ "$status" == 0 && "$diagnostics" != 0 ]]; then status=$diagnostics; fi
  python3 "$HELPERS/rust-cannon-report.py" finish "$report" "$status" || status=$?
  exit "$status"
}
trap finish EXIT
trap 'exit 130' INT
trap 'exit 143' TERM
stage_at() { python3 "$HELPERS/rust-workspace-report.py" stage-at "$report" "$@"; }
stage_at rust variants just kona-prestate-variants
python3 "$HELPERS/rust-cannon-report.py" inventory "$report"
if [[ "${CI_RUST_PROVIDER:-circleci}" == rwx ]]; then
  export CARGO_HOME="$ROOT/.ci/rust-cache/cargo" CARGO_TARGET_DIR="$ROOT/rust/target"
  export GOPATH="$ROOT/.ci/cannon-cache/go/modules" GOCACHE="$ROOT/.ci/cannon-cache/go/build"
  mkdir -p "$CARGO_HOME" "$GOPATH" "$GOCACHE"
  if [[ "$job" == env || "$job" == lint || "$job" == build || "$job" == offline ]]; then
    # Stamp COPY inputs before Docker builds as well as bound runtime sources.
    # Normalized checkout mtimes must not make old BuildKit Cargo targets fresh.
    python3 "$HELPERS/rust-target-cache.py" prepare >"$report/cache-source.json"
  fi
  if [[ "$job" == offline ]]; then
    export CARGO_INCREMENTAL=0 RUSTC_WRAPPER=sccache SCCACHE_DIR="$ROOT/.ci/cannon-cache/sccache"
    export SCCACHE_BASEDIRS="$ROOT" SCCACHE_CACHE_SIZE=10G SCCACHE_IDLE_TIMEOUT=0
    mkdir -p "$SCCACHE_DIR"
    sccache --start-server
    sccache --zero-stats
  fi
fi
config=()
while IFS= read -r value; do config+=("$value"); done < <(python3 "$HELPERS/rust-cannon-report.py" config)
if [[ "${#config[@]}" != 8 ]]; then echo 'Missing Cannon witness configuration.' >&2; exit 1; fi
verify() { python3 "$HELPERS/rust-cannon-report.py" verify "$report" "$1" "$2"; }
pull_base() {
  local base attempt=0
  base="$(python3 "$HELPERS/rust-cannon-report.py" base)"
  # Retry only a registry fetch. Compilation/test failures keep their first exit.
  until docker pull "$base"; do
    attempt=$((attempt + 1)); [[ "$attempt" -lt 5 ]] || return 1
    sleep "$((2 ** attempt))"
  done
}
prepare_image() {
  if [[ "$job" == offline && -n "${CANNON_BUILD_ARTIFACT:-}" ]]; then
    # The build recipe rebuilds its environment. Its sealed image and ELFs
    # own the runtime handoff, even when a cold rebuild changes the image ID.
    verify "$CANNON_BUILD_ARTIFACT" build
  elif [[ -n "${CANNON_ENV_ARTIFACT:-}" ]]; then
    verify "$CANNON_ENV_ARTIFACT" env
  else
    # Record the actual base digest even when BuildKit resolves only cached layers.
    export -f pull_base
    export HELPERS
    stage_at . base-pull bash -c pull_base
  fi
}
capture_elfs() {
  mkdir -p "$report/elfs"
  while read -r binary _; do
    cp "rust/target/mips64-unknown-none/release-client-lto/$binary" "$report/elfs/$binary"
  done <"$report/variants.log"
  python3 "$HELPERS/rust-cannon-report.py" elfs "$report" "$report/elfs"
}
build_go() {
  if [[ -n "${CANNON_GO_ARTIFACT:-}" ]]; then
    verify "$CANNON_GO_ARTIFACT" go
    cp "$CANNON_GO_ARTIFACT/go-binaries.json" "$report/"
  else
    stage_at cannon go just cannon
    python3 "$HELPERS/rust-cannon-report.py" go "$report"
  fi
  export PATH="$ROOT/cannon/bin:$PATH"
}
case "$job" in
  env)
    prepare_image
    stage_at rust/kona env just build-kona-env
    python3 "$HELPERS/rust-cannon-report.py" image "$report"
    ;;
  go) build_go ;;
  witness)
    stage_at . witness bash rust/kona/bin/client/scripts/fetch-witness-tar.sh "${config[6]}" "${config[7]}"
    python3 "$HELPERS/rust-cannon-report.py" witness "$report"
    ;;
  lint|build)
    prepare_image
    recipe=lint-cannon
    if [[ "$job" == build ]]; then recipe=build-cannon-client; fi
    stage_at rust/kona "$job" just "$recipe"
    python3 "$HELPERS/rust-cannon-report.py" image "$report"
    if [[ "$job" == build ]]; then capture_elfs; fi
    ;;
  offline)
    if [[ -n "${CANNON_WITNESS_ARTIFACT:-}" ]]; then
      verify "$CANNON_WITNESS_ARTIFACT" witness
    else
      stage_at . witness bash rust/kona/bin/client/scripts/fetch-witness-tar.sh "${config[6]}" "${config[7]}"
    fi
    python3 "$HELPERS/rust-cannon-report.py" witness "$report"
    build_go
    prepare_image
    # Never accept a restored state as evidence of this invocation's guest exit.
    rm -f rust/kona/state.bin.gz rust/kona/out.bin.gz rust/kona/meta.json
    stage_at rust/kona/bin/client offline just run-client-cannon-offline "${config[@]}"
    cp rust/kona/out.bin.gz "$report/out.bin.gz"
    python3 "$HELPERS/rust-workspace-report.py" json-stage "$report" guest \
      "$ROOT/cannon/bin/cannon" witness --input "$ROOT/rust/kona/out.bin.gz"
    capture_elfs
    python3 "$HELPERS/rust-cannon-report.py" image "$report"
    ;;
esac
