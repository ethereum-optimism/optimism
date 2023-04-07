// SPDX-License-Identifier: MIT
pragma solidity ^0.8.17;

import { LibSafeCall } from "src/lib/LibSafeCall.sol";
import { IBondManager } from "src/interfaces/IBondManager.sol";

/// @title DisputeGameBondManager
/// @author refcell <github.com/refcell>
contract DisputeGameBondManager is IBondManager {
    /// @notice Bond holds the bond owner and amount.
    struct Bond {
        address owner;
        uint256 value;
    }

    /// @notice The amount for each (dumb) bond.
    uint256 immutable MIN_BOND_AMOUNT;

    /// @notice The internal mapping of bond id to Bond object.
    mapping(bytes32 => Bond) internal bonds;

    /// @notice Emitted when a bond is posted.
    /// @dev Neither the owner or value are indexed since they are not sparse.
    event BondPosted(bytes32 indexed id, address owner, uint256 value);

    /// @notice Emitted when a bond is called.
    /// @dev Neither the owner or value are indexed since they are not sparse.
    event BondCalled(bytes32 indexed id, address owner, uint256 value);

    /// @notice Instantiates a new DisputeGameBondManager.
    constructor(uint256 amount) {
        MIN_BOND_AMOUNT = amount;
    }

    /// @notice Returns the next minimum bond amount.
    function next() public view returns (uint256) {
        return MIN_BOND_AMOUNT;
    }

    /// @notice Post a bond for a given game step id.
    /// @notice The id is expected to be the hash of the packed sender,
    ///         l2BlockNumber, and game step, calculated like so:
    ///         id = keccak256(abi.encodePacked(msg.sender, l2BlockNumber, step));
    function post(bytes32 id) external payable {
        require(msg.value >= next(), "DisputeGameBondManager: minimum bond amount not satisfied");
        require(bonds[id].owner == address(0), "DisputeGameBondManager: bond already posted");
        emit BondPosted(id, msg.sender, msg.value);
        bonds[id] = Bond({ owner: msg.sender, value: msg.value });
    }

    /// @notice Calls a bond for a given game step id.
    /// @notice The id is expected to be the hash of the packed sender,
    ///         l2BlockNumber, and game step, calculated like so:
    ///         id = keccak256(abi.encodePacked(msg.sender, l2BlockNumber, step));
    function call(bytes32 id, address to) external returns (uint256) {
        Bond memory bond = bonds[id];
        require(bond.owner != address(0), "DisputeGameBondManager: calling an empty bond");
        emit BondCalled(id, bond.owner, bond.value);
        delete bonds[id];
        bool success = LibSafeCall.call(payable(to), gasleft(), bond.value, hex"");
        require(success, "DisputeGameBondManager: Failed to send ether in bond call");
        return bond.value;
    }
}
