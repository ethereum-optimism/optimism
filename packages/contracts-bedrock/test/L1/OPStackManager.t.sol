// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Test } from "forge-std/Test.sol";

import { DeployOPChainInput } from "scripts/DeployOPChain.s.sol";
import { DeployOPChain_TestBase } from "test/DeployOPChain.t.sol";

import { Proxy } from "src/universal/Proxy.sol";
import { ProxyAdmin } from "src/universal/ProxyAdmin.sol";

import { L1ChugSplashProxy } from "src/legacy/L1ChugSplashProxy.sol";
import { ResolvedDelegateProxy } from "src/legacy/ResolvedDelegateProxy.sol";
import { AddressManager } from "src/legacy/AddressManager.sol";

import { DelayedWETH } from "src/dispute/weth/DelayedWETH.sol";
import { DisputeGameFactory } from "src/dispute/DisputeGameFactory.sol";
import { AnchorStateRegistry } from "src/dispute/AnchorStateRegistry.sol";
import { FaultDisputeGame } from "src/dispute/FaultDisputeGame.sol";
import { PermissionedDisputeGame } from "src/dispute/PermissionedDisputeGame.sol";

import { OPStackManager } from "src/L1/OPStackManager.sol";
import { SuperchainConfig } from "src/L1/SuperchainConfig.sol";
import { ProtocolVersions } from "src/L1/ProtocolVersions.sol";
import { OptimismPortal2 } from "src/L1/OptimismPortal2.sol";
import { SystemConfig } from "src/L1/SystemConfig.sol";
import { ResourceMetering } from "src/L1/ResourceMetering.sol";
import { L1CrossDomainMessenger } from "src/L1/L1CrossDomainMessenger.sol";
import { L1ERC721Bridge } from "src/L1/L1ERC721Bridge.sol";
import { L1StandardBridge } from "src/L1/L1StandardBridge.sol";
import { OptimismMintableERC20Factory } from "src/universal/OptimismMintableERC20Factory.sol";

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
    // This helper function is used to convert the input struct type defined in DeployOPChain.s.sol
    // to the input struct type defined in OPStackManager.sol.
    function toOPSMDeployInput(DeployOPChainInput.Input memory input)
        internal
        pure
        returns (OPStackManager.DeployInput memory)
    {
        return OPStackManager.DeployInput({
            roles: OPStackManager.Roles({
                opChainProxyAdminOwner: input.roles.opChainProxyAdminOwner,
                systemConfigOwner: input.roles.systemConfigOwner,
                batcher: input.roles.batcher,
                unsafeBlockSigner: input.roles.unsafeBlockSigner,
                proposer: input.roles.proposer,
                challenger: input.roles.challenger
            }),
            basefeeScalar: input.basefeeScalar,
            blobBasefeeScalar: input.blobBaseFeeScalar,
            l2ChainId: input.l2ChainId
        });
    }

    function test_deploy_l2ChainIdEqualsZero_reverts() public {
        deployOPChainInput.l2ChainId = 0;
        vm.expectRevert(OPStackManager.InvalidChainId.selector);
        deployImplementationsOutput.opsmSingleton.deploy(toOPSMDeployInput(deployOPChainInput));
    }

    function test_deploy_l2ChainIdEqualsCurrentChainId_reverts() public {
        deployOPChainInput.l2ChainId = block.chainid;
        vm.expectRevert(OPStackManager.InvalidChainId.selector);
        deployImplementationsOutput.opsmSingleton.deploy(toOPSMDeployInput(deployOPChainInput));
    }

    function test_deploy_succeeds() public {
        deployImplementationsOutput.opsmSingleton.deploy(toOPSMDeployInput(deployOPChainInput));
    }
}

// These tests use the harness which exposes internal functions for testing.
contract OPStackManager_InternalMethods_Test is Test {
    OPStackManager_Harness opsmHarness;

    function setUp() public {
        opsmHarness = new OPStackManager_Harness({
            _superchainConfig: SuperchainConfig(makeAddr("superchainConfig")),
            _protocolVersions: ProtocolVersions(makeAddr("protocolVersions"))
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
