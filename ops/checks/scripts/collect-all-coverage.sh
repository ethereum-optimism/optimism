#!/usr/bin/env bash
# Collect coverage for all Solidity test files in parallel.
# Usage: ./collect-all-coverage.sh [--parallel N] [--output-dir DIR]
#
# Defaults: 4 parallel jobs, output to ops/checks/coverage-data/

set -euo pipefail

PARALLEL=${1:-4}
OUTPUT_DIR="${2:-ops/checks/coverage-data}"
REPO_ROOT="$(git rev-parse --show-toplevel)"
CONTRACTS_DIR="$REPO_ROOT/packages/contracts-bedrock"
CHECKS_DIR="$REPO_ROOT/ops/checks"

mkdir -p "$OUTPUT_DIR"

# Find all test files
TEST_FILES=()
while IFS= read -r f; do
    TEST_FILES+=("$f")
done < <(find "$CONTRACTS_DIR/test" -name "*.t.sol" -type f | sort)
TOTAL=${#TEST_FILES[@]}

echo "Collecting coverage for $TOTAL test files ($PARALLEL parallel)..."
echo "Output: $OUTPUT_DIR"
echo ""

# Track progress
COMPLETED=0
FAILED=0

collect_one() {
    local test_file="$1"
    local rel_path="${test_file#$CONTRACTS_DIR/}"
    local safe_name=$(echo "$rel_path" | tr '/' '_' | sed 's/.t.sol/.json/')
    local output_path="$OUTPUT_DIR/$safe_name"

    if [ -f "$output_path" ]; then
        echo "  SKIP $rel_path (already collected)"
        return 0
    fi

    if cd "$CHECKS_DIR" && go run ./cmd/checks coverage collect \
        --lang solidity \
        --test "$rel_path" \
        --root "$REPO_ROOT" \
        --output "$output_path" 2>/dev/null; then
        echo "  OK   $rel_path"
        return 0
    else
        echo "  FAIL $rel_path"
        return 1
    fi
}

export -f collect_one
export CONTRACTS_DIR CHECKS_DIR REPO_ROOT OUTPUT_DIR

# Run in parallel using xargs
printf '%s\n' "${TEST_FILES[@]}" | xargs -P "$PARALLEL" -I {} bash -c 'collect_one "$@"' _ {}

# Count results
COLLECTED=$(find "$OUTPUT_DIR" -name "*.json" | wc -l | tr -d ' ')
echo ""
echo "Done: $COLLECTED/$TOTAL coverage reports collected in $OUTPUT_DIR"
