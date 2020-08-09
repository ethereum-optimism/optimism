pragma solidity ^0.5.0;

/* Library Imports */
import { ContractResolver } from "../optimistic-ethereum/utils/resolvers/ContractResolver.sol";

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
        assembly {
            let initialFreeMemStart := mload(0x40)
            let callSize := calldatasize()
            mstore(0x40, add(initialFreeMemStart, callSize))
            calldatacopy(
                initialFreeMemStart,
                0,
                callSize
            )
            let success := call(
                gas(), // all remaining gas, leaving enough for this to execute
                implementation,
                0,
                initialFreeMemStart,
                callSize,
                0,
                0
            )
            // write the returndata to memory
            let returnedSize := returndatasize()
            let returnDataStart := mload(0x40)
            mstore(0x40, add(returnDataStart, returnedSize))
            returndatacopy(
                returnDataStart,
                0,
                returnedSize
            )
            if eq(success, 0) {
                revert(returnDataStart,returnedSize) // surface revert up
            }
            return(returnDataStart, returnedSize)
        }
    }
}