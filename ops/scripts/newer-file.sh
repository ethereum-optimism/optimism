#!/usr/bin/env bash

# Returns 0 if the first file is newer than the second file
# Works on files or directories

if [[ ! -e "$1" ]]; then exit 1; fi
if [[ ! -e "$2" ]]; then exit 1; fi

FILE_1_AGE=$(date +%s%N --reference "$1")
FILE_2_AGE=$(date +%s%N --reference "$2")
if (("$FILE_1_AGE" > "$FILE_2_AGE")); then
  exit 0
fi
exit 1
