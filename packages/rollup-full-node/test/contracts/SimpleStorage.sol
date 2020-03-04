pragma solidity ^0.5.0;

contract SimpleStorage {
    function setStorage(address exeMgrAddr, bytes32 key, bytes32 value) public {
        // Make the low level ovmSLOAD() call
        bytes4 methodId = bytes4(keccak256("ovmSSTORE()") >> 224);
        assembly {
            let callBytes := mload(0x40)
            // Skip the address bytes, repurpose the methodID
            calldatacopy(callBytes, 0x20, calldatasize)

            // replace the first 4 bytes with the right methodID
            mstore8(callBytes, shr(24, methodId))
            mstore8(add(callBytes, 1), shr(16, methodId))
            mstore8(add(callBytes, 2), shr(8, methodId))
            mstore8(add(callBytes, 3), methodId)

            // callBytes should be 4 bytes of method ID, key, value
            let success := call(gas, exeMgrAddr, 0, callBytes, 68, 0, 500000)

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
            // Skip the address bytes, repurpose the methodID
            calldatacopy(callBytes, 0x20, calldatasize)

            // replace the first 4 bytes with the right methodID
            mstore8(callBytes, shr(24, methodId))
            mstore8(add(callBytes, 1), shr(16, methodId))
            mstore8(add(callBytes, 2), shr(8, methodId))
            mstore8(add(callBytes, 3), methodId)

            // overwrite call params
            let result := mload(0x40)
            // callBytes should be 4 bytes of method ID and key
            let success := staticcall(gas, exeMgrAddr, callBytes, 36, result, 500000)

            if eq(success, 0) {
                revert(0, 0)
            }

            return(result, returndatasize)
        }
    }
}
