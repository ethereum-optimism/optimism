pragma solidity ^0.5.0;

contract SimpleStorage {
    mapping(bytes32 => bytes32) public builtInStorage;

    function setStorage(bytes32 key, bytes32 value) public {
        builtInStorage[key] = value;
    }

    function getStorage(bytes32 key) public view returns (bytes32) {
        return builtInStorage[key];
    }

    function setStorages(bytes32 key, bytes32 value) public {
        for (uint i = 0; i < 20; i++) {
            setStorage(key, value);
        }
    }

    function getStorages(bytes32 key) public {
        for (uint i = 0; i < 20; i++) {
            getStorage(key);
        }
    }
}
