#!/usr/bin/env bash

set -eo pipefail

DIRECTORY=$1
if [ -z "$DIRECTORY" ]; then
  echo "Usage: $0 <directory>"
  exit 1
fi

# - We list all the tracked files in the specified directory
# - We blame the files (using porcelain for easy consumption)
# - We take the author emails out of the blame output
# - We replace the <brackets around the emails>
# - We sort the emails and remove duplicates
git ls-files -z "$DIRECTORY" \
    | xargs -0n1 git --no-pager blame --porcelain \
    | sed -n -e '/^author-mail /s/^author-mail //p' \
    | sed -e 's/[<>]//g' \
    | sort | uniq