// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { CommonTest } from "test/setup/CommonTest.sol";

import { IDisputeGameFactory } from "../../interfaces/dispute/IDisputeGameFactory.sol";
import { IAnchorStateRegistry } from "../../interfaces/dispute/IAnchorStateRegistry.sol";
import { AnchorStateRegistry } from "../../src/dispute/AnchorStateRegistry.sol";
import { SuperRootMigrator } from "../../src/L1/SuperRootMigrator.sol";

contract SuperRootMigrator_TestBase is CommonTest {
    SuperRootMigrator internal superRootMigrator;

    IDisputeGameFactory[] internal gameFactories;
    IAnchorStateRegistry[] internal anchorStateRegistries;
    uint256[] internal chainIDs;

    function setUp() public virtual override {
        super.setUp();

        gameFactories = new IDisputeGameFactory[](1);
        gameFactories[0] = disputeGameFactory; // From CommonTest

        anchorStateRegistries = new IAnchorStateRegistry[](1);
        anchorStateRegistries[0] = anchorStateRegistry; // From CommonTest

        chainIDs = new uint256[](1);
        chainIDs[0] = block.chainid; // TODO: validate

        superRootMigrator = new SuperRootMigrator(gameFactories, anchorStateRegistries, chainIDs);

        vm.label(address(superRootMigrator), "SuperRootMigrator");
        vm.label(address(gameFactories[0]), "DisputeGameFactory (Chain 1)");
        vm.label(address(anchorStateRegistries[0]), "AnchorStateRegistry (Chain 1)");
    }
}

contract SuperRootMigrator_Setup_Test is SuperRootMigrator_TestBase {
    function test_setUp_succeeds() external view {
        assertEq(superRootMigrator.chainIDs(0), chainIDs[0], "ChainID mismatch");
        assertEq(
            address(superRootMigrator.anchorStateRegistries(chainIDs[0])),
            address(anchorStateRegistries[0]),
            "AnchorStateRegistry mismatch"
        );
        assertEq(address(superRootMigrator.gameFactories(0)), address(gameFactories[0]), "GameFactory mismatch");
        assertEq(superRootMigrator.chainsLen(), 1, "Incorrect chains length");
    }
}
