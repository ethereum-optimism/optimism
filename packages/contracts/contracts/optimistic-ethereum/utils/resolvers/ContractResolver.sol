pragma solidity ^0.5.0;

import { AddressResolver } from "./AddressResolver.sol";

contract ContractResolver {
    AddressResolver addressResolver;

    constructor(address _addressResolver) public {
        addressResolver = AddressResolver(_addressResolver);
    }

    function resolveContract(string memory _name) public view returns (address) {
        return addressResolver.getAddress(_name);
    }
}