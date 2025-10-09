// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

// Forge
import { Script } from "forge-std/Script.sol";

// Scripts
import { DeployUtils } from "scripts/libraries/DeployUtils.sol";

// Interfaces
import { IOPContractsManager } from "interfaces/L1/IOPContractsManager.sol";
import { ISystemConfig } from "interfaces/L1/ISystemConfig.sol";
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
        IOPContractsManager opcmImpl;
        // SystemConfig contract address
        ISystemConfig systemConfigProxy;
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

        bytes memory code = vm.getDeployedCode("AddGameType.s.sol:DummyCaller");
        vm.etch(prank, code);
        vm.store(prank, bytes32(0), bytes32(uint256(uint160(address(_agi.opcmImpl)))));
        vm.label(prank, "DummyCaller");

        // Create the game input
        IOPContractsManager.AddGameInput[] memory gameConfigs = new IOPContractsManager.AddGameInput[](1);
        gameConfigs[0] = IOPContractsManager.AddGameInput({
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
        IOPContractsManager.AddGameOutput[] memory outputs = abi.decode(result, (IOPContractsManager.AddGameOutput[]));
        require(outputs.length == 1, "AddGameType: unexpected number of outputs");
        return Output({ delayedWETHProxy: outputs[0].delayedWETH, faultDisputeGameProxy: outputs[0].faultDisputeGame });
    }

    function checkOutput(Output memory _ago) internal view {
        DeployUtils.assertValidContractAddress(address(_ago.delayedWETHProxy));
        DeployUtils.assertValidContractAddress(address(_ago.faultDisputeGameProxy));
    }
}

/// @title DummyCaller
/// @notice This contract is used to mimic the contract that is used as the source of the delegatecall to the OPCM.
/// @dev This contract is used for OPCM versions 4.1.0 and above.

contract DummyCaller {
    address internal _opcmAddr;

    function addGameType(IOPContractsManager.AddGameInput[] memory _gameConfigs)
        external
        returns (bool, bytes memory)
    {
        bytes memory data = abi.encodeCall(DummyCaller.addGameType, _gameConfigs);
        (bool success, bytes memory result) = _opcmAddr.delegatecall(data);
        return (success, result);
    }
}
