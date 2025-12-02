#!/usr/bin/env bash
set -euo pipefail

# Computes GIT_VERSION for all OP Stack images based on git tags pointing at a commit.
# Replicates CircleCI logic exactly - each image can have different GIT_VERSION values.
# Outputs JSON mapping image names to their GIT_VERSION values.
#
# Usage:
#   GIT_COMMIT=$(git rev-parse HEAD) ./ops/scripts/compute-git-versions.sh
#
# Output format:
#   {"op-node":"v1.2.3","op-batcher":"v1.1.0",...}

GIT_COMMIT="${GIT_COMMIT:-$(git rev-parse HEAD)}"

IMAGES=(
  "op-node"
  "op-batcher"
  "op-deployer"
  "op-faucet"
  "op-program"
  "op-proposer"
  "op-challenger"
  "op-dispute-mon"
  "op-conductor"
  "da-server"
  "op-supervisor"
  "op-supernode"
  "op-test-sequencer"
  "cannon"
  "op-dripper"
  "op-interop-mon"
)

echo "Checking git tags pointing at $GIT_COMMIT:" >&2
tags_at_commit=$(git tag --points-at "$GIT_COMMIT" || true)
echo "Tags at commit: $tags_at_commit" >&2

declare -A VERSIONS

for IMAGE_NAME in "${IMAGES[@]}"; do
  # Replicate CircleCI logic exactly: filter tags by image name prefix
  filtered_tags=$(echo "$tags_at_commit" | grep "^${IMAGE_NAME}/" || true)
  echo "Filtered tags for ${IMAGE_NAME}: $filtered_tags" >&2

  if [ -z "$filtered_tags" ]; then
    VERSIONS["$IMAGE_NAME"]="untagged"
  else
    sorted_tags=$(echo "$filtered_tags" | sed "s|${IMAGE_NAME}/||" | sort -V)
    echo "Sorted tags for ${IMAGE_NAME}: $sorted_tags" >&2

    # prefer full release tag over "-rc" release candidate tag if both exist
    full_release_tag=$(echo "$sorted_tags" | grep -v -- "-rc" || true)
    if [ -z "$full_release_tag" ]; then
      VERSIONS["$IMAGE_NAME"]=$(echo "$sorted_tags" | tail -n 1)
    else
      VERSIONS["$IMAGE_NAME"]=$(echo "$full_release_tag" | tail -n 1)
    fi
  fi

  echo "GIT_VERSION for ${IMAGE_NAME}: ${VERSIONS[$IMAGE_NAME]}" >&2
done

# Output as JSON
JSON="{"
FIRST=true
for IMAGE_NAME in "${IMAGES[@]}"; do
  if [ "$FIRST" = true ]; then
    FIRST=false
  else
    JSON="${JSON},"
  fi
  VERSION="${VERSIONS[$IMAGE_NAME]}"
  JSON="${JSON}\"${IMAGE_NAME}\":\"${VERSION}\""
done
JSON="${JSON}}"

echo "$JSON"

