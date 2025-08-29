// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Forge
import { Test } from "forge-std/Test.sol";

// Libraries
import { JSONParserLib } from "solady/src/utils/JSONParserLib.sol";
import { SemverComp } from "src/libraries/SemverComp.sol";

/// @title SemverComp_Harness
/// @notice Exposes internal functions of `SemverComp` for testing.
contract SemverComp_Harness {
    /// @notice Parses a semver string into a Semver struct. This is a wrapper around
    ///         `SemverComp.parse` that returns the major, minor, and patch components as
    ///         separate values.
    /// @param _semver The semver string to parse.
    /// @return major_ The major version.
    /// @return minor_ The minor version.
    /// @return patch_ The patch version.
    function parse(string memory _semver) external pure returns (uint256 major_, uint256 minor_, uint256 patch_) {
        SemverComp.Semver memory v = SemverComp.parse(_semver);
        return (v.major, v.minor, v.patch);
    }
}

/// @title SemverComp_TestInit
/// @notice Reusable test initialization for `SemverComp` tests.
contract SemverComp_TestInit is Test {
    SemverComp_Harness internal harness;

    /// @notice Sets up the test environment.
    function setUp() public {
        harness = new SemverComp_Harness();
    }

    /// @notice Asserts that the parsed semver components match the expected values.
    /// @param _semver The semver string to parse.
    /// @param _major The expected major version.
    /// @param _minor The expected minor version.
    /// @param _patch The expected patch version.
    function assertParsedEq(string memory _semver, uint256 _major, uint256 _minor, uint256 _patch) internal view {
        (uint256 major, uint256 minor, uint256 patch) = harness.parse(_semver);
        assertEq(major, _major, "major mismatch");
        assertEq(minor, _minor, "minor mismatch");
        assertEq(patch, _patch, "patch mismatch");
    }
}

/// @title SemverComp_parse_Test
/// @notice Tests the `parse` function behavior.
contract SemverComp_parse_Test is SemverComp_TestInit {
    /// @notice Parses the minimal version.
    function test_parse_basicZero_succeeds() external view {
        assertParsedEq("0.0.0", 0, 0, 0);
    }

    /// @notice Parses a standard version.
    function test_parse_basic123_succeeds() external view {
        assertParsedEq("1.2.3", 1, 2, 3);
    }

    /// @notice Ignores prerelease identifiers.
    function test_parse_withPrerelease_succeeds() external view {
        assertParsedEq("1.2.3-alpha", 1, 2, 3);
        assertParsedEq("1.2.3-alpha.1", 1, 2, 3);
        assertParsedEq("10.20.30-rc.1", 10, 20, 30);
    }

    /// @notice Ignores build metadata.
    function test_parse_withBuildMetadataOnly_succeeds() external view {
        assertParsedEq("1.2.3+build.5", 1, 2, 3);
        assertParsedEq("1.2.3+20240101", 1, 2, 3);
    }

    /// @notice Ignores prerelease and build metadata together.
    function test_parse_withPrereleaseAndBuild_succeeds() external view {
        assertParsedEq("1.2.3-rc.1+build.5", 1, 2, 3);
        assertParsedEq("2.0.0-beta+exp.sha.5114f85", 2, 0, 0);
    }

    /// @notice Reverts when fewer than 3 dot-separated core parts are present.
    function test_parse_lessThanThreeParts_reverts() external {
        vm.expectRevert(SemverComp.SemverComp_InvalidSemverParts.selector);
        harness.parse("1.2");

        vm.expectRevert(SemverComp.SemverComp_InvalidSemverParts.selector);
        harness.parse("1");

        vm.expectRevert(SemverComp.SemverComp_InvalidSemverParts.selector);
        harness.parse("");
    }

    /// @notice Current behavior: extra dot-components beyond the core 3 are ignored.
    function test_parse_extraDotComponents_succeeds() external view {
        assertParsedEq("1.2.3.4", 1, 2, 3);
        assertParsedEq("1.2.3.4.5", 1, 2, 3);
    }

    /// @notice Reverts on non-numeric core parts.
    function test_parse_nonNumeric_reverts() external {
        vm.expectRevert(JSONParserLib.ParsingFailed.selector);
        harness.parse("a.b.c");

        vm.expectRevert(JSONParserLib.ParsingFailed.selector);
        harness.parse("1.b.3");

        vm.expectRevert(JSONParserLib.ParsingFailed.selector);
        harness.parse("1.2.c");
    }

    /// @notice Reverts on certain commonly malformed inputs.
    function test_parse_malformedInputs_reverts() external {
        // Leading/trailing whitespace
        vm.expectRevert(JSONParserLib.ParsingFailed.selector);
        harness.parse(" 1.2.3");
        vm.expectRevert(JSONParserLib.ParsingFailed.selector);
        harness.parse("1.2.3 ");

        // "v" prefix
        vm.expectRevert(JSONParserLib.ParsingFailed.selector);
        harness.parse("v1.2.3");
    }
}

/// @title SemverComp_Eq_Test
/// @notice Tests the `eq` function behavior.
contract SemverComp_Eq_Test is SemverComp_TestInit {
    function test_eq_succeeds() external pure {
        assertTrue(SemverComp.eq("1.2.3", "1.2.3"));

        assertFalse(SemverComp.eq("1.2.3", "1.2.4"));
        assertFalse(SemverComp.eq("1.2.3", "1.3.3"));
        assertFalse(SemverComp.eq("1.2.3", "2.2.3"));
    }
}

/// @title SemverComp_Lt_Test
/// @notice Tests the `lt` function behavior.
contract SemverComp_Lt_Test is SemverComp_TestInit {
    function test_lt_succeeds() external pure {
        assertTrue(SemverComp.lt("1.2.3", "1.2.4"));
        assertTrue(SemverComp.lt("1.2.3", "1.3.0"));
        assertTrue(SemverComp.lt("1.2.3", "2.0.0"));

        assertFalse(SemverComp.lt("1.2.3", "1.2.3"));
        assertFalse(SemverComp.lt("1.2.3", "1.2.2"));
        assertFalse(SemverComp.lt("2.0.0", "1.9.9"));
    }
}

/// @title SemverComp_Lte_Test
/// @notice Tests the `lte` function behavior.
contract SemverComp_Lte_Test is SemverComp_TestInit {
    function test_lte_succeeds() external pure {
        assertTrue(SemverComp.lte("1.2.3", "1.2.3"));
        assertTrue(SemverComp.lte("1.2.3", "1.2.4"));
        assertTrue(SemverComp.lte("1.2.3", "1.3.0"));
        assertTrue(SemverComp.lte("1.2.3", "2.0.0"));

        assertFalse(SemverComp.lte("1.2.3", "1.2.2"));
        assertFalse(SemverComp.lte("2.0.0", "1.9.9"));
    }
}

/// @title SemverComp_Gt_Test
/// @notice Tests the `gt` function behavior.
contract SemverComp_Gt_Test is SemverComp_TestInit {
    function test_gt_succeeds() external pure {
        assertTrue(SemverComp.gt("1.2.4", "1.2.3"));
        assertTrue(SemverComp.gt("1.3.0", "1.2.3"));
        assertTrue(SemverComp.gt("2.0.0", "1.2.3"));

        assertFalse(SemverComp.gt("1.2.3", "1.2.3"));
        assertFalse(SemverComp.gt("1.2.2", "1.2.3"));
        assertFalse(SemverComp.gt("1.9.9", "2.0.0"));
    }
}

/// @title SemverComp_Gte_Test
/// @notice Tests the `gte` function behavior.
contract SemverComp_Gte_Test is SemverComp_TestInit {
    function test_gte_succeeds() external pure {
        assertTrue(SemverComp.gte("1.2.3", "1.2.3"));
        assertTrue(SemverComp.gte("1.2.4", "1.2.3"));
        assertTrue(SemverComp.gte("1.3.0", "1.2.3"));
        assertTrue(SemverComp.gte("2.0.0", "1.2.3"));

        assertFalse(SemverComp.gte("1.2.2", "1.2.3"));
        assertFalse(SemverComp.gte("1.9.9", "2.0.0"));
    }
}
