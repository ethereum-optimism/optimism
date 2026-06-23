#!/usr/bin/env bash
#
# game-proposal-outputs.sh
#
# For each dispute game, ask OUR op-node what it derives, so you can tell whether
# a challenger pointed at it would agree with the proposal:
#   * optimism_outputAtBlock(<game l2BlockNumber>)   -> output root our node derives
#   * optimism_safeHeadAtL1Block(<game l1Head>)      -> our node's L2 safe head at
#                                                       the L1 view the game is anchored to
#
# A safe head BELOW the proposed block (with identical output roots between nodes)
# is the fingerprint of an incomplete safe-head DB / lagging node — the challenger
# clamps to the safe head and disputes valid proposals. Chain-agnostic.
#
# Usage:
#   game-proposal-outputs.sh <rollup-rpc> <l1-rpc> [--factory <addr> [--last N]] [game-addr ...]
#
#   <rollup-rpc>   OUR op-node rollup RPC (exposes optimism_* methods).
#   <l1-rpc>       L1 EL RPC (reads each game's l2BlockNumber()/l1Head()).
#   --factory A    DisputeGameFactory; enumerate games from it (newest first).
#   --last N       With --factory, inspect the newest N games (default 25).
#   game-addr ...  Explicit FaultDisputeGame addresses (overrides --factory).
#
# Discover games yourself instead with:
#   op-challenger list-games --game-factory-address <factory> --l1-eth-rpc <l1> --format json
#
# Env: RETRIES (default 5).

set -uo pipefail

ROLLUP_RPC="${1:-}"; L1_RPC="${2:-}"
if [[ -z "$ROLLUP_RPC" || -z "$L1_RPC" ]]; then
  echo "usage: $0 <rollup-rpc> <l1-rpc> [--factory <addr> [--last N]] [game-addr ...]" >&2
  exit 2
fi
shift 2

RETRIES="${RETRIES:-5}"
FACTORY=""; LAST=25; GAMES=()
while [[ $# -gt 0 ]]; do
  case "$1" in
    --factory) FACTORY="$2"; shift 2 ;;
    --last)    LAST="$2"; shift 2 ;;
    *)         GAMES+=("$1"); shift ;;
  esac
done

# selectors
SEL_L2BLOCK="0x8b85902b"   # l2BlockNumber()
SEL_L1HEAD="0x6361506d"    # l1Head()
SEL_STATUS="0x200d2ed2"    # status() -> uint8 (0=IN_PROGRESS,1=CHALLENGER_WINS,2=DEFENDER_WINS)
SEL_COUNT="0x4d1975b4"     # gameCount()
SEL_AT="0xbb8aa1fc"        # gameAtIndex(uint256) -> (gameType, timestamp, proxy)

rpc() { # rpc <url> <method> <params-json> -> .result (string), retries
  local url="$1" m="$2" p="$3" resp out attempt
  for ((attempt = 1; attempt <= RETRIES; attempt++)); do
    resp=$(curl -s -m 25 -X POST "$url" -H 'content-type: application/json' \
      --data "{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"$m\",\"params\":$p}")
    out=$(printf '%s' "$resp" | python3 -c 'import sys,json
try: d=json.load(sys.stdin)
except Exception: print("RETRY"); sys.exit()
r=d.get("result")
print(r if r is not None else "ERR:"+json.dumps(d.get("error","no result")))' 2>/dev/null)
    [[ "$out" == "RETRY" || -z "$out" ]] && { sleep 1; continue; }
    printf '%s' "$out"; return 0
  done
  printf ''; return 1
}
ethcall() { rpc "$L1_RPC" eth_call "[{\"to\":\"$1\",\"data\":\"$2\"},\"latest\"]"; }
hex() { printf '0x%x' "$1"; }
addr_of_word() { python3 -c "import sys;h=sys.argv[1][2:];w=h[int(sys.argv[2])*64:(int(sys.argv[2])+1)*64];print('0x'+w[24:])" "$1" "$2"; }
to_int() { python3 -c "import sys;print(int(sys.argv[1],16))" "$1"; }
is_hash32() { python3 -c "import sys;h=sys.argv[1];print('1' if h.startswith('0x') and len(h)==66 and int(h,16)!=0 else '0')" "$1" 2>/dev/null; }

# Enumerate games from a factory if no explicit games were given.
if [[ ${#GAMES[@]} -eq 0 ]]; then
  [[ -z "$FACTORY" ]] && { echo "provide game addresses or --factory <addr>" >&2; exit 2; }
  cnt_hex=$(ethcall "$FACTORY" "$SEL_COUNT"); cnt=$(to_int "$cnt_hex")
  start=$(( cnt - LAST )); (( start < 0 )) && start=0
  for (( i = cnt - 1; i >= start; i-- )); do
    res=$(ethcall "$FACTORY" "${SEL_AT}$(printf '%064x' "$i")")
    GAMES+=("$(addr_of_word "$res" 2)")
  done
fi

echo "# rollup (our node): $ROLLUP_RPC"
echo "# safeHeadAtL1Block queried per-game at each game's l1Head anchor"
printf '%-44s %-7s %-10s %-66s %-10s %s\n' "game" "status" "l2block" "outputRoot(ours)" "l1Head" "safeHead(ours)"

statusName() { case "$1" in 0) echo IN_PROG;; 1) echo CHAL_WON;; 2) echo DEF_WON;; *) echo "?$1";; esac; }

for game in "${GAMES[@]}"; do
  st=$(to_int "$(ethcall "$game" "$SEL_STATUS")" 2>/dev/null || echo -1)
  l2hex=$(ethcall "$game" "$SEL_L2BLOCK"); l2=$(to_int "$l2hex")
  l1h=$(ethcall "$game" "$SEL_L1HEAD")        # bytes32 L1 block hash
  l1num="ERR"
  if [[ "$(is_hash32 "$l1h")" == "1" ]]; then
    blk=$(rpc "$L1_RPC" eth_getBlockByHash "[\"$l1h\",false]")
    l1num=$(printf '%s' "$blk" | python3 -c 'import sys,json
try: print(int(json.load(sys.stdin)["number"],16))
except Exception: print("ERR")' 2>/dev/null)
  fi
  oroot=$(rpc "$ROLLUP_RPC" optimism_outputAtBlock "[\"$(hex "$l2")\"]" | python3 -c 'import sys,json
try: print(json.load(sys.stdin)["outputRoot"])
except Exception: print("ERR")' 2>/dev/null)
  shnum="ERR"; shhash=""
  if [[ "$l1num" != "ERR" && -n "$l1num" ]]; then
    sh=$(rpc "$ROLLUP_RPC" optimism_safeHeadAtL1Block "[\"$(hex "$l1num")\"]")
    read -r shnum shhash < <(printf '%s' "$sh" | python3 -c 'import sys,json
try:
  r=json.load(sys.stdin)["safeHead"]; n=r["number"]
  print(int(n,16) if isinstance(n,str) else int(n), r["hash"])
except Exception: print("ERR ERR")' 2>/dev/null)
  fi
  printf '%-44s %-7s %-10s %-66s %-10s %s %s\n' \
    "$game" "$(statusName "$st")" "$l2" "${oroot:-ERR}" "${l1num:-ERR}" "${shnum:-ERR}" "$shhash"
done
