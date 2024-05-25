#!/usr/bin/env bash

# Returns 0 if the first file is newer than the second file
# Works on files or directories

if [[ ! -e "$1" ]]; then exit 1; fi
if [[ ! -e "$2" ]]; then exit 1; fi

if uname | grep -q "Darwin"; then
    MOD_TIME_FMT="-f %m"
else
    MOD_TIME_FMT="-c %Y"
fi

FILE_1_AGE=$(stat "$MOD_TIME_FMT" "$1")
FILE_2_AGE=$(stat "$MOD_TIME_FMT" "$2")

if [ "$FILE_1_AGE" -gt "$FILE_2_AGE" ]; then
  exit 0
fi

exit 1