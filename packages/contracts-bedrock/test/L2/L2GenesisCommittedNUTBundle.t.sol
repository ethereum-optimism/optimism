// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { CommonTest } from "test/setup/CommonTest.sol";
import { PastNUTBundles } from "test/setup/PastNUTBundles.sol";

// Scripts
import { ExecuteNUTBundle } from "scripts/upgrade/ExecuteNUTBundle.s.sol";
import { Config, Fork, LATEST_FORK } from "scripts/libraries/Config.sol";

/// @title L2GenesisCommittedNUTBundle_LatestFork_Test
/// @notice Applies the latest fork's committed NUT bundle (the one embedded in op-node and
///         kona-node) to an L2 genesis generated from the current contracts at the preceding fork.
///         This is how new networks are deployed and then hardforked, so if any predeploy has been
///         bumped since the bundle was snapshotted, the L2CM downgrade check reverts the upgrade
///         here the same way it would at activation (see #23097).
///         L2GenesisForkUpgrade tests a bundle built fresh from source and cannot catch this drift.
///         Gated behind NUT_BUNDLE_DRIFT_TEST because a contracts PR is expected to land before the
///         re-snapshot PR (see op-core/nuts/README.md); it runs in the scheduled daily workflow.
contract L2GenesisCommittedNUTBundle_LatestFork_Test is CommonTest {
    function setUp() public override {
        if (!Config.nutBundleDriftTest()) {
            vm.skip(true);
            return;
        }

        // Start at the fork preceding the latest one, so the latest bundle performs the upgrade.
        l2Fork = Fork(uint256(LATEST_FORK) - 1);
        super.setUp();
        skipIfForkTest("committed NUT bundle drift test, not for L1 fork");
    }

    /// @notice Tests that every transaction in the latest fork's committed NUT bundle succeeds
    ///         when applied to a fresh genesis.
    function test_latestForkBundle_appliesToFreshGenesis_succeeds() public {
        PastNUTBundles.NUTBundle[] memory bundles = PastNUTBundles.fetchPastBundles();
        PastNUTBundles.NUTBundle memory latest = bundles[bundles.length - 1];

        // ExecuteNUTBundle reverts with the failing transaction's intent and revert reason.
        new ExecuteNUTBundle().executePath(latest.path);
    }
}
