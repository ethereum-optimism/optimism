#!/usr/bin/env bash
set -euo pipefail

readonly fixture="op-reth-sdm-fixture"
readonly manifest="rust/op-reth/crates/sdm-fixture-node/Cargo.toml"

if ! grep -Eq '^publish = false$' "$manifest"; then
  echo "$fixture must remain publish = false" >&2
  exit 1
fi

if awk '
  /^default-members = \[/ { in_defaults = 1 }
  in_defaults && /op-reth-sdm-fixture|sdm-fixture-node/ { found = 1 }
  in_defaults && /^\]/ { in_defaults = 0 }
  END { exit found ? 0 : 1 }
' rust/Cargo.toml; then
  echo "$fixture must not be a workspace default-member" >&2
  exit 1
fi

production_paths=(
  docker-bake.hcl
  .github/images.json
  .github/images.apko.json
  apko
  melange
  ops
  k8s
  devnets
)
existing_paths=()
for path in "${production_paths[@]}"; do
  if [[ -e "$path" ]]; then
    existing_paths+=("$path")
  fi
done

# Guard against an empty list: `grep -R` with no path argument would search the whole repo and match
# the fixture's own crate, failing for the wrong reason.
if [[ ${#existing_paths[@]} -eq 0 ]]; then
  echo "no production publication surfaces found to scan; check this guard's path list" >&2
  exit 1
fi

# `grep -R -F`, not ripgrep: this guard also runs in images that do not ship rg.
if grep -R -n -F "$fixture" "${existing_paths[@]}"; then
  echo "$fixture must not appear in production image, package, deployment, or release surfaces" >&2
  exit 1
fi

echo "$fixture is excluded from production publication and deployment surfaces"
