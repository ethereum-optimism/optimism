#!/usr/bin/env bash
set -euo pipefail

# Fails unless the commit a release is built from has green required CI.
#
# Usage: check-release-ci-green.sh <commit-sha-or-tag>
#
# Reads the commit's combined status from the GitHub API (every page: develop
# commits carry more than 100 statuses, so page one can hide the Rust gates).
# "ci/circleci: ci-gate" must be success. The other required gates must be
# success when the commit has them (not every branch runs every workflow).
# Pending or not yet posted gates are polled until RELEASE_CI_TIMEOUT_SECONDS.
# A failed gate fails at once. A required gate still missing after
# RELEASE_CI_MISSING_GRACE_SECONDS fails: CI never ran on this commit.
#
# Override: set ALLOW_RED_CI_REASON to a reason. The check then runs once,
# prints what it found and the reason, and exits 0 whatever the result.
#
# Env:
#   GITHUB_REPOSITORY                 owner/repo (default ethereum-optimism/optimism)
#   GITHUB_TOKEN, GH_TOKEN or REPO_GITHUB_TOKEN   API token (optional, never printed)
#   GITHUB_API_URL                    API base (default https://api.github.com)
#   RELEASE_CI_TIMEOUT_SECONDS        total wait for pending gates (default 2700)
#   RELEASE_CI_POLL_SECONDS           wait between polls (default 120)
#   RELEASE_CI_MISSING_GRACE_SECONDS  wait for gates not yet posted (default 600)
#   RELEASE_CI_FIXTURES               test only: read API responses from this dir
#
# Tests: ops/scripts/check-release-ci-green.test.sh

REQUIRED_CONTEXTS=("ci/circleci: ci-gate")
IF_PRESENT_CONTEXTS=(
  "ci/circleci: required-contracts-ci"
  "ci/circleci: required-rust-ci"
  "ci/circleci: required-rust-e2e"
)

REPO="${GITHUB_REPOSITORY:-ethereum-optimism/optimism}"
API="${GITHUB_API_URL:-https://api.github.com}"
TIMEOUT="${RELEASE_CI_TIMEOUT_SECONDS:-2700}"
POLL="${RELEASE_CI_POLL_SECONDS:-120}"
GRACE="${RELEASE_CI_MISSING_GRACE_SECONDS:-600}"
OVERRIDE="${ALLOW_RED_CI_REASON:-}"
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
# where state is success, pending, failure, error, missing or absent.
evaluate() {
  local statuses="$1" ctx
  for ctx in "${REQUIRED_CONTEXTS[@]}"; do
    jq -r --arg c "$ctx" '([.[] | select(.context == $c)] | first) as $s
      | if $s then "\($s.state)\t\($c)\t\($s.target_url // "")" else "missing\t\($c)\t" end' <<< "$statuses"
  done
  for ctx in "${IF_PRESENT_CONTEXTS[@]}"; do
    jq -r --arg c "$ctx" '([.[] | select(.context == $c)] | first) as $s
      | if $s then "\($s.state)\t\($c)\t\($s.target_url // "")" else "absent\t\($c)\t" end' <<< "$statuses"
  done
}

start=$SECONDS
poll=0
last_report=""
while :; do
  poll=$((poll + 1))
  elapsed=$((SECONDS - start))
  if statuses="$(fetch_statuses "$poll")"; then
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
  if grep -qxE 'failure|error' <<< "$states"; then
    verdict="red"
  elif grep -qx missing <<< "$states" && [ "$elapsed" -ge "$GRACE" ]; then
    verdict="missing"
  elif grep -qxE 'pending|missing|unknown' <<< "$states"; then
    verdict="wait"
  fi

  if [ -n "$OVERRIDE" ]; then
    annotate warning "ALLOW_RED_CI_REASON is set. Required CI on ${SHA} is '${verdict}', and the release continues anyway. Reason: ${OVERRIDE}"
    if [ -n "${GITHUB_STEP_SUMMARY:-}" ]; then
      # shellcheck disable=SC2016 # backticks are Markdown, not a command
      printf '### Release built without green CI\n\nCommit `%s` (%s), CI state `%s`.\n\nReason: %s\n' \
        "$SHA" "$REF" "$verdict" "$OVERRIDE" >> "$GITHUB_STEP_SUMMARY"
    fi
    exit 0
  fi

  case "$verdict" in
    green)
      echo "Required CI is green on ${SHA}."
      exit 0
      ;;
    red)
      annotate error "Required CI failed on ${SHA}. Rerun or fix CI on that commit, then rerun this release. For an emergency release, set ALLOW_RED_CI_REASON."
      exit 1
      ;;
    missing)
      annotate error "Required CI never reported on ${SHA} within ${GRACE}s. Releases must come from a commit CI ran on. For an emergency release, set ALLOW_RED_CI_REASON."
      exit 1
      ;;
  esac

  if [ "$elapsed" -ge "$TIMEOUT" ]; then
    annotate error "Required CI on ${SHA} is still pending after ${TIMEOUT}s. Rerun this release when it finishes. For an emergency release, set ALLOW_RED_CI_REASON."
    exit 1
  fi
  sleep "$POLL"
done
