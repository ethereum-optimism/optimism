// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Test } from "forge-std/Test.sol";

import { OPStackManager } from "src/L1/OPStackManager.sol";

// Exposes internal functions for testing.
contract OPStackManager_Harness is OPStackManager {
    constructor(
        address _releaseManager,
        string memory _latestVersion
    )
        OPStackManager(_releaseManager, _latestVersion)
    { }

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

    address releaseManager = makeAddr("releaseManager");
    string latestVersion = "op-contracts/latest";

    // Default dummy parameters for the deploy function.
    OPStackManager.Roles roles = OPStackManager.Roles({
        opChainProxyAdminOwner: makeAddr("opChainProxyAdminOwner"),
        systemConfigOwner: makeAddr("systemConfigOwner"),
        batcher: makeAddr("batcher"),
        unsafeBlockSigner: makeAddr("unsafeBlockSigner"),
        proposer: makeAddr("proposer"),
        challenger: makeAddr("challenger")
    });
    OPStackManager.DeployInput deployInput =
        OPStackManager.DeployInput({ roles: roles, basefeeScalar: 100, blobBasefeeScalar: 200, l2ChainId: 300 });

    function setUp() public {
        opsm = new OPStackManager(releaseManager, latestVersion);
    }
}

contract OPStackManager_Deploy_Test is OPStackManager_Init {
    function test_deploy_l2ChainIdEqualsZero_reverts() public {
        deployInput.l2ChainId = 0;
        vm.expectRevert(OPStackManager.InvalidChainId.selector);
        opsm.deploy(deployInput);
    }

    function test_deploy_l2ChainIdEqualsCurrentChainId_reverts() public {
        deployInput.l2ChainId = block.chainid;
        vm.expectRevert(OPStackManager.InvalidChainId.selector);
        opsm.deploy(deployInput);
    }
}

// These tests use the harness which exposes internal functions for testing.
contract OPStackManager_InternalMethods_Test is Test {
    address releaseManager = makeAddr("releaseManager");
    string latestVersion = "op-contracts/latest";

    function test_calculatesBatchInboxAddress_succeeds() public {
        OPStackManager_Harness opsmHarness = new OPStackManager_Harness(releaseManager, latestVersion);

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
