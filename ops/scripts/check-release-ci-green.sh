#!/usr/bin/env bash
set -euo pipefail

# Fails unless the commit a release is built from has green required CI.
#
# Usage: check-release-ci-green.sh <commit-sha-or-tag>
#
# The required gates come from the commit's own CircleCI config: ci-gate, plus
# each of required-contracts-ci, required-rust-ci and required-rust-e2e that
# the commit's .circleci/ defines as a job. So the commit must be in the local
# clone at RELEASE_CI_CONFIG_REPO (default: the current directory).
#
# Each gate's status ("ci/circleci: <job>") is read from the GitHub API, every
# page (develop commits carry more than 100 statuses, so page one can hide the
# Rust gates). Every gate must be success. A gate that is pending or not posted
# yet is polled until the next poll would start after RELEASE_CI_TIMEOUT_SECONDS,
# then fails. A failed gate fails at once. Jobs that run this need a limit a few
# minutes above the timeout, for the last poll's API reads.
#
# Override: set ALLOW_RED_CI_REASON to "<full commit SHA>: <reason>". It only
# applies to that commit, so a value left set cannot release another one. The
# check then reads CI once, prints what it found and the reason, and exits 0.
# A value without this commit's SHA is ignored, with a warning.
#
# Env:
#   GITHUB_REPOSITORY                 owner/repo (default: the CircleCI project,
#                                     then ethereum-optimism/optimism)
#   GITHUB_TOKEN, GH_TOKEN or REPO_GITHUB_TOKEN   API token (optional, never printed)
#   GITHUB_API_URL                    API base (default https://api.github.com)
#   RELEASE_CI_CONFIG_REPO            git clone that has the commit (default .)
#   RELEASE_CI_TIMEOUT_SECONDS        total wait for pending gates (default 2700)
#   RELEASE_CI_POLL_SECONDS           wait between polls (default 120)
#   RELEASE_CI_FIXTURES               test only: read API responses from this dir
#
# Tests: ops/scripts/check-release-ci-green.test.sh

GATE_JOB="ci-gate"
IF_DEFINED_JOBS=(required-contracts-ci required-rust-ci required-rust-e2e)

CIRCLE_REPO=""
if [ -n "${CIRCLE_PROJECT_USERNAME:-}" ] && [ -n "${CIRCLE_PROJECT_REPONAME:-}" ]; then
  CIRCLE_REPO="${CIRCLE_PROJECT_USERNAME}/${CIRCLE_PROJECT_REPONAME}"
fi
REPO="${GITHUB_REPOSITORY:-${CIRCLE_REPO:-ethereum-optimism/optimism}}"
API="${GITHUB_API_URL:-https://api.github.com}"
CONFIG_REPO="${RELEASE_CI_CONFIG_REPO:-.}"
TIMEOUT="${RELEASE_CI_TIMEOUT_SECONDS:-2700}"
POLL="${RELEASE_CI_POLL_SECONDS:-120}"
FIXTURES="${RELEASE_CI_FIXTURES:-}"
PER_PAGE=100
MAX_PAGES=20

REF="${1:-}"
if [ -z "$REF" ]; then
  echo "usage: $0 <commit-sha-or-tag>" >&2
  exit 2
fi
# The ref goes into an API path, so allow only the characters git refs here use.
if [[ ! "$REF" =~ ^[A-Za-z0-9][A-Za-z0-9._/+-]*$ ]] || [[ "$REF" == *..* ]]; then
  echo "error: '$REF' is not a plain tag or commit name" >&2
  exit 2
fi

# GitHub Actions annotation when running there, plain text otherwise.
annotate() {
  local level="$1" msg="$2"
  if [ -n "${GITHUB_ACTIONS:-}" ]; then
    echo "::${level}::${msg}"
  else
    echo "${level}: ${msg}"
  fi
}

TOKEN="${GITHUB_TOKEN:-${GH_TOKEN:-${REPO_GITHUB_TOKEN:-}}}"

# api_get <path> <poll> <page>: prints the JSON body. The token goes to curl on
# stdin, so it never shows in the process list or the log.
api_get() {
  local path="$1" poll="$2" page="$3"
  if [ -n "$FIXTURES" ]; then
    fixture_get "$path" "$poll" "$page"
    return
  fi
  local auth=""
  if [ -n "$TOKEN" ]; then
    auth="header = \"Authorization: Bearer ${TOKEN}\""
  fi
  printf '%s\n' "$auth" | curl -fsS --retry 4 --retry-all-errors --max-time 30 -K - \
    -H "Accept: application/vnd.github+json" \
    -H "X-GitHub-Api-Version: 2022-11-28" \
    "${API}/${path}"
}

# Fixture layout: commit.json for the ref lookup, status-<poll>-<page>.json for
# status pages. A poll with no file reuses the latest earlier poll.
fixture_get() {
  local path="$1" poll="$2" page="$3" n
  case "$path" in
    */status\?*)
      for ((n = poll; n >= 1; n--)); do
        if [ -f "$FIXTURES/status-$n-1.json" ]; then
          if [ -f "$FIXTURES/status-$n-$page.json" ]; then
            cat "$FIXTURES/status-$n-$page.json"
          else
            echo '{"statuses": []}'
          fi
          return
        fi
      done
      return 22
      ;;
    *) cat "$FIXTURES/commit.json" ;;
  esac
}

# Resolve a tag (annotated or not) or short SHA to the full commit SHA.
if [[ "$REF" =~ ^[0-9a-f]{40}$ ]]; then
  SHA="$REF"
else
  SHA="$(api_get "repos/${REPO}/commits/${REF}" 0 0 | jq -er '.sha')" || {
    annotate error "could not resolve '${REF}' to a commit in ${REPO}"
    exit 1
  }
fi
echo "Checking required CI on ${REPO}@${SHA} (${REF})"
if [ -n "${GITHUB_OUTPUT:-}" ]; then
  echo "sha=${SHA}" >> "$GITHUB_OUTPUT"
fi

# The override counts only when it names this exact commit.
OVERRIDE=""
if [ -n "${ALLOW_RED_CI_REASON:-}" ]; then
  if [[ "$ALLOW_RED_CI_REASON" == "${SHA}:"* ]]; then
    OVERRIDE="${ALLOW_RED_CI_REASON#"${SHA}:"}"
    OVERRIDE="${OVERRIDE#"${OVERRIDE%%[![:space:]]*}"}"
  fi
  if [ -z "$OVERRIDE" ]; then
    annotate warning "ALLOW_RED_CI_REASON is set but is not '${SHA}: <reason>', so it is ignored for this commit."
  fi
fi

# The gates this commit's own CircleCI config defines.
if ! git -C "$CONFIG_REPO" cat-file -e "${SHA}^{commit}" 2> /dev/null; then
  annotate error "commit ${SHA} is not in the clone at ${CONFIG_REPO}, so its CI config cannot be read"
  exit 1
fi
# defines_job <job>: true when the commit's .circleci/ defines that job.
defines_job() {
  local rc=0
  git -C "$CONFIG_REPO" grep -q -e "^  ${1}:" "$SHA" -- .circleci/ || rc=$?
  if [ "$rc" -gt 1 ]; then
    annotate error "could not read .circleci/ at ${SHA} in ${CONFIG_REPO}"
    exit 1
  fi
  return "$rc"
}
NO_GATE=""
REQUIRED_CONTEXTS=("ci/circleci: ${GATE_JOB}")
defines_job "$GATE_JOB" || NO_GATE=1
for job in "${IF_DEFINED_JOBS[@]}"; do
  if defines_job "$job"; then
    REQUIRED_CONTEXTS+=("ci/circleci: ${job}")
  fi
done
echo "Required gates (from .circleci/ at ${SHA}):"
printf '  %s\n' "${REQUIRED_CONTEXTS[@]}"

# fetch_statuses <poll>: prints a JSON array of {context, state, target_url},
# the latest status per context, across every page.
fetch_statuses() {
  local poll="$1" page body count all="[]"
  for ((page = 1; page <= MAX_PAGES; page++)); do
    body="$(api_get "repos/${REPO}/commits/${SHA}/status?per_page=${PER_PAGE}&page=${page}" "$poll" "$page")" || return 1
    all="$(jq -c --argjson acc "$all" '$acc + [.statuses[] | {context, state, target_url}]' <<< "$body")" || return 1
    count="$(jq -r '.statuses | length' <<< "$body")"
    if [ "$count" -lt "$PER_PAGE" ]; then
      echo "$all"
      return 0
    fi
  done
  echo "more than $((PER_PAGE * MAX_PAGES)) statuses, refusing to guess" >&2
  return 1
}

# evaluate <statuses-json>: prints one line per gate ("<state>\t<context>\t<url>")
# where state is success, pending, failure, error or missing (not posted yet).
evaluate() {
  local statuses="$1" ctx
  for ctx in "${REQUIRED_CONTEXTS[@]}"; do
    jq -r --arg c "$ctx" '([.[] | select(.context == $c)] | first) as $s
      | if $s then "\($s.state)\t\($c)\t\($s.target_url // "")" else "missing\t\($c)\t(not posted yet)" end' <<< "$statuses"
  done
}

start=$SECONDS
poll=0
last_report=""
while :; do
  poll=$((poll + 1))
  elapsed=$((SECONDS - start))
  if [ -n "$NO_GATE" ]; then
    report="$(printf 'none\t%s\t%s' "ci/circleci: ${GATE_JOB}" "(no ${GATE_JOB} job in this commit's .circleci/)")"
  elif statuses="$(fetch_statuses "$poll")"; then
    report="$(evaluate "$statuses")"
  else
    report="$(printf 'unknown\t%s\t%s' "GitHub API" "(request failed, will retry)")"
  fi
  if [ "$report" != "$last_report" ]; then
    echo "--- after ${elapsed}s:"
    awk -F'\t' '{ printf "  %-8s %s %s\n", $1, $2, $3 }' <<< "$report"
    last_report="$report"
  fi

  states="$(cut -f1 <<< "$report")"
  verdict="green"
  if grep -qx none <<< "$states"; then
    verdict="no-ci"
  elif grep -qxE 'failure|error' <<< "$states"; then
    verdict="red"
  elif grep -qxE 'pending|missing|unknown' <<< "$states"; then
    verdict="wait"
  fi

  if [ -n "$OVERRIDE" ]; then
    annotate warning "ALLOW_RED_CI_REASON names ${SHA}. Required CI there is '${verdict}', and the release continues anyway. Reason: ${OVERRIDE}"
    if [ -n "${GITHUB_STEP_SUMMARY:-}" ]; then
      # shellcheck disable=SC2016 # backticks are Markdown, not a command
      printf '### Release built without green CI\n\nCommit `%s` (%s), CI state `%s`.\n\nReason: %s\n' \
        "$SHA" "$REF" "$verdict" "$OVERRIDE" >> "$GITHUB_STEP_SUMMARY"
    fi
    exit 0
  fi

  how="For an emergency release, set ALLOW_RED_CI_REASON to '${SHA}: <reason>'."
  case "$verdict" in
    green)
      echo "Required CI is green on ${SHA}."
      exit 0
      ;;
    red)
      annotate error "Required CI failed on ${SHA}. Rerun or fix CI on that commit, then rerun this release. ${how}"
      exit 1
      ;;
    no-ci)
      annotate error "${SHA} has no ${GATE_JOB} job in its .circleci/, so there is no CI result to check. ${how}"
      exit 1
      ;;
  esac

  # Stop when the next poll would start after the limit, so the job's own limit
  # (a few minutes above it) is never reached first.
  elapsed=$((SECONDS - start))
  if [ "$elapsed" -ge "$TIMEOUT" ] || [ $((elapsed + POLL)) -gt "$TIMEOUT" ]; then
    waiting="$(awk -F'\t' '$1 != "success" { printf "%s%s (%s)", sep, $2, $1; sep = ", " }' <<< "$report")"
    annotate error "Required CI on ${SHA} is not finished after ${elapsed}s and ${poll} polls (limit ${TIMEOUT}s): ${waiting}. A gate still missing means CircleCI never posted it for this commit. Rerun this release when CI is green. ${how}"
    exit 1
  fi
  sleep "$POLL"
done
