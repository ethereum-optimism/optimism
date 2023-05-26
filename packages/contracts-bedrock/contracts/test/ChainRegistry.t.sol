// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Test } from "forge-std/Test.sol";
import { ChainRegistry } from "../periphery/ChainRegistry.sol";

/**
 * @title  ChainRegistryTest
 * @notice Test coverage of ChainRegistry
 */
contract ChainRegistryTest is Test {
    /**
     * @notice ChainRegistry
     */
    ChainRegistry public chainRegistry;

    /**
     * @notice Deploys the ChainRegistry
     */
    function setUp() public {
        chainRegistry = new ChainRegistry();
    }

    /**
     * @notice Claims a deployment
     *
     * @return Returns the name of the deployment
     */
    function _claim() public returns (string memory) {
        string memory _deployment = "deployment";
        chainRegistry.claimDeployment(_deployment, makeAddr("admin"));

        return _deployment;
    }

    /**
     * @notice Registers entries for a deployment
     *
     * @param _deployment The name of the deployment to register entries for
     *
     * @return Returns the entries registered
     */
    function _register(string memory _deployment) public returns (ChainRegistry.DeploymentEntry[] memory) {
        ChainRegistry.DeploymentEntry[] memory _entries = new ChainRegistry.DeploymentEntry[](2);
        _entries[0] = ChainRegistry.DeploymentEntry("entry1", makeAddr("entry1"));
        _entries[1] = ChainRegistry.DeploymentEntry("entry2", makeAddr("entry2"));

        vm.prank(makeAddr("admin"));
        chainRegistry.register(_deployment, _entries);

        return _entries;
    }

    /**
     * @notice A user can set a deployment name and its admin
     */
    function test_claim() public {
        _claim();
    }

    /**
     * @notice A deployment admin can transfer ownership to a new admin
     */
    function test_transferAdmin() public {
        string memory _deployment = _claim();

        vm.prank(makeAddr("admin"));
        chainRegistry.transferAdmin(_deployment, makeAddr("newAdmin"));

        assertEq(chainRegistry.deployments(_deployment), makeAddr("newAdmin"));
    }

    /**
     * @notice The deployment admin can register contract addresses for the deployment
     */
    function test_register() public {
        string memory _deployment = _claim();
        _register(_deployment);
    }

    /**
     * @notice Only the admin can register contract addresses for the deployment
     */
    function test_revertRegisterIfNotAdmin() public {
        string memory _deployment = _claim();

        vm.expectRevert(ChainRegistry.OnlyDeploymentAdmin.selector);
        chainRegistry.register(_deployment, new ChainRegistry.DeploymentEntry[](0));
    }

    /**
     * @notice A user can query contract addresses for a deployment
     */
    function test_query() public {
        string memory _deployment = _claim();
        ChainRegistry.DeploymentEntry[] memory _entries = _register(_deployment);

        for (uint256 i = 0; i < _entries.length; i++) {
            assertEq(chainRegistry.registry(_deployment, _entries[i].entryName), _entries[i].entryAddress);
        }
    }
}
