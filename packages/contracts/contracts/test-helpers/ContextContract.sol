pragma solidity ^0.5.0;

import { ExecutionManager } from "../optimistic-ethereum/ovm/ExecutionManager.sol";

/**
 * @title ContextContract
 * @notice A simple contract testing the execution manager's context functions.
 */
contract ContextContract {
    address executionManagerAddress;

    constructor(address _executionManagerAddress) public {
        executionManagerAddress = _executionManagerAddress;
    }

    function callThroughExecutionManager() public {
        // bitwise right shift 28 * 8 bits so the 4 method ID bytes are in the right-most bytes
        bytes32 methodId = keccak256("ovmCALL()") >> 224;
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

    function getCALLER() public {
        // bitwise right shift 28 * 8 bits so the 4 method ID bytes are in the right-most bytes
        bytes32 methodId = keccak256("ovmCALLER()") >> 224;
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

    function getADDRESS() public {
        // bitwise right shift 28 * 8 bits so the 4 method ID bytes are in the right-most bytes
        bytes32 methodId = keccak256("ovmADDRESS()") >> 224;
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

    function getTIMESTAMP() public {
        // bitwise right shift 28 * 8 bits so the 4 method ID bytes are in the right-most bytes
        bytes32 methodId = keccak256("ovmTIMESTAMP()") >> 224;
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

    function getCHAINID() public {
        // bitwise right shift 28 * 8 bits so the 4 method ID bytes are in the right-most bytes
        bytes32 methodId = keccak256("ovmCHAINID()") >> 224;
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

    function getGASLIMIT() public {
        // bitwise right shift 28 * 8 bits so the 4 method ID bytes are in the right-most bytes
        bytes32 methodId = keccak256("ovmGASLIMIT()") >> 224;
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

    function getFraudProofGasLimit() public {
        // bitwise right shift 28 * 8 bits so the 4 method ID bytes are in the right-most bytes
        bytes32 methodId = keccak256("ovmFraudProofGasLimit()") >> 224;
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

    function getQueueOrigin() public {
        // bitwise right shift 28 * 8 bits so the 4 method ID bytes are in the right-most bytes
        bytes32 methodId = keccak256("ovmQueueOrigin()") >> 224;
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
