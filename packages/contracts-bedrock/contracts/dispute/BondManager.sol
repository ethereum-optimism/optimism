// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

import { GameType } from "../libraries/DisputeTypes.sol";
import { GameStatus } from "../libraries/DisputeTypes.sol";
import { SafeCall } from "../libraries/SafeCall.sol";

import { IDisputeGame } from "./IDisputeGame.sol";
import { IDisputeGameFactory } from "./IDisputeGameFactory.sol";

/**
 * @title BondManager
 * @notice The Bond Manager serves as an escrow for permissionless output proposal bonds.
 */
contract BondManager {
    // The Bond Type
    struct Bond {
        address owner;
        uint256 expiration;
        bytes32 id;
        uint256 amount;
    }

    /**
     * @notice Mapping from bondId to bond.
     */
    mapping(bytes32 => Bond) public bonds;

    /**
     * @notice BondPosted is emitted when a bond is posted.
     * @param bondId is the id of the bond.
     * @param owner is the address that owns the bond.
     * @param expiration is the time at which the bond expires.
     * @param amount is the amount of the bond.
     */
    event BondPosted(bytes32 bondId, address owner, uint256 expiration, uint256 amount);

    /**
     * @notice BondSeized is emitted when a bond is seized.
     * @param bondId is the id of the bond.
     * @param owner is the address that owns the bond.
     * @param seizer is the address that seized the bond.
     * @param amount is the amount of the bond.
     */
    event BondSeized(bytes32 bondId, address owner, address seizer, uint256 amount);

    /**
     * @notice BondReclaimed is emitted when a bond is reclaimed by the owner.
     * @param bondId is the id of the bond.
     * @param claiment is the address that reclaimed the bond.
     * @param amount is the amount of the bond.
     */
    event BondReclaimed(bytes32 bondId, address claiment, uint256 amount);

    /**
     * @notice The permissioned dispute game factory.
     * @dev Used to verify the status of bonds.
     */
    IDisputeGameFactory public immutable DISPUTE_GAME_FACTORY;

    /**
     * @notice Instantiates the bond maanger with the registered dispute game factory.
     * @param _disputeGameFactory is the dispute game factory.
     */
    constructor(IDisputeGameFactory _disputeGameFactory) {
        DISPUTE_GAME_FACTORY = _disputeGameFactory;
    }

    /**
     * @notice Post a bond with a given id and owner.
     * @dev This function will revert if the provided bondId is already in use.
     * @param _bondId is the id of the bond.
     * @param _bondOwner is the address that owns the bond.
     * @param _minClaimHold is the minimum amount of time the owner
     *        must wait before reclaiming their bond.
     */
    function post(
        bytes32 _bondId,
        address _bondOwner,
        uint256 _minClaimHold
    ) external payable {
        require(bonds[_bondId].owner == address(0), "BondManager: BondId already posted.");
        require(_bondOwner != address(0), "BondManager: Owner cannot be the zero address.");
        require(msg.value > 0, "BondManager: Value must be non-zero.");

        uint256 expiration = _minClaimHold + block.timestamp;
        bonds[_bondId] = Bond({
            owner: _bondOwner,
            expiration: expiration,
            id: _bondId,
            amount: msg.value
        });

        emit BondPosted(_bondId, _bondOwner, expiration, msg.value);
    }

    /**
     * @notice Seizes the bond with the given id.
     * @dev This function will revert if there is no bond at the given id.
     * @param _bondId is the id of the bond.
     */
    function seize(bytes32 _bondId) external {
        Bond memory b = bonds[_bondId];
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

        delete bonds[_bondId];

        emit BondSeized(_bondId, b.owner, msg.sender, b.amount);

        bool success = SafeCall.send(payable(msg.sender), gasleft(), b.amount);
        require(success, "BondManager: Failed to send Ether.");
    }

    /**
     * @notice Seizes the bond with the given id and distributes it to recipients.
     * @dev This function will revert if there is no bond at the given id.
     * @param _bondId is the id of the bond.
     * @param _claimRecipients is a set of addresses to split the bond amongst.
     */
    function seizeAndSplit(bytes32 _bondId, address[] calldata _claimRecipients) external {
        Bond memory b = bonds[_bondId];
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

        delete bonds[_bondId];

        emit BondSeized(_bondId, b.owner, msg.sender, b.amount);

        uint256 len = _claimRecipients.length;
        uint256 proportionalAmount = b.amount / len;
        for (uint256 i = 0; i < len; i++) {
            bool success = SafeCall.send(
                payable(_claimRecipients[i]),
                gasleft() / len,
                proportionalAmount
            );
            require(success, "BondManager: Failed to send Ether.");
        }
    }

    /**
     * @notice Reclaims the bond of the bond owner.
     * @dev This function will revert if there is no bond at the given id.
     * @param _bondId is the id of the bond.
     */
    function reclaim(bytes32 _bondId) external {
        Bond memory b = bonds[_bondId];
        require(b.owner == msg.sender, "BondManager: Unauthorized claimant.");
        require(b.expiration <= block.timestamp, "BondManager: Bond isn't claimable yet.");

        delete bonds[_bondId];

        emit BondReclaimed(_bondId, msg.sender, b.amount);

        bool success = SafeCall.send(payable(msg.sender), gasleft(), b.amount);
        require(success, "BondManager: Failed to send Ether.");
    }
}
