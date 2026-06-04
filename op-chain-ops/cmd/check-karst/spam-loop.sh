#!/usr/bin/env bash
#
# Run the check-karst calldata spammer for one minute every 15 minutes, forever.
#
# Each cycle spams L2 for SPAM_DURATION (default 60s), then idles until the
# SPAM_PERIOD_SECONDS boundary (default 900s = 15 min) before spamming again.
# The spammer handles SIGTERM cleanly, so `timeout` stops it gracefully after
# the active window.
#
# Provide a funded L2 key via CHECK_KARST_ACCOUNT (or pass --account ...). Any
# extra args are forwarded to `check-karst spam`, e.g. --l2, --accounts,
# --fund-ether, --rps, --block-time (see `check-karst spam --help`).
#
# Examples:
#   CHECK_KARST_ACCOUNT=0x... ./spam-loop.sh
#   ./spam-loop.sh --account 0x... --l2 http://localhost:9545 --accounts 20 --fund-ether 1
#
# Env overrides:
#   SPAM_PERIOD_SECONDS   full cycle length in seconds (default 900)
#   SPAM_DURATION         active spam window, a `timeout` duration (default 60s)
#   SPAM_KILL_GRACE       SIGKILL grace after SIGTERM (default 10s)
#   CHECK_KARST_BIN       path to a prebuilt check-karst binary (skips building)

set -euo pipefail

PERIOD_SECONDS="${SPAM_PERIOD_SECONDS:-900}"   # 15 minutes
DURATION="${SPAM_DURATION:-60s}"               # 1 minute
KILL_GRACE="${SPAM_KILL_GRACE:-10s}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"

log() { printf '%s %s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$*"; }

# A funded key is required; fail fast with a clear message instead of erroring
# once per cycle forever.
if [[ -z "${CHECK_KARST_ACCOUNT:-}" && "$*" != *--account* ]]; then
  echo "error: provide a funded L2 key via CHECK_KARST_ACCOUNT env or --account flag" >&2
  exit 1
fi

# Pick a GNU `timeout` (named `gtimeout` on macOS via coreutils).
if command -v timeout >/dev/null 2>&1; then
  TIMEOUT=timeout
elif command -v gtimeout >/dev/null 2>&1; then
  TIMEOUT=gtimeout
else
  echo "error: GNU 'timeout' (coreutils) not found on PATH" >&2
  exit 1
fi

# Resolve the spammer binary: reuse $CHECK_KARST_BIN, else build once up front
# (so we don't recompile every cycle).
BIN="${CHECK_KARST_BIN:-}"
CLEANUP_BIN=""
if [[ -z "$BIN" ]]; then
  command -v go >/dev/null 2>&1 || {
    echo "error: 'go' not found on PATH (set CHECK_KARST_BIN to a prebuilt binary instead)" >&2
    exit 1
  }
  BIN="$(mktemp -t check-karst.XXXXXX)"
  CLEANUP_BIN="$BIN"
  log "building check-karst from ${SCRIPT_DIR} ..."
  ( cd "$SCRIPT_DIR" && go build -o "$BIN" . )
fi

cleanup() { [[ -n "$CLEANUP_BIN" ]] && rm -f "$CLEANUP_BIN"; }
trap cleanup EXIT
trap 'log "interrupted; exiting"; exit 0' INT TERM

log "spam loop starting: ${DURATION} of spam every ${PERIOD_SECONDS}s (Ctrl+C to stop)"

while true; do
  start=$(date +%s)
  log "spam cycle begin (${DURATION})"

  # timeout sends SIGTERM after DURATION (the spammer exits cleanly), escalating
  # to SIGKILL after KILL_GRACE if it ignores it. set +e so we can read the code.
  set +e
  "$TIMEOUT" --kill-after="$KILL_GRACE" --signal=TERM "$DURATION" "$BIN" spam "$@"
  rc=$?
  set -e

  case "$rc" in
    124) log "spam cycle done (ran the full ${DURATION})" ;;
    0)   log "spam cycle ended early (rc=0; e.g. accounts ran out of funds)" ;;
    *)   log "WARNING: spam exited with rc=${rc}; continuing to next cycle" ;;
  esac

  now=$(date +%s)
  sleep_for=$(( PERIOD_SECONDS - (now - start) ))
  if (( sleep_for < 0 )); then sleep_for=0; fi
  log "sleeping ${sleep_for}s until next cycle"
  sleep "$sleep_for"
done
