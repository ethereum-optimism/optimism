#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
test_repo="$(mktemp -d)"
trap 'rm -rf "${test_repo}"' EXIT
failures=0

git -C "${test_repo}" init -q
git -C "${test_repo}" config user.email "test@example.com"
git -C "${test_repo}" config user.name "Test"

for component in op-deployer op-acceptance-tests op-node; do
  mkdir -p "${test_repo}/${component}/pkg"
  cat >"${test_repo}/${component}/justfile" <<'EOF'
test:
    true
EOF
  printf 'package pkg\n' >"${test_repo}/${component}/pkg/example.go"
done

git -C "${test_repo}" add .
git -C "${test_repo}" commit -qm "initial"

selector_output_for() {
  local component="$1"
  git -C "${test_repo}" checkout -q -- .
  printf '\n' >>"${test_repo}/${component}/pkg/example.go"
  (cd "${test_repo}" && "${script_dir}/precommit-targets.sh" HEAD)
}

assert_contains() {
  local output="$1"
  local expected="$2"
  if [[ "${output}" != *"${expected}"* ]]; then
    printf 'expected output to contain %q:\n%s\n' "${expected}" "${output}" >&2
    failures=$((failures + 1))
  fi
}

assert_not_contains() {
  local output="$1"
  local unexpected="$2"
  if [[ "${output}" == *"${unexpected}"* ]]; then
    printf 'expected output not to contain %q:\n%s\n' "${unexpected}" "${output}" >&2
    failures=$((failures + 1))
  fi
}

for component in op-deployer op-acceptance-tests; do
  output="$(selector_output_for "${component}")"
  assert_contains "${output}" "lint-go"
  assert_not_contains "${output}" "${test_repo}/${component}/justfile"
  assert_not_contains "${output}" "go-test-packages"
done

output="$(selector_output_for op-node)"
assert_contains "${output}" "lint-go"
assert_contains "${output}" "${test_repo}/op-node/justfile"

[[ "${failures}" -eq 0 ]]
