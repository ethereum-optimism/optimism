#!/usr/bin/env bash
#
# Generates op-core/nuts/state/jovian_state.json.
# It must use jovian's OWN toolchain, so we check out jovian's commit
# in a worktree, where op-deployer and the contracts are mutually consistent,
# and dump the predeploy state there.
#
# Usage: ops/scripts/gen-seed-state.sh jovian [commit]
#   <commit> defaults to the pinned op-contracts/v5.0.0 release (JOVIAN_COMMIT below).
set -euo pipefail

if [[ $# -lt 1 ]]; then
  echo "usage: gen-seed-state.sh <fork> [commit]" >&2
  exit 1
fi

FORK="$1"
ROOT="$(git rev-parse --show-toplevel)"
LOCK="$ROOT/op-core/nuts/fork_lock.toml"

# jovian predates the NUT lock system, so its commit isn't in fork_lock.toml.
# This is the op-contracts/v5.0.0 release tag — the jovian L2 contracts release.
JOVIAN_COMMIT="d09c836f818c73ae139f60b717654c4e53712743"

COMMIT="${2:-}"
if [[ -z "$COMMIT" ]]; then
  if [[ "$FORK" == "jovian" ]]; then
    COMMIT="$JOVIAN_COMMIT"
  else
    COMMIT="$(awk -v sec="[$FORK]" '
      $1==sec {inper=1; next}
      /^\[/   {inper=0}
      inper && $1=="commit" {gsub(/"/,"",$3); print $3; exit}
    ' "$LOCK")"
  fi
fi
if [[ -z "$COMMIT" ]]; then
  echo "could not resolve a commit for fork $FORK (pass one explicitly)" >&2
  exit 1
fi

OUT="$ROOT/op-core/nuts/state/${FORK}_state.json"
WT="$(mktemp -d "${TMPDIR:-/tmp}/prefork-state-wt.XXXXXX")"

cleanup() {
  git -C "$ROOT" worktree remove --force "$WT" 2>/dev/null || true
  rm -rf "$WT" 2>/dev/null || true
}
trap cleanup EXIT

# A release-tag commit (e.g. op-contracts/v5.0.0 for the jovian seed) need not be
# on develop's mainline, so a fresh clone may lack it. Give a clear hint rather
# than a cryptic failure from `git worktree add`.
if ! git -C "$ROOT" cat-file -e "${COMMIT}^{commit}" 2>/dev/null; then
  echo "error: commit $COMMIT not found locally — fetch it first, e.g.:" >&2
  echo "  git fetch https://github.com/ethereum-optimism/optimism.git tag op-contracts/v5.0.0" >&2
  exit 1
fi

echo ">>> [1/3] worktree for $FORK at $COMMIT"
git -C "$ROOT" worktree add --detach "$WT" "$COMMIT"

echo ">>> [2/3] build ${FORK}-era forge-artifacts"
( cd "$WT/packages/contracts-bedrock" && just build-no-tests )

echo ">>> [3/3] dump predeploy-scoped $FORK state with ${FORK}-era toolchain"
# Copy the committed dumper into the worktree so it compiles against the fork-era
# packages (see ops/scripts/prefork-state-dump/main.go).
mkdir -p "$WT/ops/scripts/prefork-state-dump"
cp "$ROOT/ops/scripts/prefork-state-dump/main.go" "$WT/ops/scripts/prefork-state-dump/main.go"

# At the pinned jovian seed commit (op-contracts/v5.0.0) the predeploys package
# still lives at op-service/predeploys; it moved to op-core/predeploys later, in
# the op-core decoupling. The committed dumper uses the current op-core path so it
# compiles on develop, so rewrite it back to op-service for the worktree build.
sed -i.bak 's#op-core/predeploys#op-service/predeploys#' "$WT/ops/scripts/prefork-state-dump/main.go"
rm -f "$WT/ops/scripts/prefork-state-dump/main.go.bak"

mkdir -p "$ROOT/op-core/nuts/state"
( cd "$WT" && go run ./ops/scripts/prefork-state-dump "$FORK" "$OUT" )

echo ">>> done"
ls -lh "$OUT"
