#!/usr/bin/env bash
set -euo pipefail
SOURCE_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
HASH=0x1111111111111111111111111111111111111111111111111111111111111111
OTHER=0x2222222222222222222222222222222222222222222222222222222222222222

run_case() {
  local scenario=$1
  local expected=$2
  local root
  root=$(mktemp -d)
  mkdir -p "$root/ops/prestate-reproducibility" "$root/bin"
  cp "$SOURCE_DIR/build-prestates.sh" "$root/ops/prestate-reproducibility/"
  cat > "$root/bin/git" <<'STUB'
#!/usr/bin/env bash
set -euo pipefail
if [[ "$1" == "-C" ]]; then shift 2; fi
printf '%s\n' "$*" >> "$TEST_ROOT/git.calls"
case "$1 $2" in
  "worktree add")
    mkdir -p "$3/rust/kona/sp1/elf"
    touch "$3/rust/kona/sp1/justfile" "$3/rust/kona/sp1/elf/.gitignore"
    if [[ "$TEST_SCENARIO" == "sudo-fails" ]]; then
      mkdir -p "$3/rust/kona/sp1/programs/target/elf-compilation/docker"
    elif [[ "$TEST_SCENARIO" == "symlink-parent" ]]; then
      mkdir -p "$3/rust/kona/sp1/programs/target" "$TEST_ROOT/outside/docker"
      touch "$TEST_ROOT/outside/docker/marker"
      ln -s "$TEST_ROOT/outside" "$3/rust/kona/sp1/programs/target/elf-compilation"
    fi ;;
  "worktree remove") ;;
  "tag --list") ;;
  "rev-parse --verify")
    [[ "$3" == refs/tags/kona-sp1-program/v*'^{commit}' ]] || exit 1
    [[ "$TEST_SCENARIO" != "missing-tag" ]] || exit 1
    printf '%040d\n' 1 ;;
  "checkout --detach") [[ "$3" == "--force" && "$4" == "$(printf '%040d' 1)" ]] ;;
  "clean -fdX") find rust/kona/sp1/elf -maxdepth 1 -type f ! -name .gitignore -delete ;;
  *) echo "unexpected git call: $*" >&2; exit 1 ;;
esac
STUB
  cat > "$root/bin/rm" <<'STUB'
#!/usr/bin/env bash
set -euo pipefail
if [[ "$TEST_SCENARIO" == "sudo-fails" && "${2:-}" == */elf-compilation/docker ]]; then
  exit 1
fi
if [[ "$TEST_SCENARIO" == "symlink-parent" && "${2:-}" == */elf-compilation/docker ]]; then
  exec /bin/rm -rf "$TEST_ROOT/outside/docker"
fi
exec /bin/rm "$@"
STUB
  cat > "$root/bin/sudo" <<'STUB'
#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "$*" >> "$TEST_ROOT/sudo.calls"
exit 1
STUB
  cat > "$root/bin/mise" <<'STUB'
#!/usr/bin/env bash
set -euo pipefail
case "$1" in
  trust|install) exit 0 ;;
  exec)
    shift 2
    case "$1 $2" in
      "go run")
        while [[ "$1" != "--output" ]]; do shift; done
        printf 'snapshot\n' > "$2"
        if [[ "$TEST_SCENARIO" == "zero" ]]; then
          exit 0
        elif [[ "$TEST_SCENARIO" == "stale" || "$TEST_SCENARIO" == "stdin-consumer" ]]; then
          printf '0.0.4\n0.0.5\n'
        else
          printf '0.0.5\n'
        fi ;;
      "just build-elfs")
        [[ "$TEST_SCENARIO" != "stdin-consumer" ]] || cat > /dev/null
        count=0
        [[ ! -f "$TEST_ROOT/count" ]] || count=$(cat "$TEST_ROOT/count")
        count=$((count + 1))
        printf '%s\n' "$count" > "$TEST_ROOT/count"
        [[ "$TEST_SCENARIO" != "failed-recipe" ]] || exit 1
        [[ "$TEST_SCENARIO" != "missing-vkey" ]] || { mkdir -p elf; printf 'elf' > elf/super-aggregation-elf; exit 0; }
        [[ "$TEST_SCENARIO" != "stale" || "$count" -eq 1 ]] || exit 0
        [[ -d elf && -f elf/.gitignore ]] || { echo 'missing tracked ELF directory' >&2; exit 1; }
        printf 'elf' > elf/super-aggregation-elf
        hash="$TEST_HASH"
        [[ "$TEST_SCENARIO" != "malformed-vkey" ]] || hash=0x1234
        [[ "$TEST_SCENARIO" != "uppercase-vkey" ]] || hash=0xAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
        [[ "$TEST_SCENARIO" != "zero-vkey" ]] || hash=0x0000000000000000000000000000000000000000000000000000000000000000
        printf 'super-aggregation = "%s"\n' "$hash" > elf/vkeys.toml
        [[ "$TEST_SCENARIO" != "duplicate-vkey" ]] || printf 'super-aggregation = "%s"\n' "$hash" >> elf/vkeys.toml ;;
      "cargo metadata")
        [[ "$TEST_SCENARIO" != "failed-metadata" ]] || exit 1
        [[ "$TEST_SCENARIO" != "invalid-metadata" ]] || { printf 'invalid metadata'; exit 0; }
        features='[]'
        [[ "$TEST_SCENARIO" != "test-config-fallback" ]] || features='["test-config-fallback"]'
        printf '{"packages":[{"id":"guest","name":"kona-sp1-super-range"}],"resolve":{"nodes":[{"id":"guest","features":%s}]}}\n' "$features" ;;
      "cargo prove")
        hash="$TEST_HASH"
        [[ "$TEST_SCENARIO" != "vkey-mismatch" ]] || hash="$TEST_OTHER"
        [[ "$TEST_SCENARIO" != "derived-long" ]] || hash="${hash}1"
        [[ "$TEST_SCENARIO" != "uppercase-vkey" ]] || hash=0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
        printf 'Verification key: %s\n' "$hash" ;;
      *) echo "unexpected mise exec: $*" >&2; exit 1 ;;
    esac ;;
  *) echo "unexpected mise call: $*" >&2; exit 1 ;;
esac
STUB
  chmod +x "$root/bin/git" "$root/bin/mise" "$root/bin/rm" "$root/bin/sudo"
  local status=0
  TEST_ROOT="$root" TEST_SCENARIO="$scenario" TEST_HASH="$HASH" TEST_OTHER="$OTHER" \
    PATH="$root/bin:$PATH" BASH_ENV=/dev/null KONA_CUSTOM_CONFIGS_DIR="$([[ "$scenario" == "custom-config" ]] && printf '/tmp/custom' || true)" \
    env -u 'BASH_FUNC_mise%%' bash "$root/ops/prestate-reproducibility/build-prestates.sh" < /dev/null > "$root/output" 2>&1 || status=$?
  if [[ "$expected" == "zero" ]]; then
    [[ "$status" -eq 0 ]] || { cat "$root/output"; exit 1; }
    grep -q 'Kona SP1 registry selection: 0 entries' "$root/output"
    jq -e 'length == 0' "$root/ops/prestate-reproducibility/temp/states/versions.json" > /dev/null
  elif [[ "$expected" == "success" ]]; then
    [[ "$status" -eq 0 ]] || { cat "$root/output"; exit 1; }
    grep -q "Kona SP1 version 0.0.5: rebuilt super-aggregation vkey $HASH" "$root/output"
    jq -e --arg hash "$HASH" 'length == 1 and .[0] == {version:"0.0.5",hash:$hash,type:"kona-sp1"}' \
      "$root/ops/prestate-reproducibility/temp/states/versions.json" > /dev/null
    grep -Fq 'rev-parse --verify refs/tags/kona-sp1-program/v0.0.5^{commit}' "$root/git.calls"
  elif [[ "$expected" == "two" ]]; then
    [[ "$status" -eq 0 ]] || { cat "$root/output"; exit 1; }
    jq -e 'length == 2 and .[0].version == "0.0.4" and .[1].version == "0.0.5"' \
      "$root/ops/prestate-reproducibility/temp/states/versions.json" > /dev/null
    [[ "$(cat "$root/count")" -eq 2 ]] || { echo 'expected two SP1 builds' >&2; exit 1; }
  else
    [[ "$status" -ne 0 ]] || { echo "$scenario unexpectedly passed" >&2; exit 1; }
    grep -q "Kona SP1 version 0.0.5:" "$root/output" || { cat "$root/output"; exit 1; }
  fi
  if [[ "$scenario" == "sudo-fails" ]]; then
    grep -Fq 'worktree remove' "$root/git.calls" || { echo 'worktree cleanup was skipped' >&2; exit 1; }
    grep -Fq -- '-n rm -rf' "$root/sudo.calls" || { echo 'sudo fallback was skipped' >&2; exit 1; }
  elif [[ "$scenario" == "test-config-fallback" ]]; then
    grep -q 'test-config-fallback must be disabled' "$root/output" || { cat "$root/output"; exit 1; }
    [[ ! -f "$root/count" ]] || { echo 'unsafe guest build was started' >&2; exit 1; }
  elif [[ "$scenario" == "symlink-parent" ]]; then
    [[ -f "$root/outside/docker/marker" ]] || { echo 'cleanup followed a symlink outside the worktree' >&2; exit 1; }
  fi
  rm -rf "$root"
}

run_case success success
run_case zero zero
run_case sudo-fails success
run_case symlink-parent success
run_case stdin-consumer two
for scenario in missing-tag failed-recipe missing-vkey malformed-vkey uppercase-vkey duplicate-vkey zero-vkey vkey-mismatch derived-long custom-config stale failed-metadata invalid-metadata test-config-fallback; do
  run_case "$scenario" failure
done
echo "Kona SP1 build driver fixtures passed"
