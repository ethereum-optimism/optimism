// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import {GameType, Hash, GameStatus, Claim, Timestamp, GameTypes} from "src/dispute/lib/Types.sol";

import {Clone} from "@solady/utils/Clone.sol";
import {Bytes} from "src/libraries/Bytes.sol";
import {IDelayedWETH} from "interfaces/dispute/IDelayedWETH.sol";
import {ISemver} from "interfaces/universal/ISemver.sol";
import {IAnchorStateRegistry} from "interfaces/dispute/IAnchorStateRegistry.sol";
import {IDisputeGame} from "interfaces/dispute/IDisputeGame.sol";
import {IDisputeGameFactory} from "interfaces/dispute/IDisputeGameFactory.sol";
import {
    ClaimAlreadyExists,
    AnchorRootNotFound,
    AlreadyInitialized,
    UnexpectedRootClaim,
    BadExtraData,
    GameNotResolved,
    IncorrectBondAmount,
    GameNotInProgress
} from "src/dispute/lib/Errors.sol";

contract AggregateDisputeGame is Clone, ISemver {
    /// @notice The game type ID.
    GameType internal immutable GAME_TYPE;

    /// @notice Flag for the `initialize` function to prevent re-initialization.
    bool internal initialized;

    /// @notice The anchor state registry.
    IAnchorStateRegistry public immutable ANCHOR_STATE_REGISTRY;

    /// @notice The chain ID of the L2 network this contract argues about.
    uint256 internal immutable L2_CHAIN_ID;

    /// @notice Returns the current status of the game.
    GameStatus public status;

    GameType public immutable FAST_FINALITY_GAME_TYPE;

    /// @notice WETH contract for holding ETH.
    IDelayedWETH internal immutable WETH;

    IDisputeGameFactory internal immutable DISPUTE_GAME_FACTORY;

    address public creator;

    GameType[] public configuredGameTypes;

    mapping(GameType => uint32) public extraDataLengths;
    uint256 internal expectedExtraDataLength;

    mapping(GameType => bytes) public extraDataForGames;

    /// @notice The starting timestamp of the game
    Timestamp public createdAt;

    /// @notice A boolean for whether or not the game type was respected when the game was created.
    bool public wasRespectedGameTypeWhenCreated;

    IDisputeGame[] public games;

    struct GameConstructorParams {
        GameType gameType;
        IAnchorStateRegistry anchorStateRegistry;
        GameType fastFinalityGameType;
        IDelayedWETH weth;
        IDisputeGameFactory dgf;
        GameType[] gameTypes;
        uint32[] extraDataLengths;
        uint256 l2ChainId;
    }

    error InvalidGameTypes();
    error InvalidExtraData();
    error InvalidUnderlyingGame();

    constructor(GameConstructorParams memory _params) {
        GAME_TYPE = _params.gameType;
        ANCHOR_STATE_REGISTRY = _params.anchorStateRegistry;
        FAST_FINALITY_GAME_TYPE = _params.fastFinalityGameType;
        WETH = _params.weth;
        DISPUTE_GAME_FACTORY = _params.dgf;

        require(
            _params.gameTypes.length == _params.extraDataLengths.length, "mismatch game type and extra data lengths"
        );
        for (uint256 i = 0; i < _params.gameTypes.length; i++) {
            GameType gt = _params.gameTypes[i];
            uint32 len = _params.extraDataLengths[i];
            extraDataLengths[gt] = len;
            configuredGameTypes.push(gt);
            expectedExtraDataLength += len;
        }
        L2_CHAIN_ID = _params.l2ChainId;
    }

    /// @notice Semantic version.
    /// @custom:semver 0.0.1
    function version() public pure virtual returns (string memory) {
        return "0.0.1";
    }

    function initialize() public payable virtual {
        // INVARIANT: The game must not have already been initialized.
        if (initialized) revert AlreadyInitialized();

        // Revert if the calldata size is not the expected length.
        //
        // This is to prevent adding extra or omitting bytes from to `extraData` that result in a different game UUID
        // in the factory, but are not used by the game, which would allow for multiple dispute games for the same
        // output proposal to be created.
        //
        // Expected length: 122 bytes
        // - 4 bytes selector
        // - 20 bytes creator address
        // - 32 bytes root claim
        // - 32 bytes l1 head
        // - expectedExtraDataLen bytes
        // - 2 bytes CWIA length
        if (msg.data.length != 4 + 20 + 32 + 32 + expectedExtraDataLength + 2) {
            revert BadExtraData();
        }

        // Grab the latest anchor root.
        (Hash root, uint256 rootBlockNumber) = ANCHOR_STATE_REGISTRY.getAnchorRoot();

        // Do not allow the game to be initialized if the root claim corresponds to a block at or before the
        // configured starting block number.
        if (l2BlockNumber() <= rootBlockNumber) {
            revert UnexpectedRootClaim(rootClaim());
        }

        // Should only happen if this is a new game type that hasn't been set up yet.
        if (root.raw() == bytes32(0)) revert AnchorRootNotFound();

        if (msg.value != getRequiredBond()) revert IncorrectBondAmount();

        initialized = true;
        // Set the game's starting timestamp
        createdAt = Timestamp.wrap(uint64(block.timestamp));
        creator = gameCreator();

        // Set whether the game type was respected when the game was created.
        wasRespectedGameTypeWhenCreated =
            GameType.unwrap(ANCHOR_STATE_REGISTRY.respectedGameType()) == GameType.unwrap(GAME_TYPE);

        WETH.deposit{value: msg.value}();

        (GameType[] memory types, bytes[] memory data) = _decodeExtraData(extraData());
        for (uint256 i = 0; i < types.length; i++) {
            extraDataForGames[types[i]] = data[i];
        }
        if (extraDataForGames[FAST_FINALITY_GAME_TYPE].length == 0) {
            revert InvalidExtraData();
        }
    }

    function resolve() public virtual returns (GameStatus status_) {
        // INVARIANT: Resolution cannot occur unless the game is currently in progress.
        if (status != GameStatus.IN_PROGRESS) revert GameNotInProgress();

        // Determine if there are any games besides the fast-finality game
        bool hasOtherGames;
        IDisputeGame fastGame;
        for (uint256 i = 0; i < games.length; i++) {
            GameType gt = games[i].gameType();
            // TODO: Is it possible to have multiple fast-finality games?
            if (gt.raw() == FAST_FINALITY_GAME_TYPE.raw()) {
                fastGame = games[i];
                continue;
            }
            hasOtherGames = true;
        }

        if (!hasOtherGames && fastGame != IDisputeGame(address(0))) {
            status_ = fastGame.status();
            if (status_ != GameStatus.IN_PROGRESS) {
                status = status_;
            }
            return status_;
        }

        bool anyInProgress;
        bool anyChallengerWins;
        bool allDefenderWins = true;
        bool hasMatchingProposal;

        for (uint256 i = 0; i < games.length; i++) {
            IDisputeGame game = games[i];
            if (_hasMatchingProposal(game)) {
                hasMatchingProposal = true;
            }
            GameStatus s = game.status();

            if (s == GameStatus.IN_PROGRESS) {
                anyInProgress = true;
            } else if (s == GameStatus.CHALLENGER_WINS) {
                anyChallengerWins = true;
            }
            if (s != GameStatus.DEFENDER_WINS) {
                allDefenderWins = false;
            }
        }

        if (anyChallengerWins) {
            status = GameStatus.CHALLENGER_WINS;
            return GameStatus.CHALLENGER_WINS;
        }
        if (anyInProgress) {
            return GameStatus.IN_PROGRESS;
        }
        if (allDefenderWins) {
            status = GameStatus.DEFENDER_WINS;
            return GameStatus.DEFENDER_WINS;
        }
        return GameStatus.IN_PROGRESS;
    }

    /// @notice Getter for the required bond.
    /// @dev This is the fast-finality bond that must be paid to the dispute game contract to participate in the game.
    /// @return The required bond.
    function getRequiredBond() public view virtual returns (uint256) {
        // TODO: use min required bond for all games as a floor
        return 0.001 ether;
    }

    function claimCredit() public virtual {
        if (status == GameStatus.IN_PROGRESS) revert GameNotResolved();

        uint256 fastFinalityBond = getRequiredBond();
        address bondRecipient = creator;

        for (uint256 i = 0; i < configuredGameTypes.length; i++) {
            GameType gt = configuredGameTypes[i];
            IDisputeGame game = _findGame(gt);
            GameStatus gameStatus = game.status();
            if (gameStatus == GameStatus.CHALLENGER_WINS) {
                address payable burnAddress = payable(0x000000000000000000000000000000000000dEaD);
                // TODO: for IC burn only half the bond. Reward the caller the rest.
                bondRecipient = burnAddress;
                break;
            }
        }

        WETH.unlock(bondRecipient, fastFinalityBond);
        WETH.withdraw(bondRecipient, fastFinalityBond);

        try ANCHOR_STATE_REGISTRY.setAnchorState(IDisputeGame(address(this))) {} catch {}

        // TODO: check if game is "proper" and implement bond distribution for refunds
    }

    function addUnderlyingGame(uint256 _gameIndex) public {
        (,, IDisputeGame game) = DISPUTE_GAME_FACTORY.gameAtIndex(_gameIndex);
        if (Hash.unwrap(game.l1Head()) != Hash.unwrap(l1Head())) {
            revert InvalidUnderlyingGame();
        }
        if (
            Claim.unwrap(rootClaim()) != Claim.unwrap(game.rootClaim()) && l2SequenceNumber() != game.l2SequenceNumber()
        ) {
            revert InvalidUnderlyingGame();
        }
        for (uint256 i = 0; i < configuredGameTypes.length; i++) {
            if (configuredGameTypes[i].raw() == game.gameType().raw()) {
                games.push(game);
                return;
            }
        }
        revert InvalidGameTypes();
    }

    /// @notice Getter for the creator of the dispute game.
    /// @dev `clones-with-immutable-args` argument #1
    /// @return creator_ The creator of the dispute game.
    function gameCreator() public pure returns (address creator_) {
        creator_ = _getArgAddress(0);
    }

    /// @notice Getter for the root claim.
    /// @dev `clones-with-immutable-args` argument #2
    /// @return rootClaim_ The root claim of the DisputeGame.
    function rootClaim() public pure returns (Claim rootClaim_) {
        rootClaim_ = Claim.wrap(_getArgBytes32(20));
    }

    /// @notice Getter for the parent hash of the L1 block when the dispute game was created.
    /// @dev `clones-with-immutable-args` argument #3
    /// @return l1Head_ The parent hash of the L1 block when the dispute game was created.
    function l1Head() public pure returns (Hash l1Head_) {
        l1Head_ = Hash.wrap(_getArgBytes32(52));
    }

    /// @notice The l2BlockNumber of the disputed output root in the `L2OutputOracle`.
    function l2BlockNumber() public pure returns (uint256 l2BlockNumber_) {
        l2BlockNumber_ = _getArgUint256(84);
    }

    /// @notice The l2SequenceNumber of the disputed output root in the `L2OutputOracle` (in this case - block number).
    function l2SequenceNumber() public pure returns (uint256 l2SequenceNumber_) {
        l2SequenceNumber_ = l2BlockNumber();
    }

    /// @notice Returns the WETH contract for holding ETH.
    function weth() external view returns (IDelayedWETH weth_) {
        weth_ = WETH;
    }

    /// @notice Returns the anchor state registry contract.
    function anchorStateRegistry() external view returns (IAnchorStateRegistry registry_) {
        registry_ = ANCHOR_STATE_REGISTRY;
    }

    /// @notice Returns the chain ID of the L2 network this contract argues about.
    function l2ChainId() external view returns (uint256 l2ChainId_) {
        l2ChainId_ = L2_CHAIN_ID;
    }

    /// @notice Getter for the extra data.
    /// @dev `clones-with-immutable-args` argument #4
    /// @return extraData_ Any extra data supplied to the dispute game contract by the creator.
    function extraData() public view returns (bytes memory extraData_) {
        // The extra data starts at the second word within the cwia calldata and
        // is 56 bytes long.
        extraData_ = _getArgBytes(84, expectedExtraDataLength);
    }

    function _findGame(GameType _gameType) internal view returns (IDisputeGame game_) {
        for (uint256 i = 0; i < games.length; i++) {
            if (games[i].gameType().raw() == _gameType.raw()) {
                return games[i];
            }
        }
        require(false, "game not found");
    }

    function _hasMatchingProposal(IDisputeGame _game) internal pure returns (bool) {
        return Claim.unwrap(rootClaim()) == Claim.unwrap(_game.rootClaim()) && l2SequenceNumber() == _game.l2SequenceNumber();
    }

    function _decodeExtraData(bytes memory _extraData)
        internal
        view
        returns (GameType[] memory _decodedTypes, bytes[] memory _decodedData)
    {
        uint256 off;
        uint256 count;
        while (off < _extraData.length) {
            if (off + 4 > _extraData.length) revert InvalidExtraData();
            GameType gt = GameType.wrap(_u32be(_extraData, off));
            off += 4;

            uint256 len = extraDataLengths[gt];
            if (len == 0) revert InvalidExtraData();
            if (off + len > _extraData.length) revert InvalidExtraData();
            off += len;

            unchecked {
                count++;
            }
        }
        if (off != _extraData.length) revert InvalidExtraData();

        _decodedTypes = new GameType[](count);
        _decodedData = new bytes[](count);

        off = 0;
        for (uint256 i; i < count;) {
            GameType gt = GameType.wrap(_u32be(_extraData, off));
            off += 4;

            uint256 len = extraDataLengths[gt];
            bytes memory data = Bytes.slice(_extraData, off, len);
            off += len;

            _decodedTypes[i] = gt;
            _decodedData[i] = data;

            unchecked {
                i++;
            }
        }
    }

    function _u32be(bytes memory b, uint256 off) private pure returns (uint32 v) {
        // big-endian: b[off] is the most significant byte
        unchecked {
            v = (uint32(uint8(b[off])) << 24) | (uint32(uint8(b[off + 1])) << 16) | (uint32(uint8(b[off + 2])) << 8)
                | uint32(uint8(b[off + 3]));
        }
    }
}
