// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { Test } from "test/setup/Test.sol";

// Scripts
import { DeployMIPS } from "scripts/deploy/DeployMIPS.s.sol";
import { StandardConstants } from "scripts/deploy/StandardConstants.sol";

// Contracts
import { MIPS64 } from "src/cannon/MIPS64.sol";

// Interfaces
import { IPreimageOracle } from "interfaces/cannon/IPreimageOracle.sol";

contract DeployMIPS_Test is Test {
    DeployMIPS deployMIPS;

    // Define default input variables for testing.
    IPreimageOracle defaultPreimageOracle = IPreimageOracle(makeAddr("PreimageOracle"));
    uint256 defaultMIPSVersion = 1;

    function setUp() public {
        deployMIPS = new DeployMIPS();
    }

    function testFuzz_run_mipsVersion2_succeeds(DeployMIPS.Input memory _input) public {
        vm.assume(address(_input.preimageOracle) != address(0));
        _input.mipsVersion = StandardConstants.MIPS_VERSION;

        // Run the deployment script.
        DeployMIPS.Output memory output1 = deployMIPS.run(_input);

        // Make sure we deployed the correct MIPS
        MIPS64 mips = new MIPS64(_input.preimageOracle, _input.mipsVersion);
        assertEq(address(output1.mipsSingleton).code, address(mips).code, "100");

        // Run the deployment script again
        DeployMIPS.Output memory output2 = deployMIPS.run(_input);

        assertEq(address(output1.mipsSingleton), address(output2.mipsSingleton), "200");
    }

    function test_run_nullInput_reverts() public {
        DeployMIPS.Input memory input;

        input = defaultInput();
        input.preimageOracle = IPreimageOracle(address(0));
        vm.expectRevert("DeployMIPS: preimageOracle not set");
        deployMIPS.run(input);

        input = defaultInput();
        input.mipsVersion = 0;
        vm.expectRevert("DeployMIPS: mipsVersion not set");
        deployMIPS.run(input);
    }

    function defaultInput() internal view returns (DeployMIPS.Input memory input_) {
        input_ = DeployMIPS.Input(defaultPreimageOracle, defaultMIPSVersion);
    }
}
