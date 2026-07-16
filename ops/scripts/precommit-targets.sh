#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Usage: ops/scripts/precommit-targets.sh [--run] [git diff args...]

Print the commands for the precommit checks relevant to the changed files.
With no git diff args, changes on this branch are detected with origin/HEAD...HEAD.

Examples:
  ops/scripts/precommit-targets.sh
  ops/scripts/precommit-targets.sh --run
  ops/scripts/precommit-targets.sh origin/develop...HEAD
  ops/scripts/precommit-targets.sh --cached
  ops/scripts/precommit-targets.sh HEAD -- op-node/

Options:
  --run       Run the selected commands instead of printing them.
  -h, --help  Show this help.
EOF
}

quote() {
  local value="${1//\'/\'\\\'\'}"
  printf "'%s'" "${value}"
}

add_raw_command() {
  local key="$1"
  local line="$2"

  local existing
  for existing in "${command_keys[@]}"; do
    if [[ "${existing}" == "${key}" ]]; then
      return
    fi
  done

  command_keys+=("${key}")
  command_lines+=("${line}")
}

add_just_command() {
  local key="$1"
  local dir="$2"
  shift 2

  add_raw_command "${key}" "mise exec -- just -f $(quote "${dir}/justfile") -d $(quote "${dir}") $*"
}

add_mise_x_just_command() {
  local key="$1"
  local dir="$2"
  shift 2

  add_raw_command "${key}" "mise x -- just -f $(quote "${dir}/justfile") -d $(quote "${dir}") $*"
}

append_unique_line() {
  local var_name="$1"
  local value="$2"
  local current
  eval "current=\"\${${var_name}:-}\""
  if ! grep -Fxq -- "${value}" <<<"${current}"; then
    printf -v "${var_name}" '%s%s\n' "${current}" "${value}"
  fi
}

go_package_selector() {
  local file="$1"
  local dir
  dir="$(dirname "${file}")"
  if [[ "${dir}" == "." ]]; then
    printf '.'
  else
    printf './%s' "${dir}"
  fi
}

rust_package_for_file() {
  local file="$1"
  local dir
  dir="${repo_root}/$(dirname "${file}")"

  while [[ "${dir}" == "${repo_root}/rust"* && "${dir}" != "${repo_root}/rust" ]]; do
    if [[ -f "${dir}/Cargo.toml" ]]; then
      awk '
        /^\[package\]$/ { in_package = 1; next }
        /^\[/ { in_package = 0 }
        in_package && $1 == "name" {
          gsub(/"/, "", $3)
          print $3
          exit
        }
      ' "${dir}/Cargo.toml"
      return
    fi
    dir="$(dirname "${dir}")"
  done
}

repo_root="$(git rev-parse --show-toplevel)"
run=false
diff_args=()

while [[ $# -gt 0 ]]; do
  case "$1" in
    --run)
      run=true
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    --)
      diff_args+=("$1")
      shift
      while [[ $# -gt 0 ]]; do
        diff_args+=("$1")
        shift
      done
      ;;
    *)
      diff_args+=("$1")
      shift
      ;;
  esac
done

if [[ "${#diff_args[@]}" -eq 0 ]]; then
  base_ref="$(git -C "${repo_root}" symbolic-ref --quiet --short refs/remotes/origin/HEAD || true)"
  if [[ -z "${base_ref}" ]]; then
    base_ref="origin/develop"
  fi
  merge_base="$(git -C "${repo_root}" merge-base "${base_ref}" HEAD)"
  diff_args=("${merge_base}...HEAD")
fi

git_diff_args=(--name-only -z "${diff_args[@]}")

changed_files=()
while IFS= read -r -d '' file; do
  changed_files+=("${file}")
done < <(git -C "${repo_root}" diff "${git_diff_args[@]}")

go_lint=false
cannon_tests=false
rust_checks=false
rust_no_std=false
contracts_checks=false
go_component_test_dirs=""
go_package_tests=""
rust_package_tests=""
acceptance_package_tests=""
circleci_checks=false

for file in "${changed_files[@]}"; do
  top_dir="${file%%/*}"
  case "${file}" in
    go.mod|go.sum)
      go_lint=true
      ;;
    .golangci.yaml|linter/*|justfiles/go.just)
      go_lint=true
      ;;
    *.go)
      go_lint=true
      case "${file}" in
        cannon/*)
          cannon_tests=true
          ;;
        op-acceptance-tests/tests/*)
          append_unique_line acceptance_package_tests "$(go_package_selector "${file}")"
          ;;
        op-acceptance-tests/*)
          append_unique_line acceptance_package_tests "$(go_package_selector "${file}")"
          ;;
        rust/*)
          rust_checks=true
          rust_package="$(rust_package_for_file "${file}")"
          if [[ -n "${rust_package}" ]]; then
            append_unique_line rust_package_tests "${rust_package}"
          fi
          append_unique_line go_package_tests "$(go_package_selector "${file}")"
          ;;
        op-e2e/*)
          append_unique_line go_package_tests "$(go_package_selector "${file}")"
          ;;
        *)
          if [[ -f "${repo_root}/${top_dir}/justfile" ]] && grep -Eq '^test([[:space:]:*]|$)' "${repo_root}/${top_dir}/justfile"; then
            append_unique_line go_component_test_dirs "${top_dir}"
          else
            append_unique_line go_package_tests "$(go_package_selector "${file}")"
          fi
          ;;
      esac
      ;;
  esac

  case "${file}" in
    cannon/*)
      cannon_tests=true
      ;;
    rust/*)
      rust_checks=true
      rust_package="$(rust_package_for_file "${file}")"
      if [[ -n "${rust_package}" ]]; then
        append_unique_line rust_package_tests "${rust_package}"
      fi
      case "${file}" in
        rust/Cargo.toml|rust/Cargo.lock|rust/kona/*|rust/op-alloy/*|rust/alloy-op-evm/*|rust/alloy-op-hardforks/*|rust/op-revm/*)
          rust_no_std=true
          ;;
      esac
      ;;
    packages/contracts-bedrock/*|op-core/forks/*|op-core/nuts/*|.semgrep/*)
      contracts_checks=true
      ;;
    .circleci/*)
      circleci_checks=true
      ;;
  esac
done

command_keys=()
command_lines=()

if [[ "${go_lint}" == true ]]; then
  add_just_command go-lint "${repo_root}" lint-go
fi
if [[ "${cannon_tests}" == true ]]; then
  add_just_command cannon-tests "${repo_root}/cannon" test
fi
while IFS= read -r dir; do
  if [[ -n "${dir}" ]]; then
    add_just_command "go-component-test:${dir}" "${repo_root}/${dir}" test
  fi
done <<<"${go_component_test_dirs}"
go_package_args=""
while IFS= read -r package; do
  if [[ -n "${package}" ]]; then
    go_package_args+=" $(quote "${package}")"
  fi
done <<<"${go_package_tests}"
if [[ -n "${go_package_args}" ]]; then
  add_raw_command go-package-tests "mise exec -- just -f $(quote "${repo_root}/justfile") -d $(quote "${repo_root}") go-test-packages${go_package_args}"
fi
if [[ "${rust_checks}" == true ]]; then
  add_just_command rust-fmt "${repo_root}/rust" fmt-fix
  add_just_command rust-lint "${repo_root}/rust" lint
fi
rust_package_args=" -E $(quote '!test(test_online)')"
while IFS= read -r package; do
  if [[ -n "${package}" ]]; then
    rust_package_args+=" -p $(quote "${package}")"
  fi
done <<<"${rust_package_tests}"
if [[ "${rust_package_args}" != " -E $(quote '!test(test_online)')" ]]; then
  add_raw_command rust-test-unit-packages "mise exec -- just -f $(quote "${repo_root}/rust/justfile") -d $(quote "${repo_root}/rust") test-unit${rust_package_args}"
fi
if [[ "${rust_no_std}" == true ]]; then
  add_just_command rust-check-no-std "${repo_root}/rust" check-no-std
fi
if [[ "${contracts_checks}" == true ]]; then
  add_mise_x_just_command contracts-lint "${repo_root}/packages/contracts-bedrock" lint
  add_mise_x_just_command contracts-test-dev "${repo_root}/packages/contracts-bedrock" test-dev
fi
acceptance_package_args=""
while IFS= read -r package; do
  if [[ -n "${package}" ]]; then
    acceptance_package_args+=" $(quote "${package}")"
  fi
done <<<"${acceptance_package_tests}"
if [[ -n "${acceptance_package_args}" ]]; then
  add_raw_command acceptance-packages "RUST_JIT_BUILD=1 mise exec -- just -f $(quote "${repo_root}/op-acceptance-tests/justfile") -d $(quote "${repo_root}/op-acceptance-tests") test${acceptance_package_args}"
fi
if [[ "${circleci_checks}" == true ]]; then
  add_raw_command circleci-merge "mise exec -- bash $(quote "${repo_root}/.circleci/scripts/merge-configs.sh")"
  add_raw_command circleci-decision-tree "mise exec -- bash $(quote "${repo_root}/.circleci/scripts/test-decision-tree.sh")"
fi

if [[ "${run}" == false ]]; then
  if [[ "${#command_lines[@]}" -gt 0 ]]; then
    printf '%s\n' "${command_lines[@]}"
  fi
  exit 0
fi

for index in "${!command_lines[@]}"; do
  bash -c "${command_lines[${index}]}"
done
