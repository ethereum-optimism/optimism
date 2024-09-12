// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Test, stdStorage, StdStorage } from "forge-std/Test.sol";

import { DeployOPChainInput } from "scripts/DeployOPChain.s.sol";
import { DeployOPChain_TestBase } from "test/DeployOPChain.t.sol";

import { OPStackManager } from "src/L1/OPStackManager.sol";
import { SuperchainConfig } from "src/L1/SuperchainConfig.sol";
import { ProtocolVersions } from "src/L1/ProtocolVersions.sol";

// Exposes internal functions for testing.
contract OPStackManager_Harness is OPStackManager {
    constructor(
        SuperchainConfig _superchainConfig,
        ProtocolVersions _protocolVersions,
        Blueprints memory _blueprints
    )
        OPStackManager(_superchainConfig, _protocolVersions, _blueprints)
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
        doi.set(doi.opsm.selector, address(opsm));
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
            l2ChainId: _doi.l2ChainId()
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
        opsm.deploy(toOPSMDeployInput(doi));
    }
}

// These tests use the harness which exposes internal functions for testing.
contract OPStackManager_InternalMethods_Test is Test {
    OPStackManager_Harness opsmHarness;

    function setUp() public {
        opsmHarness = new OPStackManager_Harness({
            _superchainConfig: SuperchainConfig(makeAddr("superchainConfig")),
            _protocolVersions: ProtocolVersions(makeAddr("protocolVersions")),
            _blueprints: OPStackManager.Blueprints({
                addressManager: makeAddr("addressManager"),
                proxy: makeAddr("proxy"),
                proxyAdmin: makeAddr("proxyAdmin"),
                l1ChugSplashProxy: makeAddr("l1ChugSplashProxy"),
                resolvedDelegateProxy: makeAddr("resolvedDelegateProxy")
            })
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
