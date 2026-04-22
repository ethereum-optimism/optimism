#!/usr/bin/env bash

# Regenerate every Go binding under op-e2e/bindings/ from the current
# forge artifacts. Iterates the existing binding files, recovers the
# original contract name from each (`var <Name>MetaData = ...`), and
# re-runs the per-contract gen-binding.sh for it.
#
# Intentionally does NOT rebuild contracts — callers must ensure
# packages/contracts-bedrock/forge-artifacts/ is fresh (run
# `just -f ../packages/contracts-bedrock/justfile build` first, or
# rely on the pipeline-model selector to sequence the forge-build
# check before gen-go-bindings).
#
# Accepts an optional pattern arg (bash extglob) that filters which
# contracts to regenerate; default is all. Examples:
#   $ gen-bindings-all.sh              # all bindings
#   $ gen-bindings-all.sh 'L1*'        # only L1-prefixed contracts
#   $ gen-bindings-all.sh 'FaultDisputeGame'

set -euo pipefail

REPO_ROOT="$(git rev-parse --show-toplevel)"
cd "${REPO_ROOT}"

BINDINGS_DIR="op-e2e/bindings"
PATTERN="${1:-*}"

if [[ ! -d "${BINDINGS_DIR}" ]]; then
  echo "error: ${BINDINGS_DIR} does not exist" >&2
  exit 1
fi

matched=0
skipped=0
failed=0

for binding_file in "${BINDINGS_DIR}"/*.go; do
  # Recover the original (CamelCase) contract name from the file.
  # abigen emits `var <Name>MetaData = &bind.MetaData{...}` as the
  # first definition. Grep the first match.
  contract="$(grep -oE '^var [A-Z][A-Za-z0-9]*MetaData' "${binding_file}" \
    | head -n 1 \
    | sed -E 's/^var ([A-Za-z0-9]+)MetaData$/\1/')"

  if [[ -z "${contract}" ]]; then
    echo "warn: could not recover contract name from ${binding_file}, skipping" >&2
    ((skipped++))
    continue
  fi

  # shellcheck disable=SC2053
  if [[ ! "${contract}" == ${PATTERN} ]]; then
    continue
  fi

  ((matched++))
  echo "regen: ${contract}"
  if ! op-e2e/scripts/gen-binding.sh "${contract}"; then
    echo "  failed: ${contract}" >&2
    ((failed++))
  fi
done

if (( matched == 0 )); then
  echo "no bindings matched pattern '${PATTERN}'" >&2
  exit 1
fi

if (( failed > 0 )); then
  echo "${failed} of ${matched} regenerations failed" >&2
  exit 1
fi

echo "regenerated ${matched} bindings (${skipped} skipped)"
