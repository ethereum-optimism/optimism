#!/usr/bin/env bash
# ─────────────────────────────────────────────────────────
# interop-monitor.sh — single-pane dashboard for the
# interop stacks (supersim, v0, etc.) ↔ ponder-interop ↔ autorelayer-interop
#
# Usage:  ./scripts/interop-monitor.sh [--network supersim] [--interval 3]
# ─────────────────────────────────────────────────────────
set -euo pipefail

# ── Config ────────────────────────────────────────────────
NETWORK="supersim"
INTERVAL=3
PONDER_API="${PONDER_API:-http://127.0.0.1:42069}"

# ── Parse args ────────────────────────────────────────────
while [[ $# -gt 0 ]]; do
  case "$1" in
    --network)  NETWORK="$2"; shift 2;;
    --interval) INTERVAL="$2"; shift 2;;
    *)          echo "Unknown arg: $1"; exit 1;;
  esac
done

if [[ "$NETWORK" == "supersim" ]]; then
  L2A_RPC="${L2A_RPC:-http://127.0.0.1:9545}"
  L2B_RPC="${L2B_RPC:-http://127.0.0.1:9546}"
  RPC_PROVIDER="supersim"
elif [[ "$NETWORK" == "v0" ]]; then
  L2A_RPC="${L2A_RPC:-https://interop-v0-0.optimism.io}"
  L2B_RPC="${L2B_RPC:-https://interop-v0-1.optimism.io}"
  RPC_PROVIDER="interop-v0"
else
  echo -e "Unknown network: $NETWORK"
  echo -e "Supported networks: supersim, v0"
  exit 1
fi

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
DIM='\033[2m'
BOLD='\033[1m'
RESET='\033[0m'

rpc_call() {
  curl -sf -X POST "$1" \
    -H "Content-Type: application/json" \
    -d "{\"jsonrpc\":\"2.0\",\"method\":\"$2\",\"params\":[$3],\"id\":1}" 2>/dev/null \
    | python3 -c "import sys,json; print(json.load(sys.stdin).get('result','???'))" 2>/dev/null || echo "DOWN"
}

check_health() {
  if curl -sf "$1" >/dev/null 2>&1; then
    echo -e "${GREEN}UP${RESET}"
  else
    echo -e "${RED}DOWN${RESET}"
  fi
}

hex_to_dec() {
  python3 -c "print(int('$1', 16))" 2>/dev/null || echo "?"
}

while true; do
  clear

  echo -e "${BOLD}${CYAN}╔══════════════════════════════════════════════════════════╗${RESET}"
  echo -e "${BOLD}${CYAN}║         INTEROP STACK MONITOR  $(date +%H:%M:%S)                ║${RESET}"
  echo -e "${BOLD}${CYAN}╚══════════════════════════════════════════════════════════╝${RESET}"
  echo

  # ── Service Health ──────────────────────────────────────
  echo -e "${BOLD}▸ Service Health${RESET}"
  RPC_L2A=$(rpc_call "$L2A_RPC" "eth_chainId" "")
  RPC_L2B=$(rpc_call "$L2B_RPC" "eth_chainId" "")
  PONDER_HEALTH=$(check_health "$PONDER_API/health")

  BLOCK_A=$(rpc_call "$L2A_RPC" "eth_blockNumber" "")
  BLOCK_B=$(rpc_call "$L2B_RPC" "eth_blockNumber" "")

  if [[ "$RPC_L2A" != "DOWN" ]]; then
    CHAIN_A_ID=$(hex_to_dec "$RPC_L2A")
    BLOCK_A_DEC=$(hex_to_dec "$BLOCK_A")
    echo -e "  ${RPC_PROVIDER} L2A (chain ${CHAIN_A_ID}):  ${GREEN}UP${RESET}  block ${YELLOW}${BLOCK_A_DEC}${RESET}"
  else
    echo -e "  ${RPC_PROVIDER} L2A:  ${RED}DOWN${RESET}"
  fi

  if [[ "$RPC_L2B" != "DOWN" ]]; then
    CHAIN_B_ID=$(hex_to_dec "$RPC_L2B")
    BLOCK_B_DEC=$(hex_to_dec "$BLOCK_B")
    echo -e "  ${RPC_PROVIDER} L2B (chain ${CHAIN_B_ID}):  ${GREEN}UP${RESET}  block ${YELLOW}${BLOCK_B_DEC}${RESET}"
  else
    echo -e "  ${RPC_PROVIDER} L2B:  ${RED}DOWN${RESET}"
  fi

  echo -e "  ponder-interop:            ${PONDER_HEALTH}"

  # Check autorelayer (no health endpoint exposed locally, check process)
  if pgrep -f "autorelayer-interop" >/dev/null 2>&1 || pgrep -f "tsx.*autorelayer" >/dev/null 2>&1; then
    echo -e "  autorelayer-interop:       ${GREEN}UP${RESET}"
  else
    echo -e "  autorelayer-interop:       ${RED}DOWN${RESET}"
  fi
  echo

  # ── Ponder Message Stats ────────────────────────────────
  echo -e "${BOLD}▸ Message Stats ${DIM}(from ponder-interop)${RESET}"
  MSG_COUNT=$(curl -sf "$PONDER_API/messages/count" 2>/dev/null || echo "")
  if [[ -n "$MSG_COUNT" ]]; then
    SENT=$(echo "$MSG_COUNT" | python3 -c "import sys,json; d=json.load(sys.stdin); print(sum(x['count'] for x in d.get('sent',[])))" 2>/dev/null || echo "?")
    RELAYED=$(echo "$MSG_COUNT" | python3 -c "import sys,json; d=json.load(sys.stdin); print(sum(x['count'] for x in d.get('relayed',[])))" 2>/dev/null || echo "?")
    PENDING=$(echo "$MSG_COUNT" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('pending','?'))" 2>/dev/null || echo "?")
    echo -e "  Sent:     ${YELLOW}${SENT}${RESET}"
    echo -e "  Relayed:  ${GREEN}${RELAYED}${RESET}"
    echo -e "  Pending:  ${CYAN}${PENDING}${RESET}"
  else
    echo -e "  ${RED}Could not fetch message stats${RESET}"
  fi
  echo

  # ── Pending Messages Detail ─────────────────────────────
  echo -e "${BOLD}▸ Pending Messages${RESET}"
  PENDING_MSGS=$(curl -sf "$PONDER_API/messages/pending" 2>/dev/null || echo "[]")
  PENDING_COUNT=$(echo "$PENDING_MSGS" | python3 -c "import sys,json; print(len(json.load(sys.stdin)))" 2>/dev/null || echo "0")
  if [[ "$PENDING_COUNT" -gt 0 ]]; then
    echo "$PENDING_MSGS" | python3 -c "
import sys, json
msgs = json.load(sys.stdin)
for m in msgs[:5]:
    src = m.get('source','?')
    dst = m.get('destination','?')
    mh = m.get('messageHash','?')[:16]
    tx = m.get('transactionHash','?')[:16]
    print(f'  {src} → {dst}  msgHash={mh}…  tx={tx}…')
if len(msgs) > 5:
    print(f'  ... and {len(msgs)-5} more')
" 2>/dev/null
  else
    echo -e "  ${DIM}(none)${RESET}"
  fi
  echo

  # ── Promise Stats ───────────────────────────────────────
  echo -e "${BOLD}▸ Promises${RESET}"
  PROMISES=$(curl -sf "$PONDER_API/promises" 2>/dev/null || echo "[]")
  PROMISE_COUNT=$(echo "$PROMISES" | python3 -c "import sys,json; print(len(json.load(sys.stdin)))" 2>/dev/null || echo "0")
  if [[ "$PROMISE_COUNT" -gt 0 ]]; then
    echo "$PROMISES" | python3 -c "
import sys, json
from collections import Counter
promises = json.load(sys.stdin)
c = Counter(p['status'] for p in promises)
print(f'  Total: {len(promises)}  pending={c.get(\"pending\",0)}  resolved={c.get(\"resolved\",0)}  rejected={c.get(\"rejected\",0)}')
" 2>/dev/null
  else
    echo -e "  ${DIM}(none)${RESET}"
  fi
  echo

  # ── Chains ──────────────────────────────────────────────
  echo -e "${BOLD}▸ Indexed Chains${RESET}"
  CHAINS=$(curl -sf "$PONDER_API/chains" 2>/dev/null || echo "[]")
  echo "$CHAINS" | python3 -c "
import sys, json
chains = json.load(sys.stdin)
for c in chains:
    print(f'  {c[\"name\"]} (id={c[\"id\"]})  rpc={c[\"url\"]}')
" 2>/dev/null || echo -e "  ${RED}Could not fetch chains${RESET}"

  echo
  echo -e "${DIM}Refreshing every ${INTERVAL}s — Ctrl+C to quit${RESET}"
  sleep "$INTERVAL"
done
