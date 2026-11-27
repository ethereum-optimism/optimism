#!/usr/bin/env bash
# Run one of Circle's Cannon recipes in an RWX task. A task receives the output
# of its producer tasks through the `use` layers that .rwx/rust.yml gives it.
set -euo pipefail
HELPERS="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(git rev-parse --show-toplevel)"
cd "$ROOT"
job="${1:?Pass env, go, witness, lint, build or offline}"
case "$job" in env | go | witness | lint | build | offline) ;;
*)
  echo "Unknown Cannon job: $job" >&2
  exit 1
  ;;
esac
report="$ROOT/.ci/rust-workspace/cannon-$job"
rm -rf "$report"
mkdir -p "$report" rust/target
python3 "$HELPERS/rust-cannon-report.py" begin "$report" "$job"
finish() {
  local status=$? diagnostics=0
  trap - EXIT
  if [[ "$job" == offline ]]; then
    sccache --show-stats --stats-format json > "$report/sccache.json" || diagnostics=$?
    sccache --stop-server > "$report/sccache-stop.log" 2>&1 || diagnostics=$?
  fi
  if [[ ("$job" == env || "$job" == lint || "$job" == build || "$job" == offline) && "$status" == 0 ]]; then
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
export CARGO_HOME="$ROOT/.ci/rust-cache/cargo" CARGO_TARGET_DIR="$ROOT/rust/target"
export GOPATH="$ROOT/.ci/cannon-cache/go/modules" GOCACHE="$ROOT/.ci/cannon-cache/go/build"
mkdir -p "$CARGO_HOME" "$GOPATH" "$GOCACHE"
if [[ "$job" == env || "$job" == lint || "$job" == build || "$job" == offline ]]; then
  # Stamp COPY inputs before Docker builds as well as bound runtime sources.
  # Normalized checkout mtimes must not make old BuildKit Cargo targets fresh.
  python3 "$HELPERS/rust-target-cache.py" prepare > "$report/cache-source.json"
fi
if [[ "$job" == offline ]]; then
  export CARGO_INCREMENTAL=0 RUSTC_WRAPPER=sccache SCCACHE_DIR="$ROOT/.ci/cannon-cache/sccache"
  export SCCACHE_BASEDIRS="$ROOT" SCCACHE_CACHE_SIZE=10G SCCACHE_IDLE_TIMEOUT=0
  mkdir -p "$SCCACHE_DIR"
  sccache --start-server
  sccache --zero-stats
fi
config=()
if [[ "$job" == witness || "$job" == offline ]]; then
  while IFS= read -r value; do config+=("$value"); done < <(python3 "$HELPERS/rust-cannon-report.py" config)
  if [[ "${#config[@]}" != 8 ]]; then
    echo 'Missing Cannon witness configuration.' >&2
    exit 1
  fi
fi
pull_base() {
  local base attempt=0
  base="$(python3 "$HELPERS/rust-cannon-report.py" base)"
  # Retry only a registry fetch. Compilation/test failures keep their first exit.
  until docker pull "$base"; do
    attempt=$((attempt + 1))
    [[ "$attempt" -lt 5 ]] || return 1
    sleep "$((2 ** attempt))"
  done
}
case "$job" in
  env)
    export -f pull_base
    export HELPERS
    stage_at . base-pull bash -c pull_base
    stage_at rust/kona env just build-kona-env
    ;;
  go) stage_at cannon go just cannon ;;
  witness)
    stage_at . witness bash rust/kona/bin/client/scripts/fetch-witness-tar.sh "${config[6]}" "${config[7]}"
    python3 "$HELPERS/rust-cannon-report.py" witness
    ;;
  lint) stage_at rust/kona lint just lint-cannon ;;
  build) stage_at rust/kona build just build-cannon-client ;;
  offline)
    # The recipe runs the Cannon binary that the cannon-go task built.
    export PATH="$ROOT/cannon/bin:$PATH"
    stage_at rust/kona/bin/client offline just run-client-cannon-offline "${config[@]}"
    ;;
esac
