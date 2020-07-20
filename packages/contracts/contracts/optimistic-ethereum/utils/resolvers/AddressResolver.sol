pragma solidity ^0.5.0;

contract AddressResolver {
    mapping (bytes32 => address) addresses;

    function setAddress(string memory _name, address _address) public {
        addresses[keccak256(abi.encodePacked(_name))] = _address;
    }

    function getAddress(string memory _name) public view returns (address) {
        return addresses[keccak256(abi.encodePacked(_name))];
    }
}