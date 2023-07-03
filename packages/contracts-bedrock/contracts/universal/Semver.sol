// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { Strings } from "@openzeppelin/contracts/utils/Strings.sol";

/// @title Semver
/// @notice Semver is a simple contract for managing contract versions.
contract Semver {
    /// @notice Contract version number (major).
    uint256 private immutable MAJOR_VERSION;

    /// @notice Contract version number (minor).
    uint256 private immutable MINOR_VERSION;

    /// @notice Contract version number (patch).
    uint256 private immutable PATCH_VERSION;

    /// @param _major Version number (major).
    /// @param _minor Version number (minor).
    /// @param _patch Version number (patch).
    constructor(
        uint256 _major,
        uint256 _minor,
        uint256 _patch
    ) {
        MAJOR_VERSION = _major;
        MINOR_VERSION = _minor;
        PATCH_VERSION = _patch;
    }

    /// @notice Returns the full semver contract version.
    /// @return Semver contract version as a string.
    function version() public view returns (string memory) {
        return
            string(
                abi.encodePacked(
                    Strings.toString(MAJOR_VERSION),
                    ".",
                    Strings.toString(MINOR_VERSION),
                    ".",
                    Strings.toString(PATCH_VERSION)
                )
            );
    }
}
