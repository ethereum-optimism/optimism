// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.7.0;

import { iOVM_BondManager } from "../../iOVM/verification/iOVM_BondManager.sol";

/// Minimal contract to be inherited by contracts consumed by users that provide
/// data for fraud proofs
contract OVM_FraudContributor {
    iOVM_BondManager internal ovmBondManager;

    /// Decorate your functions with this modifier to store how much total gas was
    /// consumed by the sender, to reward users fairly
    modifier contributesToFraudProof(bytes32 preStateRoot) {
        uint startGas = gasleft();
        _;
        uint gasSpent = startGas - gasleft();
        ovmBondManager.recordGasSpent(preStateRoot, msg.sender, gasSpent);
    }
}
