// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { OptimismPortalInterop as OptimismPortal } from "src/L1/OptimismPortalInterop.sol";
import { SystemConfigInterop } from "src/L1/SystemConfigInterop.sol";
import { ConfigType } from "src/L2/L1BlockInterop.sol";
import { StaticConfig } from "src/libraries/StaticConfig.sol";

/// @title SystemConfigHolocene
/// @notice The SystemConfig contract is used to manage configuration of an Optimism network.
///         All configuration is stored on L1 and picked up by L2 as part of the derviation of
///         the L2 chain.
contract SystemConfigHolocene is SystemConfigInterop {
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

    /// @notice Internal function for updating the fee scalars as of the Ecotone upgrade.
    /// @param _basefeeScalar     New basefeeScalar value.
    /// @param _blobbasefeeScalar New blobbasefeeScalar value.
    function _setGasConfigEcotone(uint32 _basefeeScalar, uint32 _blobbasefeeScalar) internal override {
        basefeeScalar = _basefeeScalar;
        blobbasefeeScalar = _blobbasefeeScalar;

        uint256 _scalar = (uint256(0x01) << 248) | (uint256(_blobbasefeeScalar) << 32) | _basefeeScalar;
        scalar = _scalar;

        OptimismPortal(payable(optimismPortal())).setConfig(
            ConfigType.SET_GAS_CONFIG_ECOTONE, StaticConfig.encodeSetGasConfigEcotone({ _scalar: _scalar })
        );
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
