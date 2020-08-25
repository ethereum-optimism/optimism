pragma solidity ^0.5.0;

/* Library Imports */
import { ContractResolver } from "../optimistic-ethereum/utils/resolvers/ContractResolver.sol";

/* Testing Imports */
import { console } from "@nomiclabs/buidler/console.sol";

// contract proxies all calls to the contract name given in the constructor.
// Acts as an identical contract to the one it's proxying, but consumes extra gas in the process.

contract GasConsumingProxy is ContractResolver {
    string implementationName;
    constructor(address _addressResolver, string memory _implementationName) 
        public
        ContractResolver(_addressResolver)
    {
        implementationName = _implementationName;
    }
    function () external {
        address implementation = resolveContract(implementationName);
        // #if FLAG_IS_DEBUG
        // console.log("proxy activated, calling implementation", implementation);
        // #endif
        assembly {
            calldatacopy(0x0, 0x0, calldatasize)
            let result := call(gas, implementation, 0, 0x0, calldatasize, 0x0, 0)
            returndatacopy(0x0, 0x0, returndatasize)
            switch result case 0 {revert(0, 0)} default {return (0, returndatasize)}
        }
    }
}