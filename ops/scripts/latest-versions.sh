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

declare -A component_versions # hash map: component -> "space-separated versions"
declare -A latest_versions    # hash map: component -> latest version
declare -A stable_versions    # hash map: component -> stable version

# Collect all remote tags once and group by component in `component_versions`
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
    if [[ -n "${component_versions[$component]:-}" ]]; then
      component_versions["$component"]+=" $version"
    else
      component_versions["$component"]="$version"
    fi
  fi
done < <(git ls-remote --tags origin)

# Process each component once and store results in `latest_versions`, `stable_versions`
for component in "${!component_versions[@]}"; do
  result=$(find_latest_versions "${component_versions[$component]}")
  latest_versions["$component"]="${result%|*}"  # Everything before pipe delimiter
  stable_versions["$component"]="${result#*|}"  # Everything after pipe delimiter
done

# Sort components alphabetically for consistent output
mapfile -t sorted_components < <(printf '%s\n' "${!latest_versions[@]}" | sort)

# Print results in JSON format
echo "{"
for i in "${!sorted_components[@]}"; do
  component="${sorted_components[i]}"
  print_component_json "$component" \
    "${stable_versions[$component]}" \
    "${latest_versions[$component]}" \
    "$([ "$i" -eq 0 ] && echo true || echo false)"
done
echo ""
echo "}"
