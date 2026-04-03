// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

// Libraries
import { LibString } from "@solady/utils/LibString.sol";
import { JSONParserLib } from "@solady/utils/JSONParserLib.sol";

/// @notice Library for comparing semver strings. Ignores prereleases and build metadata.
library SemverComp {
    /// @notice Struct representing a semver string.
    /// @custom:field major The major version number.
    /// @custom:field minor The minor version number.
    /// @custom:field patch The patch version number.
    struct Semver {
        uint256 major;
        uint256 minor;
        uint256 patch;
    }

    /// @notice Error thrown when a semver string has less than 3 parts.
    error SemverComp_InvalidSemverParts();

    /// @notice Parses a semver string into a Semver struct. Only handles the major, minor, and
    ///         patch numerical components, ignores prereleases and build metadata.
    /// @param _semver The semver string to parse.
    /// @return The parsed Semver struct.
    function parse(string memory _semver) internal pure returns (Semver memory) {
        string[] memory parts = LibString.split(_semver, ".");

        // We need at least 3 parts to be a valid semver, but we might have more parts if the
        // semver looks like "1.2.3-beta.4+build.5".
        if (parts.length < 3) {
            revert SemverComp_InvalidSemverParts();
        }

        // Split the patch component by hyphen, if it exists. We only want the first part of the
        // patch. We're ignoring prereleases and build versions in this library. We're handling
        // cases like 1.2.3-beta.4+build.5 as well as 1.2.3+build.5.
        string[] memory patchParts = LibString.split(parts[2], "-");
        string[] memory patchParts2 = LibString.split(patchParts[0], "+");

        // Parse the major, minor, and patch components. JSONParserLib will revert if the
        // components are not valid decimal numbers.
        return Semver({
            major: JSONParserLib.parseUint(parts[0]),
            minor: JSONParserLib.parseUint(parts[1]),
            patch: JSONParserLib.parseUint(patchParts2[0])
        });
    }

    /// @notice Compares two semver strings (=). Ignores prereleases and build metadata.
    /// @param _a The first semver string.
    /// @param _b The second semver string.
    /// @return True if the semver strings are equal, false otherwise.
    function eq(string memory _a, string memory _b) internal pure returns (bool) {
        Semver memory a = parse(_a);
        Semver memory b = parse(_b);
        return a.major == b.major && a.minor == b.minor && a.patch == b.patch;
    }

    /// @notice Compares two semver strings (<). Ignores prereleases and build metadata.
    /// @param _a The first semver string.
    /// @param _b The second semver string.
    /// @return True if the first semver string is less than the second, false otherwise.
    function lt(string memory _a, string memory _b) internal pure returns (bool) {
        Semver memory a = parse(_a);
        Semver memory b = parse(_b);
        return a.major < b.major || (a.major == b.major && a.minor < b.minor)
            || (a.major == b.major && a.minor == b.minor && a.patch < b.patch);
    }

    /// @notice Compares two semver strings (<=). Ignores prereleases and build metadata.
    /// @param _a The first semver string.
    /// @param _b The second semver string.
    /// @return True if the first semver string is less than or equal to the second, false otherwise.
    function lte(string memory _a, string memory _b) internal pure returns (bool) {
        return eq(_a, _b) || lt(_a, _b);
    }

    /// @notice Compares two semver strings (>). Ignores prereleases and build metadata.
    /// @param _a The first semver string.
    /// @param _b The second semver string.
    /// @return True if the first semver string is greater than the second, false otherwise.
    function gt(string memory _a, string memory _b) internal pure returns (bool) {
        return !eq(_a, _b) && !lt(_a, _b);
    }

    /// @notice Compares two semver strings (>=). Ignores prereleases and build metadata.
    /// @param _a The first semver string.
    /// @param _b The second semver string.
    /// @return True if the first semver string is greater than or equal to the second, false otherwise.
    function gte(string memory _a, string memory _b) internal pure returns (bool) {
        return eq(_a, _b) || gt(_a, _b);
    }

    /// @notice Checks if a semver string has extra tags (prerelease "-" or build metadata "+").
    /// @param _semver The semver string to check.
    /// @return True if the semver has extra tags, false otherwise.
    function hasExtraTag(string memory _semver) internal pure returns (bool) {
        return LibString.contains(_semver, "-") || LibString.contains(_semver, "+");
    }
}
