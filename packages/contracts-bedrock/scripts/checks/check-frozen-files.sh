#!/usr/bin/env bash
set -euo pipefail

# Grab the directory of the contracts-bedrock package.
SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )

# Load semver-utils.
# shellcheck source=/dev/null
source "$SCRIPT_DIR/utils/semver-utils.sh"

# Path to semver-lock.json.
SEMVER_LOCK="snapshots/semver-lock.json"

# Create a temporary directory.
temp_dir=$(mktemp -d)
trap 'rm -rf "$temp_dir"' EXIT

# Exit early if semver-lock.json has not changed.
if ! { git diff origin/develop...HEAD --name-only; git diff --name-only; git diff --cached --name-only; } | grep -q "$SEMVER_LOCK"; then
    echo "No changes detected in semver-lock.json"
    exit 0
fi

# Get the upstream semver-lock.json.
if ! git show origin/develop:packages/contracts-bedrock/snapshots/semver-lock.json > "$temp_dir/upstream_semver_lock.json" 2>/dev/null; then
      echo "❌ Error: Could not find semver-lock.json in the snapshots/ directory of develop branch"
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
        ) | map(.key);
    changes[]
' "$temp_dir/local_semver_lock.json" "$temp_dir/upstream_semver_lock.json")

# List of files that are allowed to be modified.
# In order to prevent a file from being modified, comment it out. Do not delete it.
# All files in semver-lock.json should be in this list.
ALLOWED_FILES=(
  "src/L1/StandardValidator.sol:StandardValidator"
  "src/L1/DataAvailabilityChallenge.sol:DataAvailabilityChallenge"
  # "src/L1/ETHLockbox.sol:ETHLockbox"
  "src/L1/L1CrossDomainMessenger.sol:L1CrossDomainMessenger"
  "src/L1/L1ERC721Bridge.sol:L1ERC721Bridge"
  "src/L1/L1StandardBridge.sol:L1StandardBridge"
  "src/L1/OPContractsManager.sol:OPContractsManager"
  # "src/L1/OptimismPortal2.sol:OptimismPortal2"
  "src/L1/ProtocolVersions.sol:ProtocolVersions"
  "src/L1/SuperchainConfig.sol:SuperchainConfig"
  "src/L1/SystemConfig.sol:SystemConfig"
  "src/L2/BaseFeeVault.sol:BaseFeeVault"
  "src/L2/CrossL2Inbox.sol:CrossL2Inbox"
  "src/L2/ETHLiquidity.sol:ETHLiquidity"
  "src/L2/GasPriceOracle.sol:GasPriceOracle"
  "src/L2/L1Block.sol:L1Block"
  "src/L2/L1FeeVault.sol:L1FeeVault"
  "src/L2/L2CrossDomainMessenger.sol:L2CrossDomainMessenger"
  "src/L2/L2ERC721Bridge.sol:L2ERC721Bridge"
  "src/L2/L2StandardBridge.sol:L2StandardBridge"
  "src/L2/L2StandardBridgeInterop.sol:L2StandardBridgeInterop"
  "src/L2/L2ToL1MessagePasser.sol:L2ToL1MessagePasser"
  "src/L2/L2ToL2CrossDomainMessenger.sol:L2ToL2CrossDomainMessenger"
  "src/L2/OptimismMintableERC721.sol:OptimismMintableERC721"
  "src/L2/OptimismMintableERC721Factory.sol:OptimismMintableERC721Factory"
  "src/L2/OptimismSuperchainERC20.sol:OptimismSuperchainERC20"
  "src/L2/OptimismSuperchainERC20Beacon.sol:OptimismSuperchainERC20Beacon"
  "src/L2/OptimismSuperchainERC20Factory.sol:OptimismSuperchainERC20Factory"
  "src/L2/SequencerFeeVault.sol:SequencerFeeVault"
  "src/L2/SuperchainERC20.sol:SuperchainERC20"
  "src/L2/SuperchainTokenBridge.sol:SuperchainTokenBridge"
  "src/L2/SuperchainETHBridge.sol:SuperchainETHBridge"
  "src/L2/WETH.sol:WETH"
  "src/cannon/MIPS64.sol:MIPS64"
  "src/cannon/PreimageOracle.sol:PreimageOracle"
  # "src/dispute/AnchorStateRegistry.sol:AnchorStateRegistry"
  "src/dispute/DelayedWETH.sol:DelayedWETH"
  # "src/dispute/DisputeGameFactory.sol:DisputeGameFactory"
  "src/dispute/FaultDisputeGame.sol:FaultDisputeGame"
  "src/dispute/PermissionedDisputeGame.sol:PermissionedDisputeGame"
  "src/dispute/SuperFaultDisputeGame.sol:SuperFaultDisputeGame"
  "src/dispute/SuperPermissionedDisputeGame.sol:SuperPermissionedDisputeGame"
  "src/legacy/DeployerWhitelist.sol:DeployerWhitelist"
  "src/legacy/L1BlockNumber.sol:L1BlockNumber"
  "src/legacy/LegacyMessagePasser.sol:LegacyMessagePasser"
  "src/safe/DeputyGuardianModule.sol:DeputyGuardianModule"
  "src/safe/DeputyPauseModule.sol:DeputyPauseModule"
  "src/safe/LivenessGuard.sol:LivenessGuard"
  "src/safe/LivenessModule.sol:LivenessModule"
  "src/universal/OptimismMintableERC20.sol:OptimismMintableERC20"
  "src/universal/OptimismMintableERC20Factory.sol:OptimismMintableERC20Factory"
  "src/universal/StorageSetter.sol:StorageSetter"
  "src/vendor/asterisc/RISCV.sol:RISCV"
  "src/vendor/eas/EAS.sol:EAS"
  "src/vendor/eas/SchemaRegistry.sol:SchemaRegistry"
)

MATCHED_FILES=()
# Check each changed contract against allowed patterns
for contract in $changed_contracts; do
    is_allowed=false
    for allowed_file in "${ALLOWED_FILES[@]}"; do
        if [[ "$contract" == "$allowed_file" ]]; then
            is_allowed=true
            break
        fi
    done
    if [[ "$is_allowed" == "false" ]]; then
        MATCHED_FILES+=("$contract")
    fi
done

if [ ${#MATCHED_FILES[@]} -gt 0 ]; then
    echo "❌ Error: Changes detected in files that are not allowed to be modified."
    echo "The following files were modified but are not in the allowed list:"
    printf '  - %s\n' "${MATCHED_FILES[@]}"
    echo "Only the following files can be modified:"
    printf '  - %s\n' "${ALLOWED_FILES[@]}"
    echo "The code freeze is expected to be lifted no later than 2025-02-20."
    exit 1
fi

echo "✅ All changes are in allowed files"
exit 0
