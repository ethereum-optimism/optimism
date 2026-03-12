// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Constants } from "src/libraries/Constants.sol";
import { DevFeatures } from "src/libraries/DevFeatures.sol";
import { ISemver } from "interfaces/universal/ISemver.sol";

/// @title L2DevFeatureFlags
/// @notice Predeploy that stores the development feature bitmap. This bitmap is set at genesis by the
///         DEPOSITOR_ACCOUNT and read by the L2ContractsManager during upgrades.
/// @custom:proxied true
/// @custom:predeploy 0x420000000000000000000000000000000000002D
contract L2DevFeatureFlags is ISemver {
    /// @notice Thrown when the caller is not the depositor account.
    error L2DevFeatureFlags_Unauthorized();

    /// @dev bytes32(uint256(keccak256("l2devfeatureflags.bitmap")) - 1)
    bytes32 private constant BITMAP_SLOT = 0xc8bc8f9195cfb2d040744aac63412d02ffc186ea9bd519039edc4666ee9032bc;

    /// @notice The semantic version of the L2DevFeatureFlags contract.
    /// @custom:semver 1.0.0
    string public constant version = "1.0.0";

    /// @notice Sets the development feature bitmap. Only callable by the DEPOSITOR_ACCOUNT.
    /// @param _bitmap The new development feature bitmap.
    function setDevFeatureBitmap(bytes32 _bitmap) external {
        if (msg.sender != Constants.DEPOSITOR_ACCOUNT) revert L2DevFeatureFlags_Unauthorized();

        assembly {
            sstore(BITMAP_SLOT, _bitmap)
        }
    }

    /// @notice Returns the development feature bitmap.
    /// @return bitmap_ The current development feature bitmap.
    function devFeatureBitmap() public view returns (bytes32 bitmap_) {
        assembly {
            bitmap_ := sload(BITMAP_SLOT)
        }
    }

    /// @notice Checks if a development feature is enabled.
    /// @param _feature The feature to check.
    /// @return enabled_ True if the feature is enabled, false otherwise.
    function isDevFeatureEnabled(bytes32 _feature) external view returns (bool enabled_) {
        enabled_ = DevFeatures.isDevFeatureEnabled(devFeatureBitmap(), _feature);
    }
}
