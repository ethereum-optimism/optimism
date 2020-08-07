pragma solidity ^0.5.0;

contract SimpleStorage {
    mapping(bytes32 => string) public builtInStorage;
    function setStorage(string memory key, string memory val) public {
        builtInStorage[keccak256(abi.encode(key))] = val;
    }

    function getStorage(string memory key) public view returns (string memory) {
        return builtInStorage[keccak256(abi.encode(key))];
    }
}
