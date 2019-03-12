pragma solidity ^0.5.0;

import "./IPredicate.sol";

contract PreimagePredicate is IPredicate {
    struct PreimageData {
        bytes32 hash;
    }

    struct PreimageWitnessData {
        bytes preimage;   
    }

    function validStateTransition(bytes memory _oldState, bytes memory _newState, bytes memory _witness) public view returns (bool) {
        PreimageData memory oldStateData = parsePreimageData(_oldState);
        PreimageWitnessData memory witness = parsePreimageWitness(_witness);
        
        bool validPreimage = keccak256(witness.preimage) == oldStateData.hash;
    
        return validPreimage;
    }

    function parsePreimageData(bytes memory _state) internal pure returns (PreimageData memory) {
        bytes memory data = abi.decode(_state, (bytes));
        bytes32 hash = abi.decode(data, (bytes32));
        return PreimageData({
            hash: hash 
        });
    }

    function parsePreimageWitness(bytes memory _witness) internal pure returns (PreimageWitnessData memory) {
        bytes memory witness = abi.decode(_witness, (bytes));
        return PreimageWitnessData({
            preimage: witness
        });
    }
}
