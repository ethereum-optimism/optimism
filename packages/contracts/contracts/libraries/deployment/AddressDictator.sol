// SPDX-License-Identifier: MIT
pragma solidity ^0.8.9;

import { Lib_AddressManager } from "../resolver/Lib_AddressManager.sol";

/**
 * @title AddressDictator
 */
contract AddressDictator {
    /*********
     * Types *
     *********/

    struct NamedAddress {
        string name;
        address addr;
    }

    /*************
     * Variables *
     *************/

    Lib_AddressManager public manager;
    address public finalOwner;
    NamedAddress[] namedAddresses;

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
            "AddressDictator: Must provide an equal number of names and addresses."
        );
        for (uint256 i = 0; i < _names.length; i++) {
            namedAddresses.push(NamedAddress({ name: _names[i], addr: _addresses[i] }));
        }
    }

    /********************
     * Public Functions *
     ********************/

    function setAddresses() external {
        for (uint256 i = 0; i < namedAddresses.length; i++) {
            manager.setAddress(namedAddresses[i].name, namedAddresses[i].addr);
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

    /******************
     * View Functions *
     ******************/

    function getNamedAddresses() external view returns (NamedAddress[] memory) {
        return namedAddresses;
    }
}
