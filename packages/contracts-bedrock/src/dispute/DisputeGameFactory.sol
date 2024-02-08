// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { ClonesWithImmutableArgs } from "@cwia/ClonesWithImmutableArgs.sol";
import { OwnableUpgradeable } from "@openzeppelin/contracts-upgradeable/access/OwnableUpgradeable.sol";
import { ISemver } from "src/universal/ISemver.sol";

import { IDisputeGame } from "src/dispute/interfaces/IDisputeGame.sol";
import { IDisputeGameFactory } from "src/dispute/interfaces/IDisputeGameFactory.sol";

import { LibGameId } from "src/dispute/lib/LibGameId.sol";

import "src/libraries/DisputeTypes.sol";
import "src/libraries/DisputeErrors.sol";

/// @title DisputeGameFactory
/// @notice A factory contract for creating `IDisputeGame` contracts. All created dispute games
///         are stored in both a mapping and an append only array. The timestamp of the creation
///         time of the dispute game is packed tightly into the storage slot with the address of
///         the dispute game. This is to make offchain discoverability of playable dispute games
///         easier.
contract DisputeGameFactory is OwnableUpgradeable, IDisputeGameFactory, ISemver {
    /// @dev Allows for the creation of clone proxies with immutable arguments.
    using ClonesWithImmutableArgs for address;

    /// @notice Semantic version.
    /// @custom:semver 0.1.0
    string public constant version = "0.1.0";

    /// @inheritdoc IDisputeGameFactory
    mapping(GameType => IDisputeGame) public gameImpls;

    /// @inheritdoc IDisputeGameFactory
    mapping(GameType => uint256) public initBonds;

    /// @notice Mapping of a hash of `gameType || rootClaim || extraData` to
    ///         the deployed `IDisputeGame` clone.
    /// @dev Note: `||` denotes concatenation.
    mapping(Hash => GameId) internal _disputeGames;

    /// @notice an append-only array of disputeGames that have been created.
    /// @dev this accessor is used by offchain game solvers to efficiently
    ///      track dispute games
    GameId[] internal _disputeGameList;

    /// @notice constructs a new DisputeGameFactory contract.
    constructor() OwnableUpgradeable() {
        initialize(address(0));
    }

    /// @notice Initializes the contract.
    /// @param _owner The owner of the contract.
    function initialize(address _owner) public initializer {
        __Ownable_init();
        _transferOwnership(_owner);
    }

    /// @inheritdoc IDisputeGameFactory
    function gameCount() external view returns (uint256 gameCount_) {
        gameCount_ = _disputeGameList.length;
    }

    /// @inheritdoc IDisputeGameFactory
    function games(
        GameType _gameType,
        Claim _rootClaim,
        bytes calldata _extraData
    )
        external
        view
        returns (IDisputeGame proxy_, Timestamp timestamp_)
    {
        Hash uuid = getGameUUID(_gameType, _rootClaim, _extraData);
        (, timestamp_, proxy_) = _disputeGames[uuid].unpack();
    }

    /// @inheritdoc IDisputeGameFactory
    function gameAtIndex(uint256 _index)
        external
        view
        returns (GameType gameType_, Timestamp timestamp_, IDisputeGame proxy_)
    {
        (gameType_, timestamp_, proxy_) = _disputeGameList[_index].unpack();
    }

    /// @inheritdoc IDisputeGameFactory
    function create(
        GameType _gameType,
        Claim _rootClaim,
        bytes calldata _extraData
    )
        external
        payable
        returns (IDisputeGame proxy_)
    {
        // Grab the implementation contract for the given `GameType`.
        IDisputeGame impl = gameImpls[_gameType];

        // If there is no implementation to clone for the given `GameType`, revert.
        if (address(impl) == address(0)) revert NoImplementation(_gameType);

        // If the required initialization bond is not met, revert.
        if (msg.value < initBonds[_gameType]) revert InsufficientBond();

        // Clone the implementation contract and initialize it with the given parameters.
        proxy_ = IDisputeGame(address(impl).clone(abi.encodePacked(_rootClaim, _extraData)));
        proxy_.initialize{ value: msg.value }();

        // Compute the unique identifier for the dispute game.
        Hash uuid = getGameUUID(_gameType, _rootClaim, _extraData);

        // If a dispute game with the same UUID already exists, revert.
        if (GameId.unwrap(_disputeGames[uuid]) != bytes32(0)) revert GameAlreadyExists(uuid);

        GameId id = LibGameId.pack(_gameType, Timestamp.wrap(uint64(block.timestamp)), proxy_);

        // Store the dispute game id in the mapping & emit the `DisputeGameCreated` event.
        _disputeGames[uuid] = id;
        _disputeGameList.push(id);
        emit DisputeGameCreated(address(proxy_), _gameType, _rootClaim);
    }

    /// @inheritdoc IDisputeGameFactory
    function getGameUUID(
        GameType _gameType,
        Claim _rootClaim,
        bytes calldata _extraData
    )
        public
        pure
        returns (Hash uuid_)
    {
        uuid_ = Hash.wrap(keccak256(abi.encode(_gameType, _rootClaim, _extraData)));
    }

    /// @inheritdoc IDisputeGameFactory
    function findLatestGames(
        GameType _gameType,
        uint256 _start,
        uint256 _n
    )
        external
        view
        returns (GameSearchResult[] memory games_)
    {
        // If the `_start` index is greater than or equal to the game array length or `_n == 0`, return an empty array.
        if (_start >= _disputeGameList.length || _n == 0) return games_;

        // Allocate enough memory for the full array, but start the array's length at `0`. We may not use all of the
        // memory allocated, but we don't know ahead of time the final size of the array.
        assembly {
            games_ := mload(0x40)
            mstore(0x40, add(games_, add(0x20, shl(0x05, _n))))
        }

        // Perform a reverse linear search for the `_n` most recent games of type `_gameType`.
        for (uint256 i = _start; i >= 0 && i <= _start;) {
            GameId id = _disputeGameList[i];
            (GameType gameType, Timestamp timestamp, IDisputeGame proxy) = id.unpack();

            if (gameType.raw() == _gameType.raw()) {
                // Increase the size of the `games_` array by 1.
                // SAFETY: We can safely lazily allocate memory here because we pre-allocated enough memory for the max
                //         possible size of the array.
                assembly {
                    mstore(games_, add(mload(games_), 0x01))
                }

                bytes memory extraData = proxy.extraData();
                Claim rootClaim = proxy.rootClaim();
                games_[games_.length - 1] = GameSearchResult({
                    index: i,
                    metadata: id,
                    timestamp: timestamp,
                    rootClaim: rootClaim,
                    extraData: extraData
                });
                if (games_.length >= _n) break;
            }

            unchecked {
                i--;
            }
        }
    }

    /// @inheritdoc IDisputeGameFactory
    function setImplementation(GameType _gameType, IDisputeGame _impl) external onlyOwner {
        gameImpls[_gameType] = _impl;
        emit ImplementationSet(address(_impl), _gameType);
    }

    /// @inheritdoc IDisputeGameFactory
    function setInitBond(GameType _gameType, uint256 _initBond) external onlyOwner {
        initBonds[_gameType] = _initBond;
        emit InitBondUpdated(_gameType, _initBond);
    }
}
