#!/usr/bin/env bash
# Generic parameter collector for CircleCI dynamic configuration.
#
# Called once per type with a mode argument:
#   compute-changes.sh str        — emit all c-* env vars as JSON strings
#   compute-changes.sh bool       — emit all c-* env vars as JSON booleans (normalizes 0/1)
#   compute-changes.sh conditions — compute pipeline conditions from BRANCH and TRIGGER_SOURCE
#   compute-changes.sh detect     — treat c-* env var values as ERE patterns, match against git diff
#   compute-changes.sh workflows  — evaluate c-run_* env vars as jq expressions against collected params
#
# Each invocation appends to /tmp/pipeline-parameters.json.
# Env vars whose name starts with c- are processed; all others are ignored.
set -euo pipefail

MODE="${1:?Usage: compute-changes.sh <str|bool|conditions|detect|workflows>}"
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

  conditions)
    branch=$(printenv BRANCH || echo "")
    trigger=$(printenv TRIGGER_SOURCE || echo "")

    add_condition() {
      local key="${1}" val="${2}"
      json=$(echo "${json}" | jq --argjson v "${val}" '. + {"'"${key}"'": $v}')
      echo "  [conditions] ${key} = ${val}"
    }

    add_condition "c-is_merge_queue" "$([[ "${branch}" =~ ^gh-readonly-queue/ ]] && echo true || echo false)"
    add_condition "c-is_develop" "$([[ "${branch}" == "develop" ]] && echo true || echo false)"
    add_condition "c-is_webhook" "$([[ "${trigger}" == "webhook" ]] && echo true || echo false)"
    add_condition "c-is_api_trigger" "$([[ "${trigger}" == "api" ]] && echo true || echo false)"
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

  workflows)
    context=$(echo "${json}" | jq \
      --arg tag "$(printenv TAG || echo "")" \
      --arg schedule "$(printenv SCHEDULE_NAME || echo "")" \
      --arg trigger "$(printenv TRIGGER_SOURCE || echo "")" \
      'with_entries(.key |= (ltrimstr("c-") | gsub("-"; "_")))
       + {tag: $tag, schedule_name: $schedule, trigger_source: $trigger}')

    while IFS='=' read -r key expr; do
      [[ "${key}" == c-run_* ]] || continue
      result=$(echo "${context}" | jq "${expr}")
      json=$(echo "${json}" | jq --argjson v "${result}" '. + {"'"${key}"'": $v}')
      echo "  [workflow] ${key} = ${result}"
    done < <(env | sort)
    ;;

  *)
    echo "ERROR: Unknown mode '${MODE}'. Use: str, bool, conditions, detect, or workflows." >&2
    exit 1
    ;;
esac

echo "${json}" > "${OUTPUT}"
echo "=== Parameters so far ==="
cat "${OUTPUT}"
echo "========================="
