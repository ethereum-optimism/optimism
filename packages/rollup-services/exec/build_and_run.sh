#!/bin/bash
# build_and_run.sh -- accepts a command to run after postgres connection succeeds
# conditionally builds based on REBUILD env var and then calls wait_for_postgres_and_geth.sh

# Exits if any command fails
set -e

cmd=$@

ROOT_DIR=../..

if [ -n "$REBUILD" ]; then
  echo -e "\n\nREBUILD env var set, rebuilding...\n\n"

  if [ -n "$FETCH_DEPS" ]; then
    echo -e "\nFetching dependencies (this will take forever the first time time)..."
    yarn --verbose
    yarn --cwd $ROOT_DIR --verbose
  fi

  yarn clean

  yarn --cwd $ROOT_DIR build
  yarn build
else
  echo -e "\nStarting the microservices"
  exec $(dirname $0)/wait_for_postgres_and_geth.sh "$cmd"
fi

