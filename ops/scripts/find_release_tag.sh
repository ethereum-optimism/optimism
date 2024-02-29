#!/bin/bash

set -euo pipefail

# Function to print usage
usage() {
    echo "Usage: $0 <commit-hash> [tag-prefix]"
    echo "  <commit-hash> : The commit hash to check."
    echo "  [tag-prefix]  : Optional. The prefix for tags to check. Default is 'op-node'."
}

# Check for at least one argument
if [ "$#" -lt 1 ]; then
    usage
    exit 1
fi

commit_hash=$1
tag_prefix=${2:-"op-node"} # Default tag prefix is "op-node"

# Get all tags containing the commit, sorted by creation date
tags=$(git tag --contains "$commit_hash" --sort=taggerdate)

# Find the first release tag with the given prefix
for tag in $tags; do
    if [[ $tag == $tag_prefix/v* ]]; then
        echo "First release tag containing commit $commit_hash: $tag"
        exit 0
    fi
done

echo "Commit $commit_hash is not in any $tag_prefix/v* release tag."
