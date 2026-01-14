// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

// Forge
import { Vm } from "forge-std/Vm.sol";
import { console2 as console } from "forge-std/console2.sol";
import { stdToml } from "forge-std/StdToml.sol";

// Testing
import { EIP1967Helper } from "test/mocks/EIP1967Helper.sol";

// Libraries
import { Claim, GameTypes, GameType } from "src/dispute/lib/Types.sol";

// Interfaces
import { IOPContractsManager } from "interfaces/L1/IOPContractsManager.sol";
import { IOPContractsManagerV2 } from "interfaces/L1/opcm/IOPContractsManagerV2.sol";
import { IOPContractsManagerUtils } from "interfaces/L1/opcm/IOPContractsManagerUtils.sol";
import { ISystemConfig } from "interfaces/L1/ISystemConfig.sol";
import { ISuperchainConfig } from "interfaces/L1/ISuperchainConfig.sol";
import { IProxyAdmin } from "interfaces/universal/IProxyAdmin.sol";
import { IDisputeGameFactory } from "interfaces/dispute/IDisputeGameFactory.sol";

/// @title PastUpgrades
/// @notice Library for loading and executing past upgrades from the past_upgrades.toml file.
///         This provides a single source of truth for past upgrade configuration that can be
///         used across ForkLive.s.sol, OPContractsManager.t.sol, and OPContractsManagerV2.t.sol.
library PastUpgrades {
    using stdToml for string;

    Vm internal constant vm = Vm(address(uint160(uint256(keccak256("hevm cheat code")))));

    /// @notice Maximum number of upgrades to parse from the TOML file.
    uint256 internal constant MAX_UPGRADES = 100;

    /// @notice Maximum number of extra instructions per upgrade.
    uint256 internal constant MAX_EXTRA_INSTRUCTIONS = 20;

    /// @notice Maximum number of dispute game configs per upgrade.
    uint256 internal constant MAX_DISPUTE_GAME_CONFIGS = 10;

    /// @notice Struct representing a parsed dispute game config from the TOML file.
    struct ParsedDisputeGameConfig {
        bool enabled;
        uint256 initBond;
        string gameType; // "CANNON", "PERMISSIONED_CANNON", "CANNON_KONA", etc.
        bytes32 prestate;
        address proposer; // Only used for PERMISSIONED_CANNON
        address challenger; // Only used for PERMISSIONED_CANNON
    }

    /// @notice Struct representing a parsed past upgrade from the TOML file.
    struct PastUpgrade {
        string name;
        uint256 opcmVersion;
        address opcmAddress;
        // V1 fields
        bytes32 cannonPrestate;
        bytes32 cannonKonaPrestate;
        // V2 fields
        ParsedDisputeGameConfig[] disputeGameConfigs;
        IOPContractsManagerUtils.ExtraInstruction[] extraInstructions;
    }

    /// @notice Loads past upgrades from the TOML file for a specific chain ID.
    /// @param _chainId The chain ID to load upgrades for.
    /// @return upgrades_ Array of past upgrades for the given chain.
    function loadPastUpgrades(uint256 _chainId) internal view returns (PastUpgrade[] memory upgrades_) {
        // Read the TOML file
        string memory toml = vm.readFile("past_upgrades.toml");

        // Get the number of upgrades by reading the array
        // stdToml doesn't have a direct way to get array length, so we try to read elements
        // until we fail.
        PastUpgrade[] memory tempUpgrades = new PastUpgrade[](MAX_UPGRADES);
        uint256 validCount = 0;

        for (uint256 i = 0; i < MAX_UPGRADES; i++) {
            string memory basePath = string.concat(".upgrades[", vm.toString(i), "]");

            // Try to read the name - if it fails, we've reached the end
            try vm.parseTomlString(toml, string.concat(basePath, ".name")) returns (string memory name_) {
                // Get the OPCM version
                uint256 opcmVersion = toml.readUint(string.concat(basePath, ".opcm_version"));

                // Try to get the OPCM address for this chain ID
                string memory addrPath = string.concat(basePath, ".opcm_addresses.", vm.toString(_chainId));

                // Check if this chain has an OPCM address configured
                try vm.parseTomlAddress(toml, addrPath) returns (address opcmAddr_) {
                    if (opcmAddr_ == address(0)) continue;

                    tempUpgrades[validCount].name = name_;
                    tempUpgrades[validCount].opcmVersion = opcmVersion;
                    tempUpgrades[validCount].opcmAddress = opcmAddr_;

                    if (opcmVersion == 1) {
                        // V1: Get prestates directly
                        tempUpgrades[validCount].cannonPrestate =
                            toml.readBytes32(string.concat(basePath, ".cannon_prestate"));
                        tempUpgrades[validCount].cannonKonaPrestate =
                            toml.readBytes32(string.concat(basePath, ".cannon_kona_prestate"));
                    } else if (opcmVersion == 2) {
                        // V2: Parse dispute game configs and extra instructions
                        tempUpgrades[validCount].disputeGameConfigs = _parseDisputeGameConfigs(toml, basePath);
                        tempUpgrades[validCount].extraInstructions = _parseExtraInstructions(toml, basePath);
                    }

                    validCount++;
                } catch {
                    // No OPCM address for this chain, skip
                    continue;
                }
            } catch {
                // No more upgrades
                break;
            }
        }

        // Copy valid upgrades to correctly sized array
        upgrades_ = new PastUpgrade[](validCount);
        for (uint256 i = 0; i < validCount; i++) {
            upgrades_[i] = tempUpgrades[i];
        }
    }

    /// @notice Parses extra_instructions array from TOML for a given upgrade.
    /// @param _toml The TOML content.
    /// @param _basePath The base path to the upgrade entry.
    /// @return instructions_ The parsed extra instructions array.
    function _parseExtraInstructions(
        string memory _toml,
        string memory _basePath
    )
        private
        pure
        returns (IOPContractsManagerUtils.ExtraInstruction[] memory instructions_)
    {
        // Try to parse extra instructions - return empty array if none exist
        IOPContractsManagerUtils.ExtraInstruction[] memory temp =
            new IOPContractsManagerUtils.ExtraInstruction[](MAX_EXTRA_INSTRUCTIONS);
        uint256 count = 0;

        for (uint256 j = 0; j < MAX_EXTRA_INSTRUCTIONS; j++) {
            string memory instrPath = string.concat(_basePath, ".extra_instructions[", vm.toString(j), "]");
            try vm.parseTomlString(_toml, string.concat(instrPath, ".key")) returns (string memory key_) {
                // Try to read data as string first, then as hex
                bytes memory data;
                try vm.parseTomlString(_toml, string.concat(instrPath, ".data")) returns (string memory dataStr_) {
                    data = bytes(dataStr_);
                } catch {
                    try vm.parseTomlBytes(_toml, string.concat(instrPath, ".data_hex")) returns (bytes memory dataHex_)
                    {
                        data = dataHex_;
                    } catch {
                        data = "";
                    }
                }
                temp[count] = IOPContractsManagerUtils.ExtraInstruction({ key: key_, data: data });
                count++;
            } catch {
                break;
            }
        }

        instructions_ = new IOPContractsManagerUtils.ExtraInstruction[](count);
        for (uint256 j = 0; j < count; j++) {
            instructions_[j] = temp[j];
        }
    }

    /// @notice Parses dispute_game_configs array from TOML for a given upgrade.
    /// @param _toml The TOML content.
    /// @param _basePath The base path to the upgrade entry.
    /// @return configs_ The parsed dispute game configs array.
    function _parseDisputeGameConfigs(
        string memory _toml,
        string memory _basePath
    )
        private
        pure
        returns (ParsedDisputeGameConfig[] memory configs_)
    {
        ParsedDisputeGameConfig[] memory temp = new ParsedDisputeGameConfig[](MAX_DISPUTE_GAME_CONFIGS);
        uint256 count = 0;

        for (uint256 j = 0; j < MAX_DISPUTE_GAME_CONFIGS; j++) {
            string memory cfgPath = string.concat(_basePath, ".dispute_game_configs[", vm.toString(j), "]");
            try vm.parseTomlString(_toml, string.concat(cfgPath, ".game_type")) returns (string memory gameType_) {
                temp[count].gameType = gameType_;
                temp[count].enabled = vm.parseTomlBool(_toml, string.concat(cfgPath, ".enabled"));
                temp[count].prestate = vm.parseTomlBytes32(_toml, string.concat(cfgPath, ".prestate"));

                // Try to read init_bond (optional, defaults to 0)
                try vm.parseTomlUint(_toml, string.concat(cfgPath, ".init_bond")) returns (uint256 bond_) {
                    temp[count].initBond = bond_;
                } catch {
                    temp[count].initBond = 0;
                }

                // Try to read proposer/challenger (only for PERMISSIONED_CANNON)
                try vm.parseTomlAddress(_toml, string.concat(cfgPath, ".proposer")) returns (address p_) {
                    temp[count].proposer = p_;
                } catch {
                    temp[count].proposer = address(0);
                }
                try vm.parseTomlAddress(_toml, string.concat(cfgPath, ".challenger")) returns (address c_) {
                    temp[count].challenger = c_;
                } catch {
                    temp[count].challenger = address(0);
                }

                count++;
            } catch {
                break;
            }
        }

        configs_ = new ParsedDisputeGameConfig[](count);
        for (uint256 j = 0; j < count; j++) {
            configs_[j] = temp[j];
        }
    }

    /// @notice Executes a single V1 OPCM upgrade.
    /// @param _opcm The V1 OPCM contract address.
    /// @param _delegateCaller The address to use as the delegate caller.
    /// @param _systemConfig The SystemConfig proxy address.
    /// @param _superchainConfig The SuperchainConfig proxy address.
    /// @param _cannonPrestate The cannon prestate.
    /// @param _cannonKonaPrestate The cannon kona prestate.
    function executeV1Upgrade(
        address _opcm,
        address _delegateCaller,
        ISystemConfig _systemConfig,
        ISuperchainConfig _superchainConfig,
        Claim _cannonPrestate,
        Claim _cannonKonaPrestate
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
            cannonPrestate: _cannonPrestate,
            cannonKonaPrestate: _cannonKonaPrestate
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
    /// @param _upgrade The upgrade configuration.
    /// @param _disputeGameFactory The DisputeGameFactory to read init bonds from (used as fallback).
    function executeV2Upgrade(
        address _opcm,
        address _delegateCaller,
        ISystemConfig _systemConfig,
        ISuperchainConfig _superchainConfig,
        PastUpgrade memory _upgrade,
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

        // Build dispute game configs from parsed TOML config
        IOPContractsManagerUtils.DisputeGameConfig[] memory disputeGameConfigs =
            new IOPContractsManagerUtils.DisputeGameConfig[](_upgrade.disputeGameConfigs.length);

        for (uint256 i = 0; i < _upgrade.disputeGameConfigs.length; i++) {
            ParsedDisputeGameConfig memory parsed = _upgrade.disputeGameConfigs[i];

            // Convert game type string to GameType
            GameType gameType = _stringToGameType(parsed.gameType);

            // Use initBond from TOML if set, otherwise from factory
            uint256 initBond = parsed.initBond > 0 ? parsed.initBond : _disputeGameFactory.initBonds(gameType);

            // Build gameArgs based on game type
            bytes memory gameArgs;
            if (
                gameType.raw() == GameTypes.PERMISSIONED_CANNON.raw()
                    || gameType.raw() == GameTypes.SUPER_PERMISSIONED_CANNON.raw()
            ) {
                gameArgs = abi.encode(
                    IOPContractsManagerUtils.PermissionedDisputeGameConfig({
                        absolutePrestate: Claim.wrap(parsed.prestate),
                        proposer: parsed.proposer,
                        challenger: parsed.challenger
                    })
                );
            } else {
                gameArgs = abi.encode(
                    IOPContractsManagerUtils.FaultDisputeGameConfig({ absolutePrestate: Claim.wrap(parsed.prestate) })
                );
            }

            disputeGameConfigs[i] = IOPContractsManagerUtils.DisputeGameConfig({
                enabled: parsed.enabled,
                initBond: initBond,
                gameType: gameType,
                gameArgs: gameArgs
            });
        }

        // Sort dispute game configs by game type in ascending order (required by OPCM)
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
                        extraInstructions: _upgrade.extraInstructions
                    })
                )
            )
        );
        require(upgradeSuccess, "PastUpgrades: OPCMv2 upgrade failed");
    }

    /// @notice Converts a game type string to a GameType.
    /// @param _gameType The game type string (e.g., "CANNON", "PERMISSIONED_CANNON").
    /// @return gameType_ The GameType enum value.
    function _stringToGameType(string memory _gameType) private pure returns (GameType gameType_) {
        bytes32 hash = keccak256(bytes(_gameType));
        if (hash == keccak256("CANNON")) {
            return GameTypes.CANNON;
        } else if (hash == keccak256("PERMISSIONED_CANNON")) {
            return GameTypes.PERMISSIONED_CANNON;
        } else if (hash == keccak256("CANNON_KONA")) {
            return GameTypes.CANNON_KONA;
        } else if (hash == keccak256("SUPER_CANNON")) {
            return GameTypes.SUPER_CANNON;
        } else if (hash == keccak256("SUPER_PERMISSIONED_CANNON")) {
            return GameTypes.SUPER_PERMISSIONED_CANNON;
        } else if (hash == keccak256("SUPER_CANNON_KONA")) {
            return GameTypes.SUPER_CANNON_KONA;
        } else {
            revert(string.concat("PastUpgrades: Unknown game type: ", _gameType));
        }
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
        PastUpgrade[] memory upgrades = loadPastUpgrades(block.chainid);

        for (uint256 i = 0; i < upgrades.length; i++) {
            PastUpgrade memory upgrade = upgrades[i];
            console.log("PastUpgrades: Running %s upgrade using OPCM at %s", upgrade.name, upgrade.opcmAddress);

            if (upgrade.opcmVersion == 1) {
                executeV1Upgrade(
                    upgrade.opcmAddress,
                    _delegateCaller,
                    _systemConfig,
                    _superchainConfig,
                    Claim.wrap(upgrade.cannonPrestate),
                    Claim.wrap(upgrade.cannonKonaPrestate)
                );
            } else if (upgrade.opcmVersion == 2) {
                executeV2Upgrade(
                    upgrade.opcmAddress, _delegateCaller, _systemConfig, _superchainConfig, upgrade, _disputeGameFactory
                );
            } else {
                console.log("PastUpgrades: Skipping %s - unknown OPCM version %d", upgrade.name, upgrade.opcmVersion);
            }
        }
    }
}
