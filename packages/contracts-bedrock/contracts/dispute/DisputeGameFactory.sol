// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

import "../libraries/DisputeTypes.sol";
import "../libraries/DisputeErrors.sol";

import { ClonesWithImmutableArgs } from "@cwia/ClonesWithImmutableArgs.sol";
import {
    OwnableUpgradeable
} from "@openzeppelin/contracts-upgradeable/access/OwnableUpgradeable.sol";

import { IDisputeGame } from "./interfaces/IDisputeGame.sol";
import { IDisputeGameFactory } from "./interfaces/IDisputeGameFactory.sol";
import { IVersioned } from "./interfaces/IVersioned.sol";

/// @title DisputeGameFactory
/// @notice A factory contract for creating `IDisputeGame` contracts. All created dispute games
///         are stored in both a mapping and an append only array. The timestamp of the creation
///         time of the dispute game is packed tightly into the storage slot with the address of
///         the dispute game. This is to make offchain discoverability of playable dispute games
///         easier.
contract DisputeGameFactory is OwnableUpgradeable, IDisputeGameFactory, IVersioned {
    /// @dev Allows for the creation of clone proxies with immutable arguments.
    using ClonesWithImmutableArgs for address;

    /// @inheritdoc IDisputeGameFactory
    mapping(GameType => IDisputeGame) public gameImpls;

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

    /// @inheritdoc IVersioned
    /// @custom:semver 0.0.2
    function version() external pure returns (string memory) {
        return "0.0.2";
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
    ) external view returns (IDisputeGame proxy_, uint256 timestamp_) {
        Hash uuid = getGameUUID(_gameType, _rootClaim, _extraData);
        GameId slot = _disputeGames[uuid];
        (address addr, uint256 timestamp) = _unpackSlot(slot);
        proxy_ = IDisputeGame(addr);
        timestamp_ = timestamp;
    }

    /// @inheritdoc IDisputeGameFactory
    function gameAtIndex(uint256 _index)
        external
        view
        returns (IDisputeGame proxy_, uint256 timestamp_)
    {
        GameId slot = _disputeGameList[_index];
        (address addr, uint256 timestamp) = _unpackSlot(slot);
        proxy_ = IDisputeGame(addr);
        timestamp_ = timestamp;
    }

    /// @inheritdoc IDisputeGameFactory
    function create(
        GameType gameType,
        Claim rootClaim,
        bytes calldata extraData
    ) external returns (IDisputeGame proxy) {
        // Grab the implementation contract for the given `GameType`.
        IDisputeGame impl = gameImpls[gameType];

        // If there is no implementation to clone for the given `GameType`, revert.
        if (address(impl) == address(0)) {
            revert NoImplementation(gameType);
        }

        // Clone the implementation contract and initialize it with the given parameters.
        proxy = IDisputeGame(address(impl).clone(abi.encodePacked(rootClaim, extraData)));
        proxy.initialize();

        // Compute the unique identifier for the dispute game.
        Hash uuid = getGameUUID(gameType, rootClaim, extraData);

        // If a dispute game with the same UUID already exists, revert.
        if (GameId.unwrap(_disputeGames[uuid]) != bytes32(0)) {
            revert GameAlreadyExists(uuid);
        }

        GameId slot = _packSlot(address(proxy), block.timestamp);

        // Store the dispute game in the mapping & emit the `DisputeGameCreated` event.
        _disputeGames[uuid] = slot;
        _disputeGameList.push(slot);
        emit DisputeGameCreated(address(proxy), gameType, rootClaim);
    }

    /// @inheritdoc IDisputeGameFactory
    function getGameUUID(
        GameType gameType,
        Claim rootClaim,
        bytes memory extraData
    ) public pure returns (Hash _uuid) {
        assembly {
            // Grab the offsets of the other memory locations we will need to temporarily overwrite.
            let gameTypeOffset := sub(extraData, 0x60)
            let rootClaimOffset := add(gameTypeOffset, 0x20)
            let pointerOffset := add(rootClaimOffset, 0x20)

            // Copy the memory that we will temporarily overwrite onto the stack
            // so we can restore it later
            let tempA := mload(gameTypeOffset)
            let tempB := mload(rootClaimOffset)
            let tempC := mload(pointerOffset)

            // Overwrite the memory with the data we want to hash
            mstore(gameTypeOffset, gameType)
            mstore(rootClaimOffset, rootClaim)
            mstore(pointerOffset, 0x60)

            // Compute the length of the memory to hash
            // `0x60 + 0x20 + extraData.length` rounded to the *next* multiple of 32.
            let hashLen := and(add(mload(extraData), 0x9F), not(0x1F))

            // Hash the memory to produce the UUID digest
            _uuid := keccak256(gameTypeOffset, hashLen)

            // Restore the memory prior to `extraData`
            mstore(gameTypeOffset, tempA)
            mstore(rootClaimOffset, tempB)
            mstore(pointerOffset, tempC)
        }
    }

    /// @inheritdoc IDisputeGameFactory
    function setImplementation(GameType gameType, IDisputeGame impl) external onlyOwner {
        gameImpls[gameType] = impl;
        emit ImplementationSet(address(impl), gameType);
    }

    /// @dev Packs an address and a uint256 into a single bytes32 slot. This
    ///      is only safe for up to uint96 values.
    function _packSlot(address _addr, uint256 _num) internal pure returns (GameId slot_) {
        assembly {
            slot_ := or(shl(0xa0, _num), _addr)
        }
    }

    /// @dev Unpacks an address and a uint256 from a single bytes32 slot.
    function _unpackSlot(GameId _slot) internal pure returns (address addr_, uint256 num_) {
        assembly {
            addr_ := and(_slot, 0xffffffffffffffffffffffffffffffffffffffff)
            num_ := shr(0xa0, _slot)
        }
    }
}
