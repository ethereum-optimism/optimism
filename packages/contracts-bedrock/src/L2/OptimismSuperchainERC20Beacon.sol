// SPDX-License-Identifier: MIT
pragma solidity 0.8.25;

import { IBeacon } from "@openzeppelin/contracts-v5/proxy/beacon/IBeacon.sol";
import { ISemver } from "src/universal/ISemver.sol";

/// @custom:proxied
/// @custom:predeployed 0x4200000000000000000000000000000000000027
/// @title OptimismSuperchainERC20Beacon
/// @notice OptimismSuperchainERC20Beacon is the beacon proxy for the OptimismSuperchainERC20 implementation.
contract OptimismSuperchainERC20Beacon is IBeacon, ISemver {
    /// TODO: Replace with real implementation address
    /// @notice Address of the OptimismSuperchainERC20 implementation.
    address internal constant IMPLEMENTATION_ADDRESS = 0x0000000000000000000000000000000000000000;

    /// @notice Semantic version.
    /// @custom:semver 1.0.0-beta.1
    string public constant version = "1.0.0-beta.1";

    /// @inheritdoc IBeacon
    function implementation() external pure override returns (address) {
        return IMPLEMENTATION_ADDRESS;
    }
}
