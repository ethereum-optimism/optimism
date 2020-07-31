pragma solidity ^0.5.0;

/* Library Imports */
import { AddressResolver } from "./AddressResolver.sol";

/**
 * @title ContractResolver
 */
contract ContractResolver {
    /*
     * Contract Variables
     */

    AddressResolver internal addressResolver;


    /*
     * Constructor
     */

    constructor(
        address _addressResolver
    )
        public
    {
        addressResolver = AddressResolver(_addressResolver);
    }


    /*
     * Public Functions
     */

    function resolveContract(
        string memory _name
    )
        public
        view
        returns (address)
    {
        return addressResolver.getAddress(_name);
    }
}