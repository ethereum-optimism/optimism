// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { Script } from "forge-std/Script.sol";
import { OPContractsManager } from "src/L1/OPContractsManager.sol";
import { IOPContractsManager } from "interfaces/L1/IOPContractsManager.sol";
import { IOPContractsManagerV2 } from "interfaces/L1/opcm/IOPContractsManagerV2.sol";
import { ISystemConfig } from "interfaces/L1/ISystemConfig.sol";
import { IDisputeGameFactory } from "interfaces/dispute/IDisputeGameFactory.sol";
import { IDisputeGame } from "interfaces/dispute/IDisputeGame.sol";
import { BaseDeployIO } from "scripts/deploy/BaseDeployIO.sol";
import { DevFeatures } from "src/libraries/DevFeatures.sol";
import { GameType } from "src/dispute/lib/Types.sol";

contract UpgradeOPChainInput is BaseDeployIO {
    address internal _prank;
    OPContractsManager internal _opcm;
    bytes _opChainConfigs;

    // Setter for OPContractsManager type
    function set(bytes4 _sel, address _value) public {
        require(address(_value) != address(0), "UpgradeOPCMInput: cannot set zero address");

        if (_sel == this.prank.selector) _prank = _value;
        else if (_sel == this.opcm.selector) _opcm = OPContractsManager(_value);
        else revert("UpgradeOPCMInput: unknown selector");
    }

    function set(bytes4 _sel, OPContractsManager.OpChainConfig[] memory _value) public {
        require(_value.length > 0, "UpgradeOPCMInput: cannot set empty array");

        if (_sel == this.opChainConfigs.selector) _opChainConfigs = abi.encode(_value);
        else revert("UpgradeOPCMInput: unknown selector");
    }

    function prank() public view returns (address) {
        require(address(_prank) != address(0), "UpgradeOPCMInput: prank not set");
        return _prank;
    }

    function opcm() public view returns (OPContractsManager) {
        require(address(_opcm) != address(0), "UpgradeOPCMInput: not set");
        return _opcm;
    }

    function opChainConfigs() public view returns (bytes memory) {
        require(_opChainConfigs.length > 0, "UpgradeOPCMInput: not set");
        return _opChainConfigs;
    }
}

contract UpgradeOPChain is Script {
    function run(UpgradeOPChainInput _uoci) external {
        OPContractsManager opcm = _uoci.opcm();
        OPContractsManager.OpChainConfig[] memory opChainConfigs =
            abi.decode(_uoci.opChainConfigs(), (OPContractsManager.OpChainConfig[]));

        // Check if OPCM v2 should be used
        bool useV2 = isDevFeatureOpcmV2Enabled(address(opcm));

        // Etch DummyCaller contract. This contract is used to mimic the contract that is used
        // as the source of the delegatecall to the OPCM. In practice this will be the governance
        // 2/2 or similar.
        address prank = _uoci.prank();
        bytes memory code = vm.getDeployedCode("UpgradeOPChain.s.sol:DummyCaller");
        vm.etch(prank, code);
        vm.store(prank, bytes32(0), bytes32(uint256(uint160(address(opcm)))));
        vm.label(prank, "DummyCaller");

        if (useV2) {
            // V2 path: Call upgrade for each chain separately
            upgradeWithV2(prank, opChainConfigs);
        } else {
            // V1 path: Batch upgrade
            vm.broadcast(msg.sender);
            (bool success,) = DummyCaller(prank).upgrade(opChainConfigs);
            require(success, "UpgradeChain: upgrade failed");
        }
    }

    /// @notice Check if OPCM v2 should be used based on dev feature flag
    function isDevFeatureOpcmV2Enabled(address _opcmAddr) internal view returns (bool) {
        // Both v1 and v2 share the same interface for this function.
        return IOPContractsManager(_opcmAddr).isDevFeatureEnabled(DevFeatures.OPCM_V2);
    }

    /// @notice Upgrade using OPCMv2 - processes each chain individually
    function upgradeWithV2(address _prank, OPContractsManager.OpChainConfig[] memory _opChainConfigs) internal {
        for (uint256 i = 0; i < _opChainConfigs.length; i++) {
            OPContractsManager.OpChainConfig memory chainConfig = _opChainConfigs[i];

            // Fetch existing dispute game configs from chain to preserve them
            IOPContractsManagerV2.DisputeGameConfig[] memory existingGames = fetchExistingGameConfigs(
                address(chainConfig.systemConfigProxy)
            );

            // Build the upgrade input for v2
            IOPContractsManagerV2.UpgradeInput memory upgradeInput = IOPContractsManagerV2.UpgradeInput({
                systemConfig: chainConfig.systemConfigProxy,
                disputeGameConfigs: existingGames,
                extraInstructions: new IOPContractsManagerV2.ExtraInstruction[](0)
            });

            // Call into the DummyCallerV2 to perform the delegatecall
            // Note: DummyCaller and DummyCallerV2 are compatible - same storage slot, different function sig
            vm.broadcast(msg.sender);
            (bool success,) = DummyCallerV2(_prank).upgrade(upgradeInput);
            require(success, "UpgradeChain: v2 upgrade failed");
        }
    }

    /// @notice Fetch existing dispute game configs from the chain
    /// @dev This is critical for v2 to avoid accidentally disabling existing games
    function fetchExistingGameConfigs(address _systemConfig)
        internal
        view
        returns (IOPContractsManagerV2.DisputeGameConfig[] memory)
    {
        ISystemConfig systemConfig = ISystemConfig(_systemConfig);
        IDisputeGameFactory factory = IDisputeGameFactory(address(systemConfig.disputeGameFactory()));

        // First pass: count how many game types are configured
        uint256 configuredCount = 0;
        uint256 maxGameType = 15; // Check game types 0-15 (reasonable upper bound)

        for (uint256 i = 0; i <= maxGameType; i++) {
            GameType gameType = GameType.wrap(uint32(i));
            IDisputeGame impl = factory.gameImpls(gameType);
            if (address(impl) != address(0)) {
                configuredCount++;
            }
        }

        // Second pass: build the config array
        IOPContractsManagerV2.DisputeGameConfig[] memory configs =
            new IOPContractsManagerV2.DisputeGameConfig[](configuredCount);

        uint256 index = 0;
        for (uint256 i = 0; i <= maxGameType; i++) {
            GameType gameType = GameType.wrap(uint32(i));
            IDisputeGame impl = factory.gameImpls(gameType);

            if (address(impl) != address(0)) {
                // Game type is configured - fetch its parameters
                uint256 initBond = factory.initBonds(gameType);
                bytes memory gameArgs = factory.gameArgs(gameType);

                configs[index] = IOPContractsManagerV2.DisputeGameConfig({
                    enabled: true,
                    initBond: initBond,
                    gameType: gameType,
                    gameArgs: gameArgs
                });
                index++;
            }
        }

        return configs;
    }
}

contract DummyCaller {
    address internal _opcmAddr;

    function upgrade(OPContractsManager.OpChainConfig[] memory _opChainConfigs) external returns (bool, bytes memory) {
        bytes memory data = abi.encodeCall(DummyCaller.upgrade, _opChainConfigs);
        (bool success, bytes memory result) = _opcmAddr.delegatecall(data);
        return (success, result);
    }
}

contract DummyCallerV2 {
    address internal _opcmAddr;

    function upgrade(IOPContractsManagerV2.UpgradeInput memory _upgradeInput) external returns (bool, bytes memory) {
        bytes memory data = abi.encodeCall(DummyCallerV2.upgrade, _upgradeInput);
        (bool success, bytes memory result) = _opcmAddr.delegatecall(data);
        return (success, result);
    }
}
