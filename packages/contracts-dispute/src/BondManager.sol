/// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

import { GameType } from "src/types/Types.sol";
import { GameStatus } from "src/types/Types.sol";
import { SafeCall } from "src/lib/LibSafeCall.sol";
import { IDisputeGame } from "src/interfaces/IDisputeGame.sol";
import { IDisputeGameFactory } from "src/interfaces/IDisputeGameFactory.sol";

/// @title BondManager
/// @author clabby <https://github.com/clabby>
/// @author refcell <https://github.com/refcell>
/// @notice The Bond Manager serves as an escrow for permissionless output proposal bonds.
contract BondManager {
    // The Bond Type
    struct Bond {
        address owner;
        uint256 expiration;
        bytes32 bondId;
        uint256 amount;
    }

    /// @notice Mapping from bondId to amount.
    mapping(bytes32 => Bond) public bonds;

    /// @notice BondPosted is emitted when a bond is posted.
    event BondPosted(bytes32 bondId, address owner, uint256 expiration, uint256 amount);

    /// @notice BondSeized is emitted when a bond is seized.
    event BondSeized(bytes32 bondId, address owner, address seizer, uint256 amount);

    /// @notice BondReclaimed is emitted when a bond is reclaimed by the owner.
    event BondReclaimed(bytes32 bondId, address claiment, uint256 amount);

    /// @notice The permissioned dispute game factory.
    /// @dev Used to verify the status of bonds.
    IDisputeGameFactory public immutable DISPUTE_GAME_FACTORY;

    /// @notice Instantiates the bond maanger with the registered dispute game factory.
    constructor(IDisputeGameFactory _dgf) {
        DISPUTE_GAME_FACTORY = _dgf;
    }

    /// @notice Post a bond with a given id and owner.
    /// @dev This function will revert if the provided bondId is already in use.
    /// @param bondId is the id of the bond.
    /// @param owner is the address that owns the bond.
    /// @param minClaimHold is the minimum amount of time the owner
    ///        must wait before reclaiming their bond.
    function post(
        bytes32 bondId,
        address owner,
        uint256 minClaimHold
    ) external payable {
        require(bonds[bondId].owner == address(0), "BondManager: BondId already posted.");
        require(owner != address(0), "BondManager: Owner cannot be the zero address.");
        require(msg.value > 0, "BondManager: Value must be non-zero.");
        uint256 expiration = minClaimHold;
        unchecked {
            expiration = expiration + block.timestamp;
        }
        require(expiration >= block.timestamp, "BondManager: Invalid minimum claim hold.");

        bonds[bondId] = Bond({
            owner: owner,
            expiration: expiration,
            bondId: bondId,
            amount: msg.value
        });

        emit BondPosted(bondId, owner, expiration, msg.value);
    }

    /// @notice Seizes the bond with the given id.
    /// @dev This function will revert if there is no bond at the given id.
    /// @param bondId is the id of the bond.
    function seize(bytes32 bondId) external {
        Bond memory b = bonds[bondId];
        require(b.owner != address(0), "BondManager: The bond does not exist.");
        require(b.expiration >= block.timestamp, "BondManager: Bond expired.");

        IDisputeGame caller = IDisputeGame(msg.sender);
        IDisputeGame game = DISPUTE_GAME_FACTORY.games(
            GameType.ATTESTATION,
            caller.rootClaim(),
            caller.extraData()
        );
        require(msg.sender == address(game), "BondManager: Unauthorized seizure.");
        require(game.status() == GameStatus.CHALLENGER_WINS, "BondManager: Game incomplete.");

        delete bonds[bondId];

        emit BondSeized(bondId, b.owner, msg.sender, b.amount);

        SafeCall.call(msg.sender, gasleft(), b.amount, bytes(""));
    }

    /// @notice Seizes the bond with the given id and distributes it to recipients.
    /// @dev This function will revert if there is no bond at the given id.
    /// @param bondId is the id of the bond.
    /// @param recipients is a set of addresses to split the bond amongst.
    function seizeAndSplit(bytes32 bondId, address[] calldata recipients) external {
        Bond memory b = bonds[bondId];
        require(b.owner != address(0), "BondManager: The bond does not exist.");
        require(b.expiration >= block.timestamp, "BondManager: Bond expired.");

        IDisputeGame caller = IDisputeGame(msg.sender);
        IDisputeGame game = DISPUTE_GAME_FACTORY.games(
            GameType.ATTESTATION,
            caller.rootClaim(),
            caller.extraData()
        );
        require(msg.sender == address(game), "BondManager: Unauthorized seizure.");
        require(game.status() == GameStatus.CHALLENGER_WINS, "BondManager: Game incomplete.");

        delete bonds[bondId];

        emit BondSeized(bondId, b.owner, msg.sender, b.amount);

        uint256 len = recipients.length;
        uint256 proportionalAmount = b.amount / len;
        for (uint256 i = 0; i < len; i++) {
            SafeCall.call(recipients[i], gasleft(), proportionalAmount, bytes(""));
        }
    }

    /// @notice Reclaims the bond of the bond owner.
    /// @dev This function will revert if there is no bond at the given id.
    /// @param bondId is the id of the bond.
    function reclaim(bytes32 bondId) external {
        Bond memory b = bonds[bondId];
        require(b.owner == msg.sender, "BondManager: Unauthorized claimant.");
        require(b.expiration <= block.timestamp, "BondManager: Bond isn't claimable yet.");

        delete bonds[bondId];

        emit BondReclaimed(bondId, msg.sender, b.amount);

        SafeCall.call(msg.sender, gasleft(), b.amount, bytes(""));
    }
}
