pragma solidity ^0.5.0;

contract SimpleCallable {
  bytes public someStorage;

  constructor(bytes memory _someParameter) public {
    someStorage = _someParameter;
  }

  function update(bytes memory _someParameter) public returns (bytes memory) {
    bytes memory temp = someStorage;
    someStorage = _someParameter;
    return temp;
  }
}
