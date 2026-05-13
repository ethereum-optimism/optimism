#!/usr/bin/env bash
# Collects pipeline parameters from the environment and writes them to a JSON file.
#
# Called once per type with a mode argument:
#   collect-params.sh str    — emit all c-* env vars as JSON strings
#   collect-params.sh bool   — emit all c-* env vars as JSON booleans (normalizes 0/1)
#   collect-params.sh detect — treat c-* env var values as ERE patterns, match against git diff
#
# Each invocation appends to /tmp/pipeline-parameters.json.
# Env vars whose name starts with c- are processed; all others are ignored.
set -euo pipefail

MODE="${1:?Usage: collect-params.sh <str|bool|detect>}"
OUTPUT="/tmp/pipeline-parameters.json"

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

  detect)
    CHANGED=$(git diff --name-only "origin/${BASE_REVISION}...HEAD" 2>/dev/null \
      || git diff --name-only HEAD~1 HEAD || true)
    echo "=== Changed files ==="
    echo "${CHANGED:-<none>}"
    echo "====================="

    while IFS='=' read -r key pattern; do
      [[ "${key}" == c-* ]] || continue
      if [ -n "${CHANGED}" ] && echo "${CHANGED}" | grep -qE "${pattern}"; then
        result=true
      else
        result=false
      fi
      json=$(echo "${json}" | jq --argjson v "${result}" '. + {"'"${key}"'": $v}')
      echo "  [detect] ${key} = ${result}  (pattern: ${pattern})"
    done < <(env | sort)
    ;;

  *)
    echo "ERROR: Unknown mode '${MODE}'. Use str, bool, or detect." >&2
    exit 1
    ;;
esac

echo "${json}" > "${OUTPUT}"
echo "=== Parameters so far ==="
cat "${OUTPUT}"
echo "========================="
