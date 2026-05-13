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
import { Features } from "src/libraries/Features.sol";

// Reuse all test logic from L2ForkUpgrade — only setUp differs
import {
    L2ForkUpgrade_TestInit,
    L2ForkUpgrade_Versions_Test,
    L2ForkUpgrade_Initialization_Test,
    L2ForkUpgrade_Implementations_Test,
    L2ForkUpgrade_Events_Test,
    L2ForkUpgrade_GasProfile_Test
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

        // Skip if running with an unoptimized Foundry profile
        skipIfUnoptimized();

        // Initialize scripts
        executeScript = new ExecuteNUTBundle();
        generateScript = new GenerateNUTBundle();

        // Generate bundle
        GenerateNUTBundle.Output memory output = generateScript.run();
        delete currentBundleTxns;
        for (uint256 i = 0; i < output.txns.length; i++) {
            currentBundleTxns.push(output.txns[i]);
        }

        // Capture feature flags from deploy config.
        // Interop predeploys are upgraded only when BOTH the INTEROP sys feature (useInterop) AND
        // the OPTIMISM_PORTAL_INTEROP dev feature are enabled — mirroring the L2CM gating logic.
        commonState.isInteropEnabled = deploy.cfg().useInterop()
            && DevFeatures.isDevFeatureEnabled(deploy.cfg().devFeatureBitmap(), DevFeatures.OPTIMISM_PORTAL_INTEROP);
        console.log("L2GenesisForkUpgrade isInteropEnabled", commonState.isInteropEnabled);

        commonState.isCustomGasToken = deploy.cfg().useCustomGasToken();
        console.log("L2GenesisForkUpgrade: isCustomGasToken", commonState.isCustomGasToken);
    }

    /// @notice Genesis tests execute the bare current bundle because genesis already applies feature setup.
    function _executeCurrentBundle() internal virtual override {
        executeScript.executeAll(_currentBundleTxns());
    }
}

/// @title L2GenesisForkUpgrade_Versions_Test
/// @notice Tests that all predeploy versions are updated after the upgrade from genesis.
contract L2GenesisForkUpgrade_Versions_Test is L2GenesisForkUpgrade_TestInit, L2ForkUpgrade_Versions_Test {
    function setUp() public override(L2GenesisForkUpgrade_TestInit, L2ForkUpgrade_TestInit) {
        L2GenesisForkUpgrade_TestInit.setUp();
    }

    function _executeCurrentBundle() internal override(L2GenesisForkUpgrade_TestInit, L2ForkUpgrade_TestInit) {
        L2GenesisForkUpgrade_TestInit._executeCurrentBundle();
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

    function _executeCurrentBundle() internal override(L2GenesisForkUpgrade_TestInit, L2ForkUpgrade_TestInit) {
        L2GenesisForkUpgrade_TestInit._executeCurrentBundle();
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

    function _executeCurrentBundle() internal override(L2GenesisForkUpgrade_TestInit, L2ForkUpgrade_TestInit) {
        L2GenesisForkUpgrade_TestInit._executeCurrentBundle();
    }
}

/// @title L2GenesisForkUpgrade_Events_Test
/// @notice Tests that all predeploy proxies emit the Upgraded event during the upgrade from genesis.
contract L2GenesisForkUpgrade_Events_Test is L2GenesisForkUpgrade_TestInit, L2ForkUpgrade_Events_Test {
    function setUp() public override(L2GenesisForkUpgrade_TestInit, L2ForkUpgrade_TestInit) {
        L2GenesisForkUpgrade_TestInit.setUp();
    }

    function _executeCurrentBundle() internal override(L2GenesisForkUpgrade_TestInit, L2ForkUpgrade_TestInit) {
        L2GenesisForkUpgrade_TestInit._executeCurrentBundle();
    }
}

/// @title L2GenesisForkUpgrade_GasProfile_Test
/// @notice Gas profiling test for the upgrade bundle from genesis state.
contract L2GenesisForkUpgrade_GasProfile_Test is L2GenesisForkUpgrade_TestInit, L2ForkUpgrade_GasProfile_Test {
    function setUp() public override(L2GenesisForkUpgrade_TestInit, L2ForkUpgrade_TestInit) {
        L2GenesisForkUpgrade_TestInit.setUp();
    }

    function _executeCurrentBundle() internal override(L2GenesisForkUpgrade_TestInit, L2ForkUpgrade_TestInit) {
        L2GenesisForkUpgrade_TestInit._executeCurrentBundle();
    }
}

// ============================================================
// Interop variant — genesis deployed with useInterop=true
// ============================================================

/// @title L2GenesisForkUpgrade_Interop_TestInit
/// @notice Same as L2GenesisForkUpgrade_TestInit but enables interop before genesis deployment.
///         Calling enableInterop() before CommonTest.setUp() causes L2Genesis to run with
///         useInterop=true, so L1Block gets the INTEROP sys feature.
abstract contract L2GenesisForkUpgrade_Interop_TestInit is L2GenesisForkUpgrade_TestInit {
    function setUp() public virtual override {
        super.enableInterop();
        L2GenesisForkUpgrade_TestInit.setUp();
    }

    function _executeCurrentBundle() internal virtual override {
        L2GenesisForkUpgrade_TestInit._executeCurrentBundle();
    }
}

/// @title L2GenesisForkUpgrade_Interop_Versions_Test
contract L2GenesisForkUpgrade_Interop_Versions_Test is
    L2GenesisForkUpgrade_Interop_TestInit,
    L2ForkUpgrade_Versions_Test
{
    function setUp() public override(L2GenesisForkUpgrade_Interop_TestInit, L2ForkUpgrade_TestInit) {
        L2GenesisForkUpgrade_Interop_TestInit.setUp();
    }

    function _executeCurrentBundle() internal override(L2GenesisForkUpgrade_Interop_TestInit, L2ForkUpgrade_TestInit) {
        L2GenesisForkUpgrade_Interop_TestInit._executeCurrentBundle();
    }
}

/// @title L2GenesisForkUpgrade_Interop_Initialization_Test
contract L2GenesisForkUpgrade_Interop_Initialization_Test is
    L2GenesisForkUpgrade_Interop_TestInit,
    L2ForkUpgrade_Initialization_Test
{
    function setUp() public override(L2GenesisForkUpgrade_Interop_TestInit, L2ForkUpgrade_TestInit) {
        L2GenesisForkUpgrade_Interop_TestInit.setUp();
    }

    function _executeCurrentBundle() internal override(L2GenesisForkUpgrade_Interop_TestInit, L2ForkUpgrade_TestInit) {
        L2GenesisForkUpgrade_Interop_TestInit._executeCurrentBundle();
    }
}

/// @title L2GenesisForkUpgrade_Interop_Implementations_Test
contract L2GenesisForkUpgrade_Interop_Implementations_Test is
    L2GenesisForkUpgrade_Interop_TestInit,
    L2ForkUpgrade_Implementations_Test
{
    function setUp() public override(L2GenesisForkUpgrade_Interop_TestInit, L2ForkUpgrade_TestInit) {
        L2GenesisForkUpgrade_Interop_TestInit.setUp();
    }

    function _executeCurrentBundle() internal override(L2GenesisForkUpgrade_Interop_TestInit, L2ForkUpgrade_TestInit) {
        L2GenesisForkUpgrade_Interop_TestInit._executeCurrentBundle();
    }
}

/// @title L2GenesisForkUpgrade_Interop_Events_Test
contract L2GenesisForkUpgrade_Interop_Events_Test is
    L2GenesisForkUpgrade_Interop_TestInit,
    L2ForkUpgrade_Events_Test
{
    function setUp() public override(L2GenesisForkUpgrade_Interop_TestInit, L2ForkUpgrade_TestInit) {
        L2GenesisForkUpgrade_Interop_TestInit.setUp();
    }

    function _executeCurrentBundle() internal override(L2GenesisForkUpgrade_Interop_TestInit, L2ForkUpgrade_TestInit) {
        L2GenesisForkUpgrade_Interop_TestInit._executeCurrentBundle();
    }
}

/// @title L2GenesisForkUpgrade_Interop_GasProfile_Test
contract L2GenesisForkUpgrade_Interop_GasProfile_Test is
    L2GenesisForkUpgrade_Interop_TestInit,
    L2ForkUpgrade_GasProfile_Test
{
    function setUp() public override(L2GenesisForkUpgrade_Interop_TestInit, L2ForkUpgrade_TestInit) {
        L2GenesisForkUpgrade_Interop_TestInit.setUp();
    }

    function _executeCurrentBundle() internal override(L2GenesisForkUpgrade_Interop_TestInit, L2ForkUpgrade_TestInit) {
        L2GenesisForkUpgrade_Interop_TestInit._executeCurrentBundle();
    }
}

// ============================================================
// CGT variant — skips unless CUSTOM_GAS_TOKEN is active
// ============================================================

/// @title L2GenesisForkUpgrade_CGT_TestInit
/// @notice Same as L2GenesisForkUpgrade_TestInit but restricted to Custom Gas Token networks.
///         CGT is auto-configured from the CUSTOM_GAS_TOKEN env var inside CommonTest.setUp(),
///         so no programmatic enable is needed.
abstract contract L2GenesisForkUpgrade_CGT_TestInit is L2GenesisForkUpgrade_TestInit {
    function setUp() public virtual override {
        L2GenesisForkUpgrade_TestInit.setUp();
        skipIfSysFeatureDisabled(Features.CUSTOM_GAS_TOKEN);
    }
}

/// @title L2GenesisForkUpgrade_CGT_Versions_Test
contract L2GenesisForkUpgrade_CGT_Versions_Test is L2GenesisForkUpgrade_CGT_TestInit, L2ForkUpgrade_Versions_Test {
    function setUp() public override(L2GenesisForkUpgrade_CGT_TestInit, L2ForkUpgrade_TestInit) {
        L2GenesisForkUpgrade_CGT_TestInit.setUp();
    }

    function _executeCurrentBundle() internal override(L2GenesisForkUpgrade_TestInit, L2ForkUpgrade_TestInit) {
        L2GenesisForkUpgrade_TestInit._executeCurrentBundle();
    }
}

/// @title L2GenesisForkUpgrade_CGT_Initialization_Test
contract L2GenesisForkUpgrade_CGT_Initialization_Test is
    L2GenesisForkUpgrade_CGT_TestInit,
    L2ForkUpgrade_Initialization_Test
{
    function setUp() public override(L2GenesisForkUpgrade_CGT_TestInit, L2ForkUpgrade_TestInit) {
        L2GenesisForkUpgrade_CGT_TestInit.setUp();
    }

    function _executeCurrentBundle() internal override(L2GenesisForkUpgrade_TestInit, L2ForkUpgrade_TestInit) {
        L2GenesisForkUpgrade_TestInit._executeCurrentBundle();
    }
}

/// @title L2GenesisForkUpgrade_CGT_Implementations_Test
contract L2GenesisForkUpgrade_CGT_Implementations_Test is
    L2GenesisForkUpgrade_CGT_TestInit,
    L2ForkUpgrade_Implementations_Test
{
    function setUp() public override(L2GenesisForkUpgrade_CGT_TestInit, L2ForkUpgrade_TestInit) {
        L2GenesisForkUpgrade_CGT_TestInit.setUp();
    }

    function _executeCurrentBundle() internal override(L2GenesisForkUpgrade_TestInit, L2ForkUpgrade_TestInit) {
        L2GenesisForkUpgrade_TestInit._executeCurrentBundle();
    }
}

/// @title L2GenesisForkUpgrade_CGT_Events_Test
contract L2GenesisForkUpgrade_CGT_Events_Test is L2GenesisForkUpgrade_CGT_TestInit, L2ForkUpgrade_Events_Test {
    function setUp() public override(L2GenesisForkUpgrade_CGT_TestInit, L2ForkUpgrade_TestInit) {
        L2GenesisForkUpgrade_CGT_TestInit.setUp();
    }

    function _executeCurrentBundle() internal override(L2GenesisForkUpgrade_TestInit, L2ForkUpgrade_TestInit) {
        L2GenesisForkUpgrade_TestInit._executeCurrentBundle();
    }
}

/// @title L2GenesisForkUpgrade_CGT_GasProfile_Test
contract L2GenesisForkUpgrade_CGT_GasProfile_Test is
    L2GenesisForkUpgrade_CGT_TestInit,
    L2ForkUpgrade_GasProfile_Test
{
    function setUp() public override(L2GenesisForkUpgrade_CGT_TestInit, L2ForkUpgrade_TestInit) {
        L2GenesisForkUpgrade_CGT_TestInit.setUp();
    }

    function _executeCurrentBundle() internal override(L2GenesisForkUpgrade_TestInit, L2ForkUpgrade_TestInit) {
        L2GenesisForkUpgrade_TestInit._executeCurrentBundle();
    }
}
