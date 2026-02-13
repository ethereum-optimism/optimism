#!/usr/bin/env bash

set -eo pipefail

LOG_FILE="$1"

if [ -z "$LOG_FILE" ]; then
  echo "Usage: $0 <log.json path>" >&2
  exit 1
fi

STATUS=$(jq -r '.status // empty' "$LOG_FILE")
PR_URL=$(jq -r '.pull_request_url // empty' "$LOG_FILE")
TEST_FILE=$(jq -r '.selected_files.test_path | split("/") | .[-1]' "$LOG_FILE")

if [ "$STATUS" = "no_changes_needed" ] && [ -n "$PR_URL" ]; then
  # No changes needed but PR opened to add TOML tracking entry
  MESSAGE=$'<!subteam^S07K486JEH4> AI Contracts Test Maintenance System analyzed '"${TEST_FILE}"$' - no changes needed (test coverage already comprehensive)\n<'"${PR_URL}"$'|View PR to add no-changes tracking>'
  SLACK_JSON=$(jq -n --arg msg "$MESSAGE" '{"text": $msg}')
  echo "$SLACK_JSON"
elif [ -n "$PR_URL" ]; then
  # Normal case: PR with test improvements
  MESSAGE=$'<!subteam^S07K486JEH4> AI Contracts Test Maintenance System created a PR for '"${TEST_FILE}"$'\n<'"${PR_URL}"$'|View PR> | <https://www.notion.so/oplabs/AI-Contract-Test-Maintenance-System-PR-Reviewer-Guide-288f153ee16280478c0ed1adc5edd9f9|Reviewer Guide>'
  SLACK_JSON=$(jq -n --arg msg "$MESSAGE" '{"text": $msg}')
  echo "$SLACK_JSON"
elif [ "$STATUS" = "no_changes_needed" ]; then
  # Edge case: no changes and no PR (shouldn't happen with new workflow)
  MESSAGE=$'<!subteam^S07K486JEH4> AI Contracts Test Maintenance System analyzed '"${TEST_FILE}"$' - no changes needed (test coverage already comprehensive)'
  SLACK_JSON=$(jq -n --arg msg "$MESSAGE" '{"text": $msg}')
  echo "$SLACK_JSON"
else
  echo "No notification needed (status: $STATUS)" >&2
  echo '{}'
fi
