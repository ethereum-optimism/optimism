#!/usr/bin/env bash

# Builds op-core/superchain/superchain-configs.zip from the superchain-registry
# submodule at packages/contracts-bedrock/lib/superchain-registry.
#
# The submodule's recorded commit is the canonical pin. To bump the registry:
#   1. cd packages/contracts-bedrock/lib/superchain-registry && git checkout <new>
#   2. cd back and run `just sync-superchain`
#   3. commit the submodule pointer
#
# The zip is gitignored. CI rebuilds it on every job that needs it.
#
# Skips work if the on-disk zip already pins the same commit as the submodule.

set -euo pipefail

# Constants
SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &>/dev/null && pwd)
REPO_ROOT=$(cd "$SCRIPT_DIR/../.." && pwd)
SUBMODULE_PATH="$REPO_ROOT/packages/contracts-bedrock/lib/superchain-registry"
ZIP="$SCRIPT_DIR/superchain-configs.zip"

if [[ ! -e "$SUBMODULE_PATH/.git" ]]; then
  echo "[sync-superchain] superchain-registry submodule not initialized at $SUBMODULE_PATH" >&2
  echo "[sync-superchain] run: git submodule update --init --depth 1 packages/contracts-bedrock/lib/superchain-registry" >&2
  exit 1
fi

REGISTRY_COMMIT=$(git -C "$SUBMODULE_PATH" rev-parse HEAD)

# Short-circuit: if the existing zip already pins the same commit, do nothing.
#
# NOTE: this only checks the SR commit, not whether THIS script has changed
# since the zip was built. If you've edited sync-superchain.sh and want to
# force regeneration, delete superchain-configs.zip first:
#
#     rm op-core/superchain/superchain-configs.zip && just sync-superchain
#
# CI always rebuilds from scratch, so script changes can't escape there.
if [[ -f "$ZIP" ]]; then
  existing=$(unzip -p "$ZIP" COMMIT 2>/dev/null | tr -d '[:space:]' || true)
  if [[ "$existing" == "$REGISTRY_COMMIT" ]]; then
    echo "[sync-superchain] up to date at commit $REGISTRY_COMMIT"
    exit 0
  fi
fi

workdir=$(mktemp -d)

echo "Copying configs from submodule at $REGISTRY_COMMIT..."
cp -r "$SUBMODULE_PATH/superchain/configs" "$workdir/configs"
cp -r "$SUBMODULE_PATH/superchain/extra/genesis" "$workdir/genesis"
cp -r "$SUBMODULE_PATH/superchain/extra/dictionary" "$workdir/dictionary"

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

echo "Cleaning up..."
rm -rf "$workdir"

echo "Done."
