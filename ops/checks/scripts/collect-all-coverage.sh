#!/usr/bin/env bash
# Collect coverage for all test files across all languages.
# Usage: ./collect-all-coverage.sh [PARALLEL] [OUTPUT_DIR] [LANG]
#
# LANG: "all" (default), "solidity", "go", "rust"
# Defaults: 4 parallel jobs, output to ops/checks/coverage-data/

set -euo pipefail

PARALLEL=${1:-4}
OUTPUT_DIR="${2:-ops/checks/coverage-data}"
LANG_FILTER="${3:-all}"
REPO_ROOT="$(git rev-parse --show-toplevel)"
CONTRACTS_DIR="$REPO_ROOT/packages/contracts-bedrock"
CHECKS_DIR="$REPO_ROOT/ops/checks"

mkdir -p "$OUTPUT_DIR"

collect_one() {
    local lang="$1"
    local test_path="$2"
    local safe_name="$3"
    local output_path="$OUTPUT_DIR/$safe_name"

    if [ -f "$output_path" ]; then
        echo "  SKIP [$lang] $test_path"
        return 0
    fi

    if cd "$CHECKS_DIR" && go run ./cmd/checks coverage collect \
        --lang "$lang" \
        --test "$test_path" \
        --root "$REPO_ROOT" \
        --output "$output_path" 2>/dev/null; then
        echo "  OK   [$lang] $test_path"
        return 0
    else
        echo "  FAIL [$lang] $test_path"
        return 1
    fi
}

export -f collect_one
export CONTRACTS_DIR CHECKS_DIR REPO_ROOT OUTPUT_DIR

# === Solidity ===
if [ "$LANG_FILTER" = "all" ] || [ "$LANG_FILTER" = "solidity" ]; then
    SOL_FILES=()
    while IFS= read -r f; do
        SOL_FILES+=("$f")
    done < <(find "$CONTRACTS_DIR/test" -name "*.t.sol" -type f 2>/dev/null | sort)

    echo "Solidity: ${#SOL_FILES[@]} test files ($PARALLEL parallel)"
    for f in "${SOL_FILES[@]}"; do
        rel_path="${f#$CONTRACTS_DIR/}"
        safe_name="sol_$(echo "$rel_path" | tr '/' '_' | sed 's/.t.sol/.json/')"
        echo "solidity $rel_path $safe_name"
    done | xargs -P "$PARALLEL" -L1 bash -c 'collect_one "$@"' _
    echo ""
fi

# === Go ===
if [ "$LANG_FILTER" = "all" ] || [ "$LANG_FILTER" = "go" ]; then
    # Find Go packages that have test files
    GO_PKGS=()
    while IFS= read -r pkg; do
        GO_PKGS+=("$pkg")
    done < <(cd "$REPO_ROOT" && go list ./... 2>/dev/null | grep -v vendor | sort)

    # Filter to packages that actually have _test.go files
    GO_TEST_PKGS=()
    for pkg in "${GO_PKGS[@]}"; do
        pkg_dir="${pkg#github.com/ethereum-optimism/optimism/}"
        if ls "$REPO_ROOT/$pkg_dir"/*_test.go >/dev/null 2>&1; then
            GO_TEST_PKGS+=("$pkg")
        fi
    done

    echo "Go: ${#GO_TEST_PKGS[@]} test packages ($PARALLEL parallel)"
    for pkg in "${GO_TEST_PKGS[@]}"; do
        # Use the short package path for the safe name
        short="${pkg#github.com/ethereum-optimism/optimism/}"
        safe_name="go_$(echo "$short" | tr '/' '_').json"
        echo "go ./${short}/... $safe_name"
    done | xargs -P "$PARALLEL" -L1 bash -c 'collect_one "$@"' _
    echo ""
fi

# === Rust ===
if [ "$LANG_FILTER" = "all" ] || [ "$LANG_FILTER" = "rust" ]; then
    RUST_DIR="$REPO_ROOT/rust"
    if [ -d "$RUST_DIR" ] && command -v cargo-llvm-cov >/dev/null 2>&1; then
        RUST_CRATES=()
        while IFS= read -r crate; do
            RUST_CRATES+=("$crate")
        done < <(cd "$RUST_DIR" && cargo metadata --no-deps --format-version 1 2>/dev/null \
            | python3 -c "import json,sys; [print(p['name']) for p in json.load(sys.stdin)['packages']]" 2>/dev/null | sort)

        echo "Rust: ${#RUST_CRATES[@]} crates ($PARALLEL parallel)"
        for crate in "${RUST_CRATES[@]}"; do
            safe_name="rs_${crate}.json"
            echo "rust $crate $safe_name"
        done | xargs -P "$PARALLEL" -L1 bash -c 'collect_one "$@"' _
        echo ""
    else
        echo "Rust: skipped (no rust/ dir or cargo-llvm-cov not installed)"
    fi
fi

# Summary
COLLECTED=$(find "$OUTPUT_DIR" -name "*.json" 2>/dev/null | wc -l | tr -d ' ')
echo "Total: $COLLECTED coverage reports in $OUTPUT_DIR"
