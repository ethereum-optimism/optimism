#!/usr/bin/env bash
# ─────────────────────────────────────────────────────────
# interop-send-loop.sh — continuously sends L2-to-L2
# cross-domain messages between supersim chains so you
# can watch them flow through ponder → autorelayer.
#
# Alternates direction each iteration:
#   L2A (901) → L2B (902) → L2A (901) → …
#
# Usage:
#   ./scripts/interop-send-loop.sh [--network supersim] [--interval 6] [--count 0]
#
#   --network   network to target: supersim or v0 (default: supersim)
#   --interval  seconds between sends (default: 6)
#   --count     total messages to send, 0 = infinite (default: 0)
#
# Requires: cast (foundry), curl, python3
# ─────────────────────────────────────────────────────────
set -euo pipefail

# ── Config ────────────────────────────────────────────────
NETWORK="supersim"
INTERVAL=6
MAX_COUNT=0
PONDER_API="${PONDER_API:-http://127.0.0.1:42069}"



# Anvil account 0 — pre-funded on supersim
PRIVATE_KEY="${PRIVATE_KEY:-0x9462ff23e97802bff834fe50c50693e3f616f42a5a22985972892440652b9e65}"

# L2ToL2CrossDomainMessenger predeploy
L2_TO_L2_CDM="0x4200000000000000000000000000000000000023"

# sendMessage(uint256 _destination, address _target, bytes _message)
SEND_SIG="sendMessage(uint256,address,bytes)"

# We'll send to a dummy target with an incrementing counter as the message
DUMMY_TARGET="0x000000000000000000000000000000000000dEaD"


# ── Parse args ────────────────────────────────────────────
while [[ $# -gt 0 ]]; do
  case "$1" in
    --network)  NETWORK="$2"; shift 2;;
    --interval) INTERVAL="$2"; shift 2;;
    --count)    MAX_COUNT="$2"; shift 2;;
    *)          echo "Unknown arg: $1"; exit 1;;
  esac
done

if [[ "$NETWORK" == "supersim" ]]; then
  L2A_RPC="${L2A_RPC:-http://127.0.0.1:9545}"
  L2B_RPC="${L2B_RPC:-http://127.0.0.1:9546}"
  L2A_CHAIN_ID=901
  L2B_CHAIN_ID=902
elif [[ "$NETWORK" == "v0" ]]; then
  L2A_RPC="${L2A_RPC:-https://interop-v0-0.optimism.io}"
  L2B_RPC="${L2B_RPC:-https://interop-v0-1.optimism.io}"
  L2A_CHAIN_ID=420120046
  L2B_CHAIN_ID=420120047
else
  echo -e "Unknown network: $NETWORK"
  echo -e "Supported networks: supersim, v0"
  exit 1
fi


# ── Colors ────────────────────────────────────────────────
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
RED='\033[0;31m'
DIM='\033[2m'
BOLD='\033[1m'
RESET='\033[0m'

echo -e "${BOLD}${CYAN}Interop Cross-Chain Message Sender${RESET}"
echo -e "${DIM}Sending L2↔L2 messages every ${INTERVAL}s${RESET}"
echo -e "${DIM}L2A (${L2A_CHAIN_ID}) @ ${L2A_RPC}${RESET}"
echo -e "${DIM}L2B (${L2B_CHAIN_ID}) @ ${L2B_RPC}${RESET}"
echo

# ── Verify services are up ────────────────────────────────
for rpc in "$L2A_RPC" "$L2B_RPC"; do
  if ! curl -sf -X POST "$rpc" -H "Content-Type: application/json" \
    -d '{"jsonrpc":"2.0","method":"eth_chainId","params":[],"id":1}' >/dev/null 2>&1; then
    echo -e "${RED}ERROR: Cannot reach $rpc — is the network up?${RESET}"
    exit 1
  fi
done

if ! curl -sf "$PONDER_API/health" >/dev/null 2>&1; then
  echo -e "${YELLOW}WARN: ponder-interop not reachable at $PONDER_API — messages may not be indexed${RESET}"
fi

# ── Send loop ─────────────────────────────────────────────
SEQ=0
while true; do
  SEQ=$((SEQ + 1))

  # Alternate direction
  if (( SEQ % 2 == 1 )); then
    SRC_RPC="$L2A_RPC"
    SRC_NAME="L2A"
    SRC_CHAIN=$L2A_CHAIN_ID
    DST_CHAIN=$L2B_CHAIN_ID
    DST_NAME="L2B"
  else
    SRC_RPC="$L2B_RPC"
    SRC_NAME="L2B"
    SRC_CHAIN=$L2B_CHAIN_ID
    DST_CHAIN=$L2A_CHAIN_ID
    DST_NAME="L2A"
  fi

  # Encode a simple message: the sequence number as bytes
  MSG_HEX=$(printf "0x%064x" "$SEQ")

  echo -e "${BOLD}[#${SEQ}]${RESET} ${CYAN}${SRC_NAME}${RESET} (${SRC_CHAIN}) → ${CYAN}${DST_NAME}${RESET} (${DST_CHAIN})"

  # Send the cross-domain message via cast (JSON mode for clean parsing)
  CAST_OUT=$(cast send \
    --rpc-url "$SRC_RPC" \
    --private-key "$PRIVATE_KEY" \
    --json \
    "$L2_TO_L2_CDM" \
    "$SEND_SIG" \
    "$DST_CHAIN" \
    "$DUMMY_TARGET" \
    "$MSG_HEX" 2>/dev/null) || true

  TX_HASH=$(echo "$CAST_OUT" | python3 -c "import sys,json; print(json.load(sys.stdin).get('transactionHash',''))" 2>/dev/null) || true

  if [[ -n "$TX_HASH" ]]; then
    echo -e "  ${GREEN}tx: ${TX_HASH}${RESET}"
  else
    echo -e "  ${RED}Failed to send — check cast output${RESET}"
  fi

  # Quick peek at ponder stats
  MSG_COUNT=$(curl -sf "$PONDER_API/messages/count" 2>/dev/null || echo "")
  if [[ -n "$MSG_COUNT" ]]; then
    SENT=$(echo "$MSG_COUNT" | python3 -c "import sys,json; d=json.load(sys.stdin); print(sum(x['count'] for x in d.get('sent',[])))" 2>/dev/null || echo "?")
    RELAYED=$(echo "$MSG_COUNT" | python3 -c "import sys,json; d=json.load(sys.stdin); print(sum(x['count'] for x in d.get('relayed',[])))" 2>/dev/null || echo "?")
    PENDING=$(echo "$MSG_COUNT" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('pending','?'))" 2>/dev/null || echo "?")
    echo -e "  ${DIM}ponder: sent=${SENT} relayed=${RELAYED} pending=${PENDING}${RESET}"
  fi

  echo

  # Check if we've hit the max
  if [[ "$MAX_COUNT" -gt 0 && "$SEQ" -ge "$MAX_COUNT" ]]; then
    echo -e "${GREEN}Done! Sent ${SEQ} messages.${RESET}"
    exit 0
  fi

  sleep "$INTERVAL"
done
