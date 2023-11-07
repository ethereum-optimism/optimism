#!/usr/bin/env bash

set -euo pipefail

FILE1=$1
PATTERN1=$2
FILE2=$3
PATTERN2=$4

# shellcheck disable=SC2016
SCRIPT='
BEGIN {
    in_comment = 0;
    matches = 0;
}

/^ *\/\*/ {
    in_comment = 1;
}

in_comment && /\*\// {
    in_comment = 0;
    next;
}

!in_comment && !/^ *\/\// && $0 ~ PATTERN {
    matches++;
    matched_line = $0;
}

END {
    if (matches == 1) {
        print matched_line;
    } else if (matches > 1) {
        print "Multiple matches found. Exiting.";
        exit 1;
    } else {
        print "No matches found. Exiting.";
        exit 1;
    }
}'

VALUE1_MATCH=$(echo "$SCRIPT" | awk -v PATTERN="$PATTERN1" -f- "$FILE1")
VALUE1=$(echo "$VALUE1_MATCH" | awk -F'=' '{print $2}' | tr -d ' ;')
echo "Value from File 1: $VALUE1"

VALUE2_MATCH=$(echo "$SCRIPT" | awk -v PATTERN="$PATTERN2" -f- "$FILE2")
VALUE2=$(echo "$VALUE2_MATCH" | awk -F'=' '{print $2}' | tr -d ' ;')
echo "Value from File 2: $VALUE2"

if [ "$VALUE1" != "$VALUE2" ]; then
  echo "Error: Values from file1 ($VALUE1) and file2 ($VALUE2) don't match."
  exit 1
fi

echo "Values match!"

