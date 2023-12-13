#!/usr/bin/env bash

set -e

SLITHER_REPORT="slither-report.json"
SLITHER_REPORT_BACKUP="slither-report.json.temp"
SLITHER_TRIAGE_REPORT="slither.db.json"

# Get the absolute path of the parent directory of this script
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && cd .. && pwd )"
echo "Running slither in $DIR"
cd "$DIR"

# Clean up any previous artifacts.
# We do not check if pnpm is installed since it is used across the monorepo
# and must be installed as a prerequisite.
pnpm clean

# Check if slither is installed
# If not, provide instructions to install with `pip3 install slither-analyzer` and exit
if ! command -v slither &> /dev/null
then
    echo "Slither could not be found. Please install slither by running:"
    echo "pip3 install slither-analyzer"
    exit
fi


# Check if jq is installed and exit otherwise
if ! command -v jq &> /dev/null
then
    echo "jq could not be found. Please install jq."
    echo "On Mac: brew install jq"
    echo "On Ubuntu: sudo apt-get install jq"
    echo "For other platforms: https://stedolan.github.io/jq/download/"
    exit
fi

# Print the slither version
echo "Slither version: $(slither --version)"

# Copy the slither report if it exists to a temp file
if [ -e "$SLITHER_REPORT" ]; then
    mv $SLITHER_REPORT $SLITHER_REPORT_BACKUP
    echo "Created backup of previous slither report at $SLITHER_REPORT_BACKUP"
fi

# Slither normal mode will run slither and put all findings in the slither-report.json file.
# See slither.config.json for slither settings and to disable specific detectors.
if [[ -z "$TRIAGE_MODE" ]]; then
  echo "Running slither in normal mode"
  SLITHER_OUTPUT=$(slither . 2>&1 || true)
fi

# Slither's triage mode will run an 'interview' in the terminal.
# This allows you to review each finding, and specify which to ignore in future runs.
# Findings to keep are output to the slither-report.json output file.
# Checking in a json file is cleaner than adding slither-disable comments throughout the codebase.
# See slither.config.json for slither settings and to disable specific detectors.
if [[ -n "$TRIAGE_MODE" ]]; then
  echo "Running slither in triage mode"
  SLITHER_OUTPUT=$(slither . --triage-mode --json $SLITHER_REPORT || true)

  # If the slither report was generated successfully, and the slither triage exists, clean up the triaged output.
  if [ -f "$SLITHER_REPORT" ] && [ -f  "$SLITHER_TRIAGE_REPORT" ]; then
    # The following jq command selects a subset of fields in each of the slither triage report description and element objects.
    # This significantly slims down the output json, on the order of 100 magnitudes smaller.
    updated_json=$(cat $SLITHER_TRIAGE_REPORT | jq -r '[.[] | .id as $id | .description as $description | .check as $check | .impact as $impact | .confidence as $confidence | (.elements[] | .type as $type | .name as $name | (.source_mapping | { "id": $id, "impact": $impact, "confidence": $confidence, "check": $check, "description": $description, "type": $type, "name": $name, start, length, filename_relative } ))]')
    echo "$updated_json" > $SLITHER_TRIAGE_REPORT
    echo "Slither traige report updated at $DIR/$SLITHER_TRIAGE_REPORT"
  fi
fi

# If slither failed to generate a report, exit with an error.
if [ ! -f "$SLITHER_REPORT" ]; then
  echo "Slither output:"
  echo "$SLITHER_OUTPUT"
  echo "Slither failed to generate a report."
  if [ -e "$SLITHER_REPORT_BACKUP" ]; then
      mv $SLITHER_REPORT_BACKUP $SLITHER_REPORT
      echo "Restored previous slither report from $SLITHER_REPORT_BACKUP"
  fi
  echo "Exiting with error."
  exit 1
fi

# If slither successfully generated a report, clean up the report.
# The following jq command selects a subset of fields in each of the slither triage report description and element objects.
# This significantly slims down the output json, on the order of 100 magnitudes smaller.
echo "Slither ran successfully, generating minimzed report..."
updated_json=$(cat $SLITHER_REPORT | jq -r '[.results.detectors[] | .id as $id | .description as $description | .check as $check | .impact as $impact | .confidence as $confidence | (.elements[] | .type as $type | .name as $name | (.source_mapping | { "id": $id, "impact": $impact, "confidence": $confidence, "check": $check, "description": $description, "type": $type, "name": $name, start, length, filename_relative } ))]')
echo "$updated_json" > $SLITHER_REPORT
echo "Slither report stored at $DIR/$SLITHER_REPORT"

# Remove any items in the slither report that are also in the slither triage report.
# This prevents the same finding from being reported twice.
# Iterate over the slither-report.json file and remove any items that are in the slither.db.json file
# by matching on the id field.
if [ -f "$SLITHER_TRIAGE_REPORT" ]; then
  echo "Removing triaged items from slither report..."
  jq -s '.[0] as $slither_report | .[1] as $slither_triage_report | $slither_report - ($slither_report - $slither_triage_report)' $SLITHER_REPORT $SLITHER_TRIAGE_REPORT > $SLITHER_REPORT.temp
  mv $SLITHER_REPORT.temp $SLITHER_REPORT
  echo "Slither report stored at $DIR/$SLITHER_REPORT"
fi

# Delete the backup of the previous slither report if it exists
if [ -e "$SLITHER_REPORT_BACKUP" ]; then
    rm $SLITHER_REPORT_BACKUP
    echo "Deleted backup of previous slither report at $SLITHER_REPORT_BACKUP"
fi
