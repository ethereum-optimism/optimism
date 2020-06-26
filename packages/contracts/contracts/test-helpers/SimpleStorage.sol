pragma solidity ^0.5.0;

contract SimpleStorage {
    mapping(bytes32 => bytes32) public builtInStorage;

    function setStorage(bytes32 key, bytes32 value) public {
        builtInStorage[key] = value;
    }

    function getStorage(bytes32 key) public view returns (bytes32) {
        return builtInStorage[key];
    }
}
