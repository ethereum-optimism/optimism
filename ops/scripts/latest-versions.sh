#!/usr/bin/env bash
set -euo pipefail

# latest-versions.sh - reads all remote tags from the origin repository,
# groups them by component, and then finds the latest version for each component.

########################################################
####   FUNCTIONS                                    ####
########################################################

# find_latest_versions - finds both latest and stable versions in one pass
#
# Input: space-separated string of version numbers (e.g., "1.2.3 1.3.0-rc.1 1.2.4")
# Output: single line in format "latest_version|stable_version"
#         where stable_version is empty if no stable (vX.Y.Z only) versions exist
#
# Latest: Uses custom precedence rules (non-suffix beats suffix with same base version)
#   1. Highest semantic version wins (e.g., 1.3.0 > 1.2.9)
#   2. For same base version, non-suffixed preferred over suffixed (e.g., 1.13.6 > 1.13.6-rc.3)
#   3. Higher base version beats lower, even if suffixed (e.g., 1.13.6-rc.1 > 1.13.5)
#   4. For same base version with multiple suffixes, higher lexicographical suffix wins (e.g., 1.5.3-rc.3 > 1.5.3-rc.1)
# Stable: Highest pure X.Y.Z format (no suffixes)
find_latest_versions() {
  local versions="$1"

  # Convert space-separated string to array for iteration
  read -ra version_array <<< "$versions"

  # Create sortable versions for both latest and stable
  local sortable_versions=()
  local stable_sortable_versions=()

  for ver in "${version_array[@]}"; do
    # Extract base version (everything before first '-' suffix)
    local base="${ver%%-*}"

    # Modifies the string (while preserving the original version via | separator)
    # so lexicographical sort will work
    if [[ "$ver" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
      # stable (non-suffixed) versions: base.1.0 (priority 1, higher than any suffix)
      local sortable_ver="$base.1.0|$ver"
      sortable_versions+=("$sortable_ver")
      stable_sortable_versions+=("$sortable_ver")
    else
      # suffixed versions: base.0.suffix (priority 0, lower than stable version)
      local suffix="${ver#*-}"
      sortable_versions+=("$base.0.$suffix|$ver")
    fi
  done

  # Find highest latest version using lexicographical sort
  local latest_sortable
  latest_sortable=$(printf '%s\n' "${sortable_versions[@]}" | sort -V | tail -n1)
  local latest="${latest_sortable##*|}"

  # Find highest stable version using lexicographical sort
  local stable=""
  if [[ ${#stable_sortable_versions[@]} -gt 0 ]]; then
    local stable_sortable
    stable_sortable=$(printf '%s\n' "${stable_sortable_versions[@]}" | sort -V | tail -n1)
    stable="${stable_sortable##*|}"
  fi

  # Output in format "latest_version|stable_version"
  echo "$latest|$stable"
}

# Helper function to print component JSON
# Output example:
#   "component": {
#     "stable": "v1.0.0" (empty string if no stable version),
#     "latest": "v1.0.0"
#   }
print_component_json() {
  local component="$1"
  local stable_ver="$2"
  local latest_ver="$3"
  local is_first="$4"

  [[ "$is_first" != "true" ]] && echo ","

  local stable_field='""'
  [[ -n "$stable_ver" ]] && stable_field="\"v$stable_ver\""

  printf '  "%s": {\n    "stable": %s,\n    "latest": "v%s"\n  }' \
    "$component" "$stable_field" "$latest_ver"
}

########################################################
####   MAIN                                         ####
########################################################

# Create temporary files to simulate associative arrays
temp_dir=$(mktemp -d)
component_versions_file="$temp_dir/component_versions"
latest_versions_file="$temp_dir/latest_versions"
stable_versions_file="$temp_dir/stable_versions"

# Clean up temp files on exit
trap 'rm -rf "$temp_dir"' EXIT

# Initialize temporary files
touch "$component_versions_file" "$latest_versions_file" "$stable_versions_file"

# Helper functions to simulate associative array operations
get_component_versions() {
  local component="$1"
  grep "^$component:" "$component_versions_file" 2>/dev/null | cut -d: -f2- || true
}

set_component_versions() {
  local component="$1"
  local versions="$2"
  # Remove existing entry and add new one
  grep -v "^$component:" "$component_versions_file" > "$component_versions_file.tmp" 2>/dev/null || true
  echo "$component:$versions" >> "$component_versions_file.tmp"
  mv "$component_versions_file.tmp" "$component_versions_file"
}

set_latest_version() {
  local component="$1"
  local version="$2"
  echo "$component:$version" >> "$latest_versions_file"
}

set_stable_version() {
  local component="$1"
  local version="$2"
  echo "$component:$version" >> "$stable_versions_file"
}

get_latest_version() {
  local component="$1"
  grep "^$component:" "$latest_versions_file" 2>/dev/null | cut -d: -f2- || true
}

get_stable_version() {
  local component="$1"
  grep "^$component:" "$stable_versions_file" 2>/dev/null | cut -d: -f2- || true
}

# Collect all remote tags once and group by component
while IFS= read -r tag; do
  # Skip empty lines
  [[ -z "$tag" ]] && continue

  # Skip ^{} annotated tags completely
  [[ "$tag" == *"^{}" ]] && continue

  # git ls-remote output format: "<commit_hash> refs/tags/<tagname>"
  # Only process tags that match our refs/tags/<component>/v<version> pattern
  if [[ "$tag" =~ refs/tags/([a-zA-Z0-9_-]+)/v(.+)$ ]]; then
    component="${BASH_REMATCH[1]}"
    version="${BASH_REMATCH[2]}"

    # Append version to component's list (space-separated)
    existing_versions=$(get_component_versions "$component")
    if [[ -n "$existing_versions" ]]; then
      set_component_versions "$component" "$existing_versions $version"
    else
      set_component_versions "$component" "$version"
    fi
  fi
done < <(git ls-remote --tags origin)

# Process each component once and store results
while IFS=: read -r component versions; do
  [[ -z "$component" ]] && continue
  result=$(find_latest_versions "$versions")
  latest_version="${result%|*}"  # Everything before pipe delimiter
  stable_version="${result#*|}"  # Everything after pipe delimiter
  set_latest_version "$component" "$latest_version"
  set_stable_version "$component" "$stable_version"
done < "$component_versions_file"

# Get sorted list of components
sorted_components=$(cut -d: -f1 "$latest_versions_file" | sort)

# Print results in JSON format
echo "{"
first=true
for component in $sorted_components; do
  latest_ver=$(get_latest_version "$component")
  stable_ver=$(get_stable_version "$component")
  print_component_json "$component" "$stable_ver" "$latest_ver" "$first"
  first=false
done
echo ""
echo "}"
