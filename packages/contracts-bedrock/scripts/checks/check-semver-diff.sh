#!/usr/bin/env bash
set -euo pipefail

# Grab the directory of the contracts-bedrock package.
SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &> /dev/null && pwd)

# Load semver-utils.
# shellcheck source=/dev/null
source "$SCRIPT_DIR/utils/semver-utils.sh"

# Determine the target branch.
# shellcheck source=/dev/null
source "$SCRIPT_DIR/../ops/get-target-branch.sh"

# Path to semver-lock.json.
SEMVER_LOCK="snapshots/semver-lock.json"

# Define excluded contracts.
EXCLUDED_CONTRACTS=(
)

github_token() {
  if [ -n "${GH_TOKEN:-}" ]; then
    printf '%s' "$GH_TOKEN"
  elif [ -n "${GITHUB_TOKEN:-}" ]; then
    printf '%s' "$GITHUB_TOKEN"
  elif [ -n "${GHTOKEN:-}" ]; then
    printf '%s' "$GHTOKEN"
  fi
}

github_repo() {
  local owner="${CIRCLE_PROJECT_USERNAME:-}"
  local repo="${CIRCLE_PROJECT_REPONAME:-}"
  local origin_url

  if [ -z "$owner" ] || [ -z "$repo" ]; then
    origin_url="$(git config --get remote.origin.url || true)"
    origin_url="${origin_url%.git}"
    origin_url="${origin_url#git@github.com:}"
    origin_url="${origin_url#https://github.com/}"
    owner="${origin_url%%/*}"
    repo="${origin_url#*/}"
  fi

  if [ -n "$owner" ] && [ -n "$repo" ] && [ "$owner" != "$repo" ]; then
    printf '%s/%s' "$owner" "$repo"
  fi
}

fetch_upstream_file_from_github() {
  local path="$1"
  local output="$2"
  local token
  local repo
  local url
  local curl_args=(-fsSL -H "Accept: application/vnd.github.raw")

  token="$(github_token)"
  repo="$(github_repo)"
  if [ -z "$token" ] || [ -z "$repo" ]; then
    return 1
  fi

  curl_args+=(-H "Authorization: Bearer ${token}")
  url="https://api.github.com/repos/${repo}/contents/${path}?ref=${TARGET_BRANCH}"
  curl "${curl_args[@]}" "$url" > "$output"
}

get_upstream_file() {
  local path="$1"
  local output="$2"

  if fetch_upstream_file_from_github "$path" "$output" 2> /dev/null; then
    return 0
  fi

  git show "$UPSTREAM_REF":"$path" > "$output" 2> /dev/null
}

# Helper function to check if a contract is excluded.
is_excluded() {
  local contract="$1"
  for excluded in "${EXCLUDED_CONTRACTS[@]}"; do
    if [[ "$contract" == "$excluded" ]]; then
      return 0
    fi
  done
  return 1
}

# Create a temporary directory.
temp_dir=$(mktemp -d)
trap 'rm -rf "$temp_dir"' EXIT

# Exit early if semver-lock.json has not changed.
UPSTREAM_REF="origin/${TARGET_BRANCH}"
if ! git rev-parse --verify --quiet "$UPSTREAM_REF" > /dev/null; then
  echo "❌ Error: Could not find upstream ref $UPSTREAM_REF"
  exit 1
fi

changed_files="$temp_dir/changed_files.txt"
git diff --relative "$UPSTREAM_REF"...HEAD --name-only > "$changed_files"
git diff --relative --name-only >> "$changed_files"
git diff --relative --cached --name-only >> "$changed_files"

if ! grep -qx "$SEMVER_LOCK" "$changed_files"; then
  echo "No changes detected in semver-lock.json"
  exit 0
fi

# Get the upstream semver-lock.json.
if ! get_upstream_file packages/contracts-bedrock/snapshots/semver-lock.json "$temp_dir/upstream_semver_lock.json"; then
  echo "❌ Error: Could not find semver-lock.json in the snapshots/ directory of $TARGET_BRANCH branch"
  exit 1
fi

# Copy the local semver-lock.json.
cp "$SEMVER_LOCK" "$temp_dir/local_semver_lock.json"

# Get the changed contracts.
changed_contracts=$(jq -r '
    def changes:
        to_entries as $local
        | input as $upstream
        | $local | map(
            select(
                .key as $key
                | .value != $upstream[$key]
            )
        ) | map(.key | split(":")[0]);
    changes[]
' "$temp_dir/local_semver_lock.json" "$temp_dir/upstream_semver_lock.json")

# Flag to track if any errors are detected.
has_errors=false

# Check each changed contract for a semver version change.
for contract in $changed_contracts; do
  # Skip excluded contracts.
  if is_excluded "$contract"; then
    continue
  fi

  # Check if the contract file exists.
  if [ ! -f "$contract" ]; then
    echo "❌ Error: Contract file $contract not found"
    has_errors=true
    continue
  fi

  # Extract the old and new source files.
  old_source_file="$temp_dir/old_${contract##*/}"
  new_source_file="$temp_dir/new_${contract##*/}"
  get_upstream_file packages/contracts-bedrock/"$contract" "$old_source_file" || true
  cp "$contract" "$new_source_file"

  # Extract the old and new versions.
  old_version=$(extract_version "$old_source_file" 2> /dev/null || echo "N/A")
  new_version=$(extract_version "$new_source_file" 2> /dev/null || echo "N/A")

  # Check if the versions were extracted successfully.
  if [ "$old_version" = "N/A" ] || [ "$new_version" = "N/A" ]; then
    echo "❌ Error: unable to extract version for $contract"
    echo "          this is probably a bug in check-semver-diff.sh"
    echo "          please report or fix the issue if possible"
    has_errors=true
  fi

  # TODO: Use an existing semver comparison function since this will only
  # check if the version has changed at all and not that the version has
  # increased properly.
  # Check if the version changed.
  if [ "$old_version" = "$new_version" ]; then
    echo "❌ Error: $contract has changes in semver-lock.json but no version change"
    echo "   Old version: $old_version"
    echo "   New version: $new_version"
    has_errors=true
  else
    echo "✅ $contract: version changed from $old_version to $new_version"
  fi
done

# Exit with error if any issues were found.
if [ "$has_errors" = true ]; then
  exit 1
fi
