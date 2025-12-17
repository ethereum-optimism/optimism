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
import { IFaultDisputeGameV2 } from "interfaces/dispute/v2/IFaultDisputeGameV2.sol";
import { IPermissionedDisputeGameV2 } from "interfaces/dispute/v2/IPermissionedDisputeGameV2.sol";
import { IDelayedWETH } from "interfaces/dispute/IDelayedWETH.sol";
import { IBigStepper } from "interfaces/dispute/IBigStepper.sol";
import { IAnchorStateRegistry } from "interfaces/dispute/IAnchorStateRegistry.sol";

/// @title DeployDisputeGame
contract DeployDisputeGame is Script {
    /// We need a struct for constructor args to avoid stack-too-deep errors.
    struct Input {
        // Common inputs.
        string release;
        // Specify which game kind is being deployed here.
        string gameKind;
        // All inputs required to deploy FaultDisputeGame.
        GameType gameType;
        bytes32 absolutePrestate;
        uint256 maxGameDepth;
        uint256 splitDepth;
        uint64 clockExtension;
        uint64 maxClockDuration;
        IDelayedWETH delayedWethProxy;
        IAnchorStateRegistry anchorStateRegistryProxy;
        IBigStepper vmAddress;
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

        deployDisputeGameImplV2(_input, output_);

        assertValidOutput(_input, output_);
    }

    function deployDisputeGameImplV1(Input memory _input, Output memory _output) internal {
        // Shove the arguments into a struct to avoid stack-too-deep errors.
        IFaultDisputeGame.GameConstructorParams memory args = IFaultDisputeGame.GameConstructorParams({
            gameType: _input.gameType,
            absolutePrestate: Claim.wrap(_input.absolutePrestate),
            maxGameDepth: _input.maxGameDepth,
            splitDepth: _input.splitDepth,
            clockExtension: Duration.wrap(_input.clockExtension),
            maxClockDuration: Duration.wrap(_input.maxClockDuration),
            vm: _input.vmAddress,
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

    function deployDisputeGameImplV2(Input memory _input, Output memory _output) internal {
        // Shove the arguments into a struct to avoid stack-too-deep errors.
        IFaultDisputeGameV2.GameConstructorParams memory args = IFaultDisputeGameV2.GameConstructorParams({
            maxGameDepth: _input.maxGameDepth,
            splitDepth: _input.splitDepth,
            clockExtension: Duration.wrap(_input.clockExtension),
            maxClockDuration: Duration.wrap((_input.maxClockDuration))
        });

        // PermissionedDisputeGame is used as the type here because it is a superset of
        // FaultDisputeGame. If the user requests to deploy a FaultDisputeGame, the user will get a
        // FaultDisputeGame (and not a PermissionedDisputeGame).
        IPermissionedDisputeGame impl;
        if (LibString.eq(_input.gameKind, "FaultDisputeGame")) {
            impl = IPermissionedDisputeGame(
                DeployUtils.createDeterministic({
                    _name: "FaultDisputeGameV2",
                    _args: DeployUtils.encodeConstructor(abi.encodeCall(IFaultDisputeGameV2.__constructor__, (args))),
                    _salt: DeployUtils.DEFAULT_SALT
                })
            );
        } else {
            impl = IPermissionedDisputeGame(
                DeployUtils.createDeterministic({
                    _name: "PermissionedDisputeGameV2",
                    _args: DeployUtils.encodeConstructor(abi.encodeCall(IPermissionedDisputeGameV2.__constructor__, (args))),
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
        require(_input.absolutePrestate != bytes32(0), "DeployDisputeGame: absolutePrestate not set");
        require(_input.maxGameDepth != 0, "DeployDisputeGame: maxGameDepth not set");
        require(_input.splitDepth != 0, "DeployDisputeGame: splitDepth not set");
        require(_input.l2ChainId != 0, "DeployDisputeGame: l2ChainId not set");
        require(address(_input.delayedWethProxy) != address(0), "DeployDisputeGame: delayedWethProxy not set");
        require(
            address(_input.anchorStateRegistryProxy) != address(0),
            "DeployDisputeGame: anchorStateRegistryProxy not set"
        );
        require(address(_input.vmAddress) != address(0), "DeployDisputeGame: vmAddress not set");
        require(!LibString.eq(_input.release, ""), "DeployDisputeGame: release not set");
        require(
            LibString.eq(_input.gameKind, "FaultDisputeGame")
                || LibString.eq(_input.gameKind, "PermissionedDisputeGame"),
            "DeployDisputeGame: unknown game kind"
        );

        require(_input.proposer != address(0), "DeployDisputeGame: proposer not set");
        require(_input.challenger != address(0), "DeployDisputeGame: challenger not set");
    }

    function assertValidOutput(Input memory _input, Output memory _output) internal view {
        IPermissionedDisputeGame game = _output.disputeGameImpl;

        DeployUtils.assertValidContractAddress(address(game));

        require(game.maxGameDepth() == _input.maxGameDepth, "DG-20");
        require(game.splitDepth() == _input.splitDepth, "DG-30");
        require(game.clockExtension().raw() == uint64(_input.clockExtension), "DG-40");
        require(game.maxClockDuration().raw() == uint64(_input.maxClockDuration), "DG-50");
    }
}
