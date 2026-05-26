#!/usr/bin/env bash
# Sourceable helper functions for the workflow decision tree in config.yml.
# Provides JSON plumbing so config.yml only contains the routing policy.
#
# Usage (from config.yml inline command):
#   source .circleci/scripts/workflow-helpers.sh
#   init_json
#   ...decision tree using run/param/is_true/run_rust_and_docs...
#   finalize

OUTPUT="/tmp/pipeline-parameters.json"
_json=""

init_json() {
  _json=$(cat "${OUTPUT}")
}

run() {
  for wf in "$@"; do
    _json=$(echo "${_json}" | jq '. + {"c-run_'"${wf}"'": true}')
    echo "  [enable] c-run_${wf}"
  done
}

param() {
  echo "${_json}" | jq -r ".\"c-${1}\""
}

is_true() {
  [[ "$(param "${1}")" == "true" ]]
}

finalize() {
  local keep_params="${1:?finalize requires a comma-separated list of params to keep}"
  local jq_filter
  jq_filter=$(echo "${keep_params}" | tr ',' '\n' | sed 's/.*/"&"/' | paste -sd',' -)

  _json=$(echo "${_json}" | jq "with_entries(select(
    .key | startswith(\"c-run_\") or IN(${jq_filter})
  ))")

  echo "${_json}" > "${OUTPUT}"
  echo "=== Enabled workflows ==="
  echo "${_json}" | jq -r 'to_entries[] | select(.key | startswith("c-run_")) | select(.value == true) | "  \(.key)"'
  echo "========================="
}
