pragma solidity ^0.5.0;

contract SimpleStorage {
    mapping(bytes32 => bytes32) public builtInStorage;

    function setStorage(bytes32 key, bytes32 value) public {
        bytes memory EMcalldata = abi.encodeWithSelector(bytes4(keccak256(bytes("ovmSSTORE()"))), key, value);

        (bool success,) = msg.sender.call(EMcalldata);

    }

    function getStorage(bytes32 key) public returns (bytes32) {
        bytes memory EMcalldata = abi.encodeWithSelector(bytes4(keccak256(bytes("ovmSLOAD()"))), key);

        (bool success, bytes memory response) = msg.sender.call(EMcalldata);

        return bytesToBytes32(response);
    }

    function setSequentialSlots(uint startKey, bytes32 value, uint numIterations) public {
        for (uint i = 0; i < numIterations; i++) {
            setStorage(bytes32(startKey + i), value);
        }
    }

    function setSameSlotRepeated(bytes32 key, bytes32 value, uint numIterations) public {
        for (uint i = 0; i < numIterations; i++) {
            setStorage(key, value);
        }
    }

    function getStorages(bytes32 key, uint numIterations) public {
        for (uint i = 0; i < numIterations; i++) {
            getStorage(key);
        }
    }

    function bytesToBytes32(bytes memory source) private pure returns (bytes32 result) {
        if (source.length == 0) {
            return 0x0;
        }
        assembly {
            result := mload(add(source, 32))
        }
    }
}
