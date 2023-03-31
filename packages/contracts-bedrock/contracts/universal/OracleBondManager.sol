// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Ownable } from "@openzeppelin/contracts/access/Ownable.sol";
import { SafeCall } from "../libraries/SafeCall.sol";
import { IBondManager } from "./IBondManager.sol";

/**
 * @title OracleBondManager
 * @notice A Bond Manager implementation for the L2OutputOracleV2 contract.
 */
contract OracleBondManager is IBondManager, Ownable {
    /**
     * @notice The amount for each (dumb) bond.
     */
    uint256 immutable MIN_BOND_AMOUNT;

    /**
     * @notice The internal mapping of bond id to value.
     */
    mapping(bytes32 => uint256) internal bonds;

    /**
     * @notice Emitted when a bond is posted.
     * @dev Neither the owner or value are indexed since they are not sparse.
     */
    event BondPosted(bytes32 indexed id, address owner, uint256 value);

    /**
     * @notice Emitted when a bond is called.
     * @dev Neither the owner or value are indexed since they are not sparse.
     */
    event BondCalled(bytes32 indexed id, address owner, uint256 value);

    /**
     * @notice Instantiates a new OracleBondManager.
     */
    constructor(uint256 amount) Ownable() {
        MIN_BOND_AMOUNT = amount;
    }

    /**
     * @notice Transfers ownership.
     */
    function setOwner(address newOwner) external onlyOwner {
        _transferOwnership(newOwner);
    }

    /**
     * @notice Returns the next minimum bond amount.
     */
    function next() public returns (uint256) {
        return MIN_BOND_AMOUNT;
    }

    /**
     * @notice Post a bond for a given id.
     * @notice The id is expected to be the hash of the l2BlockNumber.
     *		   so: id = keccak256(abi.encode(l2BlockNumber));
     */
    function post(bytes32 id) external payable {
        require(msg.value >= next(), "OracleBondManager: minimum bond amount not satisfied");
        require(bonds[id] == 0, "OracleBondManager: bond already posted");
        emit BondPosted(id, msg.sender, msg.value);
        bonds[id] = msg.value;
    }

    /**
     * @notice Calls a bond for a given id.
     * @notice Only the owner can call a bond.
     * @notice The id is expected to be the hash of the l2BlockNumber.
     *		   so: id = keccak256(abi.encode(l2BlockNumber));
     */
    function call(bytes32 id, address to) external onlyOwner returns (uint256) {
        uint256 value = bonds[id];
        require(value != 0, "OracleBondManager: calling an empty bond");
        emit BondCalled(id, msg.sender, value);
        delete bonds[id];
        bool success = SafeCall.call(payable(to), gasleft(), value, hex"");
        require(success, "OracleBondManager: Failed to send ether in bond call");
        return value;
    }
}
