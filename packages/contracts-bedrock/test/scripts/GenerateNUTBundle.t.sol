// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { Test } from "test/setup/Test.sol";

// Scripts
import { GenerateNUTBundle } from "scripts/upgrade/GenerateNUTBundle.s.sol";

// Libraries
import { NetworkUpgradeTxns } from "src/libraries/NetworkUpgradeTxns.sol";
import { UpgradeUtils } from "scripts/libraries/UpgradeUtils.sol";

/// @title GenerateNUTBundleTest
/// @notice Tests that GenerateNUTBundle correctly generates Network Upgrade Transaction bundles
///         for L2 hardfork upgrades.
contract GenerateNUTBundleTest is Test {
    GenerateNUTBundle script;

    uint256 constant TEST_L1_CHAIN_ID = 1;

    function setUp() public {
        script = new GenerateNUTBundle();
        script.setUp();
    }

    /// @notice Tests that run succeeds.
    function test_run_succeeds() public {
        GenerateNUTBundle.Output memory output = script.run();

        // Verify artifact written correctly
        NetworkUpgradeTxns.NetworkUpgradeTxn[] memory readTxns =
            NetworkUpgradeTxns.readArtifact(script.upgradeBundlePath());
        assertEq(readTxns.length, output.txns.length, "Transaction count mismatch");
        for (uint256 i = 0; i < readTxns.length; i++) {
            assertEq(readTxns[i].intent, output.txns[i].intent, "Intent mismatch");
            assertEq(readTxns[i].from, output.txns[i].from, "From mismatch");
            assertEq(readTxns[i].to, output.txns[i].to, "To mismatch");
            assertEq(readTxns[i].gasLimit, uint256(output.txns[i].gasLimit), "Gas limit mismatch");
            assertEq(keccak256(readTxns[i].data), keccak256(output.txns[i].data), "Data mismatch");
        }
    }

    /// @notice Tests that transactions have correct structure.
    /// @dev Includes ConditionalDeployer and ProxyAdmin upgrades.
    function test_run_transactionStructure_succeeds() public {
        GenerateNUTBundle.Output memory output = script.run();

        // Should include:
        // 1. ConditionalDeployer deployment
        // 2. ConditionalDeployer upgrade
        // 3. All implementation deployments (StorageSetter + predeploys)
        // 4. L2ProxyAdmin upgrade
        // 5. L2ContractsManager deployment
        // 6. Upgrade execution

        // Verify ConditionalDeployer deployment
        assertEq(
            output.txns[0].intent,
            "ConditionalDeployer Deployment",
            "First transaction should be ConditionalDeployer deployment"
        );

        // Verify ConditionalDeployer upgrade
        assertEq(
            output.txns[1].intent,
            "Upgrade ConditionalDeployer Implementation",
            "Second transaction should be ConditionalDeployer upgrade"
        );

        // Verify implementation deployments
        string[] memory implementationsToUpgrade = UpgradeUtils.getImplementationsNamesToUpgrade();
        for (uint256 i = 0; i < implementationsToUpgrade.length; i++) {
            assertEq(
                output.txns[i + 2].intent,
                string.concat("Deploy ", implementationsToUpgrade[i], " Implementation"),
                string.concat("Transaction should be ", implementationsToUpgrade[i], " deployment")
            );
        }

        // Verify L2ProxyAdmin upgrade
        assertEq(
            output.txns[output.txns.length - 3].intent,
            "Upgrade L2ProxyAdmin Implementation",
            "Third to last transaction should be L2ProxyAdmin upgrade"
        );

        // Verify L2ContractsManager deployment
        assertEq(
            output.txns[output.txns.length - 2].intent,
            "Deploy L2ContractsManager Implementation",
            "Second to last transaction should be L2ContractsManager implementation deployment"
        );

        // Verify upgrade execution
        assertEq(
            output.txns[output.txns.length - 1].intent,
            "L2ProxyAdmin Upgrade Predeploys",
            "Last transaction should be L2ProxyAdmin upgrade predeploys"
        );
    }

    /// @notice Tests that multiple runs produce deterministic results.
    function test_run_deterministicOutput_succeeds() public {
        GenerateNUTBundle.Output memory output1 = script.run();
        GenerateNUTBundle.Output memory output2 = script.run();

        _compareTransactions(output1, output2);
    }

    function _compareTransactions(
        GenerateNUTBundle.Output memory _output1,
        GenerateNUTBundle.Output memory _output2
    )
        internal
        pure
    {
        assertEq(_output1.txns.length, _output2.txns.length, "Should produce same number of transactions");
        for (uint256 i = 0; i < _output1.txns.length; i++) {
            assertEq(_output1.txns[i].intent, _output2.txns[i].intent, "Transaction intent should match");
            assertEq(_output1.txns[i].from, _output2.txns[i].from, "Transaction from should match");
            assertEq(_output1.txns[i].to, _output2.txns[i].to, "Transaction to should match");
            assertEq(_output1.txns[i].gasLimit, _output2.txns[i].gasLimit, "Transaction gasLimit should match");
            assertEq(
                keccak256(_output1.txns[i].data), keccak256(_output2.txns[i].data), "Transaction data should match"
            );
        }
    }
}
