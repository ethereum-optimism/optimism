// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { AddressManager } from "../legacy/AddressManager.sol";

/**
 * @title AddressDeprecator
 * @notice Contract to deprecate addresses in the AddressManager.
 */
contract AddressDeprecator {
    /**
     * @notice AddressManager contract.
     */
    AddressManager public immutable ADDRESS_MANAGER;

    /**
     * @notice AddressDeprecator constructor.
     * @param _addressManager AddressManager contract.
     */
    constructor(AddressManager _addressManager) {
        ADDRESS_MANAGER = _addressManager;
    }

    /**
     * @notice Removes deprecated addresses from the AddressManager.
     */
    function deprecateAddresses() external {
        // Remove all deprecated addresses from the AddressManager
        string[17] memory deprecated = [
            "OVM_CanonicalTransactionChain",
            "OVM_L2CrossDomainMessenger",
            "OVM_DecompressionPrecompileAddress",
            "OVM_Sequencer",
            "OVM_Proposer",
            "OVM_ChainStorageContainer-CTC-batches",
            "OVM_ChainStorageContainer-CTC-queue",
            "OVM_CanonicalTransactionChain",
            "OVM_StateCommitmentChain",
            "OVM_BondManager",
            "OVM_ExecutionManager",
            "OVM_FraudVerifier",
            "OVM_StateManagerFactory",
            "OVM_StateTransitionerFactory",
            "OVM_SafetyChecker",
            "OVM_L1MultiMessageRelayer",
            "BondManager"
        ];

        for (uint256 i = 0; i < deprecated.length; i++) {
            AddressManager(ADDRESS_MANAGER).setAddress(deprecated[i], address(0));
        }
    }
}
