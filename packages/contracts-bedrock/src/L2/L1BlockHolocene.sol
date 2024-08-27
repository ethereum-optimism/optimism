// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { L1Block, ConfigType } from "src/L2/L1Block.sol";
import { StaticConfig } from "src/libraries/StaticConfig.sol";
import "src/libraries/L1BlockErrors.sol";

/// @custom:proxied
/// @custom:predeploy 0x4200000000000000000000000000000000000015
/// @title L1BlockHolocene
/// @notice Holocene extenstions of L1Block.
contract L1BlockHolocene is L1Block {
    /// @notice Event emitted when a new batcher hash is set.
    event BatcherHashSet(bytes32 indexed batcherHash);

    /// @notice Event emitted when new fee scalars are set.
    event FeeScalarsSet(uint32 indexed blobBasefeeScalar, uint32 indexed basefeeScalar);

    /// @notice Event emitted when a new gas limit is set.
    event GasLimitSet(uint64 indexed gasLimit);

    /// @notice The gas limit of L2 blocks in the same epoch.
    uint64 public gasLimit;

    /// @notice Sets static configuration options for the L2 system. Can only be called by the special
    ///         depositor account.
    /// @param _type  The type of configuration to set.
    /// @param _value The encoded value with which to set the configuration.
    function setConfig(ConfigType _type, bytes calldata _value) public override {
        if (msg.sender != DEPOSITOR_ACCOUNT()) revert NotDepositor();

        if (_type == ConfigType.SET_BATCHER_HASH) {
            _setBatcherHash(_value);
        } else if (_type == ConfigType.SET_FEE_SCALARS) {
            _setFeeScalars(_value);
        } else if (_type == ConfigType.SET_GAS_LIMIT) {
            _setGasLimit(_value);
        } else {
            super.setConfig(_type, _value);
        }
    }

    /// @notice Internal method to set new batcher hash.
    /// @param _value The encoded value with which to set the new batcher hash.
    function _setBatcherHash(bytes calldata _value) internal {
        bytes32 _batcherHash = StaticConfig.decodeSetBatcherHash(_value);

        batcherHash = _batcherHash;

        emit BatcherHashSet(_batcherHash);
    }

    /// @notice Internal method to set new fee scalars.
    /// @param _value The encoded value with which to set the new fee scalars.
    function _setFeeScalars(bytes calldata _value) internal {
        uint256 _scalar = StaticConfig.decodeSetFeeScalars(_value);

        (uint32 _blobBasefeeScalar, uint32 _basefeeScalar) = _decodeScalar(_scalar);

        blobBaseFeeScalar = _blobBasefeeScalar;
        baseFeeScalar = _basefeeScalar;

        emit FeeScalarsSet(_blobBasefeeScalar, _basefeeScalar);
    }

    /// @notice Internal method to decode blobBaseFeeScalar and baseFeeScalar.
    /// @return Decoded blobBaseFeeScalar and baseFeeScalar.
    function _decodeScalar(uint256 _scalar) internal pure returns (uint32, uint32) {
        // _scalar is constructed as follows:
        // uint256 _scalar = (uint256(0x01) << 248) | (uint256(_blobbasefeeScalar) << 32) | _basefeeScalar;
        // where _blobbasefeeScalar and _basefeeScalar are both uint32.

        require(0x01 == _scalar >> 248, "invalid _scalar");

        uint32 _blobBasefeeScalar = uint32((_scalar >> 32) & 0xffffffff);
        uint32 _basefeeScalar = uint32(_scalar & 0xffffffff);
        return (_blobBasefeeScalar, _basefeeScalar);
    }

    /// @notice Internal method to set new gas limit.
    /// @param _value The encoded value with which to set the new gas limit.
    function _setGasLimit(bytes calldata _value) internal {
        uint64 _gasLimit = StaticConfig.decodeSetGasLimit(_value);
        gasLimit = _gasLimit;
        emit GasLimitSet(_gasLimit);
    }

    /// @notice Updates the L1 block values for an Holocene upgraded chain.
    /// Params are packed and passed in as raw msg.data instead of ABI to reduce calldata size.
    /// Params are expected to be in the following order:
    ///   1. _sequenceNumber     Number of L2 blocks since epoch start.
    ///   2. _timestamp          L1 timestamp.
    ///   3. _number             L1 blocknumber.
    ///   4. _basefee            L1 base fee.
    ///   5. _blobBaseFee        L1 blob base fee.
    ///   6. _hash               L1 blockhash.
    function setL1BlockValuesHolocene() external {
        address depositor = DEPOSITOR_ACCOUNT();
        uint64 _sequenceNumber;
        assembly {
            // Revert if the caller is not the depositor account.
            if xor(caller(), depositor) {
                mstore(0x00, 0x3cc50b45) // 0x3cc50b45 is the 4-byte selector of "NotDepositor()"
                revert(0x1C, 0x04) // returns the stored 4-byte selector from above
            }
            // sequencenum (uint64)
            _sequenceNumber := shr(192, calldataload(4))
        }

        sequenceNumber = _sequenceNumber;

        // for each L2 block, include only the sequence number, except for L2 blocks with sequencer #0,
        // and they'd have all the L1 origin related attributes following the 0 sequence number.
        if (_sequenceNumber != 0) return;

        assembly {
            // number (uint64) and timestamp (uint64)
            sstore(number.slot, shr(128, calldataload(12)))
            sstore(basefee.slot, calldataload(28)) // uint256
            sstore(blobBaseFee.slot, calldataload(60)) // uint256
            sstore(hash.slot, calldataload(92)) // bytes32
        }
    }
}
