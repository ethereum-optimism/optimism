// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { IDelayedWETH } from "src/dispute/interfaces/IDelayedWETH.sol";
import { IAnchorStateRegistry } from "src/dispute/interfaces/IAnchorStateRegistry.sol";
import { FaultDisputeGame, IFaultDisputeGame, IBigStepper, IInitializable } from "src/dispute/FaultDisputeGame.sol";
import "src/libraries/DisputeTypes.sol";
import "src/libraries/DisputeErrors.sol";

/// @title PermissionedDisputeGame
/// @notice PermissionedDisputeGame is a contract that inherits from `FaultDisputeGame`, and contains two roles:
///         - The `challenger` role, which is allowed to challenge a dispute.
///         - The `proposer` role, which is allowed to create proposals and participate in their game.
///         This contract exists as a way for networks to support the fault proof iteration of the OptimismPortal
///         contract without needing to support a fully permissionless system. Permissionless systems can introduce
///         costs that certain networks may not wish to support. This contract can also be used as a fallback mechanism
///         in case of a failure in the permissionless fault proof system in the stage one release.
contract PermissionedDisputeGame is FaultDisputeGame {
    /// @notice The proposer role is allowed to create proposals and participate in the dispute game.
    address internal immutable PROPOSER;

    /// @notice The challenger role is allowed to participate in the dispute game.
    address internal immutable CHALLENGER;

    /// @notice Modifier that gates access to the `challenger` and `proposer` roles.
    modifier onlyAuthorized() {
        if (!(msg.sender == PROPOSER || msg.sender == CHALLENGER)) {
            revert BadAuth();
        }
        _;
    }

    /// @param _gameType The type ID of the game.
    /// @param _absolutePrestate The absolute prestate of the instruction trace.
    /// @param _maxGameDepth The maximum depth of bisection.
    /// @param _splitDepth The final depth of the output bisection portion of the game.
    /// @param _gameDuration The duration of the game.
    /// @param _vm An onchain VM that performs single instruction steps on an FPP trace.
    /// @param _weth WETH contract for holding ETH.
    /// @param _anchorStateRegistry The contract that stores the anchor state for each game type.
    /// @param _l2ChainId Chain ID of the L2 network this contract argues about.
    /// @param _proposer Address that is allowed to create instances of this contract.
    /// @param _challenger Address that is allowed to challenge instances of this contract.
    constructor(
        GameType _gameType,
        Claim _absolutePrestate,
        uint256 _maxGameDepth,
        uint256 _splitDepth,
        Duration _gameDuration,
        IBigStepper _vm,
        IDelayedWETH _weth,
        IAnchorStateRegistry _anchorStateRegistry,
        uint256 _l2ChainId,
        address _proposer,
        address _challenger
    )
        FaultDisputeGame(
            _gameType,
            _absolutePrestate,
            _maxGameDepth,
            _splitDepth,
            _gameDuration,
            _vm,
            _weth,
            _anchorStateRegistry,
            _l2ChainId
        )
    {
        PROPOSER = _proposer;
        CHALLENGER = _challenger;
    }

    /// @inheritdoc IFaultDisputeGame
    function step(
        uint256 _claimIndex,
        bool _isAttack,
        bytes calldata _stateData,
        bytes calldata _proof
    )
        public
        override
        onlyAuthorized
    {
        super.step(_claimIndex, _isAttack, _stateData, _proof);
    }

    /// @notice Generic move function, used for both `attack` and `defend` moves.
    /// @param _challengeIndex The index of the claim being moved against.
    /// @param _claim The claim at the next logical position in the game.
    /// @param _isAttack Whether or not the move is an attack or defense.
    function move(uint256 _challengeIndex, Claim _claim, bool _isAttack) public payable override onlyAuthorized {
        super.move(_challengeIndex, _claim, _isAttack);
    }

    /// @inheritdoc IInitializable
    function initialize() public payable override {
        // The creator of the dispute game must be the proposer EOA.
        if (tx.origin != PROPOSER) revert BadAuth();

        // Fallthrough initialization.
        super.initialize();
    }
}
