// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing utilities
import { CommonTest } from "test/setup/CommonTest.sol";
import { ERC20 } from "@openzeppelin/contracts/token/ERC20/ERC20.sol";

contract DeployVariations_Test is CommonTest {
    function setUp() public override {
        // prevent calling the base CommonTest.setUp() function
    }

    // Enable features which should be possible to enable or disable regardless of other options.
    function enableAddOns(
        bool _enableCGT,
        bool _enableAltDa
    ) public {
        if (_enableCGT) {
            ERC20 token = new ERC20("Silly", "SIL");
            super.enableCustomGasToken(address(token));
        }
        if (_enableAltDa) {
            super.enableAltDA();
        }
    }

    function testFuzz_enableFaultProofs(bool _enableCGT, bool _enableAltDa) public virtual {
        enableAddOns(_enableCGT, _enableAltDa);
        super.enableFaultProofs();
        super.setUp();
    }

    function test_enableInteropAndFaultProofs(bool _enableCGT, bool _enableAltDa) public virtual {
        enableAddOns(_enableCGT, _enableAltDa);
        super.enableInterop();
        super.enableFaultProofs();
        super.setUp();
    }
}
