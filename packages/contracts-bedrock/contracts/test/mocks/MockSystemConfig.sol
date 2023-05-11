// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

import { ISystemConfig } from "../../dispute/ISystemConfig.sol";

/**
 * @title MockSystemConfig
 * @notice A mock contract for the SystemConfig contract.
 */
contract MockSystemConfig is ISystemConfig {
    /**
     * @notice Enum representing different types of updates.
     *
     * @custom:value BATCHER              Represents an update to the batcher hash.
     * @custom:value GAS_CONFIG           Represents an update to txn fee config on L2.
     * @custom:value GAS_LIMIT            Represents an update to gas limit on L2.
     * @custom:value UNSAFE_BLOCK_SIGNER  Represents an update to the signer key for unsafe
     *                                    block distrubution.
     * @custom:value SIGNER_SET           Represents an update to the signer set.
     * @custom:value SIGNATURE_THRESHOLD  Represents an update to the signature threshold.
     */
    enum UpdateType {
        BATCHER,
        GAS_CONFIG,
        GAS_LIMIT,
        UNSAFE_BLOCK_SIGNER,
        SIGNER_SET,
        SIGNATURE_THRESHOLD
    }

    /**
     * @notice Storage slot that the unsafe block signer is stored at. Storing it at this
     *         deterministic storage slot allows for decoupling the storage layout from the way
     *         that `solc` lays out storage. The `op-node` uses a storage proof to fetch this value.
     */
    bytes32 public constant UNSAFE_BLOCK_SIGNER_SLOT = keccak256("systemconfig.unsafeblocksigner");

    /**
     * @notice Emitted when configuration is updated
     *
     * @param version    SystemConfig version.
     * @param updateType Type of update.
     * @param data       Encoded update data.
     */
    event ConfigUpdate(uint256 indexed version, UpdateType indexed updateType, bytes data);

    /**
     * @notice Version identifier, used for upgrades.
     */
    uint256 public constant VERSION = 0;

    /**
     * @notice The `signerSet` is a set of addresses that are allowed to issue positive attestations
     *         for alternative output proposals in the `AttestationDisputeGame`.
     */
    address[] internal _signerSet;

    /**
     * @notice The `signatureThreshold` is the number of positive attestations that must be issued
     *         for a given alternative output proposal in the `AttestationDisputeGame` before it is
     *         considered to be the canonical output.
     */
    uint256 public override signatureThreshold;

    /**
     * @notice A getter for the signer set.
     *
     * @return A list of addresses.
     */
    function signerSet() external view returns (address[] memory) {
        return _signerSet;
    }

    /**
     * @notice An external setter for the `signerSet` mapping. This method is used to
     *         authenticate or deauthenticate a signer in the `AttestationDisputeGame`.
     * @param _signer Address of the signer to authenticate or deauthenticate.
     * @param _authenticated True if the signer should be authenticated, false if the
     *        signer should be removed.
     */
    function authenticateSigner(address _signer, bool _authenticated) external {
        uint256 len = _signerSet.length;
        for (uint256 i = 0; i < len; i++) {
            if (_signerSet[i] == _signer) {
                if (_authenticated) {
                    revert("SystemConfig: signer already authenticated");
                } else {
                    // Remove the signer from the array by swapping it with the last signer
                    // and then popping the last element.
                    _signerSet[i] = _signerSet[len - 1];
                    _signerSet.pop();
                    emit ConfigUpdate(VERSION, UpdateType.SIGNER_SET, abi.encode(_signer, false));
                    return;
                }
            }
        }
        if (_authenticated) {
            _signerSet.push(_signer);
        }
        emit ConfigUpdate(VERSION, UpdateType.SIGNER_SET, abi.encode(_signer, _authenticated));
    }

    /**
     * @notice An external setter for the `signatureThreshold` variable. This method is used to
     *         set the number of signatures required to invalidate an output proposal
     *         in the `AttestationDisputeGame`.
     */
    function setSignatureThreshold(uint256 _signatureThreshold) external {
        require(
            _signatureThreshold > 0,
            "SystemConfig: signature threshold must be greater than 0"
        );

        signatureThreshold = _signatureThreshold;
        emit ConfigUpdate(VERSION, UpdateType.SIGNATURE_THRESHOLD, abi.encode(_signatureThreshold));
    }
}
