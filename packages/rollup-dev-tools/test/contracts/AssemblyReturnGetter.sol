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
  function get() public view returns (bytes memory) {
    bytes32 valToReturn;

    for (uint i = 0; i < 4; i++) {
        valToReturn |= bytes32(someStorage[i] & 0xFF) >> (i * 8);
    }

    uint numBytesToReturn = 4;
    assembly {
        function $allocate(size) -> pos {
            pos := mload(0x40)
            mstore(0x40, add(pos, size))
        }

        let return_val := valToReturn
        let return_length := numBytesToReturn
        let return_location := $allocate(return_length)
        mstore(return_location, return_val)
        return(return_location, return_length)
    }
  }
}
