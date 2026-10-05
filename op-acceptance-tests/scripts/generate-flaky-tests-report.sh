#!/bin/bash

set -euo pipefail

# Default values
BRANCH="develop"
ORG_NAME="ethereum-optimism"
REPO_NAME="optimism"
CIRCLE_API_TOKEN=""
OUTPUT_DIR="./reports"

# Parse command line arguments
while [[ $# -gt 0 ]]; do
  case $1 in
    --branch)
      if [[ $# -lt 2 ]]; then
        echo "Error: Missing value for --branch"
        exit 1
      fi
      BRANCH="$2"
      shift 2
      ;;
    --org)
      if [[ $# -lt 2 ]]; then
        echo "Error: Missing value for --org"
        exit 1
      fi
      ORG_NAME="$2"
      shift 2
      ;;
    --repo)
      if [[ $# -lt 2 ]]; then
        echo "Error: Missing value for --repo"
        exit 1
      fi
      REPO_NAME="$2"
      shift 2
      ;;
    --token)
      if [[ $# -lt 2 ]]; then
        echo "Error: Missing value for --token"
        exit 1
      fi
      CIRCLE_API_TOKEN="$2"
      shift 2
      ;;
    --output-dir)
      if [[ $# -lt 2 ]]; then
        echo "Error: Missing value for --output-dir"
        exit 1
      fi
      OUTPUT_DIR="$2"
      shift 2
      ;;
    *)
      echo "Unknown option: $1"
      exit 1
      ;;
  esac
done

# Public projects can use the API anonymously. Keep the original token option
# for authenticated Circle runs and private projects.
if [ -z "$BRANCH" ] || [ -z "$ORG_NAME" ] || [ -z "$REPO_NAME" ]; then
  echo "Error: Missing required parameters"
  echo "Usage: $0 --branch <branch> --org <org> --repo <repo> [--token <token>] [--output-dir <dir>]"
  exit 1
fi

# Create output directory
mkdir -p "$OUTPUT_DIR"

# Fetch flaky tests data
# See: https://circleci.com/docs/api/v2/index.html#tag/Insights/operation/getFlakyTests
echo "Fetching flaky tests data for branch: $BRANCH"
CURL_OPTIONS=(--fail-with-body --show-error --silent --connect-timeout 20 --max-time 90)
if [[ -n "$CIRCLE_API_TOKEN" ]]; then CURL_OPTIONS+=(-H "Circle-Token: $CIRCLE_API_TOKEN"); fi
ORIGINAL_JSON="$OUTPUT_DIR/flaky_tests.original.json"
# Retain each original failure instead of letting curl overwrite earlier bodies
# while retrying. Permanent HTTP authorization/path failures fail immediately.
for attempt in 1 2 3 4 5 6; do
  attempt_base="$OUTPUT_DIR/api-attempt-$attempt"
  if curl "${CURL_OPTIONS[@]}" --get --data-urlencode "branch=$BRANCH" \
    --output "$attempt_base.json" --write-out '%{http_code}\n' \
    "https://circleci.com/api/v2/insights/gh/$ORG_NAME/$REPO_NAME/flaky-tests" \
    > "$attempt_base.http-status.txt" 2> "$attempt_base.stderr.log"; then
    curl_status=0
  else
    curl_status=$?
  fi
  printf '%s\n' "$curl_status" > "$attempt_base.exit-code.txt"
  [[ ! -f "$attempt_base.json" ]] || cp "$attempt_base.json" "$ORIGINAL_JSON"
  cp "$attempt_base.http-status.txt" "$OUTPUT_DIR/http-status.txt"
  cat "$attempt_base.stderr.log" >&2
  http_status=$(cat "$attempt_base.http-status.txt")
  if [[ "$curl_status" == 0 && "$http_status" == 200 ]]; then break; fi
  case "$http_status" in
    400|401|403|404) echo "Error: CircleCI API HTTP $http_status" >&2; exit "$((curl_status == 0 ? 1 : curl_status))" ;;
  esac
  if [[ "$attempt" == 6 ]]; then
    echo "Error: CircleCI API fetch failed after $attempt attempts" >&2
    exit 1
  fi
  sleep "$((2 ** attempt))"
done
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
python3 "$REPO_ROOT/ops/ci/flaky-report.py" validate-api "$ORIGINAL_JSON"
API_RESPONSE=$(cat "$ORIGINAL_JSON")

# Check if we got a valid response
if [ -z "$API_RESPONSE" ]; then
  echo "Error: Empty response from CircleCI API"
  exit 1
fi

# Filter to only include acceptance tests
echo "Filtering for acceptance tests only..."
API_RESPONSE=$(echo "$API_RESPONSE" | jq '.flaky_tests = (.flaky_tests | map(select(.classname | startswith("github.com/ethereum-optimism/optimism/op-acceptance-tests/tests"))))')

# Preserve the existing acceptance-only JSON interface as well as the complete
# unfiltered original above.
echo "$API_RESPONSE" > "$OUTPUT_DIR/flaky_tests.json"
echo "Acceptance-test response saved to $OUTPUT_DIR/flaky_tests.json"

# Use acceptance-tests fresponse directly
echo "Using acceptance-tests filtered response without additional branch verification..."
FILTERED_JSON="$OUTPUT_DIR/flaky_tests.filtered.json"
cp "$OUTPUT_DIR/flaky_tests.json" "$FILTERED_JSON"
API_RESPONSE=$(cat "$FILTERED_JSON")
echo "Filtered response saved to $FILTERED_JSON"

# Check if the response contains flaky_tests
if ! echo "$API_RESPONSE" | jq -e '.flaky_tests' > /dev/null 2>&1; then
  echo "Error: Invalid JSON response or missing 'flaky_tests' field"
  echo "API Response:"
  echo "$API_RESPONSE"
  exit 1
fi

# Print the number of flaky tests found
NUM_TESTS=$(jq '.flaky_tests | length' "$FILTERED_JSON")
echo "Found $NUM_TESTS flaky tests"

# Generate CSV report
echo "Generating CSV report..."
echo '"times_flaked","test_name","classname","job_name","workflow_name","job_number","pipeline_number","job_url","first_flaked_at","last_flaked_at"' > "$OUTPUT_DIR/flaky_tests.csv"
jq -r '.flaky_tests | sort_by(.times_flaked) | reverse | .[] | [
  .times_flaked,
  (.test_name | @json),
  (.classname | @json),
  (.job_name | @json),
  (.workflow_name | @json),
  .job_number,
  .pipeline_number,
  ("https://app.circleci.com/pipelines/github/" + "'"$ORG_NAME"'" + "/" + "'"$REPO_NAME"'" + "/" + (.pipeline_number | tostring) + "/workflows/" + .workflow_id + "/jobs/" + (.job_number | tostring) | @json),
  (.workflow_created_at | @json),
  (.workflow_created_at | @json)
] | @csv' "$FILTERED_JSON" >> "$OUTPUT_DIR/flaky_tests.csv"

# Generate HTML report
echo "Generating HTML report..."
BRANCH_HTML=$(jq -nr --arg branch "$BRANCH" '$branch | @html')
cat > "$OUTPUT_DIR/flaky_tests.html" << EOF
<!DOCTYPE html>
<html>
<head>
    <title>Flaky Tests Report - Branch: $BRANCH_HTML</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        table { border-collapse: collapse; width: 100%; }
        th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
        th { background-color: #f2f2f2; }
        tr:nth-child(even) { background-color: #f9f9f9; }
        .branch-info { margin-bottom: 20px; }
    </style>
</head>
<body>
    <h1>Flaky Tests Report</h1>
    <p>
      <b>Note:</b> These tests are <i>potentially</i> flaky. They may fail for reasons other than the test itself, such as network issues, devnet issues,
      interference from other tests, etc. Be mindful of this when interpreting the results and investigating the failures.
    </p>
    <div class="branch-info">
        <h3>Branch: $BRANCH_HTML</h3>
        <h3>Total flaky tests: $NUM_TESTS</h3>
        <p>CircleCI's flaky-test API is project-wide and branch agnostic. The branch above is the requesting CI branch.</p>
    </div>

    <table>
        <tr>
            <th># Flakes (Last 14 days)</th>
            <th>Test Name</th>
            <th>Path</th>
            <th>Job Name</th>
            <th>Workflow Name</th>
            <th>Job Number</th>
            <th>Pipeline Number</th>
            <th>Job URL</th>
            <th>First Flaked At</th>
            <th>Last Flaked At</th>
        </tr>
        $(jq -r '.flaky_tests | sort_by(.times_flaked) | reverse | .[] | "<tr><td>\(.times_flaked)</td><td>\(.test_name | @html)</td><td>\(.classname | @html)</td><td>\(.job_name | @html)</td><td>\(.workflow_name | @html)</td><td>\(.job_number)</td><td>\(.pipeline_number)</td><td><a href=\"https://app.circleci.com/pipelines/github/'"$ORG_NAME"'/'"$REPO_NAME"'/\(.pipeline_number)/workflows/\(.workflow_id)/jobs/\(.job_number)\" target=\"_blank\">View Job</a></td><td>\(.workflow_created_at | @html)</td><td>\(.workflow_created_at | @html)</td></tr>"' "$FILTERED_JSON")
    </table>
</body>
</html>
EOF

# Check if HTML file was generated and has content
if [ ! -s "$OUTPUT_DIR/flaky_tests.html" ]; then
  echo "Error: HTML file is empty or was not generated"
  exit 1
fi

echo "HTML report generated"

# Output simplified text report (top10 only)
echo "Top 10 Flaky Tests for branch $BRANCH"
echo "=========================================="
jq -r '.flaky_tests | sort_by(.times_flaked) | reverse | .[0:10] | .[] | "\(.times_flaked)x: \(.test_name)"' \
  "$FILTERED_JSON"
