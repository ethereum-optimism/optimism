// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Test } from "forge-std/Test.sol";

// Interfaces
import { IPreimageOracle } from "interfaces/cannon/IPreimageOracle.sol";

import { DeployMIPS2 } from "scripts/deploy/DeployMIPS2.s.sol";
import { MIPS } from "src/cannon/MIPS.sol";
import { MIPS64 } from "src/cannon/MIPS64.sol";

contract DeployMIPS2_Test is Test {
    DeployMIPS2 deployMIPS;

    // Define default input variables for testing.
    IPreimageOracle defaultPreimageOracle = IPreimageOracle(makeAddr("PreimageOracle"));
    uint256 defaultMIPSVersion = 1;

    function setUp() public {
        deployMIPS = new DeployMIPS2();
    }

    function testFuzz_run_mipsVersion1_succeeds(DeployMIPS2.Input memory _input) public {
        vm.assume(address(_input.preimageOracle) != address(0));
        _input.mipsVersion = 1;

        // Run the deployment script.
        DeployMIPS2.Output memory output1 = deployMIPS.run(_input);

        // Make sure we deployed the correct MIPS
        MIPS mips = new MIPS(_input.preimageOracle);
        assertEq(address(output1.mipsSingleton).code, address(mips).code, "100");

        // Run the deployment script again
        DeployMIPS2.Output memory output2 = deployMIPS.run(_input);

        // Make sure the contract did not get redeployed
        assertEq(address(output1.mipsSingleton), address(output2.mipsSingleton), "200");
    }

    function testFuzz_run_mipsVersion2_succeeds(DeployMIPS2.Input memory _input) public {
        vm.assume(address(_input.preimageOracle) != address(0));
        _input.mipsVersion = 6;

        // Run the deployment script.
        DeployMIPS2.Output memory output1 = deployMIPS.run(_input);

        // Make sure we deployed the correct MIPS
        MIPS64 mips = new MIPS64(_input.preimageOracle, _input.mipsVersion);
        assertEq(address(output1.mipsSingleton).code, address(mips).code, "100");

        // Run the deployment script again
        DeployMIPS2.Output memory output2 = deployMIPS.run(_input);

        assertEq(address(output1.mipsSingleton), address(output2.mipsSingleton), "200");
    }

    function test_run_nullInput_reverts() public {
        DeployMIPS2.Input memory input;

        input = defaultInput();
        input.preimageOracle = IPreimageOracle(address(0));
        vm.expectRevert("DeployMIPS: preimageOracle not set");
        deployMIPS.run(input);

        input = defaultInput();
        input.mipsVersion = 0;
        vm.expectRevert("DeployMIPS: mipsVersion not set");
        deployMIPS.run(input);
    }

    function defaultInput() internal view returns (DeployMIPS2.Input memory input_) {
        input_ = DeployMIPS2.Input(defaultPreimageOracle, defaultMIPSVersion);
    }
}
