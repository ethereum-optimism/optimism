// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { Test } from "test/setup/Test.sol";

// Scripts
import { GenerateNUTBundle } from "scripts/upgrade/GenerateNUTBundle.s.sol";

// Libraries
import { Fork } from "scripts/libraries/Config.sol";
import { NetworkUpgradeTxns } from "src/libraries/NetworkUpgradeTxns.sol";

/// @title GenerateNUTBundleTest
/// @notice Tests that GenerateNUTBundle correctly generates Network Upgrade Transaction bundles
///         for L2 hardfork upgrades.
contract GenerateNUTBundleTest is Test {
    GenerateNUTBundle script;
    GenerateNUTBundle.Input input;

    bytes32 constant TEST_SALT = bytes32(uint256(0x1234));
    uint256 constant TEST_L1_CHAIN_ID = 1;

    function setUp() public {
        script = new GenerateNUTBundle();
        script.setUp();

        _setDefaultInput();
    }

    function _setDefaultInput() internal {
        input = GenerateNUTBundle.Input({
            fork: Fork.JOVIAN,
            salt: TEST_SALT,
            l1ChainID: TEST_L1_CHAIN_ID,
            useCustomGasToken: false
        });
    }

    /// @notice Tests that run succeeds with JOVIAN fork.
    function test_run_withJovianFork_succeeds() public {
        input.fork = Fork.JOVIAN;

        GenerateNUTBundle.Output memory output = script.run(input);

        NetworkUpgradeTxns.writeArtifact(output.txns, "deployments/nut-jovian-upgrade-test.json");
    }

    /// @notice Tests that run succeeds with ISTHMUS fork.
    function test_run_withIsthmusFork_succeeds() public {
        input.fork = Fork.ISTHMUS;

        GenerateNUTBundle.Output memory output = script.run(input);

        NetworkUpgradeTxns.writeArtifact(output.txns, "deployments/nut-isthmus-upgrade-test.json");
    }

    /// @notice Tests that run succeeds with INTEROP fork.
    function test_run_withInteropFork_succeeds() public {
        input.fork = Fork.INTEROP;

        GenerateNUTBundle.Output memory output = script.run(input);

        NetworkUpgradeTxns.writeArtifact(output.txns, "deployments/nut-interop-upgrade-test.json");
    }

    /// @notice Tests that run succeeds with custom gas token enabled.
    function test_run_withCustomGasToken_succeeds() public {
        input.useCustomGasToken = true;

        GenerateNUTBundle.Output memory output = script.run(input);

        NetworkUpgradeTxns.writeArtifact(output.txns, "deployments/nut-cgt-upgrade-test.json");
    }

    /// @notice Tests that run reverts with invalid fork (NONE).
    function test_run_invalidFork_reverts() public {
        input.fork = Fork.NONE;

        vm.expectRevert("GenerateNUTBundle: invalid fork");
        script.run(input);
    }

    /// @notice Tests that run reverts with zero salt.
    function test_run_zeroSalt_reverts() public {
        input.fork = Fork.JOVIAN;
        input.salt = bytes32(0);

        vm.expectRevert("GenerateNUTBundle: salt cannot be zero");
        script.run(input);
    }

    /// @notice Tests that run reverts with zero l1ChainID.
    function test_run_zeroL1ChainID_reverts() public {
        input.fork = Fork.JOVIAN;
        input.l1ChainID = 0;

        vm.expectRevert("GenerateNUTBundle: l1ChainID cannot be zero");
        script.run(input);
    }

    /// @notice Tests that transactions have correct structure for JOVIAN fork.
    /// @dev JOVIAN includes ConditionalDeployer and ProxyAdmin upgrades.
    function test_run_jovianTransactionStructure_succeeds() public {
        input.fork = Fork.JOVIAN;

        GenerateNUTBundle.Output memory output = script.run(input);

        // JOVIAN should include:
        // 1. ConditionalDeployer deployment
        // 2. ConditionalDeployer upgrade
        // 3. All implementation deployments (StorageSetter + predeploys)
        // 4. ProxyAdmin upgrade
        // 5. L2ContractsManager deployment
        // 6. Upgrade execution

        // Verify first transaction sourceHash corresponds to ConditionalDeployer deployment
        bytes32 expectedCDDeploymentSourceHash = NetworkUpgradeTxns.sourceHash("jovian: ConditionalDeployer Deployment");
        assertEq(
            output.txns[0].sourceHash,
            expectedCDDeploymentSourceHash,
            "First transaction should be ConditionalDeployer deployment"
        );

        bytes32 expectedCDUpgradeSourceHash =
            NetworkUpgradeTxns.sourceHash("jovian: Upgrade ConditionalDeployer Implementation");
        assertEq(
            output.txns[1].sourceHash,
            expectedCDUpgradeSourceHash,
            "Second transaction should be ConditionalDeployer upgrade"
        );

        /// TODO: Verify remaining transactions
    }

    /// @notice Tests that multiple runs produce deterministic results.
    function test_run_deterministicOutput_succeeds() public {
        input.fork = Fork.ISTHMUS;

        GenerateNUTBundle.Output memory output1 = script.run(input);
        GenerateNUTBundle.Output memory output2 = script.run(input);

        // Verify same number of transactions
        assertEq(output1.txns.length, output2.txns.length, "Should produce same number of transactions");

        _compareTransactions(output1, output2);
    }

    /// @notice Tests that multiple runs produce deterministic results.
    function test_run_deterministicOutputJovian_succeeds() public {
        input.fork = Fork.JOVIAN;

        GenerateNUTBundle.Output memory output1 = script.run(input);
        GenerateNUTBundle.Output memory output2 = script.run(input);

        // Verify same number of transactions
        assertEq(output1.txns.length, output2.txns.length, "Should produce same number of transactions");

        _compareTransactions(output1, output2);
    }

    /// @notice Tests that multiple runs produce deterministic results.
    function test_run_deterministicOutputCGT_succeeds() public {
        input.fork = Fork.ISTHMUS;
        input.useCustomGasToken = true;

        GenerateNUTBundle.Output memory output1 = script.run(input);
        GenerateNUTBundle.Output memory output2 = script.run(input);

        // Verify same number of transactions
        assertEq(output1.txns.length, output2.txns.length, "Should produce same number of transactions");

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
            assertEq(_output1.txns[i].sourceHash, _output2.txns[i].sourceHash, "Transaction sourceHash should match");
            assertEq(_output1.txns[i].from, _output2.txns[i].from, "Transaction from should match");
            assertEq(_output1.txns[i].to, _output2.txns[i].to, "Transaction to should match");
            assertEq(_output1.txns[i].mint, _output2.txns[i].mint, "Transaction mint should match");
            assertEq(_output1.txns[i].value, _output2.txns[i].value, "Transaction value should match");
            assertEq(_output1.txns[i].gas, _output2.txns[i].gas, "Transaction gas should match");
            assertEq(
                _output1.txns[i].isSystemTransaction,
                _output2.txns[i].isSystemTransaction,
                "Transaction isSystemTransaction should match"
            );
            assertEq(
                keccak256(_output1.txns[i].data), keccak256(_output2.txns[i].data), "Transaction data should match"
            );
        }
    }
}
