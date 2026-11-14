#!/usr/bin/env bash
set -euo pipefail

# Tests check-release-ci-green.sh against recorded GitHub API responses.
#
# testdata/check-release-ci-green/ holds two real commits, trimmed to the
# fields the script reads:
#   red-e16bffa    e16bffa184 (op-reth/v2.6.1, kona-node/v1.7.3): ci-gate
#                  success, required-rust-ci failure (op-reth-compact-codec)
#   green-72fcf94  72fcf9479e (op-reth/v2.9.0 and others): 134 statuses over
#                  two pages, all gates success
# The other scenarios are derived from them with jq below.
#
# Run: bash ops/scripts/check-release-ci-green.test.sh

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CHECK="${SCRIPT_DIR}/check-release-ci-green.sh"
DATA="${SCRIPT_DIR}/testdata/check-release-ci-green"
RED_SHA=e16bffa18420c86dbab693097bc33117f74a5f7d
GREEN_SHA=72fcf9479ea7ee1105b823d23da8bf58ba58136d

WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

PASS=0
FAIL=0

# run_case <name> <expected-exit> <fixture-dir> <ref> [ENV=VALUE...]
# Fast defaults: no sleeping, no grace, no timeout, unless a case sets them.
run_case() {
  local name="$1" want="$2" dir="$3" ref="$4"
  shift 4
  local got=0
  env -u GITHUB_ACTIONS -u ALLOW_RED_CI_REASON \
    RELEASE_CI_FIXTURES="$dir" RELEASE_CI_POLL_SECONDS=0 \
    RELEASE_CI_TIMEOUT_SECONDS=0 RELEASE_CI_MISSING_GRACE_SECONDS=0 \
    "$@" bash "$CHECK" "$ref" > "$WORK/out.txt" 2>&1 || got=$?
  if [ "$got" = "$want" ]; then
    PASS=$((PASS + 1))
    echo "ok   $name (exit $got)"
  else
    FAIL=$((FAIL + 1))
    echo "FAIL $name: exit $got, want $want"
    sed 's/^/     /' "$WORK/out.txt"
  fi
}

# expect_output <name> <fixed-string>: checks the last run's output.
expect_output() {
  if grep -qF -- "$2" "$WORK/out.txt"; then
    PASS=$((PASS + 1))
    echo "ok   $1"
  else
    FAIL=$((FAIL + 1))
    echo "FAIL $1: output lacks '$2'"
    sed 's/^/     /' "$WORK/out.txt"
  fi
}

# scenario <name>: new empty fixture dir, path on stdout.
scenario() {
  mkdir -p "$WORK/$1"
  echo "$WORK/$1"
}

# set_state <in> <out> <context> <state>: copy a page with one context changed.
set_state() {
  jq --arg c "$3" --arg s "$4" '.statuses |= map(if .context == $c then .state = $s else . end)' "$1" > "$2"
}

# drop_context <in> <out> <context>: copy a page without one context.
drop_context() {
  jq --arg c "$3" '.statuses |= map(select(.context != $c))' "$1" > "$2"
}

GATE="ci/circleci: ci-gate"
RUST="ci/circleci: required-rust-ci"

# 1-2. The recorded commits as they are.
run_case "red release commit fails" 1 "$DATA/red-e16bffa" "$RED_SHA"
expect_output "red release commit names the failed gate" "failure  ${RUST}"
run_case "green release commit passes (two pages)" 0 "$DATA/green-72fcf94" "$GREEN_SHA"

# 3. A tag name resolves through the commits API to the full SHA.
run_case "tag name resolves to its commit" 0 "$DATA/green-72fcf94" "op-reth/v2.9.0"
expect_output "tag name resolves to its commit (sha printed)" "@${GREEN_SHA} (op-reth/v2.9.0)"

# 4. Override: a red commit continues, and the log says why.
run_case "override lets a red commit through" 0 "$DATA/red-e16bffa" "$RED_SHA" \
  ALLOW_RED_CI_REASON="hotfix for CVE-0000-0000, approved by the release manager"
expect_output "override prints the reason" "Reason: hotfix for CVE-0000-0000, approved by the release manager"
expect_output "override prints the CI state" "is 'red'"

# 5. Pending, then green on the next poll: waits, then passes.
d="$(scenario pending-then-green)"
set_state "$DATA/green-72fcf94/status-1-1.json" "$d/status-1-1.json" "$GATE" pending
cp "$DATA/green-72fcf94/status-1-2.json" "$d/status-1-2.json"
cp "$DATA/green-72fcf94/status-1-1.json" "$d/status-2-1.json"
cp "$DATA/green-72fcf94/status-1-2.json" "$d/status-2-2.json"
run_case "pending then green passes" 0 "$d" "$GREEN_SHA" RELEASE_CI_TIMEOUT_SECONDS=60

# 6. Pending, then failure: fails.
d="$(scenario pending-then-red)"
set_state "$DATA/red-e16bffa/status-1-1.json" "$d/status-1-1.json" "$RUST" pending
cp "$DATA/red-e16bffa/status-1-1.json" "$d/status-2-1.json"
run_case "pending then red fails" 1 "$d" "$RED_SHA" RELEASE_CI_TIMEOUT_SECONDS=60

# 7. Still pending at the time limit: fails.
d="$(scenario pending-forever)"
set_state "$DATA/green-72fcf94/status-1-1.json" "$d/status-1-1.json" "$GATE" pending
cp "$DATA/green-72fcf94/status-1-2.json" "$d/status-1-2.json"
run_case "pending past the time limit fails" 1 "$d" "$GREEN_SHA"
expect_output "pending past the time limit says so" "still pending after 0s"

# 8. ci-gate never posted: fails after the grace period.
d="$(scenario gate-missing)"
drop_context "$DATA/red-e16bffa/status-1-1.json" "$d/status-1-1.json" "$GATE"
set_state "$d/status-1-1.json" "$d/status-1-1.json.tmp" "$RUST" success
mv "$d/status-1-1.json.tmp" "$d/status-1-1.json"
run_case "missing ci-gate fails" 1 "$d" "$RED_SHA" RELEASE_CI_TIMEOUT_SECONDS=60
expect_output "missing ci-gate says CI never reported" "never reported"

# 9. No statuses yet, then green: the grace period waits for CI to post.
d="$(scenario missing-then-green)"
echo '{"statuses": []}' > "$d/status-1-1.json"
cp "$DATA/green-72fcf94/status-1-1.json" "$d/status-2-1.json"
cp "$DATA/green-72fcf94/status-1-2.json" "$d/status-2-2.json"
run_case "not yet posted, then green passes" 0 "$d" "$GREEN_SHA" \
  RELEASE_CI_TIMEOUT_SECONDS=60 RELEASE_CI_MISSING_GRACE_SECONDS=60

# 10. Only ci-gate on the commit: the other gates are not required.
d="$(scenario gate-only)"
jq --arg c "$GATE" '.statuses |= map(select(.context == $c))' \
  "$DATA/green-72fcf94/status-1-1.json" > "$d/status-1-1.json"
run_case "ci-gate alone passes" 0 "$d" "$GREEN_SHA"

# 11. A failed gate on page 2 is still found.
d="$(scenario red-on-page-two)"
jq --arg c "$RUST" '.statuses |= map(select(.context != $c))' \
  "$DATA/green-72fcf94/status-1-1.json" > "$d/page1.json"
# Keep page 1 full (100 statuses) so the script reads page 2.
jq '.statuses |= (. + .[0:1] | .[0:100])' "$d/page1.json" > "$d/status-1-1.json"
jq --arg c "$RUST" '.statuses += [{"context": $c, "state": "failure", "target_url": "https://example.invalid"}]' \
  "$DATA/green-72fcf94/status-1-2.json" > "$d/status-1-2.json"
rm "$d/page1.json"
run_case "failed gate on page 2 fails" 1 "$d" "$GREEN_SHA"

# 12. state "error" counts as red.
d="$(scenario gate-error)"
set_state "$DATA/green-72fcf94/status-1-1.json" "$d/status-1-1.json" "$GATE" error
cp "$DATA/green-72fcf94/status-1-2.json" "$d/status-1-2.json"
run_case "error state fails" 1 "$d" "$GREEN_SHA"

# 13. The API never answers: fails at the time limit, the override still continues.
d="$(scenario api-down)"
run_case "API down fails" 1 "$d" "$GREEN_SHA"
run_case "API down with override continues" 0 "$d" "$GREEN_SHA" ALLOW_RED_CI_REASON="GitHub outage"

# 14. A ref that does not resolve fails at once, even with the override.
d="$(scenario bad-ref)"
echo '{"message": "No commit found for SHA: nope"}' > "$d/commit.json"
run_case "unresolvable ref fails" 1 "$d" "nope/v0.0.0" ALLOW_RED_CI_REASON="x"
run_case "ref that is not a plain name is refused" 2 "$d" "op-node/v1?per_page=1" ALLOW_RED_CI_REASON="x"
run_case "ref with .. is refused" 2 "$d" "refs/tags/../heads/develop"

echo
echo "${PASS} passed, ${FAIL} failed"
[ "$FAIL" -eq 0 ]
