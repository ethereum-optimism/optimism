// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing utilities
import { Test } from "forge-std/Test.sol";

// Target contract
import { OPStackManager } from "src/L1/OPStackManager.sol";

// Unlike other test suites, we intentionally do not inherit from CommonTest or Setup. This is
// because OPStackManager acts as a deploy script, so we start from a clean slate here and
// work OPStackManager's deployment into the existing test setup, instead of using the existing
// test setup to deploy OPStackManager.
contract OPStackManager_Test is Test {
    OPStackManager opStackManager;

    // Default dummy parameters for the deploy function.
    OPStackManager.Roles roles;
    uint256 l2ChainId = 1234;
    uint32 basefeeScalar = 1;
    uint32 blobBasefeeScalar = 1;

    function setUp() public {
        opStackManager = new OPStackManager();
    }
}

contract OPStackManager_Deploy_Test is OPStackManager_Test {
    function test_RevertsIf_L2ChainIdEqualsZero() public {
        vm.expectRevert(OPStackManager.InvalidChainId.selector);
        opStackManager.deploy(0, roles, basefeeScalar, blobBasefeeScalar);
    }

    function test_RevertsIf_L2ChainIdEqualsCurrentChainId() public {
        vm.expectRevert(OPStackManager.InvalidChainId.selector);
        opStackManager.deploy(block.chainid, roles, basefeeScalar, blobBasefeeScalar);
    }
}
