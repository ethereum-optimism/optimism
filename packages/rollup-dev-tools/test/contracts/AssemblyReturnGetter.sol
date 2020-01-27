pragma solidity ^0.5.0;

contract AssemblyReturnGetter {
  bytes public someStorage;

  constructor(bytes memory _someParameter) public {
    someStorage = _someParameter;
  }

  function update(bytes memory _someParameter) public returns (bytes memory) {
    bytes memory temp = someStorage;
    someStorage = _someParameter;
    return temp;
  }

  // this getter uses inline assembly to return a NON-ABI-encoded byte array,
  // so it's easier to work with on the recieving end (assembly).
  function get() public view returns (bytes32) {
    bytes32 valToReturn;

    for (uint i = 0; i < someStorage.length; i++) {
        valToReturn |= bytes32(someStorage[i] & 0xFF) >> (i * 8);
    }

    return valToReturn;
  }
}
