// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

import { Ownable } from "@openzeppelin/contracts/access/Ownable.sol";
import { ClonesWithImmutableArgs } from "@cwia/ClonesWithImmutableArgs.sol";

import { Claim } from "../libraries/DisputeTypes.sol";
import { Hash } from "../libraries/DisputeTypes.sol";
import { GameType } from "../libraries/DisputeTypes.sol";

import { NoImplementation } from "../libraries/DisputeErrors.sol";
import { GameAlreadyExists } from "../libraries/DisputeErrors.sol";

import { IDisputeGame } from "./IDisputeGame.sol";
import { IDisputeGameFactory } from "./IDisputeGameFactory.sol";

/**
 * @title DisputeGameFactory
 * @notice A factory contract for creating `IDisputeGame` contracts.
 */
contract DisputeGameFactory is Ownable, IDisputeGameFactory {
    /**
     * @dev Allows for the creation of clone proxies with immutable arguments.
     */
    using ClonesWithImmutableArgs for address;

    /**
     * @notice Mapping of `GameType`s to their respective `IDisputeGame` implementations.
     */
    mapping(GameType => IDisputeGame) public gameImpls;

    /**
     * @notice Mapping of a hash of `gameType . rootClaim . extraData` to
     *         the deployed `IDisputeGame` clone.
     * @dev Note: `.` denotes concatenation.
     */
    mapping(Hash => IDisputeGame) internal disputeGames;

    /**
     * @notice Constructs a new DisputeGameFactory contract.
     * @param _owner The owner of the contract.
     */
    constructor(address _owner) Ownable() {
        transferOwnership(_owner);
    }

    /**
     * @notice Retrieves the hash of `gameType . rootClaim . extraData`
     *         to the deployed `DisputeGame` clone.
     * @dev Note: `.` denotes concatenation.
     * @param gameType The type of the DisputeGame.
     *        Used to decide the implementation to clone.
     * @param rootClaim The root claim of the DisputeGame.
     * @param extraData Any extra data that should be provided to the
     *        created dispute game.
     * @return _proxy The clone of the `DisputeGame` created with the
     *         given parameters. `address(0)` if nonexistent.
     */
    function games(
        GameType gameType,
        Claim rootClaim,
        bytes calldata extraData
    ) external view returns (IDisputeGame _proxy) {
        return disputeGames[getGameUUID(gameType, rootClaim, extraData)];
    }

    /**
     * @notice Creates a new DisputeGame proxy contract.
     * @notice If a dispute game with the given parameters already exists,
     *         it will be returned.
     * @param gameType The type of the DisputeGame.
     *        Used to decide the proxy implementation.
     * @param rootClaim The root claim of the DisputeGame.
     * @param extraData Any extra data that should be provided
     *        to the created dispute game.
     * @return proxy The clone of the `DisputeGame`.
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
        bytes memory data = abi.encodePacked(rootClaim, extraData);
        proxy = IDisputeGame(address(impl).clone(data));
        proxy.initialize();

        // Compute the unique identifier for the dispute game.
        Hash uuid = getGameUUID(gameType, rootClaim, extraData);

        // If a dispute game with the same UUID already exists, revert.
        if (address(disputeGames[uuid]) != address(0)) {
            revert GameAlreadyExists(uuid);
        }

        // Store the dispute game in the mapping & emit the `DisputeGameCreated` event.
        disputeGames[uuid] = proxy;
        emit DisputeGameCreated(address(proxy), gameType, rootClaim);
    }

    /**
     * @notice Sets the implementation contract for a specific `GameType`.
     * @param gameType The type of the DisputeGame.
     * @param impl The implementation contract for the given `GameType`.
     */
    function setImplementation(GameType gameType, IDisputeGame impl) external onlyOwner {
        gameImpls[gameType] = impl;
        emit ImplementationSet(address(impl), gameType);
    }

    /**
     * @notice Returns a unique identifier for the given dispute game parameters.
     * @dev Hashes the concatenation of `gameType . rootClaim . extraData`
     *      without expanding memory.
     * @param gameType The type of the DisputeGame.
     * @param rootClaim The root claim of the DisputeGame.
     * @param extraData Any extra data that should be provided to the created dispute game.
     * @return _uuid The unique identifier for the given dispute game parameters.
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
}
