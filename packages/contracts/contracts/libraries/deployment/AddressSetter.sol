// SPDX-License-Identifier: MIT
pragma solidity ^0.8.9;

import { Lib_AddressManager } from "../resolver/Lib_AddressManager.sol";

/**
 * @title AddressSetter
 */
contract AddressSetter {
    /*************
     * Variables *
     *************/

    Lib_AddressManager manager;
    address finalOwner;
    string[] names;
    address[] addresses;

    /***************
     * Constructor *
     ***************/

    /**
     * @param _manager Address of the AddressManager contract.
     * @param _finalOwner Address to transfer AddressManager ownership to afterwards.
     * @param _names Array of names to associate an address with.
     * @param _addresses Array of addresses to associate with the name.
     */
    constructor(
        Lib_AddressManager _manager,
        address _finalOwner,
        string[] memory _names,
        address[] memory _addresses
    ) {
        // todo: this probably needs to be moved into a public function which the deployer key
        // is authed to call. Otherwise we need to predict the address of this contract, and have
        // the multisig transfer ownership here before it is deployed, which would be scary.
        manager = _manager;
        finalOwner = _finalOwner;
        require(
            _names.length == _addresses.length,
            "AddressSetter: Must provide an equal number of names and addresses."
        );
        names = _names;
        addresses = _addresses;
    }

    /********************
     * Public Functions *
     ********************/

    function setAddresses() external {
        for (uint256 i = 0; i < names.length; i++) {
            manager.setAddress(names[i], addresses[i]);
        }
        // note that this will revert if _finalOwner == currentOwner
        manager.transferOwnership(finalOwner);
    }

    /**
     * This function shouldn't be necessary, but it gives a sense of reassurance that we can recover
     * if something really surprising goes wrong.
     */
    function returnOwnership() external {
        manager.transferOwnership(finalOwner);
    }
}
