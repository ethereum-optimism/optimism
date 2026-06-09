#!/usr/bin/env bash
#
# Regenerates op-core/nuts/state/<fork>_state.json — the frozen L2 predeploy
# state as of <fork> (the state a chain is in once <fork> has activated). The
# NUT bundle activation test for the *next* fork boots from it.
#
# SCOPE: this is the BOOTSTRAP/seed generator and is normally used only for
# `jovian` (the chain's seed). It requires <fork> to be a generatable alloc mode
# at <fork>'s commit, which is true for jovian but NOT for karst-era lock commits.
# Generate every state past the seed with COMPOSE instead — run the fork's
# activation test with OP_E2E_GEN_PRESTATE=<fork> (see op-core/nuts/state/README.md);
# compose replays the already-frozen bundle and needs no fork-era build.
#
# Why this uses the era's own toolchain in a worktree: the seed must be
# built from <fork>-era contracts, but the CURRENT op-deployer cannot consume
# older contracts once their ABIs drift (e.g. DeployImplementations dropped its
# `protocolVersionsProxy` input after karst, so current Go expects 15 input
# fields where jovian/karst contracts have 16). This generates the state with
# <fork>'s OWN toolchain, run inside a worktree at <fork>'s commit, where
# op-deployer and the contracts are mutually consistent.
#
# CAVEAT (non-determinism): op-deployer randomizes the CREATE2 salt, so re-running
# produces cosmetically different L1-derived slots (the L1 counterpart addresses
# in L2CrossDomainMessenger / L2StandardBridge / L2ERC721Bridge). The committed
# state is one instance; the activation test resets those slots on entry.
#
# Usage: ops/scripts/gen-seed-state.sh <fork> [commit]
#   <commit> defaults to the fork's fork_lock.toml entry (or the pinned jovian
#   commit for jovian, which predates the NUT lock system).
set -euo pipefail

if [[ $# -lt 1 ]]; then
  echo "usage: gen-seed-state.sh <fork> [commit]" >&2
  exit 1
fi

FORK="$1"
ROOT="$(git rev-parse --show-toplevel)"
LOCK="$ROOT/op-core/nuts/fork_lock.toml"

# jovian predates the NUT lock system, so its commit isn't in fork_lock.toml.
# Jovian mainnet activation was 2025-12-02 (L1 timestamp 1764691201).
JOVIAN_COMMIT="79cee4ec028db485150db71e64d0921a78960f70"

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
WT="$(mktemp -d "${TMPDIR:-/tmp}/prestate-wt.XXXXXX")"

cleanup() {
  git -C "$ROOT" worktree remove --force "$WT" 2>/dev/null || true
  rm -rf "$WT" 2>/dev/null || true
}
trap cleanup EXIT

echo ">>> [1/3] worktree for $FORK at $COMMIT"
git -C "$ROOT" worktree add --detach "$WT" "$COMMIT"

echo ">>> [2/3] build ${FORK}-era forge-artifacts"
( cd "$WT/packages/contracts-bedrock" && just build-no-tests )

echo ">>> [3/3] dump predeploy-scoped $FORK state with ${FORK}-era toolchain"
# Copy the committed dumper into the worktree so it compiles against the fork-era
# packages (see ops/scripts/prestate-dump/main.go).
mkdir -p "$WT/ops/scripts/prestate-dump"
cp "$ROOT/ops/scripts/prestate-dump/main.go" "$WT/ops/scripts/prestate-dump/main.go"

mkdir -p "$ROOT/op-core/nuts/state"
( cd "$WT" && go run ./ops/scripts/prestate-dump "$FORK" "$OUT" )

echo ">>> done"
ls -lh "$OUT"
