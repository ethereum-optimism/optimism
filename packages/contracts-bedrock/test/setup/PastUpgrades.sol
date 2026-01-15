// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

// Forge
import { Vm } from "forge-std/Vm.sol";
import { console2 as console } from "forge-std/console2.sol";

// Scripts
import { Process } from "scripts/libraries/Process.sol";

// Testing
import { EIP1967Helper } from "test/mocks/EIP1967Helper.sol";

// Libraries
import { Claim, GameTypes } from "src/dispute/lib/Types.sol";
import { SemverComp } from "src/libraries/SemverComp.sol";

// Interfaces
import { IOPContractsManager } from "interfaces/L1/IOPContractsManager.sol";
import { IOPContractsManagerV2 } from "interfaces/L1/opcm/IOPContractsManagerV2.sol";
import { IOPContractsManagerUtils } from "interfaces/L1/opcm/IOPContractsManagerUtils.sol";
import { ISystemConfig } from "interfaces/L1/ISystemConfig.sol";
import { ISuperchainConfig } from "interfaces/L1/ISuperchainConfig.sol";
import { IProxyAdmin } from "interfaces/universal/IProxyAdmin.sol";
import { IDisputeGameFactory } from "interfaces/dispute/IDisputeGameFactory.sol";
import { ISemver } from "interfaces/universal/ISemver.sol";

/// @title PastUpgrades
/// @notice Library for loading and executing past upgrades by fetching OPCM data from the
///         superchain-registry via FFI. This provides a single source of truth for past upgrade
///         configuration that can be used across ForkLive.s.sol and OPCM tests.
library PastUpgrades {
    Vm internal constant vm = Vm(address(uint160(uint256(keccak256("hevm cheat code")))));

    /// @notice Dummy prestates used for testing (actual values don't matter for upgrade tests)
    bytes32 internal constant DUMMY_CANNON_PRESTATE = keccak256("CANNON");
    bytes32 internal constant DUMMY_CANNON_KONA_PRESTATE = keccak256("CANNON_KONA");

    /// @notice Struct representing an OPCM from the registry (returned by FFI)
    struct OPCMInfo {
        address addr;
        string version;
        bool isV1;
    }

    /// @notice Fetches OPCM addresses from the superchain-registry via FFI.
    /// @param _chainId The chain ID to fetch OPCMs for.
    /// @return opcms_ Array of OPCM info structs.
    function fetchOPCMs(uint256 _chainId) internal returns (OPCMInfo[] memory opcms_) {
        string[] memory command = new string[](3);
        command[0] = "scripts/go-ffi/go-ffi";
        command[1] = "opcm";
        command[2] = vm.toString(_chainId);

        bytes memory result = Process.run(command, true);

        // Handle empty result
        if (result.length == 0 || keccak256(result) == keccak256(bytes("[]"))) {
            return new OPCMInfo[](0);
        }

        // Decode the ABI-encoded array of structs
        opcms_ = abi.decode(result, (OPCMInfo[]));
    }

    /// @notice Tries to get the lastUsedOPCMVersion from SystemConfig.
    /// @param _systemConfig The SystemConfig proxy.
    /// @return version_ The version string, empty if call reverted.
    /// @return success_ True if the call succeeded, false if it reverted (pre-6.x.x chain).
    function tryGetLastUsedOPCMVersion(ISystemConfig _systemConfig)
        internal
        view
        returns (string memory version_, bool success_)
    {
        // eip150-safe
        try _systemConfig.lastUsedOPCMVersion() returns (string memory v_) {
            return (v_, true);
        } catch {
            return ("", false);
        }
    }

    /// @notice Gets the actual OPCM version by calling version() on the contract.
    /// @param _opcm The OPCM address.
    /// @return version_ The version string.
    function getOPCMVersion(address _opcm) internal view returns (string memory version_) {
        return ISemver(_opcm).version();
    }

    /// @notice Executes a single V1 OPCM upgrade.
    /// @param _opcm The V1 OPCM contract address.
    /// @param _delegateCaller The address to use as the delegate caller.
    /// @param _systemConfig The SystemConfig proxy address.
    /// @param _superchainConfig The SuperchainConfig proxy address.
    function executeV1Upgrade(
        address _opcm,
        address _delegateCaller,
        ISystemConfig _systemConfig,
        ISuperchainConfig _superchainConfig
    )
        internal
    {
        // Get the superchain PAO
        IProxyAdmin superchainProxyAdmin = IProxyAdmin(EIP1967Helper.getAdmin(address(_superchainConfig)));
        address superchainPAO = superchainProxyAdmin.owner();

        // Upgrade the SuperchainConfig first
        vm.prank(superchainPAO, true);
        (bool scSuccess,) =
            _opcm.delegatecall(abi.encodeCall(IOPContractsManager.upgradeSuperchainConfig, (_superchainConfig)));
        // Acceptable to fail if already up to date
        scSuccess;

        // Build the OpChainConfig for the chain being upgraded
        IOPContractsManager.OpChainConfig[] memory opChainConfigs = new IOPContractsManager.OpChainConfig[](1);
        opChainConfigs[0] = IOPContractsManager.OpChainConfig({
            systemConfigProxy: _systemConfig,
            cannonPrestate: Claim.wrap(DUMMY_CANNON_PRESTATE),
            cannonKonaPrestate: Claim.wrap(DUMMY_CANNON_KONA_PRESTATE)
        });

        // Execute the OPCMv1 chain upgrade
        vm.prank(_delegateCaller, true);
        (bool upgradeSuccess,) = _opcm.delegatecall(abi.encodeCall(IOPContractsManager.upgrade, (opChainConfigs)));
        require(upgradeSuccess, "PastUpgrades: OPCMv1 upgrade failed");
    }

    /// @notice Executes a single V2 OPCM upgrade.
    /// @param _opcm The V2 OPCM contract address.
    /// @param _delegateCaller The address to use as the delegate caller.
    /// @param _systemConfig The SystemConfig proxy address.
    /// @param _superchainConfig The SuperchainConfig proxy address.
    /// @param _disputeGameFactory The DisputeGameFactory to read init bonds from.
    function executeV2Upgrade(
        address _opcm,
        address _delegateCaller,
        ISystemConfig _systemConfig,
        ISuperchainConfig _superchainConfig,
        IDisputeGameFactory _disputeGameFactory
    )
        internal
    {
        // Get the superchain PAO
        IProxyAdmin superchainProxyAdmin = IProxyAdmin(EIP1967Helper.getAdmin(address(_superchainConfig)));
        address superchainPAO = superchainProxyAdmin.owner();

        // Upgrade the SuperchainConfig first
        vm.prank(superchainPAO, true);
        (bool scSuccess,) = _opcm.delegatecall(
            abi.encodeCall(
                IOPContractsManagerV2.upgradeSuperchain,
                (
                    IOPContractsManagerV2.SuperchainUpgradeInput({
                        superchainConfig: _superchainConfig,
                        extraInstructions: new IOPContractsManagerUtils.ExtraInstruction[](0)
                    })
                )
            )
        );
        // Acceptable to fail if already up to date
        scSuccess;

        // Build dispute game configs with dummy prestates
        IOPContractsManagerUtils.DisputeGameConfig[] memory disputeGameConfigs =
            new IOPContractsManagerUtils.DisputeGameConfig[](3);

        // CANNON (game type 0)
        disputeGameConfigs[0] = IOPContractsManagerUtils.DisputeGameConfig({
            enabled: true,
            initBond: _disputeGameFactory.initBonds(GameTypes.CANNON),
            gameType: GameTypes.CANNON,
            gameArgs: abi.encode(
                IOPContractsManagerUtils.FaultDisputeGameConfig({ absolutePrestate: Claim.wrap(DUMMY_CANNON_PRESTATE) })
            )
        });

        // PERMISSIONED_CANNON (game type 1)
        disputeGameConfigs[1] = IOPContractsManagerUtils.DisputeGameConfig({
            enabled: true,
            initBond: _disputeGameFactory.initBonds(GameTypes.PERMISSIONED_CANNON),
            gameType: GameTypes.PERMISSIONED_CANNON,
            gameArgs: abi.encode(
                IOPContractsManagerUtils.PermissionedDisputeGameConfig({
                    absolutePrestate: Claim.wrap(DUMMY_CANNON_PRESTATE),
                    proposer: address(0),
                    challenger: address(0)
                })
            )
        });

        // CANNON_KONA (game type 3)
        disputeGameConfigs[2] = IOPContractsManagerUtils.DisputeGameConfig({
            enabled: true,
            initBond: _disputeGameFactory.initBonds(GameTypes.CANNON_KONA),
            gameType: GameTypes.CANNON_KONA,
            gameArgs: abi.encode(
                IOPContractsManagerUtils.FaultDisputeGameConfig({ absolutePrestate: Claim.wrap(DUMMY_CANNON_KONA_PRESTATE) })
            )
        });

        // Sort by game type (already sorted: 0, 1, 3)
        _sortDisputeGameConfigs(disputeGameConfigs);

        // Execute the V2 upgrade
        vm.prank(_delegateCaller, true);
        (bool upgradeSuccess,) = _opcm.delegatecall(
            abi.encodeCall(
                IOPContractsManagerV2.upgrade,
                (
                    IOPContractsManagerV2.UpgradeInput({
                        systemConfig: _systemConfig,
                        disputeGameConfigs: disputeGameConfigs,
                        extraInstructions: new IOPContractsManagerUtils.ExtraInstruction[](0)
                    })
                )
            )
        );
        require(upgradeSuccess, "PastUpgrades: OPCMv2 upgrade failed");
    }

    /// @notice Sorts dispute game configs by game type in ascending numerical order.
    /// @param _configs The array to sort in-place.
    function _sortDisputeGameConfigs(IOPContractsManagerUtils.DisputeGameConfig[] memory _configs) private pure {
        uint256 n = _configs.length;
        for (uint256 i = 0; i < n; i++) {
            for (uint256 j = i + 1; j < n; j++) {
                if (_configs[j].gameType.raw() < _configs[i].gameType.raw()) {
                    IOPContractsManagerUtils.DisputeGameConfig memory temp = _configs[i];
                    _configs[i] = _configs[j];
                    _configs[j] = temp;
                }
            }
        }
    }

    /// @notice Runs all past upgrades for the current chain.
    /// @param _delegateCaller The address to use as the delegate caller.
    /// @param _systemConfig The SystemConfig proxy address.
    /// @param _superchainConfig The SuperchainConfig proxy address.
    /// @param _disputeGameFactory The DisputeGameFactory (needed for V2 upgrades).
    function runPastUpgrades(
        address _delegateCaller,
        ISystemConfig _systemConfig,
        ISuperchainConfig _superchainConfig,
        IDisputeGameFactory _disputeGameFactory
    )
        internal
    {
        // Fetch OPCMs from registry via FFI
        OPCMInfo[] memory opcms = fetchOPCMs(block.chainid);

        if (opcms.length == 0) {
            console.log("PastUpgrades: No OPCMs found for chain %d", block.chainid);
            return;
        }

        // Get the last used OPCM version to skip already-applied upgrades
        (string memory lastVersion, bool hasLastVersion) = tryGetLastUsedOPCMVersion(_systemConfig);

        if (!hasLastVersion) {
            console.log("PastUpgrades: SystemConfig.lastUsedOPCMVersion() reverted - chain is pre-6.x.x");
        } else {
            console.log("PastUpgrades: lastUsedOPCMVersion = %s", lastVersion);
        }

        for (uint256 i = 0; i < opcms.length; i++) {
            address opcmAddr = opcms[i].addr;

            // Get the actual on-chain version
            string memory actualVersion = getOPCMVersion(opcmAddr);
            SemverComp.Semver memory sv = SemverComp.parse(actualVersion);

            // Skip OPCMs before 6.x.x
            if (sv.major < 6) {
                console.log("PastUpgrades: Skipping OPCM %s (v%s) - version < 6.x.x", opcmAddr, actualVersion);
                continue;
            }

            // Skip if already applied (version <= lastUsedOPCMVersion)
            if (hasLastVersion && bytes(lastVersion).length > 0 && SemverComp.lte(actualVersion, lastVersion)) {
                console.log(
                    "PastUpgrades: Skipping OPCM %s (v%s) - already applied (lastUsed=%s)",
                    opcmAddr,
                    actualVersion,
                    lastVersion
                );
                continue;
            }

            console.log("PastUpgrades: Running upgrade with OPCM %s (v%s)", opcmAddr, actualVersion);

            if (sv.major == 6) {
                executeV1Upgrade(opcmAddr, _delegateCaller, _systemConfig, _superchainConfig);
            } else {
                executeV2Upgrade(opcmAddr, _delegateCaller, _systemConfig, _superchainConfig, _disputeGameFactory);
            }
        }
    }
}
