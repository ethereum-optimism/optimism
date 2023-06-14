// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

import "../libraries/DisputeTypes.sol";

import { SafeCall } from "../libraries/SafeCall.sol";

import { IDisputeGame } from "./interfaces/IDisputeGame.sol";
import { IDisputeGameFactory } from "./interfaces/IDisputeGameFactory.sol";
import { IBondManager } from "./interfaces/IBondManager.sol";

/**
 * @title BondManager
 * @notice The Bond Manager serves as an escrow for permissionless output proposal bonds.
 */
contract BondManager is IBondManager {
    /**
     * @notice The Bond Type
     */
    struct Bond {
        address owner;
        bytes32 id;
        uint128 expiration;
        uint128 amount;
    }

    /**
     * @notice The permissioned dispute game factory.
     * @dev Used to verify the status of bonds.
     */
    IDisputeGameFactory public immutable DISPUTE_GAME_FACTORY;

    /**
     * @notice Amount of gas used to transfer ether when splitting the bond.
     *         This is a reasonable amount of gas for a transfer, even to a smart contract.
     *         The number of participants is bound of by the block gas limit.
     */
    uint256 private constant TRANSFER_GAS = 30_000;

    /**
     * @notice Mapping from bondId to bond.
     */
    mapping(bytes32 => Bond) public bonds;

    /**
     * @notice Instantiates the bond maanger with the registered dispute game factory.
     * @param _disputeGameFactory is the dispute game factory.
     */
    constructor(IDisputeGameFactory _disputeGameFactory) {
        DISPUTE_GAME_FACTORY = _disputeGameFactory;
    }

    /**
     * @inheritdoc IBondManager
     */
    function post(
        bytes32 _bondId,
        address _bondOwner,
        uint128 _minClaimHold
    ) external payable {
        require(bonds[_bondId].owner == address(0), "BondManager: BondId already posted.");
        require(_bondOwner != address(0), "BondManager: Owner cannot be the zero address.");
        require(msg.value > 0, "BondManager: Value must be non-zero.");

        uint128 expiration = uint128(_minClaimHold + block.timestamp);
        bonds[_bondId] = Bond({
            owner: _bondOwner,
            id: _bondId,
            expiration: expiration,
            amount: uint128(msg.value)
        });

        emit BondPosted(_bondId, _bondOwner, expiration, msg.value);
    }

    /**
     * @inheritdoc IBondManager
     */
    function seize(bytes32 _bondId) external {
        Bond memory b = bonds[_bondId];
        require(b.owner != address(0), "BondManager: The bond does not exist.");
        require(b.expiration >= block.timestamp, "BondManager: Bond expired.");

        IDisputeGame caller = IDisputeGame(msg.sender);
        IDisputeGame game = DISPUTE_GAME_FACTORY.games(
            GameTypes.ATTESTATION,
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
     * @inheritdoc IBondManager
     */
    function seizeAndSplit(bytes32 _bondId, address[] calldata _claimRecipients) external {
        Bond memory b = bonds[_bondId];
        require(b.owner != address(0), "BondManager: The bond does not exist.");
        require(b.expiration >= block.timestamp, "BondManager: Bond expired.");

        IDisputeGame caller = IDisputeGame(msg.sender);
        IDisputeGame game = DISPUTE_GAME_FACTORY.games(
            GameTypes.ATTESTATION,
            caller.rootClaim(),
            caller.extraData()
        );
        require(msg.sender == address(game), "BondManager: Unauthorized seizure.");
        require(game.status() == GameStatus.CHALLENGER_WINS, "BondManager: Game incomplete.");

        delete bonds[_bondId];

        emit BondSeized(_bondId, b.owner, msg.sender, b.amount);

        uint256 len = _claimRecipients.length;
        uint256 proportionalAmount = b.amount / len;
        // Send the proportional amount to each recipient. Do not revert if a send fails as that
        // will prevent other recipients from receiving their share.
        for (uint256 i; i < len; i++) {
            SafeCall.send({
                _target: payable(_claimRecipients[i]),
                _gas: TRANSFER_GAS,
                _value: proportionalAmount
            });
        }
    }

    /**
     * @inheritdoc IBondManager
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
