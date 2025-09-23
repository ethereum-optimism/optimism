#!/bin/bash

# Script to check comment lengths in Solidity files
# Checks all .sol files in src/ directory except those in src/vendor
# Reports files and line numbers where comments exceed 100 characters

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Change to the directory containing this script
cd "$(dirname "$0")/../.."

# Check if src directory exists
if [ ! -d "src" ]; then
    echo -e "${RED}Error: src directory not found${NC}"
    exit 1
fi

echo "Checking comment lengths in src/ directory (excluding src/vendor)..."
echo "Maximum allowed comment length: 100 characters"
echo ""

errors_found=0

# Find all .sol files in src/ excluding src/vendor
while IFS= read -r -d '' file; do
    # Skip files in vendor directory
    if [[ "$file" == src/vendor* ]]; then
        continue
    fi

    line_number=0
    file_has_errors=false

    # Read file line by line
    while IFS= read -r line; do
        ((line_number++))

        # Check for single-line comments (//)
        if [[ "$line" =~ ^[[:space:]]*// ]]; then
            # Extract the comment part (everything after //)
            comment="${line#*//}"
            # Remove leading and trailing whitespace
            comment="$(echo "$comment" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')"

            if [ ${#comment} -gt 100 ]; then
                if [ "$file_has_errors" = false ]; then
                    echo -e "${RED}$file:${NC}"
                    file_has_errors=true
                    ((errors_found++))
                fi
                echo -e "  ${YELLOW}Line $line_number:${NC} Comment too long (${#comment} characters)"
                echo -e "  ${YELLOW}Content:${NC} $comment"
                echo ""
            fi
        fi

        # Check for multi-line comments (/* ... */)
        # Handle single-line multi-line comments
        if [[ "$line" =~ ^[[:space:]]*\/\*.*\*\/[[:space:]]*$ ]]; then
            # Extract content between /* and */
            comment="${line#*/*}"
            comment="${comment%*/*}"
            # Remove leading and trailing whitespace
            comment="$(echo "$comment" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')"

            if [ ${#comment} -gt 100 ]; then
                if [ "$file_has_errors" = false ]; then
                    echo -e "${RED}$file:${NC}"
                    file_has_errors=true
                    ((errors_found++))
                fi
                echo -e "  ${YELLOW}Line $line_number:${NC} Comment too long (${#comment} characters)"
                echo -e "  ${YELLOW}Content:${NC} $comment"
                echo ""
            fi
        fi

        # Check for block comments starting with /**
        if [[ "$line" =~ ^[[:space:]]*\/\*\* ]]; then
            # Extract content after /**
            comment="${line#*/**}"
            # Remove trailing */ if present
            comment="${comment%*/*}"
            # Remove leading and trailing whitespace
            comment="$(echo "$comment" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')"

            if [ ${#comment} -gt 100 ]; then
                if [ "$file_has_errors" = false ]; then
                    echo -e "${RED}$file:${NC}"
                    file_has_errors=true
                    ((errors_found++))
                fi
                echo -e "  ${YELLOW}Line $line_number:${NC} Comment too long (${#comment} characters)"
                echo -e "  ${YELLOW}Content:${NC} $comment"
                echo ""
            fi
        fi

    done < "$file"

done < <(find src -name "*.sol" -type f -print0)

echo "Comment length check completed."

if [ $errors_found -eq 0 ]; then
    echo -e "${GREEN}✓ No comment length violations found.${NC}"
    exit 0
else
    echo -e "${RED}✗ Found comment length violations in $errors_found file(s).${NC}"
    echo "Please shorten comments to 100 characters or less."
    exit 1
fi
