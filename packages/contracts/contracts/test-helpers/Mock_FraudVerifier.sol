// SPDX-License-Identifier: MIT
pragma solidity ^0.7.0;

import { OVM_BondManager } from "./../optimistic-ethereum/OVM/verification/OVM_BondManager.sol";

contract Mock_FraudVerifier {
    OVM_BondManager bondManager;

    mapping (bytes32 => address) transitioners;

    function setBondManager(OVM_BondManager _bondManager) public {
        bondManager = _bondManager;
    }

    function setStateTransitioner(bytes32 preStateRoot, address addr) public {
        transitioners[preStateRoot] = addr;
    }

    function getStateTransitioner(bytes32 preStateRoot) public view returns (address) {
        return transitioners[preStateRoot];
    }

    function finalize(bytes32 _preStateRoot, uint256 batchIndex, address publisher, uint256 timestamp) public {
        bondManager.finalize(_preStateRoot, batchIndex, publisher, timestamp);
    }
}
