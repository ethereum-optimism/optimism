// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { Ownable } from "@openzeppelin/contracts/access/Ownable.sol";
import { AddressAliasHelper } from "../vendor/AddressAliasHelper.sol";

/**
 * @title CrossDomainOwnable
 * @notice This contract extends the OpenZeppelin `Ownable` contract
 *         for L2 contracts to be owned by contracts on L1. Note that
 *         this contract is only safe to be used if the CrossDomainMessenger
 *         system is bypassed and the caller on L1 is calling the
 *         OptimismPortal directly.
 */
abstract contract CrossDomainOwnable is Ownable {
    /**
     * @notice Set the owner of the contract. OpenZeppelin's ownable by default
     *         will set the owner of the contract to be `msg.sender` and it is
     *         up to the implementer to transfer the ownership if using the
     *         deployer as the owner is not desired. This makes ownership
     *         transfer much more explicit.
     */
    constructor(address _owner) {
        _transferOwnership(_owner);
    }

    /**
     * @notice Overrides the implementation of the `onlyOwner` modifier
     *         to check that the unaliased `msg.sender` is the owner
     *         of the contract.
     */
    function _checkOwner() internal view override {
        require(
            owner() == AddressAliasHelper.undoL1ToL2Alias(msg.sender),
            "CrossDomainOwnable: caller is not the owner"
        );
    }
}
