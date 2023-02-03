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
abstract contract CrossDomainOwnable2 is Ownable {
    /**
     * @notice If true, the contract uses the cross domain _checkOwner function override. If false
     *         it uses the standard Ownable _checkOwner function.
     */
    bool internal crossDomainSwitch = false;

    /**
     * @notice The local owner of the contract.
     */
    address private _localOwner = _msgSender();

    /**
     * @notice Thrown when the local owner changes.
     */
    event LocalOwnershipTransferred(address indexed previousOwner, address indexed newOwner);

    /**
     * @notice Throws if called by any account other than the localOwner.
     */
    modifier onlyLocalOwner() {
        _checkLocalOwner();
        _;
    }

    /**
     * @notice Throws if the sender is not the localOwner.
     */
    function _checkLocalOwner() internal view {
        require(_msgSender() == _localOwner, "CrossDomainOwnable2: caller is not the localOwner");
    }

    /**
     * @notice Overrides the implementation of the `onlyOwner` modifier to check that the unaliased
     *         `xDomainMessageSender` is the owner of the contract. This value is set to the caller
     *         of the L1CrossDomainMessenger.
     */
    function _checkOwner() internal view override {
        if (crossDomainSwitch) {
            L2CrossDomainMessenger messenger = L2CrossDomainMessenger(
                Predeploys.L2_CROSS_DOMAIN_MESSENGER
            );

            require(
                msg.sender == address(messenger),
                "CrossDomainOwnable2: caller is not the messenger"
            );

            require(
                owner() == messenger.xDomainMessageSender(),
                "CrossDomainOwnable2: caller is not the owner"
            );
        } else {
            _checkLocalOwner();
        }
    }

    /**
     * @notice Allows the localOwner to turn on the cross domain ownership check.
     */
    function flipTheSwitch() external onlyLocalOwner {
        require(_msgSender() == _localOwner, "CrossDomainOwnable2: caller is not the localOwner");
        crossDomainSwitch = !crossDomainSwitch;
    }

    /**
     * @notice Allows the localOwner to transfer ownership to a .
     * @param newOwner The address of the new owner.
     */
    function transferLocalOwnership(address newOwner) external onlyLocalOwner {
        require(
            crossDomainSwitch == false,
            "CrossDomainOwnable2: cross domain ownership turned on"
        );
        require(newOwner != address(0), "Ownable: new owner is the zero address");
        address oldOwner = _localOwner;
        _localOwner = newOwner;

        emit LocalOwnershipTransferred(oldOwner, newOwner);
    }
}
