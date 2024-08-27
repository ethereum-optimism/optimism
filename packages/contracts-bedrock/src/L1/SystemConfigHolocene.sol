// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { OptimismPortalInterop as OptimismPortal } from "src/L1/OptimismPortalInterop.sol";
import { SystemConfig } from "src/L1/SystemConfig.sol";
import { ConfigType } from "src/L2/L1BlockInterop.sol";
import { StaticConfig } from "src/libraries/StaticConfig.sol";

/// @title SystemConfigHolocene
/// @notice The SystemConfig contract is used to manage configuration of an Optimism network.
///         All configuration is stored on L1 and picked up by L2 as part of the derviation of
///         the L2 chain.
contract SystemConfigHolocene is SystemConfig {
    error Deprecated();

    /// @notice Internal function for updating the gas config.
    ///         `setGasConfig` is deprecated, use `setFeeScalars` instead.
    ///         `setGasConfig` will be removed when SystemConfigHolocene is
    ///         pulled into SystemConfig.
    function _setGasConfig(uint256, uint256) internal pure override {
        revert Deprecated();
    }

    /// @notice Internal function for updating the fee scalars as of the Ecotone upgrade.
    ///         Deprecated, use `setFeeScalars` instead.
    /// @param _basefeeScalar     New basefeeScalar value.
    /// @param _blobBasefeeScalar New blobBasefeeScalar value.
    function _setGasConfigEcotone(uint32 _basefeeScalar, uint32 _blobBasefeeScalar) internal override {
        _setFeeScalars(_basefeeScalar, _blobBasefeeScalar);
    }

    /// @notice Updates the fee scalars. Can only be called by the owner.
    /// @param _basefeeScalar     New basefeeScalar value.
    /// @param _blobBasefeeScalar New blobBasefeeScalar value.
    function setFeeScalars(uint32 _basefeeScalar, uint32 _blobBasefeeScalar) external onlyOwner {
        _setFeeScalars(_basefeeScalar, _blobBasefeeScalar);
    }

    /// @notice Internal function for updating the fee scalars.
    /// @param _basefeeScalar     New basefeeScalar value.
    /// @param _blobBasefeeScalar New blobBasefeeScalar value.
    function _setFeeScalars(uint32 _basefeeScalar, uint32 _blobBasefeeScalar) internal {
        basefeeScalar = _basefeeScalar;
        blobbasefeeScalar = _blobBasefeeScalar;

        uint256 _scalar = (uint256(0x01) << 248) | (uint256(_blobBasefeeScalar) << 32) | _basefeeScalar;
        scalar = _scalar;

        OptimismPortal(payable(optimismPortal())).setConfig(
            ConfigType.SET_FEE_SCALARS, StaticConfig.encodeSetFeeScalars({ _scalar: _scalar })
        );
    }

    /// @notice Internal setter for the batcher hash.
    /// @param _batcherHash New batcher hash.
    function _setBatcherHash(bytes32 _batcherHash) internal override {
        if (batcherHash != _batcherHash) {
            batcherHash = _batcherHash;
            OptimismPortal(payable(optimismPortal())).setConfig(
                ConfigType.SET_BATCHER_HASH, StaticConfig.encodeSetBatcherHash({ _batcherHash: _batcherHash })
            );
        }
    }

    /// @notice Internal function for updating the L2 gas limit.
    /// @param _gasLimit New gas limit.
    function _setGasLimit(uint64 _gasLimit) internal override {
        require(_gasLimit >= minimumGasLimit(), "SystemConfig: gas limit too low");
        require(_gasLimit <= maximumGasLimit(), "SystemConfig: gas limit too high");
        gasLimit = _gasLimit;

        OptimismPortal(payable(optimismPortal())).setConfig(
            ConfigType.SET_GAS_LIMIT, StaticConfig.encodeSetGasLimit({ _gasLimit: _gasLimit })
        );
    }
}
