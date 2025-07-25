// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Forge
import { Script } from "forge-std/Script.sol";

// Scripts
import { DeployUtils } from "scripts/libraries/DeployUtils.sol";

// Libraries
import { GameType, Claim, Duration } from "src/dispute/lib/Types.sol";
import { LibString } from "@solady/utils/LibString.sol";

// Interfaces
import { IFaultDisputeGame } from "interfaces/dispute/IFaultDisputeGame.sol";
import { IPermissionedDisputeGame } from "interfaces/dispute/IPermissionedDisputeGame.sol";
import { IDelayedWETH } from "interfaces/dispute/IDelayedWETH.sol";
import { IBigStepper } from "interfaces/dispute/IBigStepper.sol";
import { IAnchorStateRegistry } from "interfaces/dispute/IAnchorStateRegistry.sol";

/// @title DeployDisputeGame
contract DeployDisputeGame2 is Script {
    /// We need a struct for constructor args to avoid stack-too-deep errors.
    struct Input {
        // Common inputs.
        string release;
        string standardVersionsToml;
        // Specify which game kind is being deployed here.
        string gameKind;
        // All inputs required to deploy FaultDisputeGame.
        uint256 gameType;
        bytes32 absolutePrestate;
        uint256 maxGameDepth;
        uint256 splitDepth;
        uint256 clockExtension;
        uint256 maxClockDuration;
        IDelayedWETH delayedWethProxy;
        IAnchorStateRegistry anchorStateRegistryProxy;
        IBigStepper vm;
        uint256 l2ChainId;
        // Additional inputs required to deploy PermissionedDisputeGame.
        address proposer;
        address challenger;
    }

    struct Output {
        IPermissionedDisputeGame disputeGameImpl;
    }

    function run(Input memory _input) public returns (Output memory output_) {
        assertValidInput(_input);

        deployDisputeGameImpl(_input, output_);

        assertValidOutput(_input, output_);
    }

    function deployDisputeGameImpl(Input memory _input, Output memory _output) internal {
        // Shove the arguments into a struct to avoid stack-too-deep errors.
        IFaultDisputeGame.GameConstructorParams memory args = IFaultDisputeGame.GameConstructorParams({
            gameType: GameType.wrap(uint32(_input.gameType)),
            absolutePrestate: Claim.wrap(_input.absolutePrestate),
            maxGameDepth: _input.maxGameDepth,
            splitDepth: _input.splitDepth,
            clockExtension: Duration.wrap(uint64(_input.clockExtension)),
            maxClockDuration: Duration.wrap(uint64(_input.maxClockDuration)),
            vm: _input.vm,
            weth: _input.delayedWethProxy,
            anchorStateRegistry: _input.anchorStateRegistryProxy,
            l2ChainId: _input.l2ChainId
        });

        // PermissionedDisputeGame is used as the type here because it is a superset of
        // FaultDisputeGame. If the user requests to deploy a FaultDisputeGame, the user will get a
        // FaultDisputeGame (and not a PermissionedDisputeGame).
        IPermissionedDisputeGame impl;
        if (LibString.eq(_input.gameKind, "FaultDisputeGame")) {
            impl = IPermissionedDisputeGame(
                DeployUtils.createDeterministic({
                    _name: "FaultDisputeGame",
                    _args: DeployUtils.encodeConstructor(abi.encodeCall(IFaultDisputeGame.__constructor__, (args))),
                    _salt: DeployUtils.DEFAULT_SALT
                })
            );
        } else {
            impl = IPermissionedDisputeGame(
                DeployUtils.createDeterministic({
                    _name: "PermissionedDisputeGame",
                    _args: DeployUtils.encodeConstructor(
                        abi.encodeCall(IPermissionedDisputeGame.__constructor__, (args, _input.proposer, _input.challenger))
                    ),
                    _salt: DeployUtils.DEFAULT_SALT
                })
            );
        }

        vm.label(address(impl), string.concat(_input.gameKind, "Impl"));
        _output.disputeGameImpl = impl;
    }

    // A release is considered a 'develop' release if it does not start with 'op-contracts'.
    function isDevelopRelease(string memory _release) internal pure returns (bool) {
        return !LibString.startsWith(_release, "op-contracts");
    }

    function assertValidInput(Input memory _input) internal pure {
        require(_input.gameType <= type(uint32).max, "DeployDisputeGame: gameType must fit inside uint32");
        require(_input.maxGameDepth != 0, "DeployDisputeGame: maxGameDepth not set");
        require(_input.splitDepth != 0, "DeployDisputeGame: splitDepth not set");
        require(_input.clockExtension <= type(uint64).max, "DeployDisputeGame: clockExtension must fit inside uint64");
        require(_input.clockExtension != 0, "DeployDisputeGame: clockExtension not set");
        require(
            _input.maxClockDuration <= type(uint64).max, "DeployDisputeGame: maxClockDuration must fit inside uint64"
        );
        require(_input.maxClockDuration != 0, "DeployDisputeGame: maxClockDuration not set");
        require(_input.l2ChainId != 0, "DeployDisputeGame: l2ChainId not set");
        require(address(_input.delayedWethProxy) != address(0), "DeployDisputeGame: delayedWethProxy not set");
        require(
            address(_input.anchorStateRegistryProxy) != address(0),
            "DeployDisputeGame: anchorStateRegistryProxy not set"
        );
        require(address(_input.vm) != address(0), "DeployDisputeGame: vm not set");
        require(!LibString.eq(_input.release, ""), "DeployDisputeGame: release not set");
        require(!LibString.eq(_input.standardVersionsToml, ""), "DeployDisputeGame: standardVersionsToml not set");
        require(
            LibString.eq(_input.gameKind, "FaultDisputeGame")
                || LibString.eq(_input.gameKind, "PermissionedDisputeGame"),
            "DeployDisputeGame: unknown game kind"
        );

        if (LibString.eq(_input.gameKind, "FaultDisputeGame")) {
            require(_input.proposer == address(0), "DeployDisputeGame: proposer must be empty");
            require(_input.challenger == address(0), "DeployDisputeGame: challenger must be empty");
        } else {
            require(_input.proposer != address(0), "DeployDisputeGame: proposer not set");
            require(_input.challenger != address(0), "DeployDisputeGame: challenger not set");
        }
    }

    function assertValidOutput(Input memory _input, Output memory _output) internal view {
        IPermissionedDisputeGame game = _output.disputeGameImpl;

        DeployUtils.assertValidContractAddress(address(game));

        require(game.gameType().raw() == uint32(_input.gameType), "DG-10");
        require(game.maxGameDepth() == _input.maxGameDepth, "DG-20");
        require(game.splitDepth() == _input.splitDepth, "DG-30");
        require(game.clockExtension().raw() == uint64(_input.clockExtension), "DG-40");
        require(game.maxClockDuration().raw() == uint64(_input.maxClockDuration), "DG-50");
        require(game.vm() == _input.vm, "DG-60");
        require(game.weth() == _input.delayedWethProxy, "DG-70");
        require(game.anchorStateRegistry() == _input.anchorStateRegistryProxy, "DG-80");
        require(game.l2ChainId() == _input.l2ChainId, "DG-90");

        if (LibString.eq(_input.gameKind, "PermissionedDisputeGame")) {
            require(game.proposer() == _input.proposer, "DG-100");
            require(game.challenger() == _input.challenger, "DG-110");
        }
    }
}
