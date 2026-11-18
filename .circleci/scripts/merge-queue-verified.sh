#!/usr/bin/env bash
# Records which of main, rust-ci and rust-e2e-ci the merge queue already passed
# on this develop push's commit, as c-mq_verified_<key> booleans in the
# pipeline parameters JSON. compute-workflow-conditions.sh turns them into the
# c-develop_mq_verified_* params that halt test-only jobs on the develop push.
#
# GitHub's merge queue tests each PR on gh-readonly-queue/develop/pr-<n>-<base>.
# When the required checks pass, it fast-forwards develop to that same commit,
# so the develop push builds a commit the queue has just tested.
#
# A workflow counts as verified only if a merge queue pipeline for CIRCLE_SHA1
# ran a workflow with exactly that name and the workflow's gate job succeeded:
#   main        -> ci-gate
#   rust-ci     -> required-rust-ci
#   rust-e2e-ci -> required-rust-e2e
# The queue's fast paths (ci-gate-skip, rust-ci-gate-short, rust-e2e-gate-skip)
# emit the same gate names from other workflows without running the tests, so
# they never count.
#
# Fails open. An API error or timeout, no queue pipeline (an admin merge or a
# direct push), or a gate that failed or hasn't finished leaves the workflow
# unverified, and develop runs it in full. The script always exits 0.
#
# It only calls the API for webhook pushes to develop in
# ethereum-optimism/optimism. Every other pipeline records false.
#
# Inputs: BRANCH, TRIGGER_SOURCE and TAG (set by the config.yml step),
# CIRCLE_SHA1, CIRCLE_PROJECT_USERNAME and CIRCLE_PROJECT_REPONAME.
# Optional: OUTPUT (the params JSON), CIRCLECI_API (API base URL),
# MQ_VERIFY_DEADLINE (seconds for all requests, default 30) and CIRCLE_MQ_TOKEN.
#
# The endpoints used here need no token while the project is public. If it goes
# private, set CIRCLE_MQ_TOKEN (a personal API token) in a restricted context.
# v2 pipeline/<id>/workflow needs a token even now, so the queue pipeline's
# workflows come from the v1.1 job list of its branch instead.
set -uo pipefail

OUTPUT="${OUTPUT:-/tmp/pipeline-parameters.json}"
API="${CIRCLECI_API:-https://circleci.com/api}"
DEADLINE="${MQ_VERIFY_DEADLINE:-30}"
SHA="${CIRCLE_SHA1:-}"
SLUG="gh/${CIRCLE_PROJECT_USERNAME:-}/${CIRCLE_PROJECT_REPONAME:-}"
QUEUE_PREFIX="gh-readonly-queue/develop/"
# The queue pipeline is a few pipelines older than the develop push (at most
# 8 apart in a week of data), and a page holds 20.
MAX_PIPELINE_PAGES=5

# <param key>:<workflow name>:<gate job>
CHECKS="main:main:ci-gate rust_ci:rust-ci:required-rust-ci rust_e2e:rust-e2e-ci:required-rust-e2e"

VERIFIED=""
TOKEN_HEADER=()
if [ -n "${CIRCLE_MQ_TOKEN:-}" ]; then
  TOKEN_HEADER=(-H "Circle-Token: ${CIRCLE_MQ_TOKEN}")
fi

log() {
  echo "$*" >&2
}

# Adds one boolean per check to OUTPUT (true only for the keys in VERIFIED) and
# exits 0. If OUTPUT can't be updated it is left as it was, which reads as false.
finish() {
  local json check key value filter="."
  for check in ${CHECKS}; do
    key=${check%%:*}
    value=false
    case " ${VERIFIED} " in *" ${key} "*) value=true ;; esac
    filter="${filter} + {\"c-mq_verified_${key}\": ${value}}"
    echo "  [mq] c-mq_verified_${key} = ${value}"
  done
  if json=$(jq "${filter}" "${OUTPUT}"); then
    echo "${json}" > "${OUTPUT}"
  fi
  exit 0
}

# api_get <url> <file>: GET within what is left of the deadline.
api_get() {
  local left=$((DEADLINE - SECONDS))
  if [ "${left}" -le 0 ]; then
    log "Out of time (${DEADLINE}s) for CircleCI API requests"
    return 1
  fi
  [ "${left}" -lt 10 ] || left=10
  curl -fsS --max-time "${left}" --retry 2 --retry-delay 1 --retry-max-time "${left}" \
    ${TOKEN_HEADER[@]+"${TOKEN_HEADER[@]}"} -o "$2" "$1"
}

# Prints the merge queue branches whose pipelines built SHA.
queue_branches() {
  local page=1 token=""
  while [ "${page}" -le "${MAX_PIPELINE_PAGES}" ]; do
    api_get "${API}/v2/project/${SLUG}/pipeline${token:+?page-token=${token}}" "${TMP}/pipelines.json" || return 1
    # An empty or item-less body is an error, not an empty page.
    jq -e '.items | type == "array"' "${TMP}/pipelines.json" > /dev/null || return 1
    # A pipeline triggered again on the same queue branch lists it twice.
    jq -r --arg sha "${SHA}" --arg prefix "${QUEUE_PREFIX}" \
      '[.items[] | select(.vcs.revision == $sha) | .vcs.branch // empty | select(startswith($prefix))] | unique | .[]' \
      "${TMP}/pipelines.json" > "${TMP}/found" || return 1
    if [ -s "${TMP}/found" ]; then
      cat "${TMP}/found"
      return 0
    fi
    token=$(jq -r '.next_page_token // empty | @uri' "${TMP}/pipelines.json") || return 1
    [ -n "${token}" ] || return 0
    page=$((page + 1))
  done
}

# queue_workflows <branch>: prints "<workflow name> <workflow id>" for each of
# the branch's jobs that built SHA.
queue_workflows() {
  local branch offset rows
  branch=$(jq -rn --arg b "$1" '$b | @uri') || return 1
  for offset in 0 100 200; do
    api_get "${API}/v1.1/project/${SLUG}/tree/${branch}?shallow=true&limit=100&offset=${offset}" "${TMP}/jobs.json" || return 1
    jq -r --arg sha "${SHA}" \
      '.[] | select(.vcs_revision == $sha and .workflows != null) | "\(.workflows.workflow_name) \(.workflows.workflow_id)"' \
      "${TMP}/jobs.json" || return 1
    rows=$(jq 'length' "${TMP}/jobs.json") || return 1
    [ "${rows}" -eq 100 ] || return 0
  done
}

# gate_status <workflow id> <gate job>: prints the gate job's status, or
# "missing" if the workflow has no such job.
gate_status() {
  local page token="" status
  for page in 1 2 3; do
    api_get "${API}/v2/workflow/$1/job${token:+?page-token=${token}}" "${TMP}/workflow.json" || return 1
    status=$(jq -r --arg gate "$2" '[.items[] | select(.name == $gate) | .status][0] // empty' "${TMP}/workflow.json") || return 1
    if [ -n "${status}" ]; then
      echo "${status}"
      return 0
    fi
    token=$(jq -r '.next_page_token // empty | @uri' "${TMP}/workflow.json") || return 1
    [ -n "${token}" ] || break
  done
  echo "missing"
}

if [[ "${TRIGGER_SOURCE:-}" != "webhook" || "${BRANCH:-}" != "develop" || -n "${TAG:-}" ]]; then
  log "Not a push to develop, so the merge queue check is skipped."
  finish
fi
if [[ "${SLUG}" != "gh/ethereum-optimism/optimism" ]]; then
  log "Not ethereum-optimism/optimism, so the merge queue check is skipped."
  finish
fi
if ! [[ "${SHA}" =~ ^[0-9a-f]{40}$ ]]; then
  log "CIRCLE_SHA1 '${SHA}' is not a commit hash, so the merge queue check is skipped."
  finish
fi

TMP=$(mktemp -d) || finish
trap 'rm -rf "${TMP}"' EXIT

if ! branches=$(queue_branches); then
  log "Could not list pipelines. Develop runs every workflow in full."
  finish
fi
if [ -z "${branches}" ]; then
  log "No merge queue pipeline built ${SHA} among the newest pipelines listed (an admin merge or a direct push?). Develop runs every workflow in full."
  finish
fi

: > "${TMP}/workflows"
for branch in ${branches}; do
  log "Merge queue pipeline for ${SHA}: ${branch}"
  if ! queue_workflows "${branch}" >> "${TMP}/workflows"; then
    log "Could not list the jobs on ${branch}. Develop runs every workflow in full."
    finish
  fi
done

for check in ${CHECKS}; do
  key=${check%%:*}
  rest=${check#*:}
  workflow=${rest%%:*}
  gate=${rest#*:}
  ids=$(awk -v w="${workflow}" '$1 == w { print $2 }' "${TMP}/workflows" | sort -u)
  reason="the queue pipeline has no ${workflow} workflow"
  for id in ${ids}; do
    [[ "${id}" =~ ^[0-9a-f-]{36}$ ]] || continue
    if ! status=$(gate_status "${id}" "${gate}"); then
      reason="could not list the jobs of workflow ${id}"
      continue
    fi
    reason="${gate} is ${status} in workflow ${id}"
    if [ "${status}" = "success" ]; then
      VERIFIED="${VERIFIED} ${key}"
      break
    fi
  done
  case " ${VERIFIED} " in
    *" ${key} "*) log "${workflow}: verified (${reason})" ;;
    *) log "${workflow}: not verified, runs in full (${reason})" ;;
  esac
done

finish
