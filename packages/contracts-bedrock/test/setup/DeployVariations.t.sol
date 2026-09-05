// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing utilities
import { CommonTest } from "test/setup/CommonTest.sol";

contract DeployVariations_Test is CommonTest {
    function setUp() public override {
        // Prevent calling the base CommonTest.setUp() function, as we will run it within the test functions
        // after setting the feature flags
    }

    /// @dev It should be possible to enable Fault Proofs.
    function test_enableFaultProofs_succeeds() public virtual {
        super.setUp();
    }

    /// @dev It should be possible to enable Fault Proofs and Interop.
    function test_enableInteropAndFaultProofs_succeeds() public virtual {
        super.enableInterop();

        super.setUp();
    }
}
