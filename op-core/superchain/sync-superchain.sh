#!/usr/bin/env bash

# Syncs the superchain-registry configs into op-core/superchain/superchain-configs.zip.
#
# Reads from the superchain-registry git submodule at the repo root — the single
# canonical commit pin. The resulting zip is gitignored; the committed
# superchain-configs.zip.sha256 pins it.
#
# The bundle is reproducible from the submodule data alone (it embeds no commit),
# so it can be regenerated git-free inside a Docker build, where the submodule
# data is copied into the context but the gitlink can't be resolved.
#
# Modes:
#   * default (verify) — used as a build prerequisite and inside Docker. If the
#     on-disk zip already matches the committed .sha256, do nothing. Otherwise
#     regenerate it from the submodule and assert the result matches the committed
#     .sha256, failing loudly on drift. Never rewrites the .sha256.
#   * refresh (OP_CORE_SYNC_SUPERCHAIN=1) — regenerate the zip AND rewrite the
#     committed .sha256. This is what `just sync-superchain [<ref>]` runs to bump
#     the registry; it's also how you force a rebuild after editing this script.
#   * external (SUPERCHAIN_REGISTRY_DIR + SUPERCHAIN_CONFIGS_OUT) — build a bundle
#     from a caller-provided superchain-registry checkout to a throwaway output
#     path, for an arbitrary registry commit (used by op-chain-ops/check-prestate).
#     No .sha256 gate; the submodule is left untouched.

set -euo pipefail

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &>/dev/null && pwd)
# Repo root via git, falling back to the known layout (<root>/op-core/superchain)
# for environments without a .git dir, e.g. inside a Docker build.
REPO_ROOT=$(git -C "$SCRIPT_DIR" rev-parse --show-toplevel 2>/dev/null || (cd "$SCRIPT_DIR/../.." && pwd))
SR_DIR="${SUPERCHAIN_REGISTRY_DIR:-$REPO_ROOT/superchain-registry}"
ZIP="${SUPERCHAIN_CONFIGS_OUT:-$SCRIPT_DIR/superchain-configs.zip}"
SHA_FILE="$SCRIPT_DIR/superchain-configs.zip.sha256"

external=""
[[ -n "${SUPERCHAIN_REGISTRY_DIR:-}" ]] && external=1
refresh=""
[[ -n "${OP_CORE_SYNC_SUPERCHAIN:-}" ]] && refresh=1

expected_sha=$(awk '{print $1}' "$SHA_FILE" 2>/dev/null || true)

# Fast path (default mode): the on-disk zip already matches the committed .sha256,
# so there's nothing to do. To force a rebuild from the current superchain-registry
# submodule, run `just sync-superchain` (or set OP_CORE_SYNC_SUPERCHAIN=1).
if [[ -z "$external" && -z "$refresh" && -f "$ZIP" && "$(sha256sum "$ZIP" | awk '{print $1}')" == "$expected_sha" ]]; then
  echo "[sync-superchain] up to date (run \`just sync-superchain\` to force a rebuild from the submodule)"
  exit 0
fi

if [[ -z "$external" && ! -d "$SR_DIR/superchain" ]]; then
  echo "[sync-superchain] superchain-registry submodule not initialized; initializing..." >&2
  (cd "$REPO_ROOT" && just update-superchain-registry-submodule)
fi

workdir=$(mktemp -d)

echo "Copying configs from superchain-registry submodule..."
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

# Archive the configs as a ZIP file. ZIP is used since it can be efficiently used as a filesystem.
echo "Archiving configs..."
# We need to normalize the lastmod dates and permissions to ensure the ZIP file is deterministic.
find . -exec touch -t 198001010000.00 {} +
chmod -R 755 ./*
files=$(find . -type f | LC_ALL=C sort)
echo -n "$files" | xargs zip -9 -oX --quiet superchain-configs.zip
mv superchain-configs.zip "$ZIP"

got_sha=$(sha256sum "$ZIP" | awk '{print $1}')

if [[ -n "$external" ]]; then
  # Throwaway bundle for a caller-provided registry checkout; not pinned by .sha256.
  echo "[sync-superchain] built $ZIP from $SR_DIR"
elif [[ -n "$refresh" ]]; then
  # Persist the bundle's SHA256 alongside it. The hash is committed to git
  # (the zip itself isn't); any drift between what a developer/CI builds and what
  # was approved in review surfaces as a .sha256 diff.
  echo "$got_sha  superchain-configs.zip" >"$SHA_FILE"
  echo "[sync-superchain] refreshed bundle and .sha256 ($got_sha)"
else
  if [[ "$got_sha" != "$expected_sha" ]]; then
    echo "ERROR: regenerated superchain-configs.zip sha256 $got_sha != committed $expected_sha." >&2
    echo "The submodule data and the committed .sha256 disagree — run 'just sync-superchain'." >&2
    exit 1
  fi
  echo "[sync-superchain] regenerated bundle matches committed .sha256"
fi

echo "Cleaning up..."
rm -rf "$workdir"

echo "Done."
