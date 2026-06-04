#!/usr/bin/env bash
set -euo pipefail

# Tests git.just VERSION resolution logic.
#
# Scenarios covered:
#   1. Single-service tag on a commit          → resolves that service's version
#   2. Multi-service tags on a shared commit   → each service resolves only its own version
#   3. RC-only tag                             → resolves the RC version
#   4. Both RC and release tags               → prefers the release over the RC
#   5. No tag on commit                       → resolves to "untagged"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
JUSTFILES_DIR="${REPO_ROOT}/justfiles"

PASS=0
FAIL=0

# ── helpers ────────────────────────────────────────────────────────────────────

# assert_version <scenario> <project-dir> <gitcommit> <expected-version>
assert_version() {
  local scenario="$1"
  local project_dir="$2"
  local gitcommit="$3"
  local expected="$4"

  local actual
  actual=$(cd "${project_dir}" && GITCOMMIT="${gitcommit}" just --evaluate 2>/dev/null | awk '/^VERSION / { print $NF }' | tr -d '"')

  if [ "${actual}" = "${expected}" ]; then
    echo "  PASS: ${scenario}"
    PASS=$((PASS + 1))
  else
    echo "  FAIL: ${scenario}"
    echo "        expected: ${expected}"
    echo "        actual:   ${actual}"
    FAIL=$((FAIL + 1))
  fi
}

# setup_repo creates a temp git repo with a series of annotated tags and
# returns (via stdout) the path to the repo followed by the SHA of each commit,
# one per line.  Call as:
#   mapfile -t INFO < <(setup_repo)
#   REPO="${INFO[0]}" SHA1="${INFO[1]}" SHA2="${INFO[2]}"
setup_repo() {
  local repo
  repo="$(mktemp -d)"

  git -C "${repo}" init -q
  git -C "${repo}" config user.email "test@test.com"
  git -C "${repo}" config user.name "Test"

  # commit 1 – will carry the single-service and RC/release pairs
  git -C "${repo}" commit -q --allow-empty -m "commit1"
  local sha1
  sha1="$(git -C "${repo}" rev-parse HEAD)"

  # commit 2 – multi-service shared release commit
  git -C "${repo}" commit -q --allow-empty -m "commit2"
  local sha2
  sha2="$(git -C "${repo}" rev-parse HEAD)"

  # commit 3 – untagged
  git -C "${repo}" commit -q --allow-empty -m "commit3"
  local sha3
  sha3="$(git -C "${repo}" rev-parse HEAD)"

  # Tags on sha1: single service (conductor) + RC-only (proposer) + RC+release pair (batcher)
  git -C "${repo}" tag -a "op-conductor/v0.9.3"      -m "op-conductor/v0.9.3"      "${sha1}"
  git -C "${repo}" tag -a "op-proposer/v2.0.0-rc.1"  -m "op-proposer/v2.0.0-rc.1"  "${sha1}"
  git -C "${repo}" tag -a "op-batcher/v1.0.0-rc.1"   -m "op-batcher/v1.0.0-rc.1"   "${sha1}"
  git -C "${repo}" tag -a "op-batcher/v1.0.0"        -m "op-batcher/v1.0.0"        "${sha1}"

  # Tags on sha2: multiple services on the same commit (the shared-release scenario)
  git -C "${repo}" tag -a "op-node/v1.16.10-rc.1"    -m "op-node/v1.16.10-rc.1"    "${sha2}"
  git -C "${repo}" tag -a "op-node/v1.16.10"         -m "op-node/v1.16.10"         "${sha2}"
  git -C "${repo}" tag -a "op-batcher/v1.16.6-rc.1"  -m "op-batcher/v1.16.6-rc.1"  "${sha2}"
  git -C "${repo}" tag -a "op-batcher/v1.16.6"       -m "op-batcher/v1.16.6"       "${sha2}"
  git -C "${repo}" tag -a "op-proposer/v1.16.2-rc.1" -m "op-proposer/v1.16.2-rc.1" "${sha2}"
  git -C "${repo}" tag -a "op-proposer/v1.16.2"      -m "op-proposer/v1.16.2"      "${sha2}"

  echo "${repo}"
  echo "${sha1}"
  echo "${sha2}"
  echo "${sha3}"
}

# make_project <repo-path> <project-name> → creates a justfile that imports git.just
make_project() {
  local repo="$1"
  local project="$2"
  local dir="${repo}/${project}"
  mkdir -p "${dir}"
  cat > "${dir}/justfile" << EOF
import '${JUSTFILES_DIR}/go.just'
EOF
  echo "${dir}"
}

# ── main ───────────────────────────────────────────────────────────────────────

echo "Setting up test repo..."
mapfile -t INFO < <(setup_repo)
REPO="${INFO[0]}"
SHA1="${INFO[1]}"
SHA2="${INFO[2]}"
SHA3="${INFO[3]}"

# Create one project directory per service under the fake repo so that
# `basename justfile_directory()` resolves to the right project name.
DIR_CONDUCTOR="$(make_project "${REPO}" op-conductor)"
DIR_BATCHER="$(make_project "${REPO}"   op-batcher)"
DIR_PROPOSER="$(make_project "${REPO}"  op-proposer)"
DIR_NODE="$(make_project "${REPO}"      op-node)"

echo "Running tests..."

# 1. Single-service tag on a commit
assert_version \
  "single-service tag resolves correctly" \
  "${DIR_CONDUCTOR}" "${SHA1}" "v0.9.3"

# 2a. Multi-service shared commit – op-batcher
assert_version \
  "multi-service shared commit: op-batcher gets its own version" \
  "${DIR_BATCHER}" "${SHA2}" "v1.16.6"

# 2b. Multi-service shared commit – op-proposer (was broken before the fix)
assert_version \
  "multi-service shared commit: op-proposer gets its own version" \
  "${DIR_PROPOSER}" "${SHA2}" "v1.16.2"

# 2c. Multi-service shared commit – op-node
assert_version \
  "multi-service shared commit: op-node gets its own version" \
  "${DIR_NODE}" "${SHA2}" "v1.16.10"

# 3. RC-only tag on a commit → picks the RC
assert_version \
  "RC-only tag: resolves to the RC version" \
  "${DIR_PROPOSER}" "${SHA1}" "v2.0.0-rc.1"

# 4. Both RC and release tags on same commit → prefers the release over the RC
assert_version \
  "RC + release tags: prefers release over RC" \
  "${DIR_BATCHER}" "${SHA1}" "v1.0.0"

# 5. No tag on commit → "untagged"
assert_version \
  "no tag on commit: resolves to untagged" \
  "${DIR_CONDUCTOR}" "${SHA3}" "untagged"

# ── cleanup & summary ──────────────────────────────────────────────────────────

rm -rf "${REPO}"

echo ""
echo "Results: ${PASS} passed, ${FAIL} failed"
[ "${FAIL}" -eq 0 ]
