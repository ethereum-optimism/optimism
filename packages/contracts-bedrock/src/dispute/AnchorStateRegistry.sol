// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Contracts
import { Initializable } from "@openzeppelin/contracts/proxy/utils/Initializable.sol";

// Libraries
import { GameType, Proposal, Claim, GameStatus, Hash } from "src/dispute/lib/Types.sol";

// Interfaces
import { ISemver } from "interfaces/universal/ISemver.sol";
import { IFaultDisputeGame } from "interfaces/dispute/IFaultDisputeGame.sol";
import { IDisputeGame } from "interfaces/dispute/IDisputeGame.sol";
import { IDisputeGameFactory } from "interfaces/dispute/IDisputeGameFactory.sol";
import { ISystemConfig } from "interfaces/L1/ISystemConfig.sol";
import { ISuperchainConfig } from "interfaces/L1/ISuperchainConfig.sol";

/// @custom:proxied true
/// @title AnchorStateRegistry
/// @notice The AnchorStateRegistry is a contract that stores the latest "anchor" state for each available
///         FaultDisputeGame type. The anchor state is the latest state that has been proposed on L1 and was not
///         challenged within the challenge period. By using stored anchor states, new FaultDisputeGame instances can
///         be initialized with a more recent starting state which reduces the amount of required offchain computation.
contract AnchorStateRegistry is Initializable, ISemver {
    /// @notice Semantic version.
    /// @custom:semver 3.3.0
    string public constant version = "3.3.0";

    /// @notice The dispute game finality delay in seconds.
    uint256 internal immutable DISPUTE_GAME_FINALITY_DELAY_SECONDS;

    /// @notice Address of the SystemConfig contract.
    ISystemConfig public systemConfig;

    /// @notice Address of the DisputeGameFactory contract.
    IDisputeGameFactory public disputeGameFactory;

    /// @notice The game whose claim is currently being used as the anchor state.
    IFaultDisputeGame public anchorGame;

    /// @notice The starting anchor root.
    Proposal internal startingAnchorRoot;

    /// @notice Mapping of blacklisted dispute games.
    mapping(IDisputeGame => bool) public disputeGameBlacklist;

    /// @notice The respected game type.
    GameType public respectedGameType;

    /// @notice The retirement timestamp. All games created before or at this timestamp are
    ///         considered retired and are therefore not valid games. Retirement is used as a
    ///         blanket invalidation mechanism if games resolve incorrectly.
    uint64 public retirementTimestamp;

    /// @notice Emitted when an anchor state is updated.
    /// @param game Game that was used as the new anchor game.
    event AnchorUpdated(IFaultDisputeGame indexed game);

    /// @notice Emitted when the respected game type is set.
    /// @param gameType The new respected game type.
    event RespectedGameTypeSet(GameType gameType);

    /// @notice Emitted when the retirement timestamp is set.
    /// @param timestamp The new retirement timestamp.
    event RetirementTimestampSet(uint256 timestamp);

    /// @notice Emitted when a dispute game is blacklisted.
    /// @param disputeGame The dispute game that was blacklisted.
    event DisputeGameBlacklisted(IDisputeGame indexed disputeGame);

    /// @notice Thrown when an invalid anchor game is provided.
    error AnchorStateRegistry_InvalidAnchorGame();

    /// @notice Thrown when an unauthorized caller attempts to set the anchor state.
    error AnchorStateRegistry_Unauthorized();

    /// @param _disputeGameFinalityDelaySeconds The dispute game finality delay in seconds.
    constructor(uint256 _disputeGameFinalityDelaySeconds) {
        DISPUTE_GAME_FINALITY_DELAY_SECONDS = _disputeGameFinalityDelaySeconds;
        _disableInitializers();
    }

    /// @notice Initializes the contract.
    /// @param _systemConfig The address of the SystemConfig contract.
    /// @param _disputeGameFactory The address of the DisputeGameFactory contract.
    /// @param _startingAnchorRoot The starting anchor root.
    function initialize(
        ISystemConfig _systemConfig,
        IDisputeGameFactory _disputeGameFactory,
        Proposal memory _startingAnchorRoot,
        GameType _startingRespectedGameType
    )
        external
        initializer
    {
        systemConfig = _systemConfig;
        disputeGameFactory = _disputeGameFactory;
        startingAnchorRoot = _startingAnchorRoot;
        respectedGameType = _startingRespectedGameType;
        retirementTimestamp = uint64(block.timestamp);
    }

    /// @notice Returns whether the contract is paused.
    function paused() public view returns (bool) {
        return systemConfig.paused();
    }

    /// @notice Returns the SuperchainConfig contract.
    /// @return ISuperchainConfig The SuperchainConfig contract.
    function superchainConfig() public view returns (ISuperchainConfig) {
        return systemConfig.superchainConfig();
    }

    /// @notice Returns the dispute game finality delay in seconds.
    function disputeGameFinalityDelaySeconds() external view returns (uint256) {
        return DISPUTE_GAME_FINALITY_DELAY_SECONDS;
    }

    /// @notice Allows the Guardian to set the respected game type.
    /// @param _gameType The new respected game type.
    function setRespectedGameType(GameType _gameType) external {
        if (msg.sender != systemConfig.guardian()) revert AnchorStateRegistry_Unauthorized();
        respectedGameType = _gameType;
        emit RespectedGameTypeSet(_gameType);
    }

    /// @notice Allows the Guardian to update the retirement timestamp.
    function updateRetirementTimestamp() external {
        if (msg.sender != systemConfig.guardian()) revert AnchorStateRegistry_Unauthorized();
        retirementTimestamp = uint64(block.timestamp);
        emit RetirementTimestampSet(block.timestamp);
    }

    /// @notice Allows the Guardian to blacklist a dispute game.
    /// @param _disputeGame Dispute game to blacklist.
    function blacklistDisputeGame(IDisputeGame _disputeGame) external {
        if (msg.sender != systemConfig.guardian()) revert AnchorStateRegistry_Unauthorized();
        disputeGameBlacklist[_disputeGame] = true;
        emit DisputeGameBlacklisted(_disputeGame);
    }

    /// @custom:legacy
    /// @notice Returns the anchor root. Note that this is a legacy deprecated function and will
    ///         be removed in a future release. Use getAnchorRoot() instead. Anchor roots are no
    ///         longer stored per game type, so this function will return the same root for all
    ///         game types.
    function anchors(GameType /* unused */ ) external view returns (Hash, uint256) {
        return getAnchorRoot();
    }

    /// @notice Returns the current anchor root.
    /// @return The anchor root.
    function getAnchorRoot() public view returns (Hash, uint256) {
        // Return the starting anchor root if there is no anchor game.
        if (address(anchorGame) == address(0)) {
            return (startingAnchorRoot.root, startingAnchorRoot.l2SequenceNumber);
        }

        // Otherwise, return the anchor root.
        return (Hash.wrap(anchorGame.rootClaim().raw()), anchorGame.l2SequenceNumber());
    }

    /// @notice Determines whether a game is registered by checking that it was created by the
    ///         DisputeGameFactory and that it uses this AnchorStateRegistry.
    /// @param _game The game to check.
    /// @return Whether the game is registered.
    function isGameRegistered(IDisputeGame _game) public view returns (bool) {
        // Grab the game and game data.
        (GameType gameType, Claim rootClaim, bytes memory extraData) = _game.gameData();

        // Grab the verified address of the game based on the game data.
        (IDisputeGame factoryRegisteredGame,) =
            disputeGameFactory.games({ _gameType: gameType, _rootClaim: rootClaim, _extraData: extraData });

        // Grab the AnchorStateRegistry from the game. Awkward type conversion here but
        // IDisputeGame probably needs to have this function eventually anyway.
        address asr = address(IFaultDisputeGame(address(_game)).anchorStateRegistry());

        // Return whether the game is factory registered and uses this AnchorStateRegistry. We
        // check for both of these conditions because the game could be using a different
        // AnchorStateRegistry if the registry was updated at some point. We mitigate the risks of
        // an outdated AnchorStateRegistry by invalidating all previous games in the initializer of
        // this contract, but an explicit check avoids potential footguns in the future.
        return address(factoryRegisteredGame) == address(_game) && asr == address(this);
    }

    /// @notice Determines whether a game is of a respected game type.
    /// @param _game The game to check.
    /// @return Whether the game is of a respected game type.
    function isGameRespected(IDisputeGame _game) public view returns (bool) {
        // We don't do a try/catch here for legacy games because by the time this code is live on
        // mainnet, users won't be using legacy games anymore. Avoiding the try/catch simplifies
        // the logic.
        return _game.wasRespectedGameTypeWhenCreated();
    }

    /// @notice Determines whether a game is blacklisted.
    /// @param _game The game to check.
    /// @return Whether the game is blacklisted.
    function isGameBlacklisted(IDisputeGame _game) public view returns (bool) {
        return disputeGameBlacklist[_game];
    }

    /// @notice Determines whether a game is retired.
    /// @param _game The game to check.
    /// @return Whether the game is retired.
    function isGameRetired(IDisputeGame _game) public view returns (bool) {
        // Must be created after the retirementTimestamp. Note that this means all games created in
        // the same block as the retirementTimestamp are considered retired.
        return _game.createdAt().raw() <= retirementTimestamp;
    }

    /// @notice Returns whether a game is resolved.
    /// @param _game The game to check.
    /// @return Whether the game is resolved.
    function isGameResolved(IDisputeGame _game) public view returns (bool) {
        return _game.resolvedAt().raw() != 0
            && (_game.status() == GameStatus.DEFENDER_WINS || _game.status() == GameStatus.CHALLENGER_WINS);
    }

    /// @notice **READ THIS FUNCTION DOCUMENTATION CAREFULLY.**
    ///         Determines whether a game resolved properly and the game was not subject to any
    ///         invalidation conditions. The root claim of a proper game IS NOT guaranteed to be
    ///         valid. The root claim of a proper game CAN BE incorrect and still be a proper game.
    ///         DO NOT USE THIS FUNCTION ALONE TO DETERMINE IF A ROOT CLAIM IS VALID.
    /// @dev Note that isGameProper previously checked that the game type was equal to the
    ///      respected game type. However, it should be noted that it is possible for a game other
    ///      than the respected game type to resolve without being invalidated. Since isGameProper
    ///      exists to determine if a game has (or has not) been invalidated, we now allow any game
    ///      type to be considered a proper game. We enforce checks on the game type in
    ///      isGameClaimValid().
    /// @param _game The game to check.
    /// @return Whether the game is a proper game.
    function isGameProper(IDisputeGame _game) public view returns (bool) {
        // Must be registered in the DisputeGameFactory.
        if (!isGameRegistered(_game)) {
            return false;
        }

        // Must not be blacklisted.
        if (isGameBlacklisted(_game)) {
            return false;
        }

        // Must be created at or after the retirement timestamp.
        if (isGameRetired(_game)) {
            return false;
        }

        // Must not be paused, temporarily causes game to be considered improper.
        if (paused()) {
            return false;
        }

        return true;
    }

    /// @notice Returns whether a game is finalized.
    /// @param _game The game to check.
    /// @return Whether the game is finalized.
    function isGameFinalized(IDisputeGame _game) public view returns (bool) {
        // Game must be resolved.
        if (!isGameResolved(_game)) {
            return false;
        }

        // Game must be beyond the "airgap period" - time since resolution must be at least
        // "dispute game finality delay" seconds in the past.
        if (block.timestamp - _game.resolvedAt().raw() <= DISPUTE_GAME_FINALITY_DELAY_SECONDS) {
            return false;
        }

        return true;
    }

    /// @notice Returns whether a game's root claim is valid.
    /// @param _game The game to check.
    /// @return Whether the game's root claim is valid.
    function isGameClaimValid(IDisputeGame _game) public view returns (bool) {
        // Game must be a proper game.
        if (!isGameProper(_game)) {
            return false;
        }

        // Must be respected.
        if (!isGameRespected(_game)) {
            return false;
        }

        // Game must be finalized.
        if (!isGameFinalized(_game)) {
            return false;
        }

        // Game must be resolved in favor of the defender.
        if (_game.status() != GameStatus.DEFENDER_WINS) {
            return false;
        }

        return true;
    }

    /// @notice Updates the anchor game.
    /// @param _game New candidate anchor game.
    function setAnchorState(IDisputeGame _game) public {
        // Convert game to FaultDisputeGame.
        // We can't use FaultDisputeGame in the interface because this function is called from the
        // FaultDisputeGame contract which can't import IFaultDisputeGame by convention. We should
        // likely introduce a new interface (e.g., StateDisputeGame) that can act as a more useful
        // version of IDisputeGame in the future.
        IFaultDisputeGame game = IFaultDisputeGame(address(_game));

        // Check if the candidate game claim is valid.
        if (!isGameClaimValid(game)) {
            revert AnchorStateRegistry_InvalidAnchorGame();
        }

        // Must be newer than the current anchor game.
        (, uint256 anchorL2BlockNumber) = getAnchorRoot();
        if (game.l2SequenceNumber() <= anchorL2BlockNumber) {
            revert AnchorStateRegistry_InvalidAnchorGame();
        }

        // Update the anchor game.
        anchorGame = game;
        emit AnchorUpdated(game);
    }
}
