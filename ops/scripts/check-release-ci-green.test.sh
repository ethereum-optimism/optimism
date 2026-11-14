#!/usr/bin/env bash
set -euo pipefail

# Tests check-release-ci-green.sh with generated GitHub API responses and a
# throwaway git repo for the commit's CI config. No network.
#
# Run: bash ops/scripts/check-release-ci-green.test.sh

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CHECK="${SCRIPT_DIR}/check-release-ci-green.sh"

WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

PASS=0
FAIL=0

# A repo with three commits, each defining a different set of gate jobs.
REPO="$WORK/repo"
git_t() { git -C "$REPO" -c user.name=t -c user.email=t@t.invalid -c commit.gpgsign=false "$@"; }
mkdir -p "$REPO/.circleci/continue"
git init -q "$REPO"
# commit_with <job...>: a commit whose .circleci/ defines exactly these jobs.
commit_with() {
  local job
  rm -f "$REPO"/.circleci/continue/*.yml
  {
    echo "jobs:"
    for job in "$@"; do
      printf '  %s:\n    docker: []\n' "$job"
    done
    # Workflow references and similar names must not count as a definition.
    printf 'workflows:\n  ci-gate-skip:\n    jobs:\n      - required-rust-e2e:\n          always-succeed: true\n'
  } > "$REPO/.circleci/continue/main.yml"
  git_t add -A .circleci
  git_t commit -q --allow-empty -m "$*"
  git_t rev-parse HEAD
}
ALL_SHA="$(commit_with ci-gate required-contracts-ci required-rust-ci required-rust-e2e)"
GATE_ONLY_SHA="$(commit_with ci-gate)"
NO_GATE_SHA="$(commit_with build-only)"
NOT_HERE_SHA=0123456789abcdef0123456789abcdef01234567

# run_case <name> <expected-exit> <fixture-dir> <ref> [ENV=VALUE...]
# Fast defaults: no sleeping and no waiting, unless a case sets them.
run_case() {
  local name="$1" want="$2" dir="$3" ref="$4"
  shift 4
  local got=0
  env -u GITHUB_ACTIONS -u GITHUB_OUTPUT -u GITHUB_STEP_SUMMARY -u ALLOW_RED_CI_REASON \
    -u GITHUB_REPOSITORY -u CIRCLE_PROJECT_USERNAME -u CIRCLE_PROJECT_REPONAME \
    RELEASE_CI_FIXTURES="$dir" RELEASE_CI_CONFIG_REPO="$REPO" RELEASE_CI_POLL_SECONDS=0 \
    RELEASE_CI_TIMEOUT_SECONDS=0 \
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

# page <out> <filler-prefix> <filler-count> [job=state...]: a status page with
# <filler-count> green non-gate contexts, then the given gates.
page() {
  local out="$1" prefix="$2" count="$3" kv
  shift 3
  for kv in "$@"; do printf '%s\n' "$kv"; done \
    | jq -R 'split("=") | {context: ("ci/circleci: " + .[0]), state: .[1], target_url: "https://circleci.com/gh/o/r/1"}' \
    | jq -s --arg p "$prefix" --argjson n "$count" \
      '{statuses: ([range($n) | {context: "ci/circleci: \($p)-\(.)", state: "success", target_url: ""}] + .)}' > "$out"
}

# green_pages <dir> <poll> [job=state...]: a two-page commit like develop's
# (134 statuses). Page 1 is full and has ci-gate. Page 2 has the other three
# gates, each success unless overridden by an argument (job=state, or job=-
# to leave it out).
green_pages() {
  local dir="$1" poll="$2" gate=success contracts=success rust=success e2e=success kv
  shift 2
  for kv in "$@"; do
    case "$kv" in
      ci-gate=*) gate="${kv#*=}" ;;
      required-contracts-ci=*) contracts="${kv#*=}" ;;
      required-rust-ci=*) rust="${kv#*=}" ;;
      required-rust-e2e=*) e2e="${kv#*=}" ;;
    esac
  done
  local p1=() p2=()
  [ "$gate" = - ] || p1+=("ci-gate=$gate")
  [ "$contracts" = - ] || p2+=("required-contracts-ci=$contracts")
  [ "$rust" = - ] || p2+=("required-rust-ci=$rust")
  [ "$e2e" = - ] || p2+=("required-rust-e2e=$e2e")
  page "$dir/status-$poll-1.json" a $((100 - ${#p1[@]})) ${p1[@]+"${p1[@]}"}
  page "$dir/status-$poll-2.json" b 30 ${p2[@]+"${p2[@]}"}
}

# 1-2. Like e16bffa184 (red required-rust-ci, green ci-gate) and 72fcf9479e.
d="$(scenario red)"
green_pages "$d" 1 required-rust-ci=failure
run_case "red rust gate fails though ci-gate is green" 1 "$d" "$ALL_SHA"
expect_output "red rust gate is named" "failure  ci/circleci: required-rust-ci"
RED="$d"
d="$(scenario green)"
green_pages "$d" 1
run_case "all gates green over two pages passes" 0 "$d" "$ALL_SHA"
expect_output "required gates come from the commit's config" "  ci/circleci: required-rust-e2e"
GREEN="$d"

# 3. A tag name resolves through the commits API to the full SHA.
printf '{"sha": "%s"}\n' "$ALL_SHA" > "$GREEN/commit.json"
run_case "tag name resolves to its commit" 0 "$GREEN" "op-reth/v2.9.0"
expect_output "tag name resolves to its commit (sha printed)" "@${ALL_SHA} (op-reth/v2.9.0)"
run_case "resolved sha goes to GITHUB_OUTPUT" 0 "$GREEN" "op-reth/v2.9.0" GITHUB_OUTPUT="$WORK/gh-output"
if grep -qx "sha=${ALL_SHA}" "$WORK/gh-output"; then
  PASS=$((PASS + 1))
  echo "ok   GITHUB_OUTPUT has the sha"
else
  FAIL=$((FAIL + 1))
  echo "FAIL GITHUB_OUTPUT lacks sha=${ALL_SHA}"
fi

# 4-6. The override applies only when it names this commit.
run_case "override naming the commit lets a red commit through" 0 "$RED" "$ALL_SHA" \
  ALLOW_RED_CI_REASON="${ALL_SHA}: hotfix for CVE-0000-0000, approved by the release manager"
expect_output "override prints the reason" "Reason: hotfix for CVE-0000-0000, approved by the release manager"
expect_output "override prints the CI state" "there is 'red'"
run_case "override naming another commit is ignored" 1 "$RED" "$ALL_SHA" \
  ALLOW_RED_CI_REASON="${GATE_ONLY_SHA}: some other release"
expect_output "ignored override warns" "so it is ignored for this commit"
run_case "override without a sha is ignored" 1 "$RED" "$ALL_SHA" ALLOW_RED_CI_REASON="hotfix"
run_case "override with a sha and no reason is ignored" 1 "$RED" "$ALL_SHA" ALLOW_RED_CI_REASON="${ALL_SHA}:  "

# 7. Pending, then green on the next poll: waits, then passes.
d="$(scenario pending-then-green)"
green_pages "$d" 1 ci-gate=pending
green_pages "$d" 2
run_case "pending then green passes" 0 "$d" "$ALL_SHA" RELEASE_CI_TIMEOUT_SECONDS=60

# 8. Pending, then failure: fails.
d="$(scenario pending-then-red)"
green_pages "$d" 1 required-rust-ci=pending
green_pages "$d" 2 required-rust-ci=failure
run_case "pending then red fails" 1 "$d" "$ALL_SHA" RELEASE_CI_TIMEOUT_SECONDS=60

# 9. Still pending at the time limit: fails.
d="$(scenario pending-forever)"
green_pages "$d" 1 ci-gate=pending
run_case "pending past the time limit fails" 1 "$d" "$ALL_SHA"
expect_output "pending past the time limit says so" "1 polls (limit 0s): ci/circleci: ci-gate (pending)"

# 9b. It stops when the next poll would pass the limit, not after it: with a
# 1 s limit and a 2 s interval it polls once and does not sleep. With a 3 s
# limit and a 2 s interval it polls twice (a second of slack for a slow first
# poll).
run_case "next poll past the limit stops without sleeping" 1 "$d" "$ALL_SHA" \
  RELEASE_CI_TIMEOUT_SECONDS=1 RELEASE_CI_POLL_SECONDS=2
expect_output "next poll past the limit stops after one poll" "and 1 polls (limit 1s)"
run_case "a poll that fits the limit still runs" 1 "$d" "$ALL_SHA" \
  RELEASE_CI_TIMEOUT_SECONDS=3 RELEASE_CI_POLL_SECONDS=2
expect_output "a poll that fits the limit still runs (two polls)" "and 2 polls (limit 3s)"

# 10. ci-gate green before required-rust-ci posted (seen on 72fcf9479e): that
# is pending, not a pass. It passes once the gate posts green.
d="$(scenario gate-not-posted-yet)"
green_pages "$d" 1 required-rust-ci=-
run_case "gate not posted yet is not a pass" 1 "$d" "$ALL_SHA"
expect_output "gate not posted yet is reported as missing" "ci/circleci: required-rust-ci (missing)"
green_pages "$d" 2
run_case "gate not posted yet, then green, passes" 0 "$d" "$ALL_SHA" RELEASE_CI_TIMEOUT_SECONDS=60
expect_output "gate not posted yet is shown as such" "missing  ci/circleci: required-rust-ci (not posted yet)"

# 11. No statuses at all yet, then green.
d="$(scenario nothing-then-green)"
page "$d/status-1-1.json" a 0
green_pages "$d" 2
run_case "nothing posted, then green, passes" 0 "$d" "$ALL_SHA" RELEASE_CI_TIMEOUT_SECONDS=60

# 12. A commit whose config has only ci-gate needs only ci-gate.
d="$(scenario gate-only)"
page "$d/status-1-1.json" a 5 ci-gate=success
run_case "only ci-gate defined, only ci-gate needed" 0 "$d" "$GATE_ONLY_SHA"
run_case "the same statuses on a commit that defines all four wait" 1 "$d" "$ALL_SHA"

# 13. A commit with no ci-gate job: nothing to check, fails at once.
run_case "no ci-gate in the config fails at once" 1 "$d" "$NO_GATE_SHA" RELEASE_CI_TIMEOUT_SECONDS=600
expect_output "no ci-gate says so" "has no ci-gate job in its .circleci/"
run_case "no ci-gate with the override continues" 0 "$d" "$NO_GATE_SHA" \
  ALLOW_RED_CI_REASON="${NO_GATE_SHA}: release from before ci-gate existed"

# 14. A commit that is not in the clone fails, even with the override.
run_case "commit not in the clone fails" 1 "$d" "$NOT_HERE_SHA" \
  ALLOW_RED_CI_REASON="${NOT_HERE_SHA}: x"
expect_output "commit not in the clone says so" "is not in the clone"

# 15. A failed gate on page 2 is still found.
d="$(scenario red-on-page-two)"
green_pages "$d" 1 required-rust-e2e=failure
run_case "failed gate on page 2 fails" 1 "$d" "$ALL_SHA"
expect_output "failed gate on page 2 is read as failed, not missing" "failure  ci/circleci: required-rust-e2e"

# 16. state "error" counts as red.
d="$(scenario gate-error)"
green_pages "$d" 1 ci-gate=error
run_case "error state fails" 1 "$d" "$ALL_SHA" RELEASE_CI_TIMEOUT_SECONDS=600

# 17. The API never answers: fails at the time limit, the override still continues.
d="$(scenario api-down)"
run_case "API down fails" 1 "$d" "$ALL_SHA"
run_case "API down with override continues" 0 "$d" "$ALL_SHA" ALLOW_RED_CI_REASON="${ALL_SHA}: GitHub outage"

# 18. A ref that does not resolve fails at once, even with an override.
d="$(scenario bad-ref)"
echo '{"message": "No commit found for SHA: nope"}' > "$d/commit.json"
run_case "unresolvable ref fails" 1 "$d" "nope/v0.0.0" ALLOW_RED_CI_REASON="x"
run_case "ref that is not a plain name is refused" 2 "$d" "op-node/v1?per_page=1" ALLOW_RED_CI_REASON="x"
run_case "ref with .. is refused" 2 "$d" "refs/tags/../heads/develop"

# 19. The repo: GITHUB_REPOSITORY, else the CircleCI project, else the default.
run_case "repo defaults to the CircleCI project" 0 "$GREEN" "$ALL_SHA" \
  CIRCLE_PROJECT_USERNAME=someorg CIRCLE_PROJECT_REPONAME=somerepo
expect_output "repo defaults to the CircleCI project (printed)" "on someorg/somerepo@${ALL_SHA}"
run_case "GITHUB_REPOSITORY wins over the CircleCI project" 0 "$GREEN" "$ALL_SHA" \
  GITHUB_REPOSITORY=gh/repo CIRCLE_PROJECT_USERNAME=someorg CIRCLE_PROJECT_REPONAME=somerepo
expect_output "GITHUB_REPOSITORY wins (printed)" "on gh/repo@${ALL_SHA}"
run_case "repo default without either" 0 "$GREEN" "$ALL_SHA"
expect_output "repo default without either (printed)" "on ethereum-optimism/optimism@${ALL_SHA}"

# --- The CI config around the script (needs yq) ---
ROOT="$(git -C "$SCRIPT_DIR" rev-parse --show-toplevel)"
MAIN_YML="$ROOT/.circleci/continue/main.yml"
check() {
  if [ "$2" = true ]; then
    PASS=$((PASS + 1))
    echo "ok   $1"
  else
    FAIL=$((FAIL + 1))
    echo "FAIL $1: $3"
  fi
}

# 20. Job limits sit a few minutes above the script's own wait. The steps are
# the ones that run the script (with an argument), not the one that copies it.
# shellcheck disable=SC2016 # the $ is in the pattern, not an expansion
default_timeout="$(sed -n 's/^TIMEOUT="\${RELEASE_CI_TIMEOUT_SECONDS:-\([0-9]*\)}"$/\1/p' "$CHECK")"
need_minutes=$((default_timeout / 60 + 5))
minutes() {
  local v="${1%m}"
  [[ "$v" =~ ^[0-9]+$ ]] && echo "$v" || echo 0
}
for job in release-ci-green publish-contract-artifacts; do
  v="$(yq -r ".jobs.\"$job\".steps[] | select(.run.command // \"\" | test(\"check-release-ci-green.sh \\\"\")) | .run.no_output_timeout" "$MAIN_YML")"
  check "CircleCI $job check step limit ($v) is at least ${need_minutes}m" \
    "$([ "$(minutes "$v")" -ge "$need_minutes" ] && echo true)" "got '$v'"
done
for wf in build-images.apko.yaml release-op-deployer.yaml; do
  v="$(yq -r '.jobs.release-ci-green.timeout-minutes' "$ROOT/.github/workflows/$wf")"
  check "$wf release-ci-green limit (${v} min) is at least ${need_minutes} min" \
    "$([ "$(minutes "$v")" -ge "$need_minutes" ] && echo true)" "got '$v'"
done

# 21. The manual publish check waits like release-ci-green: no shorter timeout.
publish_cmd="$(yq -r '.jobs.publish-contract-artifacts.steps[] | select(.run.name == "Check required CI on a release commit") | .run.command' "$MAIN_YML")"
check "manual publish check runs the script" \
  "$(grep -q 'check-release-ci-green.sh' <<< "$publish_cmd" && echo true)" "script not called"
check "manual publish check keeps the normal wait" \
  "$(grep -q 'RELEASE_CI_TIMEOUT_SECONDS' <<< "$publish_cmd" || echo true)" "it sets RELEASE_CI_TIMEOUT_SECONDS"

# 22. The get-target-branch command on backport pushes. A develop line
# D0 <- D1, a backport branch D0 <- B1 <- B2 (HEAD), and X, a commit that was
# on the branch before a force push (not an ancestor of HEAD).
TB="$WORK/target-branch"
git init -q "$TB"
tb_git() { git -C "$TB" -c user.name=t -c user.email=t@t.invalid -c commit.gpgsign=false "$@"; }
mkdir -p "$TB/packages/contracts-bedrock/scripts/ops"
# Stub for the repo script: it gives develop for a non-PR push.
echo 'TARGET_BRANCH=develop' > "$TB/packages/contracts-bedrock/scripts/ops/get-target-branch.sh"
tb_git add -A
tb_git commit -q -m D0
D0="$(tb_git rev-parse HEAD)"
tb_git commit -q --allow-empty -m D1
tb_git update-ref refs/remotes/origin/develop HEAD
tb_git checkout -q -b backports/x/v1 "$D0"
tb_git commit -q --allow-empty -m X
X="$(tb_git rev-parse HEAD)"
tb_git reset -q --hard "$D0"
tb_git commit -q --allow-empty -m B1
B1="$(tb_git rev-parse HEAD)"
tb_git commit -q --allow-empty -m B2
B2="$(tb_git rev-parse HEAD)"
TB_CMD="$(yq -r '.commands.get-target-branch.steps[0].run.command' "$MAIN_YML")"

# target_case <name> <branch> <base-revision> <want>
target_case() {
  local name="$1" branch="$2" base="$3" want="$4" got cmd bash_env
  cmd="${TB_CMD//<< pipeline.git.base_revision >>/$base}"
  bash_env="$WORK/bash_env"
  : > "$bash_env"
  # shellcheck disable=SC2016 # the inner bash expands $1 and TARGET_BRANCH
  got="$(cd "$TB/packages/contracts-bedrock" \
    && env -u CIRCLE_PULL_REQUEST CIRCLE_BRANCH="$branch" CIRCLE_SHA1="$B2" BASH_ENV="$bash_env" \
      bash -eo pipefail -c "$cmd" > "$WORK/out.txt" 2>&1 \
    && env -i bash -c 'source "$1" && printf %s "$TARGET_BRANCH"' _ "$bash_env")" || got="(command failed)"
  check "target branch: $name" "$([ "$got" = "$want" ] && echo true)" "got '$got', want '$want'"
}
target_case "backport push uses the base revision" backports/x/v1 "$B1" "$B1"
target_case "force push (base not an ancestor) uses the merge-base" backports/x/v1 "$X" "$D0"
target_case "unfetchable base revision uses the merge-base" backports/x/v1 "$NOT_HERE_SHA" "$D0"
target_case "no base revision uses the merge-base" backports/x/v1 "" "$D0"
target_case "all-zero base revision uses the merge-base" backports/x/v1 0000000000000000000000000000000000000000 "$D0"
target_case "base revision equal to HEAD uses the merge-base" backports/x/v1 "$B2" "$D0"
target_case "proposal push compares against itself" proposal/p "$B1" proposal/p
target_case "develop push compares against develop" develop "$B1" develop
# shellcheck disable=SC2016 # a literal $(c) in the branch name
target_case "branch name with shell characters stays one value" 'backports/a;b$(c)' "" "$D0"
tb_git update-ref -d refs/remotes/origin/develop
target_case "no develop ref compares against itself" backports/x/v1 "" backports/x/v1

echo
echo "${PASS} passed, ${FAIL} failed"
[ "$FAIL" -eq 0 ]
