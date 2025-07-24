// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { Test } from "forge-std/Test.sol";

import { DeployPreimageOracle2 } from "scripts/deploy/DeployPreimageOracle2.s.sol";

contract DeployPreimageOracle2_Test is Test {
    DeployPreimageOracle2 deployPreimageOracle;

    uint256 defaultMinProposalSize = 1000;
    uint256 defaultChallengePeriod = 7 days;

    function setUp() public {
        deployPreimageOracle = new DeployPreimageOracle2();
    }

    function testFuzz_run_succeeds(DeployPreimageOracle2.Input memory _input, uint64 _challengePeriod) public {
        vm.assume(_input.minProposalSize != 0);
        vm.assume(_challengePeriod != 0);

        _input.challengePeriod = _challengePeriod;

        DeployPreimageOracle2.Output memory output = deployPreimageOracle.run(_input);

        assertNotEq(address(output.preimageOracle), address(0));
        assertEq(output.preimageOracle.minProposalSize(), _input.minProposalSize);
        assertEq(output.preimageOracle.challengePeriod(), _input.challengePeriod);
    }

    function test_run_nullInputs_reverts() public {
        DeployPreimageOracle2.Input memory input;

        input = defaultInput();
        input.minProposalSize = 0;
        vm.expectRevert("DeployPreimageOracle: minProposalSize not set");
        deployPreimageOracle.run(input);

        input = defaultInput();
        input.challengePeriod = 0;
        vm.expectRevert("DeployPreimageOracle: challengePeriod not set");
        deployPreimageOracle.run(input);
    }

    function defaultInput() internal view returns (DeployPreimageOracle2.Input memory input_) {
        input_ = DeployPreimageOracle2.Input({
            minProposalSize: defaultMinProposalSize,
            challengePeriod: defaultChallengePeriod
        });
    }
}
