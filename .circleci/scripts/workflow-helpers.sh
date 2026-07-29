#!/usr/bin/env bash
# Sourceable helper functions for the workflow routing policy in
# compute-workflow-conditions.sh. Provides JSON plumbing and routing.yml
# readers so the policy script only contains the routing logic and routing.yml
# only contains the data.
#
# Usage (from compute-workflow-conditions.sh):
#   source .circleci/scripts/workflow-helpers.sh
#   init_json
#   ...routing logic using run/run_group/param/is_true...
#   finalize

# OUTPUT is overridable so tests can point at an isolated file instead of the
# pipeline's real params file.
OUTPUT="${OUTPUT:-/tmp/pipeline-parameters.json}"
# routing.yml lives one directory up from this script (.circleci/routing.yml),
# resolved from the script location so it works regardless of the caller's cwd.
ROUTING="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/routing.yml"
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

# run_group <section> <key>: enable every workflow listed under
# routing.yml's <section>.<key>. A missing key enables nothing.
run_group() {
  local section="${1}" key="${2}"
  local wfs
  wfs=$(yq -r ".${section}.\"${key}\"[]?" "${ROUTING}")
  if [[ -n "${wfs}" ]]; then
    # shellcheck disable=SC2086  # intentional word-splitting of the name list
    run ${wfs}
  fi
}

param() {
  echo "${_json}" | jq -r ".\"c-${1}\""
}

is_true() {
  [[ "$(param "${1}")" == "true" ]]
}

# Strip intermediate params, keeping only c-run_* flags and the passthrough
# params declared in routing.yml, then write the final JSON.
finalize() {
  local jq_filter
  jq_filter=$(yq -r '.passthrough_params[]' "${ROUTING}" | sed 's/.*/"&"/' | paste -sd',' -)

  _json=$(echo "${_json}" | jq "with_entries(select(
    .key | startswith(\"c-run_\") or IN(${jq_filter})
  ))")

  echo "${_json}" > "${OUTPUT}"
  echo "=== Enabled workflows ==="
  echo "${_json}" | jq -r 'to_entries[] | select(.key | startswith("c-run_")) | select(.value == true) | "  \(.key)"'
  echo "========================="
}
