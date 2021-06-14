// SPDX-License-Identifier: MIT
pragma solidity >=0.7.0;

contract SetStorageAndDeployCode {
    // deploys arbitrary given bytecode after setting an arbitrary storage slot, all specified via constructor.
    constructor(
        bytes32 _key,
        bytes32 _value,
        bytes memory _codeToDeploy
    ) {
        assembly {
            sstore(_key, _value)
            return(add(_codeToDeploy, 0x20), mload(_codeToDeploy))
        }
    }
}
