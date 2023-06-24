// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { CommonTest } from "./CommonTest.t.sol";
import { Semver } from "../universal/Semver.sol";
import { Proxy } from "../universal/Proxy.sol";

/// @notice Test the Semver contract that is used for semantic versioning
///         of various contracts.
contract Semver_Test is CommonTest {
    /// @notice Global semver contract deployed in setUp. This is used in
    ///         the test cases.
    Semver semver;

    /// @notice Deploy a Semver contract
    function setUp() public virtual override {
        semver = new Semver(7, 8, 0);
    }

    /// @notice Test the version getter
    function test_version_succeeds() external {
        assertEq(semver.version(), "7.8.0");
    }

    /// @notice Since the versions are all immutable, they should
    ///         be able to be accessed from behind a proxy without needing
    ///         to initialize the contract.
    function test_behindProxy_succeeds() external {
        Proxy proxy = new Proxy(alice);
        vm.prank(alice);
        proxy.upgradeTo(address(semver));

        assertEq(Semver(address(proxy)).version(), "7.8.0");
    }
}
