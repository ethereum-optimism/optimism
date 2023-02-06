// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { Predeploys } from "../libraries/Predeploys.sol";
import { L2CrossDomainMessenger } from "./L2CrossDomainMessenger.sol";
import { Ownable } from "@openzeppelin/contracts/access/Ownable.sol";

/**
 * @title CrossDomainOwnable2
 * @notice This contract extends the OpenZeppelin `Ownable` contract for L2 contracts to be owned
 *         by contracts on L1. Note that this contract is meant to be used with systems that use
 *         the CrossDomainMessenger system. It will not work if the OptimismPortal is used
 *         directly.
 */
abstract contract CrossDomainOwnable3 is Ownable {
    /**
     * @notice If true, the contract uses the cross domain _checkOwner function override. If false
     *         it uses the standard Ownable _checkOwner function.
     */
    bool internal isLocal = true;

    /**
     * @notice Overrides the implementation of the `onlyOwner` modifier to check that the unaliased
     *         `xDomainMessageSender` is the owner of the contract. This value is set to the caller
     *         of the L1CrossDomainMessenger.
     */
    function _checkOwner() internal view override {
        if (isLocal) {
            super._checkOwner();
        } else {
            L2CrossDomainMessenger messenger = L2CrossDomainMessenger(
                Predeploys.L2_CROSS_DOMAIN_MESSENGER
            );

            require(
                msg.sender == address(messenger),
                "CrossDomainOwnable3: caller is not the messenger"
            );

            require(
                owner() == messenger.xDomainMessageSender(),
                "CrossDomainOwnable3: caller is not the owner"
            );
        }
    }

    /**
     * @notice Overrides the implementation of the `transferOwnership` function to allow
     * for local ownership.
     * @param newOwner The new owner of the contract.
     * @param _isLocal If false, the contract uses the cross domain _checkOwner function override.
     * If false it uses the standard Ownable _checkOwner function.
     */
    function transferOwnership(address newOwner, bool _isLocal) external override {
        isLocal = _isLocal;

        super.transferOwnership(newOwner);
    }
}
