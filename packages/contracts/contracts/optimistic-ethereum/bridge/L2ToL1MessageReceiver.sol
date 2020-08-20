pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

/* Library Imports */
import { DataTypes } from "../utils/libraries/DataTypes.sol";

/**
 * @title L2ToL1MessageReceiver
 */
contract L2ToL1MessageReceiver {
    enum MessageStatus { unverified, verified, activated }
    function verifyMessage (DataTypes.L2ToL1Proof memory proof) public {
        bytes32 msgDigest = proof.value;
        // First verify which state root batch this state root is in.
        // StateCommitmentChain.verifyElement(
        //     proof.stateTrieRoot,
        //     proof.stateRootIndex, 
        //     proof.stateChainWitness
        // )
        // Verify that the state root was committed over a week ago
        // require(proof.stateChainWitness.batchHeader.timestamp + 1 week < now);

        //
        // EthMerkleTrie.proveAccountStorageSlotValue(
        //     L2ToL1MessagePasserAddress,
        //     proof.key,
        //     proof.value,
        //     proof.stateTrieWitness,
        //     proof.storageTrieWitness,
        //     proof.stateTrieRoot
        // )
    }
}