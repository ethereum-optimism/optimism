#!/usr/bin/env bash

set -eo pipefail

DIRECTORY=$1
if [ -z "$DIRECTORY" ]; then
  echo "Usage: $0 <directory>"
  exit 1
fi

# Extract authorship data of target directory from git history
echo -n "dt,author,commit"
cd "$DIRECTORY"
git ls-files | while read -r file; do
  git --no-pager log --pretty=format:"%ad,%ae,%H%n" --date=format:"%Y-%m-%d %H:%M:%S" -- "$file" 2>/dev/null
done | sort | uniq
