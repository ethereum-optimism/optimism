pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

contract Commitment {
   struct Range {
    uint256 start;
    uint256 end;
  }

  struct StateObject {
    address predicateAddress;
    bytes data;
  }

  struct StateUpdate {
    Range range;
    StateObject stateObject;
    address plasmaContract;
    uint256 plasmaBlockNumber;
  }
    function verifyInclusion(StateUpdate memory _stateUpdate, bytes memory _inclusionProof) public returns (bool) {}
}
