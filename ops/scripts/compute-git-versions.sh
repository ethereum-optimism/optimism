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

# Read image list from the single source of truth
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
IMAGES_JSON="$REPO_ROOT/.github/docker-images.json"

if [ ! -f "$IMAGES_JSON" ]; then
  echo "Error: $IMAGES_JSON not found" >&2
  exit 1
fi

mapfile -t IMAGES < <(jq -r '.images | keys[]' "$IMAGES_JSON" | sort)

echo "Checking git tags pointing at $GIT_COMMIT:" >&2
tags_at_commit=$(git tag --points-at "$GIT_COMMIT" || true)
echo "Tags at commit: $tags_at_commit" >&2

declare -A VERSIONS

for IMAGE_NAME in "${IMAGES[@]}"; do
  # Filter tags by exact image name prefix (use bash matching, not regex, to avoid injection)
  filtered_tags=""
  while IFS= read -r tag; do
    [[ -z "$tag" ]] && continue
    if [[ "$tag" == "${IMAGE_NAME}/"* ]]; then
      filtered_tags+="$tag"$'\n'
    fi
  done <<< "$tags_at_commit"
  filtered_tags="${filtered_tags%$'\n'}"
  echo "Filtered tags for ${IMAGE_NAME}: $filtered_tags" >&2

  if [ -z "$filtered_tags" ]; then
    VERSIONS["$IMAGE_NAME"]="untagged"
  else
    # Strip image name prefix using parameter expansion (not sed) to avoid metacharacter issues
    sorted_tags=""
    while IFS= read -r tag; do
      [[ -z "$tag" ]] && continue
      sorted_tags+="${tag#"${IMAGE_NAME}/"}"$'\n'
    done <<< "$filtered_tags"
    sorted_tags=$(echo "$sorted_tags" | sort -V)
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

# Output as JSON (use jq to safely encode keys/values)
JSON="{}"
for IMAGE_NAME in "${IMAGES[@]}"; do
  VERSION="${VERSIONS[$IMAGE_NAME]}"
  JSON=$(echo "$JSON" | jq -c --arg k "$IMAGE_NAME" --arg v "$VERSION" '. + {($k): $v}')
done

echo "$JSON"

