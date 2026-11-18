#!/usr/bin/env bash
# Offline test for merge-queue-verified.sh. A stub curl answers each request
# with a CircleCI API response built below, holding only the fields the script
# reads, or fails the way curl does, so every scenario is deterministic.
#
# Usage:
#   bash .circleci/scripts/test-merge-queue-verified.sh
#
# Requires: jq
set -euo pipefail

SCRIPT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/merge-queue-verified.sh"
WORK="$(mktemp -d)"
trap 'rm -rf "${WORK}"' EXIT

SHA=1111111111111111111111111111111111111111
BASE_SHA=2222222222222222222222222222222222222222
QUEUE="gh-readonly-queue/develop/pr-1-${BASE_SHA}"
PIPELINES="*/v2/project/gh/ethereum-optimism/optimism/pipeline"
TREE="*/v1.1/project/gh/ethereum-optimism/optimism/tree/gh-readonly-queue%2Fdevelop%2Fpr-1-${BASE_SHA}?shallow=true&limit=100&offset=0"
ID_MAIN=10000000-0000-4000-8000-000000000000
ID_RUST_CI=20000000-0000-4000-8000-000000000000
ID_RUST_E2E=30000000-0000-4000-8000-000000000000
ID_FAST_1=40000000-0000-4000-8000-000000000000
ID_FAST_2=50000000-0000-4000-8000-000000000000
ID_FAST_3=60000000-0000-4000-8000-000000000000
# Sorts before ID_MAIN, so the script checks it first.
ID_FAILED_MAIN=00000000-0000-4000-8000-000000000000

# page <file> <next page token or ""> <branch>=<sha>...: a v2 pipeline list page.
page() {
  local file="$1" next="$2" pair
  shift 2
  for pair in "$@"; do
    jq -n --arg b "${pair%%=*}" --arg s "${pair#*=}" '{vcs: {branch: $b, revision: $s}}'
  done | jq -s --arg next "${next}" '{items: ., next_page_token: (if $next == "" then null else $next end)}' > "${WORK}/${file}"
}

# tree <file> <sha> <workflow name>=<workflow id>...: a v1.1 job list with one
# job per workflow.
tree() {
  local file="$1" sha="$2" pair
  shift 2
  for pair in "$@"; do
    jq -n --arg s "${sha}" --arg w "${pair%%=*}" --arg id "${pair#*=}" \
      '{vcs_revision: $s, workflows: {workflow_name: $w, workflow_id: $id}}'
  done | jq -s . > "${WORK}/${file}"
}

# job_list <file> <job>=<status>...: a v2 workflow job list.
job_list() {
  local file="$1" pair
  shift
  for pair in "$@"; do
    jq -n --arg n "${pair%%=*}" --arg s "${pair#*=}" '{name: $n, status: $s}'
  done | jq -s '{items: ., next_page_token: null}' > "${WORK}/${file}"
}

page queue.json "" "feat/x=${BASE_SHA}" "develop=${SHA}" "${QUEUE}=${SHA}"
page queue-twice.json "" "${QUEUE}=${SHA}" "develop=${SHA}" "${QUEUE}=${SHA}"
page no-queue.json "page-2" "feat/x=${BASE_SHA}" "develop=${SHA}"
page no-queue-again.json "page-n" "feat/y=${BASE_SHA}"
tree full.json "${SHA}" main="${ID_MAIN}" rust-ci="${ID_RUST_CI}" rust-e2e-ci="${ID_RUST_E2E}"
tree full-other-sha.json "${BASE_SHA}" main="${ID_MAIN}" rust-ci="${ID_RUST_CI}" rust-e2e-ci="${ID_RUST_E2E}"
tree rerun.json "${SHA}" main="${ID_FAILED_MAIN}" main="${ID_MAIN}" rust-ci="${ID_RUST_CI}" rust-e2e-ci="${ID_RUST_E2E}"
# The queue's fast paths report the gate names as success without testing.
tree rust-fast.json "${SHA}" main="${ID_MAIN}" rust-ci-gate-short="${ID_FAST_2}" rust-e2e-gate-skip="${ID_FAST_3}"
tree docs-fast.json "${SHA}" ci-gate-skip="${ID_FAST_1}" rust-ci-gate-short="${ID_FAST_2}" rust-e2e-gate-skip="${ID_FAST_3}"
job_list main.json go-tests=success ci-gate=success
job_list main-failed.json go-tests=failed ci-gate=failed
job_list rust-ci.json required-rust-ci=success
job_list rust-ci-running.json required-rust-ci=running
job_list rust-e2e.json required-rust-e2e=success
job_list fast-1.json ci-gate=success
job_list fast-2.json required-rust-ci=success
job_list fast-3.json required-rust-e2e=success
echo "garbage" > "${WORK}/not-json.json"
: > "${WORK}/empty.json"

# Stub curl. Each line of $ROUTES is "<glob> <response>", first match wins. The
# response is a file in $WORK or "exit:<code>". An unrouted URL fails like an
# HTTP 404 (exit 22). Every URL is logged to $CURL_LOG.
mkdir "${WORK}/bin"
cat > "${WORK}/bin/curl" << 'EOF'
#!/usr/bin/env bash
out="" url=""
while [ $# -gt 0 ]; do
  case "$1" in
    -o) out="$2"; shift 2 ;;
    http*) url="$1"; shift ;;
    *) shift ;;
  esac
done
echo "${url}" >> "${CURL_LOG}"
while read -r pattern response; do
  [ -n "${pattern}" ] || continue
  # shellcheck disable=SC2254 # the route is a glob
  case "${url}" in
    ${pattern})
      case "${response}" in
        exit:*) exit "${response#exit:}" ;;
        *) cp "${WORK}/${response}" "${out}" ;;
      esac
      exit 0
      ;;
  esac
done < "${ROUTES}"
echo "curl stub: no route for ${url}" >&2
exit 22
EOF
chmod +x "${WORK}/bin/curl"
export WORK

# routes <name> <route>...: writes a routes file. Each route is "<glob> <response>".
# The workflow job lists are always routed, so only the script's own checks keep
# a fast path or a failed gate from counting.
routes() {
  local name="$1"
  shift
  {
    printf '%s\n' "$@"
    echo "*/v2/workflow/${ID_MAIN}/job main.json"
    echo "*/v2/workflow/${ID_FAILED_MAIN}/job main-failed.json"
    echo "*/v2/workflow/${ID_RUST_CI}/job rust-ci.json"
    echo "*/v2/workflow/${ID_RUST_E2E}/job rust-e2e.json"
    echo "*/v2/workflow/${ID_FAST_1}/job fast-1.json"
    echo "*/v2/workflow/${ID_FAST_2}/job fast-2.json"
    echo "*/v2/workflow/${ID_FAST_3}/job fast-3.json"
  } > "${WORK}/routes-${name}"
}

PASS=0
FAIL=0

# scenario <name> <routes name> <expected main> <expected rust_ci> <expected rust_e2e> <max requests>
# Runs the script with the environment in ENV_* (a develop push of $SHA by default).
scenario() {
  local name="$1" expected="$3 $4 $5" max_requests="$6" got requests status=0
  export ROUTES="${WORK}/routes-$2" CURL_LOG="${WORK}/curl.log"
  : > "${CURL_LOG}"
  echo '{"c-rust_changes_detected": false}' > "${WORK}/params.json"
  PATH="${WORK}/bin:${PATH}" OUTPUT="${WORK}/params.json" \
    BRANCH="${ENV_BRANCH:-develop}" TRIGGER_SOURCE="${ENV_TRIGGER:-webhook}" TAG="${ENV_TAG:-}" \
    CIRCLE_SHA1="${SHA}" CIRCLE_PROJECT_USERNAME="${ENV_ORG:-ethereum-optimism}" \
    CIRCLE_PROJECT_REPONAME="optimism" MQ_VERIFY_DEADLINE="${ENV_DEADLINE:-30}" \
    bash "${SCRIPT}" > "${WORK}/out.log" 2>&1 || status=$?
  got=$(jq -r '[."c-mq_verified_main", ."c-mq_verified_rust_ci", ."c-mq_verified_rust_e2e"] | map(tostring) | join(" ")' \
    "${WORK}/params.json")
  requests=$(wc -l < "${CURL_LOG}" | tr -d ' ')
  if [ "${status}" -ne 0 ]; then
    echo "FAIL: ${name}: exited ${status}"
  elif [ "${got}" != "${expected}" ]; then
    echo "FAIL: ${name}: main rust_ci rust_e2e = ${got}, expected ${expected}"
  elif [ "$(jq -r '."c-rust_changes_detected"' "${WORK}/params.json")" != "false" ]; then
    echo "FAIL: ${name}: dropped the params already in the JSON"
  elif [ "${requests}" -gt "${max_requests}" ]; then
    echo "FAIL: ${name}: ${requests} API requests, expected at most ${max_requests}"
  else
    echo "PASS: ${name}"
    PASS=$((PASS + 1))
    return 0
  fi
  sed 's/^/    /' "${WORK}/out.log" "${CURL_LOG}"
  FAIL=$((FAIL + 1))
}

# expect_requests <name> <pattern> <count>: the last scenario requested exactly
# <count> URLs containing <pattern>.
expect_requests() {
  local n
  n=$(grep -c -- "$2" "${WORK}/curl.log" || true)
  if [ "${n}" -ne "$3" ]; then
    echo "FAIL: $1: ${n} requests for $2, expected $3"
    FAIL=$((FAIL + 1))
  fi
}

echo "=== merge-queue-verified.sh offline tests ==="

routes verified "${PIPELINES} queue.json" "${TREE} full.json"
scenario "queue passed main, rust-ci and rust-e2e-ci" verified true true true 5

routes rust-fast "${PIPELINES} queue.json" "${TREE} rust-fast.json"
scenario "queue ran the Rust fast paths" rust-fast true false false 3

# The docs fast paths' pipeline is on the second page of the pipeline list.
routes docs-fast "${PIPELINES} no-queue.json" "${PIPELINES}?page-token=page-2 queue.json" "${TREE} docs-fast.json"
scenario "queue ran only the docs fast paths" docs-fast false false false 3
expect_requests "docs fast paths" "/pipeline" 2
expect_requests "docs fast paths" "/tree/" 1

routes page-2 "${PIPELINES} no-queue.json" "${PIPELINES}?page-token=page-2 queue.json" "${TREE} full.json"
scenario "queue pipeline on the second page" page-2 true true true 6
expect_requests "second page" "/pipeline" 2

# As when the queue branch's pipeline is triggered again.
routes twice "${PIPELINES} queue-twice.json" "${TREE} full.json"
scenario "queue branch listed twice" twice true true true 5
expect_requests "queue branch listed twice" "/tree/" 1

routes rerun "${PIPELINES} queue.json" "${TREE} rerun.json"
scenario "main failed, then passed on a re-run" rerun true true true 6
expect_requests "re-run" "/v2/workflow/${ID_FAILED_MAIN}/job" 1

routes gates "*/v2/workflow/${ID_MAIN}/job main-failed.json" "*/v2/workflow/${ID_RUST_CI}/job rust-ci-running.json" \
  "${PIPELINES} queue.json" "${TREE} full.json"
scenario "ci-gate failed, required-rust-ci still running" gates false false true 5

routes other-sha "${PIPELINES} queue.json" "${TREE} full-other-sha.json"
scenario "queue branch's jobs built another commit" other-sha false false false 2

routes no-queue "${PIPELINES} no-queue.json" "${PIPELINES}?page-token=* no-queue-again.json"
scenario "no queue pipeline (admin merge or direct push) stops after 5 pages" no-queue false false false 5
expect_requests "no queue pipeline" "/pipeline" 5
expect_requests "no queue pipeline" "/tree/" 0

routes http-error "${PIPELINES} exit:22"
scenario "pipeline list returns an HTTP error" http-error false false false 1

routes tree-timeout "*/tree/* exit:28" "${PIPELINES} queue.json"
scenario "job list times out" tree-timeout false false false 2

routes wf-timeout "*/v2/workflow/${ID_MAIN}/job exit:28" "${PIPELINES} queue.json" "${TREE} full.json"
scenario "one workflow's job list times out" wf-timeout false true true 5

routes not-json "${PIPELINES} not-json.json"
scenario "pipeline list is not JSON" not-json false false false 1

routes empty "${PIPELINES} empty.json"
scenario "pipeline list is an empty 200" empty false false false 1
if ! grep -q "Could not list pipelines" "${WORK}/out.log"; then
  echo "FAIL: empty pipeline list: not reported as a listing error"
  FAIL=$((FAIL + 1))
fi

ENV_DEADLINE=0 scenario "deadline already passed" verified false false false 0

# Every other pipeline records false without a request.
ENV_BRANCH="feat/my-thing" scenario "PR branch push" verified false false false 0
ENV_BRANCH="${QUEUE}" scenario "merge queue push" verified false false false 0
ENV_TRIGGER="scheduled_pipeline" scenario "scheduled pipeline on develop" verified false false false 0
ENV_TRIGGER="api" scenario "API-triggered pipeline on develop" verified false false false 0
ENV_ORG="other-org" scenario "develop push outside ethereum-optimism/optimism" verified false false false 0

echo ""
echo "=== Results: ${PASS} passed, ${FAIL} failed ==="
[ "${FAIL}" -eq 0 ]
