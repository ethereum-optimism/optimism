// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { CommonTest } from "test/setup/CommonTest.sol";
import { console2 as console } from "forge-std/console2.sol";

// Scripts
import { ExecuteNUTBundle } from "scripts/upgrade/ExecuteNUTBundle.s.sol";
import { GenerateNUTBundle } from "scripts/upgrade/GenerateNUTBundle.s.sol";

// Libraries
import { DevFeatures } from "src/libraries/DevFeatures.sol";

// Reuse all test logic from L2ForkUpgrade — only setUp differs
import {
    L2ForkUpgrade_TestInit,
    L2ForkUpgrade_Versions_Test,
    L2ForkUpgrade_Initialization_Test,
    L2ForkUpgrade_Implementations_Test,
    L2ForkUpgrade_Events_Test
} from "test/L2/fork/L2ForkUpgrade.t.sol";

/// @title L2GenesisForkUpgrade_TestInit
/// @notice Provides a genesis-based setUp for the L2 upgrade tests.
///         Reuses all test logic from L2ForkUpgrade by inheriting L2ForkUpgrade_TestInit,
///         but replaces its setUp to start from locally-deployed L2 genesis state instead
///         of a live forked L2 chain.
abstract contract L2GenesisForkUpgrade_TestInit is L2ForkUpgrade_TestInit {
    function setUp() public virtual override {
        // Directly call CommonTest.setUp() to run L1 + L2 genesis deployment,
        // bypassing L2ForkUpgrade_TestInit.setUp() which requires a live fork.
        CommonTest.setUp();

        // Skip if running against any fork — this test targets local genesis state only
        skipIfForkTest("genesis upgrade test, not for L1 fork");

        // Skip if L2CM dev feature is not enabled
        skipIfDevFeatureDisabled(DevFeatures.L2CM);

        // Initialize scripts
        executeScript = new ExecuteNUTBundle();
        generateScript = new GenerateNUTBundle();

        // Generate bundle
        generateScript.run();

        // Capture feature flags from deploy config (genesis state)
        commonState.isInteropEnabled =
            DevFeatures.isDevFeatureEnabled(deploy.cfg().devFeatureBitmap(), DevFeatures.OPTIMISM_PORTAL_INTEROP);
        console.log("L2GenesisForkUpgrade isInteropEnabled", commonState.isInteropEnabled);

        commonState.isCustomGasToken = deploy.cfg().useCustomGasToken();
        console.log("L2GenesisForkUpgrade: isCustomGasToken", commonState.isCustomGasToken);
    }
}

/// @title L2GenesisForkUpgrade_Versions_Test
/// @notice Tests that all predeploy versions are updated after the upgrade from genesis.
contract L2GenesisForkUpgrade_Versions_Test is L2GenesisForkUpgrade_TestInit, L2ForkUpgrade_Versions_Test {
    function setUp() public override(L2GenesisForkUpgrade_TestInit, L2ForkUpgrade_TestInit) {
        L2GenesisForkUpgrade_TestInit.setUp();
    }
}

/// @title L2GenesisForkUpgrade_Initialization_Test
/// @notice Tests that all initialization configurations are preserved after the upgrade from genesis.
contract L2GenesisForkUpgrade_Initialization_Test is
    L2GenesisForkUpgrade_TestInit,
    L2ForkUpgrade_Initialization_Test
{
    function setUp() public override(L2GenesisForkUpgrade_TestInit, L2ForkUpgrade_TestInit) {
        L2GenesisForkUpgrade_TestInit.setUp();
    }
}

/// @title L2GenesisForkUpgrade_Implementations_Test
/// @notice Tests that all predeploy implementations are correctly upgraded from genesis.
contract L2GenesisForkUpgrade_Implementations_Test is
    L2GenesisForkUpgrade_TestInit,
    L2ForkUpgrade_Implementations_Test
{
    function setUp() public override(L2GenesisForkUpgrade_TestInit, L2ForkUpgrade_TestInit) {
        L2GenesisForkUpgrade_TestInit.setUp();
    }
}

/// @title L2GenesisForkUpgrade_Events_Test
/// @notice Tests that all predeploy proxies emit the Upgraded event during the upgrade from genesis.
contract L2GenesisForkUpgrade_Events_Test is L2GenesisForkUpgrade_TestInit, L2ForkUpgrade_Events_Test {
    function setUp() public override(L2GenesisForkUpgrade_TestInit, L2ForkUpgrade_TestInit) {
        L2GenesisForkUpgrade_TestInit.setUp();
    }
}
