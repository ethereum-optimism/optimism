#!/usr/bin/env bash
set -euo pipefail

# latest-versions.sh - reads all remote tags from the origin repository,
# groups them by component, and then finds the latest version for each component.

# Create hash map with key/value pairs: component -> "space-separated list of versions"
declare -A component_versions

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
    if [[ -n "${component_versions[$component]:-}" ]]; then
      component_versions["$component"]+=" $version"
    else
      component_versions["$component"]="$version"
    fi
  fi
done < <(git ls-remote --tags origin)

# find_latest_version - determines the latest version from a list of versions
#
# Input: space-separated string of version numbers (e.g., "1.2.3 1.3.0-rc.1 1.2.4")
#        Each version should be in format: X.Y.Z or X.Y.Z-rc.N (with optional additional suffixes)
#        Examples: "0.4.2", "1.13.6-rc.3", "0.1.0-rc.2"
#
# Output: single version string representing the latest version
#
# Semver precedence rules applied:
#   1. Highest semantic version wins (e.g., 1.3.0 > 1.2.9)
#   2. For same base version, non-rc preferred over -rc (e.g., 1.13.6 > 1.13.6-rc.3)
#   3. Higher base version beats lower, even if -rc (e.g., 1.13.6-rc.1 > 1.13.5)
#   4. For same base version with multiple RCs, higher -rc wins (e.g., 1.5.3-rc.3 > 1.5.3-rc.1)
find_latest_version() {
  local versions="$1"

  # Convert space-separated string to array for iteration
  read -ra version_array <<< "$versions"

  # Create sortable versions that encode precedence rules
  local sortable_versions=()
  for ver in "${version_array[@]}"; do
    local base="${ver%%-rc*}"

    # Modifies the string (while preserving the original version via | separator)
    # so lexicographical sort will work
    if [[ "$ver" =~ -rc ]]; then
      local rc_suffix="${ver##*-rc}"
      # -rc versions: base.0.rc_suffix (priority 0)
      sortable_versions+=("$base.0.$rc_suffix|$ver")
    else
      # non-rc versions: base.1.0 (priority 1)
      sortable_versions+=("$base.1.0|$ver")
    fi
  done

  # Sort all versions lexicographically and get the highest one
  local latest_sortable
  latest_sortable=$(printf '%s\n' "${sortable_versions[@]}" | sort -V | tail -n1)
  echo "${latest_sortable##*|}"
}

declare -A latest_versions

# Process each component once
for component in "${!component_versions[@]}"; do
  latest_versions["$component"]=$(find_latest_version "${component_versions[$component]}")
done


# Sort components alphabetically for consistent output,
# then print in JSON format
mapfile -t sorted_components < <(printf '%s\n' "${!latest_versions[@]}" | sort)
echo "{"
for i in "${!sorted_components[@]}"; do
  component="${sorted_components[i]}"
  if [[ $i -gt 0 ]]; then
    echo ","
  fi
  printf '  "%s": "v%s"' "$component" "${latest_versions[$component]}"
done
echo ""
echo "}"
