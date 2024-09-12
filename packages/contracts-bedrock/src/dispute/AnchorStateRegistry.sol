// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Contracts
import { Initializable } from "@openzeppelin/contracts/proxy/utils/Initializable.sol";

// Libraries
import { Unauthorized } from "src/libraries/errors/CommonErrors.sol";
import { UnregisteredGame, InvalidGameStatus, OldGame, BlacklistedGame } from "src/dispute/lib/Errors.sol";
import "src/dispute/lib/Types.sol";

// Interfaces
import { ISemver } from "src/universal/interfaces/ISemver.sol";
import { IAnchorStateRegistry } from "src/dispute/interfaces/IAnchorStateRegistry.sol";
import { IFaultDisputeGame } from "src/dispute/interfaces/IFaultDisputeGame.sol";
import { IDisputeGame } from "src/dispute/interfaces/IDisputeGame.sol";
import { IDisputeGameFactory } from "src/dispute/interfaces/IDisputeGameFactory.sol";
import { ISuperchainConfig } from "src/L1/interfaces/ISuperchainConfig.sol";
import { IOptimismPortal2 } from "src/L1/interfaces/IOptimismPortal2.sol";

/// @custom:proxied true
/// @title AnchorStateRegistry
/// @notice The AnchorStateRegistry is a contract that stores the latest "anchor" state for each available
///         FaultDisputeGame type. The anchor state is the latest state that has been proposed on L1 and was not
///         challenged within the challenge period. By using stored anchor states, new FaultDisputeGame instances can
///         be initialized with a more recent starting state which reduces the amount of required offchain computation.
contract AnchorStateRegistry is Initializable, IAnchorStateRegistry, ISemver {
    /// @notice Describes an initial anchor state for a game type.
    struct StartingAnchorRoot {
        GameType gameType;
        OutputRoot outputRoot;
    }

    /// @notice Semantic version.
    /// @custom:semver 2.1.0-beta.3
    string public constant version = "2.1.0-beta.3";

    /// @notice DisputeGameFactory address.
    IDisputeGameFactory internal immutable DISPUTE_GAME_FACTORY;

    /// @inheritdoc IAnchorStateRegistry
    mapping(GameType => OutputRoot) public anchors;

    /// @notice Address of the SuperchainConfig contract.
    ISuperchainConfig public superchainConfig;

    /// @notice The address of the OptimismPortal2 contract.
    IOptimismPortal2 public optimismPortal2;

    /// @param _disputeGameFactory DisputeGameFactory address.
    constructor(IDisputeGameFactory _disputeGameFactory) {
        DISPUTE_GAME_FACTORY = _disputeGameFactory;
        _disableInitializers();
    }

    /// @notice Initializes the contract.
    /// @param _startingAnchorRoots An array of starting anchor roots.
    /// @param _optimismPortal2 The address of the OptimismPortal2 contract.
    /// @param _superchainConfig The address of the SuperchainConfig contract.
    function initialize(
        StartingAnchorRoot[] memory _startingAnchorRoots,
        IOptimismPortal2 _optimismPortal2,
        ISuperchainConfig _superchainConfig
    )
        public
        initializer
    {
        for (uint256 i = 0; i < _startingAnchorRoots.length; i++) {
            StartingAnchorRoot memory startingAnchorRoot = _startingAnchorRoots[i];
            anchors[startingAnchorRoot.gameType] = startingAnchorRoot.outputRoot;
        }
        superchainConfig = _superchainConfig;
        optimismPortal2 = _optimismPortal2;
    }

    /// @inheritdoc IAnchorStateRegistry
    function disputeGameFactory() external view returns (IDisputeGameFactory) {
        return DISPUTE_GAME_FACTORY;
    }

    /// @inheritdoc IAnchorStateRegistry
    function tryUpdateAnchorState() external {
        _tryUpdateAnchorState(IFaultDisputeGame(msg.sender), false);
    }

    /// @inheritdoc IAnchorStateRegistry
    function setAnchorState(IFaultDisputeGame _game) external {
        if (msg.sender != superchainConfig.guardian()) revert Unauthorized();
        _tryUpdateAnchorState(_game, true);
    }

    /// @notice Attempts to update the anchor state.
    /// @param _game Game to use to update the anchor state.
    /// @param _override Whether or not to override the anchor state if the provided game is older.
    function _tryUpdateAnchorState(IFaultDisputeGame _game, bool _override) internal {
        // Get the metadata of the game.
        (GameType gameType, Claim rootClaim, bytes memory extraData) = _game.gameData();

        // Grab the verified address of the game based on the game data.
        // slither-disable-next-line unused-return
        (IDisputeGame factoryRegisteredGame,) =
            DISPUTE_GAME_FACTORY.games({ _gameType: gameType, _rootClaim: rootClaim, _extraData: extraData });

        // Check that the game was actually created by the factory.
        if (address(factoryRegisteredGame) != address(_game)) revert UnregisteredGame();

        // Check that the game resolved in favor of the defender.
        if (_game.status() != GameStatus.DEFENDER_WINS) revert InvalidGameStatus();

        // Check if the game is older than the current anchor state (or bypass if overriding).
        if (!_override && _game.l2BlockNumber() <= anchors[gameType].l2BlockNumber) revert OldGame();

        // Check if the game is blacklisted.
        if (optimismPortal2.disputeGameBlacklist(_game)) revert BlacklistedGame();

        // Update the anchor.
        anchors[gameType] =
            OutputRoot({ l2BlockNumber: _game.l2BlockNumber(), root: Hash.wrap(_game.rootClaim().raw()) });
    }
}
