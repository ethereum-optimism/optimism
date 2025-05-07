#!/usr/bin/env bash

set -e  # Exit immediately if a command exits with a non-zero status
set -o pipefail  # Ensure failures in pipelines are detected

# Usage function
usage() {
    echo "Usage: $0 <vm-profile> <baseline-report>"
    echo "Valid profiles: cannon-singlethreaded-32, cannon-multithreaded-64"
    exit 1
}

# Validate input
if [[ $# -ne 2 ]]; then
    usage
fi

VM_PROFILE=$1
BASELINE_REPORT=$2

if [[ "$VM_PROFILE" != "cannon-singlethreaded-32" && "$VM_PROFILE" != "cannon-multithreaded-64" ]]; then
    echo "Error: Invalid vm-profile '$VM_PROFILE'"
    usage
fi

ANALYZER_BIN="vm-compat"

if ! command -v llvm-objdump &>/dev/null; then
    echo "❌ Error: llvm-objdump is required but not found in \$PATH"
    echo "Please install it using one of the following commands, based on your OS:"
    echo "  Ubuntu/Debian: sudo apt-get install -y llvm"
    echo "  Fedora: sudo dnf install -y llvm"
    echo "  Arch Linux: sudo pacman -Sy llvm"
    echo "  macOS (Homebrew): brew install llvm"
    exit 1
fi

echo "✅ llvm-objdump found at $(which llvm-objdump)"

# Check if 'vm-compat' is installed
if ! command -v vm-compat &>/dev/null; then
    echo "❌ Error: 'vm-compat' is required but not found in \$PATH"
    echo "Please install it using:"
    echo "  mise use -g ubi:ChainSafe/vm-compat@1.1.0"
    echo "Or manually download from:"
    echo "  https://github.com/ChainSafe/vm-compat/releases"
    exit 1
fi

echo "✅ vm-compat found at $(which vm-compat)"

# Run the analyzer
echo "Running analysis with VM profile: $VM_PROFILE using baseline report: $BASELINE_REPORT..."
OUTPUT_FILE=$(mktemp)

"$ANALYZER_BIN" analyze --with-trace=true --format=json --vm-profile "$VM_PROFILE" --baseline-report "$BASELINE_REPORT" --report-output-path "$OUTPUT_FILE" ./client/cmd/main.go

# Check if JSON output contains any issues
ISSUE_COUNT=$(jq 'length' "$OUTPUT_FILE")

if [ "$ISSUE_COUNT" -gt 0 ]; then
    echo "❌ Analysis found $ISSUE_COUNT issues!"
    cat "$OUTPUT_FILE"
    rm -f "$OUTPUT_FILE"
    exit 1
else
    echo "✅ No issues found."
    rm -f "$OUTPUT_FILE"
    exit 0
fi
