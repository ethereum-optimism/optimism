// SPDX-License-Identifier: MIT
pragma solidity ^0.8.9;

/**
 * @title Semver
 * @notice Semver is a simple contract for managing contract versions.
 */
contract Semver {
    /**
     * @notice Contract version number (major).
     */
    uint256 public immutable MAJOR_VERSION;

    /**
     * @notice Contract version number (minor).
     */
    uint256 public immutable MINOR_VERSION;

    /**
     * @notice Contract version number (patch).
     */
    uint256 public immutable PATCH_VERSION;

    /**
     * @param _major Version number (major).
     * @param _minor Version number (minor).
     * @param _patch Version number (patch).
     */
    constructor(
        uint256 _major,
        uint256 _minor,
        uint256 _patch
    ) {
        MAJOR_VERSION = _major;
        MINOR_VERSION = _minor;
        PATCH_VERSION = _patch;
    }
}
