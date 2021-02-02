// SPDX-License-Identifier: MIT
pragma solidity >0.5.0 <0.8.0;

import { OVM_BondManager } from "./../optimistic-ethereum/OVM/verification/OVM_BondManager.sol";

contract Mock_FraudVerifier {
    OVM_BondManager bondManager;

    mapping (bytes32 => address) transitioners;

    function setBondManager(OVM_BondManager _bondManager) public {
        bondManager = _bondManager;
    }

    function setStateTransitioner(bytes32 preStateRoot, bytes32 txHash, address addr) public {
        transitioners[keccak256(abi.encodePacked(preStateRoot, txHash))] = addr;
    }

    function getStateTransitioner(
        bytes32 _preStateRoot,
        bytes32 _txHash
    )
        public
        view
        returns (
            address
        )
    {
        return transitioners[keccak256(abi.encodePacked(_preStateRoot, _txHash))];
    }

    function finalize(bytes32 _preStateRoot, address publisher, uint256 timestamp) public {
        bondManager.finalize(_preStateRoot, publisher, timestamp);
    }
}
