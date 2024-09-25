// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { IBeacon } from "@openzeppelin/contracts/proxy/beacon/IBeacon.sol";
import { ISemver } from "src/universal/interfaces/ISemver.sol";

/// @custom:proxied
/// @custom:predeployed 0x4200000000000000000000000000000000000027
/// @title OptimismSuperchainERC20Beacon
/// @notice OptimismSuperchainERC20Beacon is the beacon proxy for the OptimismSuperchainERC20 implementation.
contract OptimismSuperchainERC20Beacon is IBeacon, ISemver {
    /// @notice Address of the OptimismSuperchainERC20 implementation.
    address internal immutable IMPLEMENTATION;

    /// @notice Semantic version.
    /// @custom:semver 1.0.0-beta.1
    string public constant version = "1.0.0-beta.1";

    constructor(address _implementation) {
        IMPLEMENTATION = _implementation;
    }

    /// @inheritdoc IBeacon
    function implementation() external view override returns (address) {
        return IMPLEMENTATION;
    }
}
