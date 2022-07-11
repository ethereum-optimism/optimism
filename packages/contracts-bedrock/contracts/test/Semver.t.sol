// SPDX-License-Identifier: MIT
pragma solidity ^0.8.9;

import { CommonTest } from "./CommonTest.t.sol";
import { Semver } from "../universal/Semver.sol";
import { Proxy } from "../universal/Proxy.sol";

/**
 * @notice Test the Semver contract that is used for semantic versioning
 *         of various contracts.
 */
contract Semver_Test is CommonTest {
    /**
     * @notice Global semver contract deployed in setUp. This is used in
     *         the test cases.
     */
    Semver semver;

    /**
     * @notice Deploy a Semver contract
     */
    function setUp() external {
        semver = new Semver(7, 8, 9);
    }

    /**
     * @notice Test the getter of the major version
     */
    function test_major() external {
        assertEq(
            semver.MAJOR_VERSION(),
            7
        );
    }

    /**
     * @notice Test the getter of the minor version
     */
    function test_minor() external {
        assertEq(
            semver.MINOR_VERSION(),
            8
        );
    }

    /**
     * @notice Test the getter of the patch version
     */
    function test_patch() external {
        assertEq(
            semver.PATCH_VERSION(),
            9
        );
    }

    /**
     * @notice Since the versions are all immutable, they should
     *         be able to be accessed from behind a proxy without needing
     *         to initialize the contract.
     */
    function test_behindProxy() external {
        Proxy proxy = new Proxy(alice);
        vm.prank(alice);
        proxy.upgradeTo(address(semver));

        assertEq(
            Semver(address(proxy)).MAJOR_VERSION(),
            7
        );

        assertEq(
            Semver(address(proxy)).MINOR_VERSION(),
            8
        );

        assertEq(
            Semver(address(proxy)).PATCH_VERSION(),
            9
        );
    }
}
