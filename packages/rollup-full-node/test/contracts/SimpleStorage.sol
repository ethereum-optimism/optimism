pragma solidity ^0.5.0;

contract SimpleStorage {
    function setStorage(address exeMgrAddr, bytes32 key, bytes32 value) public {
        // Make the low level ovmSLOAD() call
        bytes4 methodId = bytes4(keccak256("ovmSSTORE()") >> 224);
        bytes32 result;
        assembly {
            let callBytes := mload(0x40)
            calldatacopy(callBytes, 0, calldatasize)

            // replace the first 4 bytes with the right methodID
            mstore8(callBytes, shr(24, methodId))
            mstore8(add(callBytes, 1), shr(16, methodId))
            mstore8(add(callBytes, 2), shr(8, methodId))
            mstore8(add(callBytes, 3), methodId)
            // Add the key to the calldata
            mstore(add(callBytes, 4), key)
            // Add the value to the calldata
            mstore(add(callBytes, 36), value)

            // overwrite call params
            result := mload(0x40)
            let success := call(gas, exeMgrAddr, 0, callBytes, 68, result, 500000)

            if eq(success, 0) {
                revert(0, 0)
            }
        }
    }

    function getStorage(address exeMgrAddr, bytes32 key) public view returns (bytes32) {
        // Make the low level ovmSLOAD() call
        bytes4 methodId = bytes4(keccak256("ovmSLOAD()") >> 224);
        assembly {
            let callBytes := mload(0x40)
            calldatacopy(callBytes, 0, calldatasize)

            // replace the first 4 bytes with the right methodID
            mstore8(callBytes, shr(24, methodId))
            mstore8(add(callBytes, 1), shr(16, methodId))
            mstore8(add(callBytes, 2), shr(8, methodId))
            mstore8(add(callBytes, 3), methodId)
            // Add the key to the calldata
            mstore(add(callBytes, 4), key)

            // overwrite call params
            let result := mload(0x40)
            let success := staticcall(gas, exeMgrAddr, callBytes, 36, result, 500000)

            if eq(success, 0) {
                revert(0, 0)
            }

            return(result, returndatasize)
        }
    }
}
