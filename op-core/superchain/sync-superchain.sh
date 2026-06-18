#!/usr/bin/env bash

# Syncs the superchain-registry configs into op-core/superchain.
#
# Reads from the superchain-registry git submodule at the repo root — the single
# canonical commit pin. Initialize it first with
# `just update-superchain-registry-submodule` (the root `just sync-superchain`
# target does this for you). The resulting zip is gitignored; the committed
# .sha256 pins it. Skips work if the on-disk zip already matches the submodule's
# commit.

set -euo pipefail

# Constants
SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &>/dev/null && pwd)
REPO_ROOT=$(git -C "$SCRIPT_DIR" rev-parse --show-toplevel)
SR_DIR="$REPO_ROOT/superchain-registry"
ZIP="$SCRIPT_DIR/superchain-configs.zip"

if [[ ! -d "$SR_DIR/superchain" ]]; then
  echo "[sync-superchain] superchain-registry submodule not initialized; initializing..." >&2
  (cd "$REPO_ROOT" && just update-superchain-registry-submodule)
fi

REGISTRY_COMMIT=$(git -C "$SR_DIR" rev-parse HEAD)

# Short-circuit: if the existing zip already pins the same commit, do nothing.
#
# NOTE: this only checks the SR commit, not whether THIS script has changed
# since the zip was built. If you've edited sync-superchain.sh and want to
# force regeneration, delete superchain-configs.zip first:
#
#     rm op-core/superchain/superchain-configs.zip && just sync-superchain
#
# CI is the safety net for un-rebuilt script changes — it always rebuilds
# (cache key includes the script's hash) and fails on a .sha256 diff.
if [[ -f "$ZIP" ]]; then
  existing=$(unzip -p "$ZIP" COMMIT 2>/dev/null | tr -d '[:space:]' || true)
  if [[ "$existing" == "$REGISTRY_COMMIT" ]]; then
    echo "[sync-superchain] up to date at commit $REGISTRY_COMMIT"
    exit 0
  fi
fi

workdir=$(mktemp -d)

echo "Copying configs from superchain-registry submodule at $REGISTRY_COMMIT..."
cp -r "$SR_DIR/superchain/configs" "$workdir/configs"
cp -r "$SR_DIR/superchain/extra/genesis" "$workdir/genesis"
cp -r "$SR_DIR/superchain/extra/dictionary" "$workdir/dictionary"

cd "$workdir"
echo "Using $workdir as workdir..."

# Create a simple mapping of chain id -> config name to make looking up chains by their ID easier.
echo "Generating index of configs..."

echo "{}" >chains.json

# Function to process each network directory
process_network_dir() {
    local network_dir="$1"
    local network_name
    network_name=$(basename "$network_dir")

    echo "Processing chains in $network_name superchain..."

    # Find all TOML files in the network directory
    find "$network_dir" -type f -name "*.toml" | LC_ALL=C sort | while read -r toml_file; do
        if [[ "$toml_file" == "configs/$network_name/superchain.toml" ]]; then
            continue
        fi

        echo "Processing $toml_file..."
        # Extract chain_id from TOML file using yq
        chain_id=$(yq -r '.chain_id' "$toml_file")
        chain_name="$(basename "${toml_file%.*}")"

        if [[ -z "$chain_id"
              # Boba Sepolia
              || "$chain_id" -eq 28882
              # Boba Mainnet
              || "$chain_id" -eq 288
              # Celo Mainnet: non-standard genesis format (forked from Ethereum, then converted to L2)
              || "$chain_id" -eq 42220 ]];
        then
            echo "Skipping $network_name/$chain_name ($chain_id)"
            rm "$toml_file"
            rm -f "genesis/$network_name/$chain_name.json.zst"
            continue
        fi

        # Create JSON object for this config
        config_json=$(jq -n \
            --arg name "$chain_name" \
            --arg network "$network_name" \
            '{
                "name": $name,
                "network": $network
            }')

        # Add this config to the result JSON using the chain_id as the key
        jq --argjson config "$config_json" \
            --arg chain_id "$chain_id" \
            '. + {($chain_id): $config}' chains.json >temp.json
        mv temp.json chains.json
    done
}

# Process each network directory in configs
for network_dir in configs/*; do
    if [ -d "$network_dir" ]; then
        process_network_dir "$network_dir"
    fi
done

# Archive the genesis configs as a ZIP file. ZIP is used since it can be efficiently used as a filesystem.
echo "Archiving configs..."
echo "$REGISTRY_COMMIT" >COMMIT
# We need to normalize the lastmod dates and permissions to ensure the ZIP file is deterministic.
find . -exec touch -t 198001010000.00 {} +
chmod -R 755 ./*
files=$(find . -type f | LC_ALL=C sort)
echo -n "$files" | xargs zip -9 -oX --quiet superchain-configs.zip
zipinfo superchain-configs.zip
mv superchain-configs.zip "$SCRIPT_DIR/superchain-configs.zip"

# Persist the bundle's SHA256 alongside it. The hash is committed to git
# (the zip itself isn't); this gives strong consistency: any drift between
# what a developer/CI builds and what was approved in review surfaces as a
# .sha256 diff.
sha256sum "$SCRIPT_DIR/superchain-configs.zip" \
  | awk '{print $1 "  superchain-configs.zip"}' \
  > "$SCRIPT_DIR/superchain-configs.zip.sha256"

echo "Cleaning up..."
rm -rf "$workdir"

echo "Done."
