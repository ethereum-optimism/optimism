// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing utilities
import { CommonTest } from "test/setup/CommonTest.sol";

contract DeployVariations_Test is CommonTest {
    function setUp() public override {
        // Prevent calling the base CommonTest.setUp() function, as we will run it within the test functions
        // after setting the feature flags
    }

    // Enable features which should be possible to enable or disable regardless of other options.
    function enableAddOns(bool _enableAltDa) public {
        if (_enableAltDa) {
            super.enableAltDA();
        }
    }

    /// @dev It should be possible to enable Fault Proofs without Alt-DA.
    function test_enableFaultProofs_succeeds() public virtual {
        enableAddOns(false);

        super.setUp();
    }

    /// @dev It should be possible to enable Fault Proofs with Alt-DA.
    function test_enableFaultProofsWithAltDa_succeeds() public virtual {
        enableAddOns(true);

        super.setUp();
    }

    /// @dev It should be possible to enable Fault Proofs and Interop without Alt-DA.
    function test_enableInteropAndFaultProofs_succeeds() public virtual {
        enableAddOns(false);
        super.enableInterop();

        super.setUp();
    }

    /// @dev It should be possible to enable Fault Proofs and Interop with Alt-DA.
    function test_enableInteropAndFaultProofsWithAltDa_succeeds() public virtual {
        enableAddOns(true);
        super.enableInterop();

        super.setUp();
    }
}
