// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

// Testing
import { Test } from "test/setup/Test.sol";

// Scripts
import { DeployAlphabetVM } from "scripts/deploy/DeployAlphabetVM.s.sol";

// Interfaces
import { IPreimageOracle } from "interfaces/cannon/IPreimageOracle.sol";

contract DeployAlphabetVM2_Test is Test {
    DeployAlphabetVM deployAlphanetVM;

    IPreimageOracle private preimageOracle = IPreimageOracle(makeAddr("preimageOracle"));
    bytes32 private absolutePrestate = bytes32(uint256(1));

    function setUp() public {
        deployAlphanetVM = new DeployAlphabetVM();
    }

    function test_run_succeeds() public {
        DeployAlphabetVM.Input memory input = defaultInput();
        DeployAlphabetVM.Output memory output = deployAlphanetVM.run(input);

        assertNotEq(address(output.alphabetVM), address(0), "100");
        assertEq(address(output.alphabetVM.oracle()), address(input.preimageOracle), "200");
    }

    /// @notice Tests that a second run finds the VM of the first, so that a deployment that resumes
    ///         after its dispute game was registered registers the same VM again.
    function test_run_twice_succeeds() public {
        DeployAlphabetVM.Input memory input = defaultInput();
        DeployAlphabetVM.Output memory output1 = deployAlphanetVM.run(input);
        DeployAlphabetVM.Output memory output2 = deployAlphanetVM.run(input);

        assertEq(address(output1.alphabetVM), address(output2.alphabetVM), "100");
    }

    function test_run_nullInput_reverts() public {
        DeployAlphabetVM.Input memory input;

        input = defaultInput();
        input.absolutePrestate = hex"";
        vm.expectRevert("DeployAlphabetVM: absolutePrestate not set");
        deployAlphanetVM.run(input);

        input = defaultInput();
        input.preimageOracle = IPreimageOracle(address(0));
        vm.expectRevert("DeployAlphabetVM: preimageOracle not set");
        deployAlphanetVM.run(input);
    }

    function defaultInput() private view returns (DeployAlphabetVM.Input memory input_) {
        input_ = DeployAlphabetVM.Input(absolutePrestate, preimageOracle);
    }
}
