// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

import "src/types/Errors.sol";
import { IOwnable } from "src/interfaces/IOwnable.sol";

/// @notice Simple single owner contract.
/// @author Adapted from Solmate (https://github.com/transmissions11/solmate/blob/main/src/auth/Owned.sol)
abstract contract Ownable is IOwnable {
    event OwnershipTransferred(address indexed user, address indexed newOwner);

    /// @dev The owner of the contract.
    address internal _owner;

    modifier onlyOwner() virtual {
        if (msg.sender != _owner) {
            revert NotOwner();
        }
        _;
    }

    constructor(address initialOwner) {
        _owner = initialOwner;
        emit OwnershipTransferred(address(0), initialOwner);
    }

    /// @notice Returns the owner of the contract
    function owner() public view returns (address) {
        return _owner;
    }

    /// @notice Transfer ownership to the passed address
    /// @param newOwner The address to transfer ownership to
    function transferOwnership(address newOwner) public virtual onlyOwner {
        _owner = newOwner;
        emit OwnershipTransferred(msg.sender, newOwner);
    }
}
