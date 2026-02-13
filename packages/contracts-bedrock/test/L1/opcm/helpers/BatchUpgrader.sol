// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { IOPContractsManagerV2 } from "interfaces/L1/opcm/IOPContractsManagerV2.sol";
import { LibString } from "@solady/utils/LibString.sol";

/// @title BatchUpgrader
/// @notice Helper contract for testing that multiple upgrade operations can be executed
///         within a single transaction. Used to enforce the OPCMV2 invariant that
///         approximately 5 upgrade operations should be executable in one transaction.
contract BatchUpgrader {
    /// @notice The OPContractsManagerV2 instance to use for upgrades.
    IOPContractsManagerV2 public immutable opcm;

    /// @param _opcm The OPContractsManagerV2 contract address.
    constructor(IOPContractsManagerV2 _opcm) {
        opcm = _opcm;
    }

    /// @notice Executes multiple upgrade operations sequentially in a single transaction.
    /// @param _inputs Array of upgrade inputs, one per chain to upgrade.
    function batchUpgrade(IOPContractsManagerV2.UpgradeInput[] memory _inputs) external {
        for (uint256 i = 0; i < _inputs.length; i++) {
            (bool success, bytes memory returnData) =
                address(opcm).delegatecall(abi.encodeCall(IOPContractsManagerV2.upgrade, (_inputs[i])));
            require(
                success,
                string.concat(
                    "BatchUpgrader: upgrade failed for chain ", LibString.toString(i), ": ", string(returnData)
                )
            );
        }
    }
}
