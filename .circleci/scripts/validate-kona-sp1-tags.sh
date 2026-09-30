#!/usr/bin/env bash
set -euo pipefail

repo_root=$(git rev-parse --show-toplevel)
trial_dir=$(mktemp -d)
logs_dir="${repo_root}/ops/prestate-reproducibility/temp/logs/tagged-validation"
mkdir -p "$logs_dir"

cleanup() {
  if [[ -n "${disk_monitor_pid:-}" ]]; then
    kill "$disk_monitor_pid" 2>/dev/null || true
  fi
  for path in "$trial_dir"/build-*; do
    [[ -d "$path" ]] || continue
    sudo rm -rf "$path/rust/kona/sp1/programs/target/elf-compilation/docker" || true
    git -C "$repo_root" worktree remove --force "$path" || true
  done
  rm -rf "$trial_dir"
}
trap cleanup EXIT

[[ "$(uname -s -m)" == "Linux x86_64" ]] || {
  echo "This trial requires a native linux/amd64 machine" >&2
  exit 1
}
[[ -z "${KONA_CUSTOM_CONFIGS_DIR:-}" ]] || {
  echo "KONA_CUSTOM_CONFIGS_DIR must be unset" >&2
  exit 1
}

(
  while true; do
    printf '%s ' "$(date -u +%FT%TZ)"
    df -Pk "$repo_root" | awk 'NR == 2 { print $3, $4 }'
    sleep 15
  done
) > "${logs_dir}/disk-kib.txt" &
disk_monitor_pid=$!

build_tag() {
  local tag=$1 label=$2
  local ref="refs/tags/kona-sp1-proposer/${tag}"
  local commit worktree started elapsed manifest elf key derived_output derived sp1_tag docker_dir cleanup_errors first_other_dir
  commit=$(git -C "$repo_root" rev-parse --verify "${ref}^{commit}")
  worktree="${trial_dir}/build-${label}"
  git -C "$repo_root" worktree add --detach "$worktree" "$commit"
  sp1_tag=$(sed -nE 's/^SP1_TAG := "([^"]+)"$/\1/p' "${worktree}/rust/kona/sp1/justfile")
  [[ -n "$sp1_tag" ]]
  echo "TRIAL START label=${label} tag=${ref} commit=${commit} sp1_toolchain_tag=${sp1_tag}"
  started=$(date +%s)

  (
    cd "$worktree"
    mise trust
    mise install -v -y go just jq 'github:succinctlabs/sp1'
    cd rust/kona/sp1
    mise exec -- just install-sp1-toolchain
    mise exec -- just build-elfs
  ) 2>&1 | awk '
    BEGIN { previous = systime(); max_gap = 0 }
    {
      now = systime()
      if (now - previous > max_gap) max_gap = now - previous
      print
      fflush()
      previous = now
    }
    END {
      trailing_gap = systime() - previous
      if (trailing_gap > max_gap) max_gap = trailing_gap
      print "TRIAL longest_output_gap_seconds=" max_gap
      fflush()
    }
  ' | tee "${logs_dir}/${label}.txt"
  echo "TRIAL Docker build completed label=${label}" | tee -a "${logs_dir}/summary.txt"

  manifest="${worktree}/rust/kona/sp1/elf/vkeys.toml"
  elf="${worktree}/rust/kona/sp1/elf/super-aggregation-elf"
  [[ -s "$elf" && -f "$manifest" ]]
  echo "TRIAL artifacts present label=${label}" | tee -a "${logs_dir}/summary.txt"
  key=$(grep -E '^[[:space:]]*super-aggregation[[:space:]]*=' "$manifest" | sed -nE 's/^[[:space:]]*super-aggregation[[:space:]]*=[[:space:]]*"(0x[0-9a-fA-F]{64})"[[:space:]]*$/\1/p')
  [[ "$key" =~ ^0x[0-9a-fA-F]{64}$ ]]
  derived_output=$(cd "$worktree" && SP1_PROVER=cpu mise exec -- cargo prove vkey --elf "$elf")
  derived=$(printf '%s\n' "$derived_output" | grep -oE '0x[0-9a-fA-F]{64}')
  [[ "$derived" =~ ^0x[0-9a-fA-F]{64}$ ]]
  [[ "${key,,}" == "${derived,,}" ]]
  echo "TRIAL manifest and ELF agree label=${label} vkey=${key}" | tee -a "${logs_dir}/summary.txt"
  elapsed=$(($(date +%s) - started))
  echo "TRIAL PASS label=${label} tag=${ref} commit=${commit} vkey=${key} elapsed_seconds=${elapsed}" | tee -a "${logs_dir}/summary.txt"
  docker_dir="${worktree}/rust/kona/sp1/programs/target/elf-compilation/docker"
  cleanup_errors="${logs_dir}/${label}-cleanup-errors.txt"
  echo "TRIAL job identity: $(id)" | tee -a "${logs_dir}/summary.txt"
  echo "TRIAL Docker security options: $(docker info --format '{{json .SecurityOptions}}')" | tee -a "${logs_dir}/summary.txt"
  stat -c 'TRIAL output directory uid=%u gid=%g mode=%a path=%n' "$docker_dir" | tee -a "${logs_dir}/summary.txt"
  sudo -n find "$docker_dir" -type d -printf '%U:%G %m %p\n' |
    awk -v uid="$(id -u)" '
      { total++; if (split($1, owner, ":") && owner[1] != uid) { other++; if (other <= 20) print "TRIAL other-owned directory: " $0 } }
      END { printf "TRIAL directories=%d other-owned=%d\n", total, other }
    ' | tee -a "${logs_dir}/summary.txt"
  if command -v getfacl >/dev/null; then
    sudo -n getfacl -p "$docker_dir" | tee -a "${logs_dir}/summary.txt"
    first_other_dir=$(sudo -n find "$docker_dir" -type d ! -user "$(id -u)" -print -quit)
    if [[ -n "$first_other_dir" ]]; then
      sudo -n getfacl -p "$first_other_dir" | tee -a "${logs_dir}/summary.txt"
    fi
  fi
  if rm -rf "$docker_dir" 2>"$cleanup_errors"; then
    echo "TRIAL unprivileged cleanup succeeded; sudo was unnecessary" | tee -a "${logs_dir}/summary.txt"
  else
    echo "TRIAL unprivileged cleanup failed; first errors:" | tee -a "${logs_dir}/summary.txt"
    sed -n '1,20p' "$cleanup_errors" | tee -a "${logs_dir}/summary.txt"
    sudo -n rm -rf "$docker_dir"
    [[ ! -e "$docker_dir" ]]
    echo "TRIAL passwordless sudo cleanup succeeded after unprivileged failure" | tee -a "${logs_dir}/summary.txt"
  fi
  git -C "$repo_root" worktree remove --force "$worktree"
}

build_tag v0.0.5 first

kill "$disk_monitor_pid" 2>/dev/null || true
wait "$disk_monitor_pid" 2>/dev/null || true
disk_monitor_pid=
awk 'BEGIN { peak = 0 } $2 > peak { peak = $2 } END { print "TRIAL peak_filesystem_used_kib=" peak }' "${logs_dir}/disk-kib.txt"
