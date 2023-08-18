#!/bin/bash

rm -rf artifacts forge-artifacts

# See slither.config.json for slither settings
if [[ -z "$TRIAGE_MODE" ]]; then
  echo "Building contracts"
  forge build --build-info --force
  echo "Running slither"
  slither --ignore-compile .
else
  echo "Running slither in triage mode"
  # Slither's triage mode will run an 'interview' in the terminal, allowing you to review each of
  # its findings, and specify which should be ignored in future runs of slither. This will update
  # (or create) the slither.db.json file. This DB is a cleaner alternative to adding slither-disable
  # comments throughout the codebase.
  # Triage mode should only be run manually, and can be used to update the db when new findings are
  # causing a CI failure.
  slither . --triage-mode

  # For whatever reason the slither db contains a filename_absolute property which includes the full
  # local path to source code on the machine where it was generated. This property does not
  # seem to be required for slither to run, so we remove it.
  DB=slither.db.json
  TEMP_DB=temp-slither.db.json
  mv $DB $TEMP_DB
  jq 'walk(if type == "object" then del(.filename_absolute) else . end)' $TEMP_DB > $DB
  rm -f $TEMP_DB
fi
