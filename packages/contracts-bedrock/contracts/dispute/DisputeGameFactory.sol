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

/**
 * @title DisputeGameFactory
 * @notice A factory contract for creating `IDisputeGame` contracts.
 */
contract DisputeGameFactory is OwnableUpgradeable, IDisputeGameFactory, IVersioned {
    /**
     * @dev Allows for the creation of clone proxies with immutable arguments.
     */
    using ClonesWithImmutableArgs for address;

    /**
     * @inheritdoc IDisputeGameFactory
     */
    mapping(GameType => IDisputeGame) public gameImpls;

    /**
     * @notice Mapping of a hash of `gameType . rootClaim . extraData` to
     *         the deployed `IDisputeGame` clone.
     * @dev Note: `.` denotes concatenation.
     */
    mapping(Hash => IDisputeGame) internal disputeGames;

    /**
     * @notice An append-only array of disputeGames that have been created.
     * @dev This accessor is used by offchain game solvers to efficiently
     *      track dispute games
     */
    IDisputeGame[] public disputeGameList;

    /**
     * @notice Constructs a new DisputeGameFactory contract. Set the owner
     *         to `address(0)` to prevent accidental usage of the implementation.
     */
    constructor() OwnableUpgradeable() {
        _transferOwnership(address(0));
    }

    /**
     * @notice Initializes the contract.
     * @param _owner The owner of the contract.
     */
    function initialize(address _owner) external initializer {
        __Ownable_init();
        _transferOwnership(_owner);
    }

    /**
     * @inheritdoc IVersioned
     */
    function version() external pure returns (string memory) {
        return "0.0.1";
    }

    /**
     * @inheritdoc IDisputeGameFactory
     */
    function gameCount() external view returns (uint256 _gameCount) {
        _gameCount = disputeGameList.length;
    }

    /**
     * @inheritdoc IDisputeGameFactory
     */
    function games(
        GameType gameType,
        Claim rootClaim,
        bytes calldata extraData
    ) external view returns (IDisputeGame _proxy) {
        return disputeGames[getGameUUID(gameType, rootClaim, extraData)];
    }

    /**
     * @inheritdoc IDisputeGameFactory
     */
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
        if (address(disputeGames[uuid]) != address(0)) {
            revert GameAlreadyExists(uuid);
        }

        // Store the dispute game in the mapping & emit the `DisputeGameCreated` event.
        disputeGames[uuid] = proxy;
        disputeGameList.push(proxy);
        emit DisputeGameCreated(address(proxy), gameType, rootClaim);
    }

    /**
     * @inheritdoc IDisputeGameFactory
     */
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

    /**
     * @inheritdoc IDisputeGameFactory
     */
    function setImplementation(GameType gameType, IDisputeGame impl) external onlyOwner {
        gameImpls[gameType] = impl;
        emit ImplementationSet(address(impl), gameType);
    }
}
