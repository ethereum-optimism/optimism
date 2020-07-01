pragma solidity ^0.5.0;

import { ExecutionManager } from "../optimistic-ethereum/ovm/ExecutionManager.sol";

/**
 * @title SimpleTxOrigin
 * @notice A simple contract testing the transaction origin opcode.
 */
contract SimpleTxOrigin {
    address executionManagerAddress;

    /**
     * Constructor currently accepts an execution manager & stores that in storage.
     * Note this should be the only storage that this contract ever uses & it should be replaced
     * by a hardcoded value once we have the transpiler.
     */
    constructor(address _executionManagerAddress) public {
        executionManagerAddress = _executionManagerAddress;
    }

    function getOrigin() public {
        // bitwise right shift 28 * 8 bits so the 4 method ID bytes are in the right-most bytes
        bytes32 methodId = keccak256("ovmORIGIN()") >> 224;
        address addr = executionManagerAddress;

        assembly {
            let callBytes := mload(0x40)
            calldatacopy(callBytes, 0, calldatasize)

            // replace the first 4 bytes with the right methodID
            mstore8(callBytes, shr(24, methodId))
            mstore8(add(callBytes, 1), shr(16, methodId))
            mstore8(add(callBytes, 2), shr(8, methodId))
            mstore8(add(callBytes, 3), methodId)

            // overwrite call params
            let result := mload(0x40)
            let success := call(gas, addr, 0, callBytes, calldatasize, result, 500000)

            if eq(success, 0) {
                revert(0, 0)
            }

            return(result, returndatasize)
        }
    }
}
