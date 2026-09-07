#!/usr/bin/env bash
# Temporary, CI-only diagnostic against the immutable archive from pipeline 133962.
set -euo pipefail
[[ "${CIRCLECI:-}" == true ]] || { echo 'Run this diagnostic in CI.' >&2; exit 1; }

logs=test/kontrol/logs
container=withdrawal-segment-profile
mkdir -p "$logs"
curl --fail --location --retry 3 --retry-delay 5 --max-time 300 \
  'https://output.circle-artifacts.com/output/job/4b03459c-f95f-4603-baa5-c6c670bdc3a6/artifacts/0/packages/contracts-bedrock/test/kontrol/logs/kontrol-results_latest.tar.gz' \
  --output /tmp/withdrawal-profile-input.tar.gz
printf '%s  %s\n' '60cd44aab230356145b88a3e58e145aa9ad6f3dfeb9a3a10058ea524d7503895' \
  /tmp/withdrawal-profile-input.tar.gz | sha256sum --check
docker run --detach --name "$container" --network none --entrypoint sleep \
  runtimeverificationinc/kontrol@sha256:858f004144d61b005997f56bb8b7cd15673850286c96e0e5ec0502d9c9a9e204 infinity
cleanup() {
  local result_status=$?
  trap - EXIT
  if ! docker exec "$container" tar -czf /tmp/profiles.tar.gz -C /tmp/withdrawal-profile profiles \
    || ! docker cp "$container:/tmp/profiles.tar.gz" "$logs/segment-profiles.tar.gz"; then
    result_status=1
  fi
  docker rm --force "$container" >/dev/null || true
  exit "$result_status"
}
trap cleanup EXIT
docker cp /tmp/withdrawal-profile-input.tar.gz "$container:/tmp/input.tar.gz"
docker cp test/kontrol/scripts/profile-withdrawal.py "$container:/tmp/profile-withdrawal.py"
docker exec "$container" mkdir -p /tmp/withdrawal-profile/profiles
docker exec "$container" tar -xzf /tmp/input.tar.gz -C /tmp/withdrawal-profile

# Run sequentially with compact logs to avoid competing for memory or CPU.
first_status=0
second_status=0
docker exec "$container" timeout --signal=TERM --kill-after=15s 20m \
  python3 /tmp/profile-withdrawal.py /tmp/withdrawal-profile/kout-proofs 23 \
  > "$logs/segment-23.log" 2>&1 || first_status=$?
docker exec "$container" timeout --signal=TERM --kill-after=15s 20m \
  python3 /tmp/profile-withdrawal.py /tmp/withdrawal-profile/kout-proofs 30 \
  > "$logs/segment-30.log" 2>&1 || second_status=$?
printf 'Diagnostic replay only: node 23 exit=%s; node 30 exit=%s\n' "$first_status" "$second_status" \
  | tee "$logs/segment-status.txt"
[[ "$first_status" == 0 && "$second_status" == 0 ]]
