// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing utilities
import { Test } from "forge-std/Test.sol";

// Target contract
import { OPStackManager } from "src/L1/OPStackManager.sol";

// Exposes internal functions for testing.
contract OPStackManager_Harness is OPStackManager {
    function chainIdToBatchInboxAddress_exposed(uint256 l2ChainId) public pure returns (address) {
        return super.chainIdToBatchInboxAddress(l2ChainId);
    }
}

// Unlike other test suites, we intentionally do not inherit from CommonTest or Setup. This is
// because OPStackManager acts as a deploy script, so we start from a clean slate here and
// work OPStackManager's deployment into the existing test setup, instead of using the existing
// test setup to deploy OPStackManager.
contract OPStackManager_Init is Test {
    OPStackManager opsm;

    // Default dummy parameters for the deploy function.
    OPStackManager.Roles roles;
    uint256 l2ChainId = 1234;
    uint32 basefeeScalar = 1;
    uint32 blobBasefeeScalar = 1;

    function setUp() public {
        opsm = new OPStackManager();
    }
}

contract OPStackManager_Deploy_Test is OPStackManager_Init {
    function test_deploy_l2ChainIdEqualsZero_reverts() public {
        vm.expectRevert(OPStackManager.InvalidChainId.selector);
        opsm.deploy(0, basefeeScalar, blobBasefeeScalar, roles);
    }

    function test_deploy_l2ChainIdEqualsCurrentChainId_reverts() public {
        vm.expectRevert(OPStackManager.InvalidChainId.selector);
        opsm.deploy(block.chainid, basefeeScalar, blobBasefeeScalar, roles);
    }
}

// These tests use the harness which exposes internal functions for testing.
contract OPStackManager_InternalMethods_Test is Test {
    function test_calculatesBatchInboxAddress_succeeds() public {
        OPStackManager_Harness opsmHarness = new OPStackManager_Harness();

        // These test vectors were calculated manually:
        //   1. Compute the bytes32 encoding of the chainId: bytes32(uint256(chainId));
        //   2. Hash it and manually take the first 19 bytes, and prefixed it with 0x00.
        uint256 chainId = 1234;
        address expected = 0x0017FA14b0d73Aa6A26D6b8720c1c84b50984f5C;
        address actual = opsmHarness.chainIdToBatchInboxAddress_exposed(chainId);
        vm.assertEq(expected, actual);

        chainId = type(uint256).max;
        expected = 0x00a9C584056064687E149968cBaB758a3376D22A;
        actual = opsmHarness.chainIdToBatchInboxAddress_exposed(chainId);
        vm.assertEq(expected, actual);
    }
}
