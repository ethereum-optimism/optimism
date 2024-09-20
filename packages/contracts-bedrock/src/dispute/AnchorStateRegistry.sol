// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Initializable } from "@openzeppelin/contracts/proxy/utils/Initializable.sol";
import { ISemver } from "src/universal/interfaces/ISemver.sol";

import { IAnchorStateRegistry } from "src/dispute/interfaces/IAnchorStateRegistry.sol";
import { IFaultDisputeGame } from "src/dispute/interfaces/IFaultDisputeGame.sol";
import { IDisputeGame } from "src/dispute/interfaces/IDisputeGame.sol";
import { IDisputeGameFactory } from "src/dispute/interfaces/IDisputeGameFactory.sol";
import { ISuperchainConfig } from "src/L1/interfaces/ISuperchainConfig.sol";

import "src/dispute/lib/Types.sol";
import { Unauthorized } from "src/libraries/errors/CommonErrors.sol";
import { UnregisteredGame, InvalidGameStatus, GameNotNewer } from "src/dispute/lib/Errors.sol";

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
    /// @custom:semver 2.0.1-beta.2
    string public constant version = "2.0.1-beta.2";

    /// @notice DisputeGameFactory address.
    IDisputeGameFactory internal immutable DISPUTE_GAME_FACTORY;

    /// @notice The delay between when a dispute game is resolved and when a withdrawal proven against it may be
    ///         finalized.
    uint256 internal immutable DISPUTE_GAME_FINALITY_DELAY_SECONDS;

    /// @inheritdoc IAnchorStateRegistry
    mapping(GameType => OutputRoot) public anchors;

    /// @notice Address of the SuperchainConfig contract.
    ISuperchainConfig public superchainConfig;

    /// @notice Mapping from game to whether it has been blacklisted.
    mapping (IFaultDisputeGame => bool) internal blacklisted;

    /// @notice Mapping from game type to a linked list of games that resolved in favor of the defender.
    mapping (GameType => IFaultDisputeGame[]) internal games;

    /// @notice Mapping from game type to the index of the game in games array that is the anchor.
    mapping (GameType => uint256) internal anchorGameIndex;

    /// @param _disputeGameFactory DisputeGameFactory address.
    constructor(IDisputeGameFactory _disputeGameFactory, uint256 _disputeGameFinalityDelaySeconds) {
        DISPUTE_GAME_FACTORY = _disputeGameFactory;
        DISPUTE_GAME_FINALITY_DELAY_SECONDS = _disputeGameFinalityDelaySeconds;
        _disableInitializers();
    }

    /// @notice Initializes the contract.
    /// @param _startingAnchorRoots An array of starting anchor roots.
    /// @param _superchainConfig The address of the SuperchainConfig contract.
    function initialize(
        StartingAnchorRoot[] memory _startingAnchorRoots,
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
    }

    /// @inheritdoc IAnchorStateRegistry
    function disputeGameFactory() external view returns (IDisputeGameFactory) {
        return DISPUTE_GAME_FACTORY;
    }

    /// @inheritdoc IAnchorStateRegistry
    function tryUpdateAnchorState() external {
        IFaultDisputeGame game = IFaultDisputeGame(msg.sender);
        (GameType gameType, Claim rootClaim, bytes memory extraData) = game.gameData();

        // Grab the verified address of the game based on the game data.
        // slither-disable-next-line unused-return
        (IDisputeGame factoryRegisteredGame,) =
            DISPUTE_GAME_FACTORY.games({ _gameType: gameType, _rootClaim: rootClaim, _extraData: extraData });

        // Must be a valid game.
        if (address(factoryRegisteredGame) != address(game)) revert UnregisteredGame();

        // Add this game to the list of successful games as long as it resolved for the defender.
        if (game.status() == GameStatus.DEFENDER_WINS) {
            games[gameType].push(game);
        }

        // Binary search for the most recent game that has passed the finality delay (air-gap) and
        // has not been blacklisted.
        uint256 start = anchorGameIndex[gameType];
        uint256 end = games[gameType].length;
        uint256 mid;
        uint256 res = type(uint256).max;
        while (start < end) {
            mid = (start + end) / 2;
            IFaultDisputeGame tgt = games[gameType][mid];
            if (block.timestamp >= tgt.resolvedAt().raw() + DISPUTE_GAME_FINALITY_DELAY_SECONDS) {
                // If the game is not blacklisted, it is a potential result.
                if (!blacklisted[tgt]) {
                    res = mid;
                }

                // Continue searching anyway.
                start = mid + 1;
            } else {
                end = mid;
            }
        }

        // If a valid game was found, update the anchor state.
        if (res != type(uint256).max) {
            IFaultDisputeGame found = games[gameType][res];
            anchors[gameType] = OutputRoot({ l2BlockNumber: found.l2BlockNumber(), root: Hash.wrap(found.rootClaim().raw()) });
        }
    }
}
