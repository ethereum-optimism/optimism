// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Test } from "forge-std/Test.sol";

import { DeployUtils } from "scripts/libraries/DeployUtils.sol";

// Interfaces
import { IPreimageOracle } from "interfaces/cannon/IPreimageOracle.sol";

import { DeployAsterisc2 } from "scripts/deploy/DeployAsterisc2.s.sol";

contract DeployAsterisc2_Test is Test {
    DeployAsterisc2 deployAsterisc;

    // Define default input variables for testing.
    IPreimageOracle defaultPreimageOracle = IPreimageOracle(makeAddr("preimageOracle"));

    function setUp() public {
        deployAsterisc = new DeployAsterisc2();
    }

    function test_run_succeeds(DeployAsterisc2.Input memory _input) public {
        vm.assume(address(_input.preimageOracle) != address(0));

        DeployAsterisc2.Output memory output = deployAsterisc.run(_input);

        DeployUtils.assertValidContractAddress(address(output.asteriscSingleton));
        assertEq(address(output.asteriscSingleton.oracle()), address(_input.preimageOracle), "100");
    }

    function test_run_nullInput_reverts() public {
        DeployAsterisc2.Input memory input;

        input = defaultInput();
        input.preimageOracle = IPreimageOracle(address(0));
        vm.expectRevert("DeployAsterisc: preimageOracle not set");
        deployAsterisc.run(input);
    }

    function defaultInput() internal view returns (DeployAsterisc2.Input memory input_) {
        input_ = DeployAsterisc2.Input(defaultPreimageOracle);
    }
}
