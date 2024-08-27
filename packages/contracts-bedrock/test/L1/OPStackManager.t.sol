// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Test } from "forge-std/Test.sol";

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
// test setup to deploy OPStackManager.
contract OPStackManager_Init is Test {
    OPStackManager opsm;

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

    OPStackManager.ImplementationSetter[] setters;

    function setUp() public {
        setters.push(
            OPStackManager.ImplementationSetter({
                name: "L1ERC721Bridge",
                info: OPStackManager.Implementation(makeAddr("l1ERC721Bridge"), L1ERC721Bridge.initialize.selector)
            })
        );
        setters.push(
            OPStackManager.ImplementationSetter({
                name: "OptimismPortal",
                info: OPStackManager.Implementation(makeAddr("optimismPortal"), OptimismPortal2.initialize.selector)
            })
        );
        setters.push(
            OPStackManager.ImplementationSetter({
                name: "SystemConfig",
                info: OPStackManager.Implementation(makeAddr("systemConfig"), SystemConfig.initialize.selector)
            })
        );
        setters.push(
            OPStackManager.ImplementationSetter({
                name: "OptimismMintableERC20Factory",
                info: OPStackManager.Implementation(
                    makeAddr("optimismMintableERC20Factory"), OptimismMintableERC20Factory.initialize.selector
                )
            })
        );
        setters.push(
            OPStackManager.ImplementationSetter({
                name: "L1CrossDomainMessenger",
                info: OPStackManager.Implementation(
                    makeAddr("l1CrossDomainMessenger"), L1CrossDomainMessenger.initialize.selector
                )
            })
        );

        opsm = new OPStackManager({
            _superchainConfig: SuperchainConfig(makeAddr("superchainConfig")),
            _protocolVersions: ProtocolVersions(makeAddr("protocolVersions"))
        });
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

    function test_deploy_succeeds() public {
        // Currently skipped because OPSM is not fully implemented yet, so the deploy method reverts.
        // This is also why we don't yet use the DeployOPChain script here.
        vm.skip(true);
        opsm.deploy(deployInput);
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
