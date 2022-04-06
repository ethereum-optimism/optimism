//SPDX-License-Identifier: MIT
pragma solidity 0.8.10;

/* Library Imports */
import {
    Lib_SecureMerkleTrie
} from "../../../lib/optimism/packages/contracts/contracts/libraries/trie/Lib_SecureMerkleTrie.sol";

/**
 * @title WithdrawalVerifier
 * @notice A library with helper functions for verifying a withdrawal on L1.
 */
library WithdrawalVerifier {
    /// @notice A struct containing the elements hashed together to generate the output root.
    struct OutputRootProof {
        bytes32 version;
        bytes32 stateRoot;
        bytes32 withdrawerStorageRoot;
        bytes32 latestBlockhash;
    }

    /**
     * @notice Derives the output root corresponding to the elements provided in the proof.
     * @param _outputRootProof The elements which were hashed together to generate the output root.
     * @return Whether or not the output root matches the hashed output of the proof.
     */
    function _deriveOutputRoot(OutputRootProof calldata _outputRootProof)
        internal
        pure
        returns (bytes32)
    {
        return
            keccak256(
                abi.encode(
                    _outputRootProof.version,
                    _outputRootProof.stateRoot,
                    _outputRootProof.withdrawerStorageRoot,
                    _outputRootProof.latestBlockhash
                )
            );
    }

    /**
     * @notice Verifies a proof that a given withdrawal hash is present in the Withdrawer contract's
     * withdrawals mapping.
     * @param _withdrawalHash Keccak256 hash of the withdrawal transaction data.
     * @param _withdrawerStorageRoot Storage root of the withdrawer predeploy contract.
     * @param _withdrawalProof Merkle trie inclusion proof for the desired node.
     * @return Whether or not the inclusion proof was successful.
     */
    function _verifyWithdrawalInclusion(
        bytes32 _withdrawalHash,
        bytes32 _withdrawerStorageRoot,
        bytes calldata _withdrawalProof
    ) internal pure returns (bool) {
        bytes32 storageKey = keccak256(
            abi.encode(
                _withdrawalHash,
                uint256(1) // The withdrawals mapping is at the second slot in the layout.
            )
        );

        return
            Lib_SecureMerkleTrie.verifyInclusionProof(
                abi.encodePacked(storageKey),
                hex"01",
                _withdrawalProof,
                _withdrawerStorageRoot
            );
    }
}
