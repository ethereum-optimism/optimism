// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

// Forge
import { Script } from "forge-std/Script.sol";

// Scripts
import { DeployUtils } from "scripts/libraries/DeployUtils.sol";
import { SemverComp } from "src/libraries/SemverComp.sol";

// Interfaces
import { OPContractsManager } from "src/L1/OPContractsManager.sol";
import { IOPContractsManagerPre4_1_0 } from "interfaces/L1/IOPContractsManager.sol";
import { ISystemConfig } from "interfaces/L1/ISystemConfig.sol";
import { IProxyAdmin } from "interfaces/universal/IProxyAdmin.sol";
import { IDelayedWETH } from "interfaces/dispute/IDelayedWETH.sol";
import { IBigStepper } from "interfaces/dispute/IBigStepper.sol";
import { GameType, Duration, Claim } from "src/dispute/lib/Types.sol";
import { IFaultDisputeGame } from "interfaces/dispute/IFaultDisputeGame.sol";

/// @title AddGameType
contract AddGameType is Script {
    struct Input {
        // Address that will be used for the DummyCaller contract
        address prank;
        // OPCM contract address
        OPContractsManager opcmImpl;
        // SystemConfig contract address
        ISystemConfig systemConfigProxy;
        // ProxyAdmin contract address
        IProxyAdmin opChainProxyAdmin;
        // DelayedWETH contract address (optional)
        IDelayedWETH delayedWETHProxy;
        // Game type to add
        GameType disputeGameType;
        // Absolute prestate for the game
        Claim disputeAbsolutePrestate;
        // Maximum game depth
        uint256 disputeMaxGameDepth;
        // Split depth for the game
        uint256 disputeSplitDepth;
        // Clock extension duration
        Duration disputeClockExtension;
        // Maximum clock duration
        Duration disputeMaxClockDuration;
        // Initial bond amount
        uint256 initialBond;
        // VM contract address
        IBigStepper vm;
        // Whether this is a permissioned game
        bool permissioned;
        // Salt mixer for deterministic addresses
        string saltMixer;
    }

    struct Output {
        IDelayedWETH delayedWETHProxy;
        IFaultDisputeGame faultDisputeGameProxy;
    }

    function run(Input memory _agi) public returns (Output memory) {
        // Etch DummyCaller contract
        address prank = _agi.prank;

        // From OPCM version 4.1.0, the proxyAdmin was removed from the OpChainConfig struct so we do create support for
        // both interface variants.
        if (SemverComp.lt(_agi.opcmImpl.version(), "4.1.0")) {
            bytes memory code = vm.getDeployedCode("AddGameType.s.sol:OldDummyCaller");
            vm.etch(prank, code);
            vm.store(prank, bytes32(0), bytes32(uint256(uint160(address(_agi.opcmImpl)))));
            vm.label(prank, "DummyCaller");

            // Create the game input
            IOPContractsManagerPre4_1_0.AddGameInput[] memory gameConfigs =
                new IOPContractsManagerPre4_1_0.AddGameInput[](1);
            gameConfigs[0] = IOPContractsManagerPre4_1_0.AddGameInput({
                saltMixer: _agi.saltMixer,
                systemConfig: _agi.systemConfigProxy,
                proxyAdmin: _agi.opChainProxyAdmin,
                delayedWETH: _agi.delayedWETHProxy,
                disputeGameType: _agi.disputeGameType,
                disputeAbsolutePrestate: _agi.disputeAbsolutePrestate,
                disputeMaxGameDepth: _agi.disputeMaxGameDepth,
                disputeSplitDepth: _agi.disputeSplitDepth,
                disputeClockExtension: _agi.disputeClockExtension,
                disputeMaxClockDuration: _agi.disputeMaxClockDuration,
                initialBond: _agi.initialBond,
                vm: _agi.vm,
                permissioned: _agi.permissioned
            });

            // Call into the DummyCaller to perform the delegatecall
            vm.broadcast(msg.sender);

            (bool success, bytes memory result) = DummyCallerPreOPCM4_1_0(prank).addGameType(gameConfigs);
            require(success, "AddGameType: addGameType failed");

            // Decode the result and set it in the output
            OPContractsManager.AddGameOutput[] memory outputs = abi.decode(result, (OPContractsManager.AddGameOutput[]));
            require(outputs.length == 1, "AddGameType: unexpected number of outputs");
            return
                Output({ delayedWETHProxy: outputs[0].delayedWETH, faultDisputeGameProxy: outputs[0].faultDisputeGame });
        } else {
            bytes memory code = vm.getDeployedCode("AddGameType.s.sol:DummyCaller");
            vm.etch(prank, code);
            vm.store(prank, bytes32(0), bytes32(uint256(uint160(address(_agi.opcmImpl)))));
            vm.label(prank, "DummyCaller");

            // Create the game input
            OPContractsManager.AddGameInput[] memory gameConfigs = new OPContractsManager.AddGameInput[](1);
            gameConfigs[0] = OPContractsManager.AddGameInput({
                saltMixer: _agi.saltMixer,
                systemConfig: _agi.systemConfigProxy,
                delayedWETH: _agi.delayedWETHProxy,
                disputeGameType: _agi.disputeGameType,
                disputeAbsolutePrestate: _agi.disputeAbsolutePrestate,
                disputeMaxGameDepth: _agi.disputeMaxGameDepth,
                disputeSplitDepth: _agi.disputeSplitDepth,
                disputeClockExtension: _agi.disputeClockExtension,
                disputeMaxClockDuration: _agi.disputeMaxClockDuration,
                initialBond: _agi.initialBond,
                vm: _agi.vm,
                permissioned: _agi.permissioned
            });

            // Call into the DummyCaller to perform the delegatecall
            vm.broadcast(msg.sender);

            (bool success, bytes memory result) = DummyCaller(prank).addGameType(gameConfigs);
            require(success, "AddGameType: addGameType failed");

            // Decode the result and set it in the output
            OPContractsManager.AddGameOutput[] memory outputs = abi.decode(result, (OPContractsManager.AddGameOutput[]));
            require(outputs.length == 1, "AddGameType: unexpected number of outputs");
            return
                Output({ delayedWETHProxy: outputs[0].delayedWETH, faultDisputeGameProxy: outputs[0].faultDisputeGame });
        }
    }

    function checkOutput(Output memory _ago) internal view {
        DeployUtils.assertValidContractAddress(address(_ago.delayedWETHProxy));
        DeployUtils.assertValidContractAddress(address(_ago.faultDisputeGameProxy));
    }
}

/// @title DummyCallerPreOPCM4_1_0
/// @notice This contract is used to mimic the contract that is used as the source of the delegatecall to the OPCM.
/// @dev This contract is used for OPCM versions 4.1.0 and below.
contract DummyCallerPreOPCM4_1_0 {
    address internal _opcmAddr;

    function addGameType(IOPContractsManagerPre4_1_0.AddGameInput[] memory _gameConfigs)
        external
        returns (bool, bytes memory)
    {
        bytes memory data = abi.encodeCall(DummyCallerPreOPCM4_1_0.addGameType, _gameConfigs);
        (bool success, bytes memory result) = _opcmAddr.delegatecall(data);
        return (success, result);
    }
}

/// @title DummyCaller
/// @notice This contract is used to mimic the contract that is used as the source of the delegatecall to the OPCM.
/// @dev This contract is used for OPCM versions 4.1.0 and above.
contract DummyCaller {
    address internal _opcmAddr;

    function addGameType(OPContractsManager.AddGameInput[] memory _gameConfigs) external returns (bool, bytes memory) {
        bytes memory data = abi.encodeCall(DummyCaller.addGameType, _gameConfigs);
        (bool success, bytes memory result) = _opcmAddr.delegatecall(data);
        return (success, result);
    }
}
