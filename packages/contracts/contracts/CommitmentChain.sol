pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

/* Internal Imports */
import {DataTypes as dt} from "./DataTypes.sol";

contract CommitmentChain {
    function verifyInclusion(dt.StateUpdate memory _stateUpdate, bytes memory _inclusionProof) public returns (bool) {
        // Always return true for now until we can verify inclusion proofs.
        return true;
    }

    function getAbiPacked(
        dt.StateSubtreeNode memory _leftSibling,
        dt.StateSubtreeNode memory _rightSibling
    ) public pure returns (bytes memory) {
        bytes memory packed =
            abi.encodePacked(
                _leftSibling.hashValue,
                _leftSibling.start,
                _rightSibling.hashValue,
                _rightSibling.start
            );
        return packed;
    }
    function stateSubtreeParent(
        dt.StateSubtreeNode memory _leftSibling,
        dt.StateSubtreeNode memory _rightSibling
    ) public pure returns (dt.StateSubtreeNode memory) {
        dt.StateSubtreeNode memory parent;
        bytes32 computedHash = keccak256(
            abi.encodePacked(
                _leftSibling.hashValue,
                _leftSibling.start,
                _rightSibling.hashValue,
                _rightSibling.start
            )
        );
        parent.hashValue = computedHash;
        parent.start = _leftSibling.start;
        return parent;
    }
}
