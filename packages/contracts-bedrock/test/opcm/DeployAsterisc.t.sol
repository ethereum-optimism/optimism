// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { Test } from "test/setup/Test.sol";

// Scripts
import { DeployUtils } from "scripts/libraries/DeployUtils.sol";
import { DeployAsterisc } from "scripts/deploy/DeployAsterisc.s.sol";

// Interfaces
import { IPreimageOracle } from "interfaces/cannon/IPreimageOracle.sol";

contract DeployAsterisc_Test is Test {
    DeployAsterisc deployAsterisc;

    // Define default input variables for testing.
    IPreimageOracle defaultPreimageOracle = IPreimageOracle(makeAddr("preimageOracle"));

    function setUp() public {
        deployAsterisc = new DeployAsterisc();
    }

    function test_run_succeeds(DeployAsterisc.Input memory _input) public {
        vm.assume(address(_input.preimageOracle) != address(0));

        DeployAsterisc.Output memory output = deployAsterisc.run(_input);

        DeployUtils.assertValidContractAddress(address(output.asteriscSingleton));
        assertEq(address(output.asteriscSingleton.oracle()), address(_input.preimageOracle), "100");
    }

    function test_run_nullInput_reverts() public {
        DeployAsterisc.Input memory input;

        input = defaultInput();
        input.preimageOracle = IPreimageOracle(address(0));
        vm.expectRevert("DeployAsterisc: preimageOracle not set");
        deployAsterisc.run(input);
    }

    function defaultInput() internal view returns (DeployAsterisc.Input memory input_) {
        input_ = DeployAsterisc.Input(defaultPreimageOracle);
    }
}
