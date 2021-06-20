// SPDX-License-Identifier: MIT
pragma solidity >0.5.0 <0.8.0;

/* Library Imports */
import { MVM_AddressManager } from "./MVM_AddressManager.sol";

/**
 * @title MVM_AddressResolver
 */
abstract contract MVM_AddressResolver {

    /*************
     * Variables *
     *************/
     
    MVM_AddressManager public mvmAddressManager;


    /***************
     * Constructor *
     ***************/

    /**
     * @param _libAddressManager Address of the MVM_AddressManager.
     */
    constructor(
        address _libAddressManager
    ) {
        mvmAddressManager = MVM_AddressManager(_libAddressManager);
    }

    /********************
     * Public Functions *
     ********************/

    /**
     * Resolves the address for MVM associated with a given name.
     * @param _name Name to resolve an address for.
     * @return Address associated with the given name.
     */
    function resolveFromMvm(
        string memory _name
    )
        public
        view
        returns (
            address
        )
    {
        return mvmAddressManager.getAddress(_name);
    }
}
