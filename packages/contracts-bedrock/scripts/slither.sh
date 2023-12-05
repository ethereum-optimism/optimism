#!/usr/bin/env bash

set -e

SLITHER_REPORT="slither-report.json"
SLITHER_REPORT_BACKUP="slither-report.json.temp"

# Get the absolute path of the parent directory of this script
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && cd .. && pwd )"
echo "Running slither in $DIR"
cd $DIR

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

# Slither's triage mode will run an 'interview' in the terminal, allowing you to review each of
# its findings, and specify which should be ignored in future runs of slither. This will update
# (or create) the slither.db.json file. This DB is a cleaner alternative to adding slither-disable
# comments throughout the codebase.
# Triage mode should only be run manually, and can be used to update the db when new findings are
# causing a CI failure.
# See slither.config.json for slither settings
if [[ -z "$TRIAGE_MODE" ]]; then
  echo "Running slither in normal mode"
  # Run slither and store the output in a variable to be used later
  SLITHER_OUTPUT=$(slither . 2>&1 || true)

  # If slither failed to generate a report, exit with an error.
  if [ ! -f "$SLITHER_REPORT" ]; then
    echo "Slither output:\n$SLITHER_OUTPUT"
    echo "Slither failed to generate a report."
    if [ -e "$SLITHER_REPORT_BACKUP" ]; then
        mv $SLITHER_REPORT_BACKUP $SLITHER_REPORT
        echo "Restored previous slither report from $SLITHER_REPORT_BACKUP"
    fi
    echo "Exiting with error."
    exit 1
  fi

  echo "Slither ran successfully, generating minimzed report..."
  json=$(cat $SLITHER_REPORT)
  updated_json=$(cat $SLITHER_REPORT | jq -r '[.results.detectors[] | .description as $description | .check as $check | .impact as $impact | .confidence as $confidence | (.elements[] | .type as $type | .name as $name | (.source_mapping | { "impact": $impact, "confidence": $confidence, "check": $check, "description": $description, "type": $type, "name": $name, start, length, filename_relative } ))]')
  echo "$updated_json" > $SLITHER_REPORT
  echo "Slither report stored at $DIR/$SLITHER_REPORT"
else
  echo "Running slither in triage mode"
  slither . --triage-mode

  # The slither json report contains a `filename_absolute` property which includes the full
  # local path to source code on the machine where it was generated. This property breaks
  # cross-platform report comparisons, so it's removed here.
  mv $SLITHER_REPORT temp-slither-report.json
  jq 'walk(if type == "object" then del(.filename_absolute) else . end)' temp-slither-report.json > $SLITHER_REPORT
  rm -f temp-slither-report.json
fi

# Delete the backup of the previous slither report if it exists
if [ -e "$SLITHER_REPORT_BACKUP" ]; then
    rm $SLITHER_REPORT_BACKUP
    echo "Deleted backup of previous slither report at $SLITHER_REPORT_BACKUP"
fi
