pragma solidity ^0.5.0;

import { ExecutionManager } from "../optimistic-ethereum/ovm/ExecutionManager.sol";

/**
 * @title SimpleCall
 * @notice A simple contract testing the execution manager's CALL.
 */
contract SimpleCall {
    address executionManagerAddress;

    /**
     * Constructor currently accepts an execution manager & stores that in storage.
     * Note this should be the only storage that this contract ever uses & it should be replaced
     * by a hardcoded value once we have the transpiler.
     */
    constructor(address _executionManagerAddress) public {
        executionManagerAddress = _executionManagerAddress;
    }

    // expects _targetContract (address as bytes32), _calldata (variable-length bytes).
    // returns variable-length bytes result.
    function makeCall() public {
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

    // expects _targetContract (address as bytes32), _calldata (variable-length bytes).
    // returns variable-length bytes result.
    function makeStaticCall() public {
        // bitwise right shift 28 * 8 bits so the 4 method ID bytes are in the right-most bytes
        bytes32 methodId = keccak256("ovmSTATICCALL()") >> 224;
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

    // expects _targetContract (address as bytes32), _calldata (variable-length bytes).
    // returns variable-length bytes result.
    function makeDelegateCall() public {
        // bitwise right shift 28 * 8 bits so the 4 method ID bytes are in the right-most bytes
        bytes32 methodId = keccak256("ovmDELEGATECALL()") >> 224;
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

    // Does a call to ovmSSTORE, assuming a 32-byte key and a 32-byte value are passed in
    function notStaticFriendlySSTORE() public {
        // bitwise right shift 28 * 8 bits so the 4 method ID bytes are in the right-most bytes
        bytes32 methodId = keccak256("ovmSSTORE()") >> 224;
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

    // Does a call to ovmCREATE with provided contract initcode
    function notStaticFriendlyCREATE() public {
        // bitwise right shift 28 * 8 bits so the 4 method ID bytes are in the right-most bytes
        bytes32 methodId = keccak256("ovmCREATE()") >> 224;
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

    // Does a call to ovmCREATE2 with provided contract salt and initcode
    function notStaticFriendlyCREATE2() public {
        // bitwise right shift 28 * 8 bits so the 4 method ID bytes are in the right-most bytes
        bytes32 methodId = keccak256("ovmCREATE2()") >> 224;
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

    // Does a call to ovmSLOAD assuming a 32-byte KEY is passed in
    function staticFriendlySLOAD() public {
        // bitwise right shift 28 * 8 bits so the 4 method ID bytes are in the right-most bytes
        bytes32 methodId = keccak256("ovmSLOAD()") >> 224;
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

    // expects 32-byte left-padded value for this contract's OVM address
    // does not return data
    function makeStaticCallThenCall() public {
        // bitwise right shift 28 * 8 bits so the 4 method ID bytes are in the right-most bytes
        bytes32 staticCallMethodId = keccak256("ovmSTATICCALL()") >> 224;
        bytes32 callMethodId = keccak256("ovmCALL()") >> 224;
        bytes32 sloadMethodId = keccak256("staticFriendlySLOAD()") >> 224;
        bytes32 sstoreMethodId = keccak256("notStaticFriendlySSTORE()") >> 224;

        address emAddr = executionManagerAddress;
        uint key = 1;
        uint value = 2;

        assembly {
            let myAddress := calldataload(4)

            function $writeMethodId(methodHash, writeToThis) {
                // replace the first 4 bytes with the right methodID
                mstore8(writeToThis, shr(24, methodHash))
                mstore8(add(writeToThis, 1), shr(16, methodHash))
                mstore8(add(writeToThis, 2), shr(8, methodHash))
                mstore8(add(writeToThis, 3), methodHash)
            }

            let callBytes := mload(0x40)
            $writeMethodId(staticCallMethodId, callBytes)
            mstore(add(callBytes, 4), myAddress)
            $writeMethodId(sloadMethodId, add(callBytes, 0x24))
            mstore(add(callBytes, 0x28), key)

            // overwrite call params
            let result := mload(0x40)
            let success := call(gas, emAddr, 0, callBytes, 0x48, result, 500000)

            if eq(success, 0) {
                revert(0,0)
            }

            // overwrite result to make next call
            callBytes := mload(0x40)
            //$writeMethodId(staticContextMethodId, callBytes)

            $writeMethodId(callMethodId, callBytes)
            mstore(add(callBytes, 4), myAddress)
            $writeMethodId(sstoreMethodId, add(callBytes, 0x24))
            mstore(add(callBytes, 0x28), key)
            mstore(add(callBytes, 0x48), value)

            // overwrite call params
            result := mload(0x40)
            success := call(gas, emAddr, 0, callBytes, 0x68, result, 500000)

            if eq(success, 0) {
                revert(0,0)
            }
        }
    }
}
