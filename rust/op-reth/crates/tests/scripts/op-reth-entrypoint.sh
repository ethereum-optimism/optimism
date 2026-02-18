#!/bin/sh
set -e

# Variables to extract
DATADIR=""
PROOFS_PATH=""
CHAIN=""

# Helper: require a value after flag
require_value() {
  if [ -z "$2" ] || printf "%s" "$2" | grep -q "^--"; then
    echo "ERROR: Missing value for $1" >&2
    exit 1
  fi
}

# Parse arguments using a prev_flag pattern to avoid eval
prev_flag=""
for arg in "$@"; do
  if [ -n "$prev_flag" ]; then
    require_value "$prev_flag" "$arg"
    case "$prev_flag" in
      --datadir) DATADIR="$arg" ;;
      --proofs-history.storage-path) PROOFS_PATH="$arg" ;;
      --chain) CHAIN="$arg" ;;
    esac
    prev_flag=""
    continue
  fi

  case "$arg" in
    --datadir=*)
      DATADIR="${arg#*=}"
      ;;
    --datadir)
      prev_flag="$arg"
      ;;
    --proofs-history.storage-path=*)
      PROOFS_PATH="${arg#*=}"
      ;;
    --proofs-history.storage-path)
      prev_flag="$arg"
      ;;
    --chain=*)
      CHAIN="${arg#*=}"
      ;;
    --chain)
      prev_flag="$arg"
      ;;
  esac
done

# Check if a flag was left without a value at the end
if [ -n "$prev_flag" ]; then
  echo "ERROR: Missing value for $prev_flag" >&2
  exit 1
fi

# Log extracted values
echo "extracted --datadir: ${DATADIR:-<not-set>}"
echo "extracted --proofs-history.storage-path: ${PROOFS_PATH:-<not-set>}"
echo "extracted --chain: ${CHAIN:-<not-set>}"

echo "Initializing op-reth"
op-reth init --datadir="$DATADIR" --chain="$CHAIN"
echo "Initializing op-reth proofs"
op-reth proofs init --datadir="$DATADIR" --chain="$CHAIN" --proofs-history.storage-path="$PROOFS_PATH"
echo "Starting op-reth with args: $*"
op-reth "$@"
