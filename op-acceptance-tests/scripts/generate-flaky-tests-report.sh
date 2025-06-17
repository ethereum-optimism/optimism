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

# Validate required parameters
if [ -z "$BRANCH" ] || [ -z "$ORG_NAME" ] || [ -z "$REPO_NAME" ] || [ -z "$CIRCLE_API_TOKEN" ]; then
  echo "Error: Missing required parameters"
  echo "Usage: $0 --branch <branch> --org <org> --repo <repo> --token <token> [--output-dir <dir>]"
  echo "Debug information:"
  echo "BRANCH: $BRANCH"
  echo "ORG_NAME: $ORG_NAME"
  echo "REPO_NAME: $REPO_NAME"
  echo "CIRCLE_API_TOKEN length: ${#CIRCLE_API_TOKEN}"
  echo "CIRCLE_API_TOKEN first 10 chars: ${CIRCLE_API_TOKEN:0:10}..."
  exit 1
fi

# Create output directory
mkdir -p "$OUTPUT_DIR"

# Fetch flaky tests data
# See: https://circleci.com/docs/api/v2/index.html#tag/Insights/operation/getFlakyTests
echo "Fetching flaky tests data for branch: $BRANCH"
API_RESPONSE=$(curl -s -H "Circle-Token: $CIRCLE_API_TOKEN" \
  "https://circleci.com/api/v2/insights/gh/$ORG_NAME/$REPO_NAME/flaky-tests?branch=$BRANCH")

# Check if we got a valid response
if [ -z "$API_RESPONSE" ]; then
  echo "Error: Empty response from CircleCI API"
  exit 1
fi

# Filter to only include acceptance tests
echo "Filtering for acceptance tests only..."
API_RESPONSE=$(echo "$API_RESPONSE" | jq '.flaky_tests = (.flaky_tests | map(select(.classname | startswith("github.com/ethereum-optimism/optimism/op-acceptance-tests/tests"))))')


# Save the raw response for debugging
echo "$API_RESPONSE" > "$OUTPUT_DIR/flaky_tests.json"
echo "Raw API response saved to $OUTPUT_DIR/flaky_tests.json"

# Check if the response contains flaky_tests
if ! echo "$API_RESPONSE" | jq -e '.flaky_tests' > /dev/null 2>&1; then
  echo "Error: Invalid JSON response or missing 'flaky_tests' field"
  echo "API Response:"
  echo "$API_RESPONSE"
  exit 1
fi

# Check if we have any flaky tests
if ! echo "$API_RESPONSE" | jq -e '.flaky_tests | length > 0' > /dev/null 2>&1; then
  echo "No flaky tests found for branch $BRANCH"
  echo "API Response:"
  echo "$API_RESPONSE"
  exit 0
fi

# Print the number of flaky tests found
NUM_TESTS=$(echo "$API_RESPONSE" | jq '.flaky_tests | length')
echo "Found $NUM_TESTS flaky tests"

# Generate CSV report
echo "Generating CSV report..."
jq -r '.flaky_tests[] | [
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
] | @csv' "$OUTPUT_DIR/flaky_tests.json" > "$OUTPUT_DIR/flaky_tests.csv"

# Check if CSV file was generated and has content
if [ ! -s "$OUTPUT_DIR/flaky_tests.csv" ]; then
  echo "Error: CSV file is empty or was not generated"
  echo "Contents of flaky_tests.json:"
  cat "$OUTPUT_DIR/flaky_tests.json"
  exit 1
fi

# Generate HTML report
echo "Generating HTML report..."
cat > "$OUTPUT_DIR/flaky_tests.html" << EOF
<!DOCTYPE html>
<html>
<head>
    <title>Flaky Tests Report - Branch: $BRANCH</title>
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
    <div class="branch-info">
        <h2>Branch: $BRANCH</h2>
        <p>Total flaky tests: $NUM_TESTS</p>
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
            <th>Build URL</th>
            <th>First Flaked At</th>
            <th>Last Flaked At</th>
        </tr>
        $(jq -r '.flaky_tests[] | "<tr><td>\(.times_flaked)</td><td>\(.test_name)</td><td>\(.classname)</td><td>\(.job_name)</td><td>\(.workflow_name)</td><td>\(.job_number)</td><td>\(.pipeline_number)</td><td><a href=\"https://app.circleci.com/pipelines/github/'"$ORG_NAME"'/'"$REPO_NAME"'/\(.pipeline_number)/workflows/\(.workflow_id)/jobs/\(.job_number)\" target=\"_blank\">View Build</a></td><td>\(.workflow_created_at)</td><td>\(.workflow_created_at)</td></tr>"' "$OUTPUT_DIR/flaky_tests.json")
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
  "$OUTPUT_DIR/flaky_tests.json"