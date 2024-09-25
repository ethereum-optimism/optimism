// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Test, stdStorage, StdStorage } from "forge-std/Test.sol";

import { DeployOPChainInput } from "scripts/DeployOPChain.s.sol";
import { DeployOPChain_TestBase } from "test/DeployOPChain.t.sol";

import { OPStackManager } from "src/L1/OPStackManager.sol";
import { SuperchainConfig } from "src/L1/SuperchainConfig.sol";
import { ProtocolVersions } from "src/L1/ProtocolVersions.sol";
import { SystemConfig } from "src/L1/SystemConfig.sol";

// Exposes internal functions for testing.
contract OPStackManager_Harness is OPStackManager {
    constructor(
        SuperchainConfig _superchainConfig,
        ProtocolVersions _protocolVersions
    )
        OPStackManager(_superchainConfig, _protocolVersions)
    { }

    function chainIdToBatchInboxAddress_exposed(uint256 l2ChainId) public pure returns (address) {
        return super.chainIdToBatchInboxAddress(l2ChainId);
    }
}

// Unlike other test suites, we intentionally do not inherit from CommonTest or Setup. This is
// because OPStackManager acts as a deploy script, so we start from a clean slate here and
// work OPStackManager's deployment into the existing test setup, instead of using the existing
// test setup to deploy OPStackManager. We do however inherit from DeployOPChain_TestBase so
// we can use its setup to deploy the implementations similarly to how a real deployment would
// happen.
contract OPStackManager_Deploy_Test is DeployOPChain_TestBase {
    using stdStorage for StdStorage;

    event Deployed(uint256 indexed l2ChainId, SystemConfig indexed systemConfig);

    function setUp() public override {
        DeployOPChain_TestBase.setUp();

        doi.set(doi.opChainProxyAdminOwner.selector, opChainProxyAdminOwner);
        doi.set(doi.systemConfigOwner.selector, systemConfigOwner);
        doi.set(doi.batcher.selector, batcher);
        doi.set(doi.unsafeBlockSigner.selector, unsafeBlockSigner);
        doi.set(doi.proposer.selector, proposer);
        doi.set(doi.challenger.selector, challenger);
        doi.set(doi.basefeeScalar.selector, basefeeScalar);
        doi.set(doi.blobBaseFeeScalar.selector, blobBaseFeeScalar);
        doi.set(doi.l2ChainId.selector, l2ChainId);
        doi.set(doi.opsmProxy.selector, address(opsm));
    }

    // This helper function is used to convert the input struct type defined in DeployOPChain.s.sol
    // to the input struct type defined in OPStackManager.sol.
    function toOPSMDeployInput(DeployOPChainInput _doi) internal view returns (OPStackManager.DeployInput memory) {
        return OPStackManager.DeployInput({
            roles: OPStackManager.Roles({
                opChainProxyAdminOwner: _doi.opChainProxyAdminOwner(),
                systemConfigOwner: _doi.systemConfigOwner(),
                batcher: _doi.batcher(),
                unsafeBlockSigner: _doi.unsafeBlockSigner(),
                proposer: _doi.proposer(),
                challenger: _doi.challenger()
            }),
            basefeeScalar: _doi.basefeeScalar(),
            blobBasefeeScalar: _doi.blobBaseFeeScalar(),
            l2ChainId: _doi.l2ChainId(),
            startingAnchorRoots: _doi.startingAnchorRoots()
        });
    }

    function test_deploy_l2ChainIdEqualsZero_reverts() public {
        OPStackManager.DeployInput memory deployInput = toOPSMDeployInput(doi);
        deployInput.l2ChainId = 0;
        vm.expectRevert(OPStackManager.InvalidChainId.selector);
        opsm.deploy(deployInput);
    }

    function test_deploy_l2ChainIdEqualsCurrentChainId_reverts() public {
        OPStackManager.DeployInput memory deployInput = toOPSMDeployInput(doi);
        deployInput.l2ChainId = block.chainid;

        vm.expectRevert(OPStackManager.InvalidChainId.selector);
        opsm.deploy(deployInput);
    }

    function test_deploy_succeeds() public {
        vm.expectEmit(true, false, true, true); // TODO precompute the system config address.
        emit Deployed(doi.l2ChainId(), SystemConfig(address(1)));
        opsm.deploy(toOPSMDeployInput(doi));
    }
}

// These tests use the harness which exposes internal functions for testing.
contract OPStackManager_InternalMethods_Test is Test {
    OPStackManager_Harness opsmHarness;

    function setUp() public {
        SuperchainConfig superchainConfigProxy = SuperchainConfig(makeAddr("superchainConfig"));
        ProtocolVersions protocolVersionsProxy = ProtocolVersions(makeAddr("protocolVersions"));
        vm.etch(address(superchainConfigProxy), hex"01");
        vm.etch(address(protocolVersionsProxy), hex"01");

        opsmHarness = new OPStackManager_Harness({
            _superchainConfig: superchainConfigProxy,
            _protocolVersions: protocolVersionsProxy
        });
    }

    function test_calculatesBatchInboxAddress_succeeds() public view {
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
