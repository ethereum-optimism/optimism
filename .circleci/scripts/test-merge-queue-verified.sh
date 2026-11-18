#!/usr/bin/env bash
# Offline test for merge-queue-verified.sh. A stub curl answers each request
# from recorded CircleCI API responses in testdata/merge-queue-verified, or
# fails the way curl does, so every scenario is deterministic.
#
# The fixtures are real responses, trimmed to the fields the script reads:
#   pipelines-page{1,2}.json  v2 project pipeline list (page tokens rewritten)
#   tree-<case>-0.json        v1.1 job list of the case's merge queue branch
#   jobs-<case>-<wf>.json     v2 job list of each workflow in that pipeline
# The cases are three queue pipelines: "verified" ran main, rust-ci and
# rust-e2e-ci; "rustskip" ran main with the Rust gates on their fast paths;
# "docsonly" ran only the fast paths.
#
# Usage:
#   bash .circleci/scripts/test-merge-queue-verified.sh
#
# Requires: jq
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPT="${SCRIPT_DIR}/merge-queue-verified.sh"
FIXTURES="${SCRIPT_DIR}/testdata/merge-queue-verified"
WORK="$(mktemp -d)"
trap 'rm -rf "${WORK}"' EXIT

SHA_VERIFIED=89b4a22c130ef374fd15859bf5137b9a1b65b1da
SHA_RUSTSKIP=416753f094b0edcfa37b378b43d8a629eec83d53
SHA_DOCSONLY=e5423ae69b154100a77361d74eb6d7170b2c34aa
SHA_UNKNOWN=0123456789abcdef0123456789abcdef01234567

# Stub curl. Each line of $ROUTES is "<glob> <response>", first match wins. The
# response is a file (relative to $FIXTURES, or absolute) or "exit:<code>". An
# unrouted URL fails like an HTTP 404 (exit 22). Every URL is logged to $CURL_LOG.
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
        /*) cp "${response}" "${out}" ;;
        *) cp "${FIXTURES}/${response}" "${out}" ;;
      esac
      exit 0
      ;;
  esac
done < "${ROUTES}"
echo "curl stub: no route for ${url}" >&2
exit 22
EOF
chmod +x "${WORK}/bin/curl"
export FIXTURES

# routes <case> <sha>: the routes that serve a case's recorded responses. The
# pipeline list is two pages; the second page names a third page that is the
# second page again, so a SHA that is on neither page exhausts the page limit.
routes() {
  local case_name="$1" sha="$2" branch name id
  echo "*/v2/project/gh/ethereum-optimism/optimism/pipeline pipelines-page1.json"
  echo "*/v2/project/gh/ethereum-optimism/optimism/pipeline?page-token=fixture-page-2 pipelines-page2.json"
  echo "*/v2/project/gh/ethereum-optimism/optimism/pipeline?page-token=fixture-page-3 pipelines-page2.json"
  branch=$(jq -r --arg sha "${sha}" \
    '.items[] | select(.vcs.revision == $sha and (.vcs.branch | startswith("gh-readonly-queue/"))) | .vcs.branch | @uri' \
    "${FIXTURES}"/pipelines-page*.json)
  [ -n "${branch}" ] || return 0
  echo "*/v1.1/project/gh/ethereum-optimism/optimism/tree/${branch}?shallow=true&limit=100&offset=0 tree-${case_name}-0.json"
  jq -r '[.[] | "\(.workflows.workflow_name) \(.workflows.workflow_id)"] | unique | .[]' \
    "${FIXTURES}/tree-${case_name}-0.json" > "${WORK}/workflows"
  while read -r name id; do
    if [ -f "${FIXTURES}/jobs-${case_name}-${name}.json" ]; then
      echo "*/v2/workflow/${id}/job jobs-${case_name}-${name}.json"
    fi
  done < "${WORK}/workflows"
}

# workflow_id <case> <workflow name>
workflow_id() {
  jq -r --arg w "$2" '[.[] | select(.workflows.workflow_name == $w) | .workflows.workflow_id][0]' \
    "${FIXTURES}/tree-$1-0.json"
}

# with_status <case> <workflow name> <gate> <status>: a copy of the recorded job
# list with the gate's status replaced. Prints its path.
with_status() {
  local path="${WORK}/jobs-$1-$2-$4.json"
  jq --arg gate "$3" --arg status "$4" \
    '.items |= map(if .name == $gate then .status = $status else . end)' \
    "${FIXTURES}/jobs-$1-$2.json" > "${path}"
  echo "${path}"
}

PASS=0
FAIL=0

# scenario <name> <routes file> <expected main> <expected rust_ci> <expected rust_e2e> <max requests>
# Runs the script with the environment in ENV_* (a develop push by default).
scenario() {
  local name="$1" expected="$3 $4 $5" max_requests="$6" got requests status=0
  export ROUTES="$2" CURL_LOG="${WORK}/curl.log"
  : > "${CURL_LOG}"
  echo '{"c-rust_changes_detected": false}' > "${WORK}/params.json"
  PATH="${WORK}/bin:${PATH}" OUTPUT="${WORK}/params.json" \
    BRANCH="${ENV_BRANCH:-develop}" TRIGGER_SOURCE="${ENV_TRIGGER:-webhook}" TAG="${ENV_TAG:-}" \
    CIRCLE_SHA1="${ENV_SHA}" CIRCLE_PROJECT_USERNAME="${ENV_ORG:-ethereum-optimism}" \
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

routes verified "${SHA_VERIFIED}" > "${WORK}/routes-verified"
routes rustskip "${SHA_RUSTSKIP}" > "${WORK}/routes-rustskip"
routes docsonly "${SHA_DOCSONLY}" > "${WORK}/routes-docsonly"

ENV_SHA="${SHA_VERIFIED}" scenario \
  "queue passed main, rust-ci and rust-e2e-ci" \
  "${WORK}/routes-verified" true true true 5

# rust-ci-gate-short and rust-e2e-gate-skip report required-rust-ci and
# required-rust-e2e as success without testing. Their job lists are routed, so
# only the workflow name check keeps them from counting.
ENV_SHA="${SHA_RUSTSKIP}" scenario \
  "queue ran the Rust fast paths" \
  "${WORK}/routes-rustskip" true false false 3

# ci-gate-skip reports ci-gate as success; the SHA is on the second page.
ENV_SHA="${SHA_DOCSONLY}" scenario \
  "queue ran only the docs fast paths" \
  "${WORK}/routes-docsonly" false false false 3
expect_requests "docs fast paths" "/pipeline" 2
expect_requests "docs fast paths" "/tree/" 1

# The verified queue pipeline is on the second page of the pipeline list.
jq --arg sha "${SHA_VERIFIED}" '.items |= map(select(.vcs.revision != $sha))' \
  "${FIXTURES}/pipelines-page1.json" > "${WORK}/page1-without.json"
jq --arg sha "${SHA_VERIFIED}" --slurpfile p1 "${FIXTURES}/pipelines-page1.json" \
  '.items += [$p1[0].items[] | select(.vcs.revision == $sha)]' \
  "${FIXTURES}/pipelines-page2.json" > "${WORK}/page2-with.json"
{
  echo "*/v2/project/gh/ethereum-optimism/optimism/pipeline ${WORK}/page1-without.json"
  echo "*/v2/project/gh/ethereum-optimism/optimism/pipeline?page-token=fixture-page-2 ${WORK}/page2-with.json"
  grep -v '/pipeline' "${WORK}/routes-verified"
} > "${WORK}/routes-page2"
ENV_SHA="${SHA_VERIFIED}" scenario \
  "queue pipeline on the second page" \
  "${WORK}/routes-page2" true true true 6
expect_requests "second page" "/pipeline" 2

# The same queue branch twice on a page, as when its pipeline is triggered again.
jq --arg sha "${SHA_VERIFIED}" \
  '.items += [.items[] | select(.vcs.revision == $sha and (.vcs.branch | startswith("gh-readonly-queue/")))]' \
  "${FIXTURES}/pipelines-page1.json" > "${WORK}/page1-dup.json"
{
  echo "*/v2/project/gh/ethereum-optimism/optimism/pipeline ${WORK}/page1-dup.json"
  grep -v '/pipeline' "${WORK}/routes-verified"
} > "${WORK}/routes-dup"
ENV_SHA="${SHA_VERIFIED}" scenario \
  "queue branch listed twice" \
  "${WORK}/routes-dup" true true true 5
expect_requests "queue branch listed twice" "/tree/" 1

# A re-run: the queue pipeline's first main workflow failed and a second one
# passed. The failed one's id sorts first, so it is checked first.
RERUN_ID=00000000-0000-4000-8000-000000000000
jq --arg id "${RERUN_ID}" \
  '. + [[.[] | select(.workflows.workflow_name == "main")][0] | .workflows.workflow_id = $id]' \
  "${FIXTURES}/tree-verified-0.json" > "${WORK}/tree-rerun.json"
{
  sed -n 's|^\(\*/v1.1/.*/tree/[^ ]*\) .*|\1 '"${WORK}"'/tree-rerun.json|p' "${WORK}/routes-verified"
  echo "*/v2/workflow/${RERUN_ID}/job $(with_status verified main ci-gate failed)"
  grep -v '/tree/' "${WORK}/routes-verified"
} > "${WORK}/routes-rerun"
ENV_SHA="${SHA_VERIFIED}" scenario \
  "main failed, then passed on a re-run" \
  "${WORK}/routes-rerun" true true true 6
expect_requests "re-run" "/v2/workflow/${RERUN_ID}/job" 1

{
  echo "*/v2/workflow/$(workflow_id verified main)/job $(with_status verified main ci-gate failed)"
  echo "*/v2/workflow/$(workflow_id verified rust-ci)/job $(with_status verified rust-ci required-rust-ci running)"
  cat "${WORK}/routes-verified"
} > "${WORK}/routes-gates"
ENV_SHA="${SHA_VERIFIED}" scenario \
  "ci-gate failed, required-rust-ci still running" \
  "${WORK}/routes-gates" false false true 5

# The rustskip queue branch answers with the verified pipeline's jobs, which
# built another commit, so none of them count.
{
  sed -n 's|^\(\*/v1.1/.*/tree/[^ ]*\) .*|\1 tree-verified-0.json|p' "${WORK}/routes-rustskip"
  grep '/v2/workflow/' "${WORK}/routes-verified"
  grep -v '/tree/' "${WORK}/routes-rustskip"
} > "${WORK}/routes-other-sha"
ENV_SHA="${SHA_RUSTSKIP}" scenario \
  "queue branch's jobs built another commit" \
  "${WORK}/routes-other-sha" false false false 5

ENV_SHA="${SHA_UNKNOWN}" scenario \
  "no queue pipeline (admin merge or direct push) stops after 5 pages" \
  "${WORK}/routes-verified" false false false 5
expect_requests "no queue pipeline" "/pipeline" 5
expect_requests "no queue pipeline" "/tree/" 0

echo "*/pipeline exit:22" > "${WORK}/routes-error"
ENV_SHA="${SHA_VERIFIED}" scenario \
  "pipeline list returns an HTTP error" \
  "${WORK}/routes-error" false false false 1

{
  echo "*/tree/* exit:28"
  cat "${WORK}/routes-verified"
} > "${WORK}/routes-timeout"
ENV_SHA="${SHA_VERIFIED}" scenario \
  "job list times out" \
  "${WORK}/routes-timeout" false false false 2

{
  echo "*/v2/workflow/$(workflow_id verified main)/job exit:28"
  cat "${WORK}/routes-verified"
} > "${WORK}/routes-wf-timeout"
ENV_SHA="${SHA_VERIFIED}" scenario \
  "one workflow's job list times out" \
  "${WORK}/routes-wf-timeout" false true true 5

cp "${FIXTURES}/pipelines-page1.json" "${WORK}/not-json.json"
echo "garbage" >> "${WORK}/not-json.json"
echo "*/pipeline ${WORK}/not-json.json" > "${WORK}/routes-bad-json"
ENV_SHA="${SHA_VERIFIED}" scenario \
  "pipeline list is not JSON" \
  "${WORK}/routes-bad-json" false false false 1

: > "${WORK}/empty.json"
echo "*/pipeline ${WORK}/empty.json" > "${WORK}/routes-empty"
ENV_SHA="${SHA_VERIFIED}" scenario \
  "pipeline list is an empty 200" \
  "${WORK}/routes-empty" false false false 1
if ! grep -q "Could not list pipelines" "${WORK}/out.log"; then
  echo "FAIL: empty pipeline list: not reported as a listing error"
  FAIL=$((FAIL + 1))
fi

ENV_SHA="${SHA_VERIFIED}" ENV_DEADLINE=0 scenario \
  "deadline already passed" \
  "${WORK}/routes-verified" false false false 0

# Every other pipeline records false without a request.
ENV_SHA="${SHA_VERIFIED}" ENV_BRANCH="feat/my-thing" scenario \
  "PR branch push" "${WORK}/routes-verified" false false false 0
ENV_SHA="${SHA_VERIFIED}" ENV_BRANCH="gh-readonly-queue/develop/pr-24242-416753f094b0edcfa37b378b43d8a629eec83d53" scenario \
  "merge queue push" "${WORK}/routes-verified" false false false 0
ENV_SHA="${SHA_VERIFIED}" ENV_TRIGGER="scheduled_pipeline" scenario \
  "scheduled pipeline on develop" "${WORK}/routes-verified" false false false 0
ENV_SHA="${SHA_VERIFIED}" ENV_TRIGGER="api" scenario \
  "API-triggered pipeline on develop" "${WORK}/routes-verified" false false false 0
ENV_SHA="${SHA_VERIFIED}" ENV_ORG="other-org" scenario \
  "develop push outside ethereum-optimism/optimism" "${WORK}/routes-verified" false false false 0

echo ""
echo "=== Results: ${PASS} passed, ${FAIL} failed ==="
[ "${FAIL}" -eq 0 ]
