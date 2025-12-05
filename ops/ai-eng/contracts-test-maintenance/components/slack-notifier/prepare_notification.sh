#!/usr/bin/env bash
set -euo pipefail

# Prepare Slack notification for AI Contracts Test Maintenance System results
# Outputs a JSON message to be used with CircleCI Slack orb

LOG_FILE="../../log.json"

# Extract data from log file
PR_URL=$(jq -r '.pull_request_url // empty' "$LOG_FILE")
TEST_FILE=$(jq -r '.selected_files.test_path | split("/") | .[-1]' "$LOG_FILE")
STATUS=$(jq -r '.status // empty' "$LOG_FILE")

# Prepare message based on outcome
if [ -n "$PR_URL" ]; then
  # PR was created - notify team for review
  MESSAGE=$'<!subteam^S07K486JEH4> AI Contracts Test Maintenance System created a PR for '"${TEST_FILE}"$'\n<'"${PR_URL}"$'|View PR> | <https://www.notion.so/oplabs/AI-Contract-Test-Maintenance-System-PR-Reviewer-Guide-288f153ee16280478c0ed1adc5edd9f9|Reviewer Guide>'
  SLACK_JSON=$(jq -n --arg msg "$MESSAGE" '{"text": $msg}')
  echo "export AI_PR_SLACK_TEMPLATE='${SLACK_JSON}'"
elif [ "$STATUS" = "finished_no_changes" ]; then
  # Analysis complete but no changes needed - informational only
  MESSAGE=$'AI Contracts Test Maintenance System analyzed '"${TEST_FILE}"$' - no changes needed (test coverage is already comprehensive)'
  SLACK_JSON=$(jq -n --arg msg "$MESSAGE" '{"text": $msg}')
  echo "export AI_PR_SLACK_TEMPLATE='${SLACK_JSON}'"
else
  # No notification needed
  echo "No PR created, skipping notification"
  echo "export AI_PR_SLACK_TEMPLATE=''"
fi
