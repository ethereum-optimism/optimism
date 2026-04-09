#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
CHECKER="$SCRIPT_DIR/check-component-release.py"

# Default components to check when no explicit component list is provided.
DEFAULT_COMPONENTS=(
  op-node
  op-batcher
  kona-node
)

usage() {
  cat <<'EOF'
Usage:
  check-component-releases.sh [component ...] [checker-args...]
  check-component-releases.sh [component ...] -- [checker-args...]

Examples:
  check-component-releases.sh
  check-component-releases.sh op-node kona-node
  check-component-releases.sh -vvv
  check-component-releases.sh --ref origin/develop --fetch
  check-component-releases.sh op-batcher --include-merges
  check-component-releases.sh op-batcher -- --include-merges

Notes:
  - If no components are provided, the script uses its DEFAULT_COMPONENTS list.
  - Common checker flags like -v/-vvv, --ref, --fetch, --include-merges, and --json
    may be passed directly.
  - Anything after '--' is forwarded to check-component-release.py unchanged.
EOF
}

if [[ ! -x "$CHECKER" ]]; then
  echo "error: checker script is not executable: $CHECKER" >&2
  exit 1
fi

components=()
checker_args=()

while [[ $# -gt 0 ]]; do
  case "$1" in
    --help|-h)
      usage
      exit 0
      ;;
    --)
      shift
      checker_args+=("$@")
      break
      ;;
    --ref)
      if [[ $# -lt 2 ]]; then
        echo "error: --ref requires a value" >&2
        exit 1
      fi
      checker_args+=("$1" "$2")
      shift 2
      ;;
    --ref=*)
      checker_args+=("$1")
      shift
      ;;
    --fetch|--include-merges|--json|--verbose|-v|-vv|-vvv|-vvvv|-vvvvv)
      checker_args+=("$1")
      shift
      ;;
    -*)
      checker_args+=("$1")
      shift
      ;;
    *)
      components+=("$1")
      shift
      ;;
  esac
done

if [[ ${#components[@]} -eq 0 ]]; then
  components=("${DEFAULT_COMPONENTS[@]}")
fi

failures=0

for component in "${components[@]}"; do
  echo "============================================================"
  echo "Component: $component"
  echo "============================================================"

  if [[ ${#checker_args[@]} -gt 0 ]]; then
    if ! python3 "$CHECKER" "$component" "${checker_args[@]}"; then
      failures=$((failures + 1))
    fi
  else
    if ! python3 "$CHECKER" "$component"; then
      failures=$((failures + 1))
    fi
  fi

  echo
  echo

done

if [[ $failures -gt 0 ]]; then
  echo "completed with $failures failure(s)" >&2
  exit 1
fi
