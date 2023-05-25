// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Test } from "forge-std/Test.sol";
import { OpStackChainRegistry } from "../periphery/OpStackChainRegistry.sol";

/**
 * @title  OpStackChainRegistryTest
 * @notice Test coverage of OpStackChainRegistry
 */
contract OpStackChainRegistryTest is Test {
    /**
     * @notice OpStackChainRegistry
     */
    OpStackChainRegistry public chainRegistry;

    function setUp() public {
        chainRegistry = new OpStackChainRegistry();
    }

    function _claim() public returns (OpStackChainRegistry.Deployment memory) {
        OpStackChainRegistry.Deployment memory _deployment = OpStackChainRegistry.Deployment("deployment", makeAddr("admin"));
        chainRegistry.claimDeployment(_deployment);

        return _deployment;
    }

    function _register(OpStackChainRegistry.Deployment memory _deployment) public returns (OpStackChainRegistry.DeploymentEntry[] memory) {
        OpStackChainRegistry.DeploymentEntry[] memory _entries = new OpStackChainRegistry.DeploymentEntry[](2);
        _entries[0] = OpStackChainRegistry.DeploymentEntry("entry1", makeAddr("entry1"));
        _entries[1] = OpStackChainRegistry.DeploymentEntry("entry2", makeAddr("entry2"));

        vm.prank(makeAddr("admin"));
        chainRegistry.register(_deployment, _entries);

        return _entries;
    }

    function test_claim() public {
        _claim();
    }

    function test_register() public {
        OpStackChainRegistry.Deployment memory _deployment = _claim();
        _register(_deployment);
    }

    function test_revertRegisterIfNotAdmin() public {
        OpStackChainRegistry.Deployment memory _deployment = _claim();

        vm.expectRevert(OpStackChainRegistry.OnlyDeploymentAdmin.selector);
        chainRegistry.register(_deployment, new OpStackChainRegistry.DeploymentEntry[](0));
    }

    function test_query() public {
        OpStackChainRegistry.Deployment memory _deployment = _claim();
        OpStackChainRegistry.DeploymentEntry[] memory _entries = _register(_deployment);

        for (uint256 i = 0; i < _entries.length; i++) {
            assertEq(chainRegistry.registry(_deployment.deploymentName, _entries[i].entryName), _entries[i].entryAddress);
        }
    }
}
