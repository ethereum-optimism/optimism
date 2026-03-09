#!/usr/bin/env bash
# check-shard-coverage.sh — Verify all test packages are covered by CI shard gates.
#
# Compares test packages on disk against the union of all ci-shard-* gates in
# acceptance-tests.yaml. Fails if any package with test files exists on disk
# but is not covered by a shard gate (or explicitly listed as excluded).
#
# Run from op-acceptance-tests/:
#   ./scripts/check-shard-coverage.sh

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
YAML="$ROOT_DIR/acceptance-tests.yaml"
TESTS_DIR="$ROOT_DIR/tests"

MODULE="github.com/ethereum-optimism/optimism/op-acceptance-tests/tests"

# --- Known exclusions (covered by develop-CI, nightly, or flake-shake) ---
# If you add a package here, add a comment explaining where it's covered.
EXCLUDED_PACKAGES=(
  # Fault-proof tests — covered by develop-CI gates (isthmus, pre-interop, interop)
  "base/withdrawal/cannon"
  "base/withdrawal/cannon_kona"
  "base/withdrawal/permissioned"
  "isthmus/preinterop"
  "isthmus/preinterop-singlechain"
  "interop/proofs"
  "interop/proofs/fpp"
  "interop/proofs/serial"
  "interop/proofs/withdrawal"
  "interop/proofs-singlechain"
  "proofs/cannon"
  # External-network daily tests — covered by sync-test-op-node gate
  "sync_tester/sync_tester_ext_el"
  "sync_tester/sync_tester_hfs_ext"
  # Flake-shake quarantine — temporarily excluded, tracked in flake-shake gate
  "supernode/interop/activation"
  "depreqres/syncmodereqressync/elsync"
  "depreqres/reqressyncdisabled"
)

# --- Find all test packages on disk that contain actual test functions ---
disk_packages=()
while IFS= read -r dir; do
  # Only count packages with actual Test functions (not just _test.go boilerplate)
  if grep -rql '^func Test' "$dir"/*_test.go 2>/dev/null; then
    rel="${dir#"$TESTS_DIR"/}"
    disk_packages+=("$rel")
  fi
done < <(find "$TESTS_DIR" -name '*_test.go' -printf '%h\n' | sort -u)

# --- Extract packages from ci-shard-* gates in acceptance-tests.yaml ---
# Handles both exact packages and .../... wildcard packages.
shard_packages=()
in_shard=false
while IFS= read -r line; do
  if [[ "$line" =~ id:\ *ci-shard- ]]; then
    in_shard=true
    continue
  fi
  if $in_shard && [[ "$line" =~ ^[[:space:]]*-\ *id: ]]; then
    in_shard=false
    continue
  fi
  if $in_shard && [[ "$line" =~ package:.*$MODULE/ ]]; then
    pkg="${line##*"$MODULE"/}"
    pkg="${pkg%%\"*}"
    pkg="${pkg%%\'*}"
    pkg="$(echo "$pkg" | xargs)"  # trim whitespace
    shard_packages+=("$pkg")
  fi
done < "$YAML"

# --- Check coverage ---
missing=()
for disk_pkg in "${disk_packages[@]}"; do
  covered=false

  # Check explicit exclusions
  for excl in "${EXCLUDED_PACKAGES[@]}"; do
    if [[ "$disk_pkg" == "$excl" ]]; then
      covered=true
      break
    fi
  done
  $covered && continue

  # Check shard gates
  for shard_pkg in "${shard_packages[@]}"; do
    # Exact match
    if [[ "$disk_pkg" == "$shard_pkg" ]]; then
      covered=true
      break
    fi
    # Wildcard match: "foo/..." covers "foo", "foo/bar", "foo/bar/baz"
    if [[ "$shard_pkg" == *"/..." ]]; then
      prefix="${shard_pkg%/...}"
      if [[ "$disk_pkg" == "$prefix" || "$disk_pkg" == "$prefix/"* ]]; then
        covered=true
        break
      fi
    fi
  done

  if ! $covered; then
    missing+=("$disk_pkg")
  fi
done

if [[ ${#missing[@]} -gt 0 ]]; then
  echo "ERROR: The following test packages are not covered by any ci-shard-* gate"
  echo "       and are not in the exclusion list:"
  echo ""
  for pkg in "${missing[@]}"; do
    echo "  - tests/$pkg"
  done
  echo ""
  echo "To fix: add the package to a ci-shard-* gate in acceptance-tests.yaml,"
  echo "        or add it to EXCLUDED_PACKAGES in this script with a comment"
  echo "        explaining where it's covered (develop-CI, nightly, flake-shake)."
  exit 1
fi

echo "OK: All ${#disk_packages[@]} test packages are covered by ci-shard gates (${#shard_packages[@]} shard entries, ${#EXCLUDED_PACKAGES[@]} exclusions)."
