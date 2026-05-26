#!/usr/bin/env bash

# This script is used to sync superchain configs in the registry with op-core/superchain.
#
# The resulting zip is gitignored. Run this script before building the Go workspace.
# Skips work if the on-disk zip already matches the pinned commit.

set -euo pipefail

# Constants
SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &>/dev/null && pwd)
REGISTRY_COMMIT=$(cat "$SCRIPT_DIR/superchain-registry-commit.txt")
ZIP="$SCRIPT_DIR/superchain-configs.zip"

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

repodir=$(mktemp -d)
workdir=$(mktemp -d)

# Clone the registry
echo "Cloning SR..."
cd "$repodir"
git clone --no-checkout --depth 1 --shallow-submodules https://github.com/ethereum-optimism/superchain-registry.git
cd "$repodir/superchain-registry"
git fetch --depth 1 origin "$REGISTRY_COMMIT"
git checkout "$REGISTRY_COMMIT"

echo "Copying configs..."
cp -r superchain/configs "$workdir/configs"
cp -r superchain/extra/genesis "$workdir/genesis"
cp -r superchain/extra/dictionary "$workdir/dictionary"

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
rm -rf "$repodir"
rm -rf "$workdir"

echo "Done."
