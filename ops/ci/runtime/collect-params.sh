#!/usr/bin/env bash
# Collects pipeline parameters from the environment and writes them to a JSON file.
#
# Called once per type with a mode argument:
#   collect-params.sh str        — emit all c-* env vars as JSON strings
#   collect-params.sh bool       — emit all c-* env vars as JSON booleans (normalizes 0/1)
#   collect-params.sh detect     — match routing.yml change_patterns.any against changed files; c-<name> true iff ANY file matches
#   collect-params.sh detect_all — match routing.yml change_patterns.all against changed files; c-<name> true iff EVERY file matches (and there is at least one)
#
# str/bool read c-* env vars. detect/detect_all read
# their ERE patterns from routing.yml so the patterns are declarative data.
# OUTPUT selects the JSON parameters file. CHANGED_FILES_FILE may supply a
# newline-delimited path list; otherwise diff against CI_BASE_REVISION (develop).
set -euo pipefail

MODE="${1:?Usage: collect-params.sh <str|bool|detect|detect_all>}"
OUTPUT="${OUTPUT:-/tmp/pipeline-parameters.json}"
ROUTING="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/routing.yml"

[ -f "${OUTPUT}" ] || echo '{}' > "${OUTPUT}"

to_bool() {
  case "${1}" in 1|true|True|TRUE) echo "true" ;; *) echo "false" ;; esac
}

json=$(cat "${OUTPUT}")

case "${MODE}" in
  str)
    while IFS='=' read -r key value; do
      [[ "${key}" == c-* ]] || continue
      json=$(echo "${json}" | jq --arg v "${value}" '. + {"'"${key}"'": $v}')
      echo "  [str] ${key} = ${value}"
    done < <(env | sort)
    ;;

  bool)
    while IFS='=' read -r key value; do
      [[ "${key}" == c-* ]] || continue
      json=$(echo "${json}" | jq --argjson v "$(to_bool "${value}")" '. + {"'"${key}"'": $v}')
      echo "  [bool] ${key} = $(to_bool "${value}")"
    done < <(env | sort)
    ;;

  detect|detect_all)
    [[ "${MODE}" == "detect_all" ]] && section=".change_patterns.all" || section=".change_patterns.any"

    if [[ -n "${CHANGED_FILES_FILE:-}" ]]; then
      if [[ ! -f "${CHANGED_FILES_FILE}" || ! -r "${CHANGED_FILES_FILE}" ]]; then
        echo "ERROR: CHANGED_FILES_FILE must name a readable file: ${CHANGED_FILES_FILE}" >&2
        exit 1
      fi
      CHANGED=$(cat "${CHANGED_FILES_FILE}")
    else
      CHANGED=$(git diff --name-only "origin/${CI_BASE_REVISION:-develop}...HEAD" 2>/dev/null \
        || git diff --name-only HEAD~1 HEAD || true)
    fi
    echo "=== Changed files ==="
    echo "${CHANGED:-<none>}"
    echo "====================="

    while IFS= read -r name; do
      [[ -n "${name}" ]] || continue
      pattern=$(yq -r "${section}.\"${name}\"" "${ROUTING}")
      if [ -z "${CHANGED}" ]; then
        result=false
      elif [[ "${MODE}" == "detect_all" ]]; then
        # True iff every changed file matches the pattern (i.e., no file fails to match).
        if grep -qvE "${pattern}" <<<"${CHANGED}"; then
          result=false
        else
          result=true
        fi
      else
        # detect: true iff at least one changed file matches the pattern.
        # A pipe plus grep -q can SIGPIPE the writer after an early match.
        # With pipefail that incorrectly reports a non-match on large diffs.
        if grep -qE "${pattern}" <<<"${CHANGED}"; then
          result=true
        else
          result=false
        fi
      fi
      json=$(echo "${json}" | jq --argjson v "${result}" '. + {"c-'"${name}"'": $v}')
      echo "  [${MODE}] c-${name} = ${result}  (pattern: ${pattern})"
    done < <(yq -r "${section} | keys | .[]" "${ROUTING}")
    ;;

  *)
    echo "ERROR: Unknown mode '${MODE}'. Use str, bool, detect, or detect_all." >&2
    exit 1
    ;;
esac

echo "${json}" > "${OUTPUT}"
echo "=== Parameters so far ==="
cat "${OUTPUT}"
echo "========================="
