//SPDX-License-Identifier: MIT
pragma solidity 0.8.10;

/* Library Imports */
import { Lib_SecureMerkleTrie } from "./trie/Lib_SecureMerkleTrie.sol";
import { Lib_CrossDomainUtils } from "./Lib_CrossDomainUtils.sol";

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
     * @notice Derives the withdrawal hash according to the encoding in the L2 Withdrawer contract
     * @param _nonce Nonce for the provided message.
     * @param _sender Message sender address on L2.
     * @param _target Target address on L1.
     * @param _value ETH to send to the target.
     * @param _gasLimit Gas to be forwarded to the target.
     * @param _data Data to send to the target.
     */
    function withdrawalHash(
        uint256 _nonce,
        address _sender,
        address _target,
        uint256 _value,
        uint256 _gasLimit,
        bytes memory _data
    ) internal pure returns (bytes32) {
        return keccak256(abi.encode(_nonce, _sender, _target, _value, _gasLimit, _data));
    }

    /**
     * @notice Derives the output root corresponding to the elements provided in the proof.
     * @param _outputRootProof The elements which were hashed together to generate the output root.
     * @return Whether or not the output root matches the hashed output of the proof.
     */
    function _deriveOutputRoot(OutputRootProof memory _outputRootProof)
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
        bytes memory _withdrawalProof
    ) internal pure returns (bool) {
        bytes32 storageKey = keccak256(
            abi.encode(
                _withdrawalHash,
                uint256(0) // The withdrawals mapping is at the first slot in the layout.
            )
        );

        return
            Lib_SecureMerkleTrie.verifyInclusionProof(
                abi.encode(storageKey),
                hex"01",
                _withdrawalProof,
                _withdrawerStorageRoot
            );
    }
}
