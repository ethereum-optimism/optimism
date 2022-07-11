// SPDX-License-Identifier: MIT
pragma solidity 0.8.10;

import { Encoding } from "./Encoding.sol";

/**
 * @title Hashing
 * @notice Hashing handles Optimism's various different hashing schemes.
 */
library Hashing {
    /// @notice A struct containing the elements hashed together to generate the output root.
    struct OutputRootProof {
        bytes32 version;
        bytes32 stateRoot;
        bytes32 withdrawerStorageRoot;
        bytes32 latestBlockhash;
    }

    /**
     * @notice Compute the L2 transaction hash given
     * data about an L1 deposit transaction. This is useful for
     * environments that do not have access to arbitrary
     * RLP encoding functionality but have access to the
     * standard web3 API
     * TODO: rearrange args in a sane way
     * @param _l1BlockHash The L1 block hash corresponding to the block
     * the deposit was included in
     * @param _logIndex The log index of the event that the deposit was
     * created from. This can be found on the transaction receipt
     * @param _from The sender of the deposit
     * @param _to The L2 contract to be called by the deposit transaction
     * @param _isCreate Indicates if the deposit creates a contract
     * @param _mint The amount of ETH being minted by the transaction
     * @param _value The amount of ETH send in the L2 call
     * @param _gas The gas limit for the L2 call
     */
    function L2TransactionHash(
        bytes32 _l1BlockHash,
        uint256 _logIndex,
        address _from,
        address _to,
        bool _isCreate,
        uint256 _mint,
        uint256 _value,
        uint256 _gas,
        bytes memory _data
    ) internal pure returns (bytes32) {
        bytes memory raw = Encoding.L2Transaction(
            _l1BlockHash,
            _logIndex,
            _from,
            _to,
            _isCreate,
            _mint,
            _value,
            _gas,
            _data
        );

        return keccak256(raw);
    }

    /**
     * @notice Compute the deposit transaction source hash.
     * This value ensures that the L2 transaction hash is unique
     * and deterministic based on L1 execution
     * @param l1BlockHash The L1 blockhash corresponding to the block including
     * the deposit
     * @param logIndex The index of the log that created the deposit transaction
     */
    function sourceHash(bytes32 l1BlockHash, uint256 logIndex) internal pure returns (bytes32) {
        bytes32 depositId = keccak256(abi.encode(l1BlockHash, logIndex));
        return keccak256(abi.encode(bytes32(0), depositId));
    }

    /**
     * @notice Compute the cross domain hash based on the versioned nonce
     */
    function getVersionedHash(
        uint256 _nonce,
        address _sender,
        address _target,
        uint256 _value,
        uint256 _gasLimit,
        bytes memory _data
    ) internal pure returns (bytes32) {
        uint16 version = Encoding.getVersionFromNonce(_nonce);
        if (version == 0) {
            return getHashV0(_target, _sender, _data, _nonce);
        } else if (version == 1) {
            return getHashV1(_nonce, _sender, _target, _value, _gasLimit, _data);
        }

        revert("Unknown version.");
    }

    /**
     * @notice Compute the legacy hash of a cross domain message
     */
    function getHashV0(
        address _target,
        address _sender,
        bytes memory _data,
        uint256 _nonce
    ) internal pure returns (bytes32) {
        return keccak256(Encoding.getEncodingV0(_target, _sender, _data, _nonce));
    }

    /**
     * @notice Compute the V1 hash of a cross domain message
     */
    function getHashV1(
        uint256 _nonce,
        address _sender,
        address _target,
        uint256 _value,
        uint256 _gasLimit,
        bytes memory _data
    ) internal pure returns (bytes32) {
        return
            keccak256(Encoding.getEncodingV1(_nonce, _sender, _target, _value, _gasLimit, _data));
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
}
