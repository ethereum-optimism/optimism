// SPDX-License-Identifier: MIT
pragma solidity ^0.8.17;

/// @notice Simple single owner contract.
/// @author Adapted from Solmate (https://github.com/transmissions11/solmate/blob/main/src/auth/Owned.sol)
abstract contract Owner {
    event OwnershipTransferred(address indexed user, address indexed newOwner);

    address internal _owner;

    modifier onlyOwner() virtual {
        require(msg.sender == _owner, "UNAUTHORIZED");
        _;
    }

    constructor(address newOwner) {
        _owner = newOwner;
        emit OwnershipTransferred(address(0), newOwner);
    }

    function transferOwnership(address newOwner) public virtual onlyOwner {
        _owner = newOwner;
        emit OwnershipTransferred(msg.sender, newOwner);
    }
}
