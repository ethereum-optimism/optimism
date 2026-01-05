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

# Parse arguments
i=1
while [ "$i" -le "$#" ]; do
  eval arg="\${$i}"

  case "$arg" in
    --datadir=*)
      DATADIR="${arg#*=}"
      ;;

    --datadir)
      eval next="\${$((i+1))}"
      require_value "$arg" "$next"
      DATADIR="$next"
      i=$((i+1))
      ;;

    --proofs-history.storage-path=*)
      PROOFS_PATH="${arg#*=}"
      ;;

    --proofs-history.storage-path)
      eval next="\${$((i+1))}"
      require_value "$arg" "$next"
      PROOFS_PATH="$next"
      i=$((i+1))
      ;;

    --chain=*)
      CHAIN="${arg#*=}"
      ;;

    --chain)
      eval next="\${$((i+1))}"
      require_value "$arg" "$next"
      CHAIN="$next"
      i=$((i+1))
      ;;

    *)
      # ignore unknown args—OR log them
      ;;
  esac

  i=$((i+1))
done

# Log extracted values
echo "extracted --datadir: ${DATADIR:-<not-set>}"
echo "extracted --proofs-history.storage-path: ${PROOFS_PATH:-<not-set>}"
echo "extracted --chain: ${CHAIN:-<not-set>}"

echo "Initializing op-reth"
op-reth init --datadir="$DATADIR" --chain="$CHAIN"
echo "Initializing op-reth proofs"
op-reth initialize-op-proofs --datadir="$DATADIR" --chain="$CHAIN" --proofs-history.storage-path="$PROOFS_PATH"
echo "Starting op-reth with args: $*"
op-reth "$@"
