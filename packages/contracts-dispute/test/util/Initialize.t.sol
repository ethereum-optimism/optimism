// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

import { Test } from "forge-std/Test.sol";
import { Initializable } from "src/util/Initializable.sol";

/// @title Initialize_Test
/// @notice Tests the [Initialize] contract.
contract Initialize_Test is Test {
    TestInitialize internal tInit;

    /// @notice An event emitted by the [Initialize] contract upon initialization.
    event Initialized();

    /// @notice An error thrown by the [Initialize] contract when attempting to initialize more than once.
    error AlreadyInitialized();

    function setUp() public {
        tInit = new TestInitialize();
    }

    /// @notice Asserts that the [Initialize] contract can be initialized.
    function test_init_succeeds() public {
        // Should succeed
        vm.expectEmit(false, false, false, false);
        emit Initialized();
        tInit.initialize();

        // Assert that the contract was initialized correctly
        assertTrue(tInit.initialized());
        assertEq(tInit.a(), 1);
    }

    /// @notice Asserts that the [Initialize] contract cannot be initialized more than once.
    function test_init_cannotInitializeTwice_reverts() public {
        test_init_succeeds();

        // Assert that the contract cannot be initialized twice.
        vm.expectRevert(AlreadyInitialized.selector);
        tInit.initialize();
    }
}

/// @title TestInitialize
/// @notice A mock [Initialize] contract.
contract TestInitialize is Initializable {
    uint256 public a;

    /// @notice Initializes the test contract
    function initialize() external initializer {
        a = 1;
    }
}
