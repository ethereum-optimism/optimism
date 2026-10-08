#!/usr/bin/env bash
# Waits until at least 40 GB are available (shared-host memory rule), then runs the checks under a 16 GB cap.
cd "$(dirname "$0")" || exit 1
# Needs mise on PATH (q.sh runs `mise exec node@22 -- quint`).
while (( $(awk "/MemAvailable/ {print int(\$2/1048576)}" /proc/meminfo) < 40 )); do sleep 30; done
echo "start $(date)"
exec systemd-run --user --scope -p MemoryMax=16G -p MemorySwapMax=0 \
  env LOGDIR="${LOGDIR:-logs}" DEPTH="${DEPTH:-15}" SAFE_DEPTH="${SAFE_DEPTH:-11}" FULL_DEPTH="${FULL_DEPTH:-10}" JOBS="${JOBS:-2}" BASE_PORT="${BASE_PORT:-9300}" ONLY="${ONLY:-.}" SKIP="${SKIP:-^$}" QUINT="$PWD/q.sh" ./run.sh verify
