// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

import "../libraries/DisputeTypes.sol";
import "../libraries/DisputeErrors.sol";

import { Ownable } from "@openzeppelin/contracts/access/Ownable.sol";
import { ClonesWithImmutableArgs } from "@cwia/ClonesWithImmutableArgs.sol";

import { IDisputeGame } from "./interfaces/IDisputeGame.sol";
import { IDisputeGameFactory } from "./interfaces/IDisputeGameFactory.sol";

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
     * @notice Constructs a new DisputeGameFactory contract.
     * @param _owner The owner of the contract.
     */
    constructor(address _owner) Ownable() {
        transferOwnership(_owner);
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
