// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { Test } from "test/setup/Test.sol";

// Scripts
import { GenerateNUTBundle } from "scripts/upgrade/GenerateNUTBundle.s.sol";

// Libraries
import { NetworkUpgradeTxns } from "src/libraries/NetworkUpgradeTxns.sol";
import { UpgradeUtils } from "scripts/libraries/UpgradeUtils.sol";
import { Constants } from "src/libraries/Constants.sol";
import { L2ContractsManagerTypes } from "src/libraries/L2ContractsManagerTypes.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";

/// @title GenerateNUTBundle_Harness
/// @notice Harness contract that exposes internal functions for testing.
contract GenerateNUTBundle_Harness is GenerateNUTBundle {
    /// @notice Builds the upgrade transaction bundle Output struct without writing to disk.
    function buildOutput() external returns (Output memory) {
        return _buildOutput();
    }

    /// @notice Returns the fork name used by the generated bundle.
    function upgradeName() external pure returns (string memory) {
        return UPGRADE_NAME;
    }

    /// @notice Asserts that the given output is valid.
    function assertValidOutput(Output memory _output) external pure {
        _assertValidOutput(_output);
    }
}

/// @title GenerateNUTBundleTest
/// @notice Tests that GenerateNUTBundle correctly generates Network Upgrade Transaction bundles
///         for L2 hardfork upgrades.
contract GenerateNUTBundleTest is Test {
    GenerateNUTBundle_Harness script;

    uint256 constant TEST_L1_CHAIN_ID = 1;

    function setUp() public {
        script = new GenerateNUTBundle_Harness();
        script.setUp();
    }

    /// @notice Tests that run succeeds.
    function test_run_succeeds() public {
        GenerateNUTBundle.Output memory output = script.run();

        assertEq(output.fork, script.upgradeName(), "fork mismatch");

        // Verify artifact written correctly
        NetworkUpgradeTxns.NetworkUpgradeTxn[] memory readTxns =
            NetworkUpgradeTxns.readArtifact(Constants.CURRENT_BUNDLE_PATH);
        assertEq(readTxns.length, output.txns.length, "Transaction count mismatch");
        for (uint256 i = 0; i < readTxns.length; i++) {
            assertEq(readTxns[i].intent, output.txns[i].intent, "Intent mismatch");
            assertEq(readTxns[i].from, output.txns[i].from, "From mismatch");
            assertEq(readTxns[i].to, output.txns[i].to, "To mismatch");
            assertEq(readTxns[i].gasLimit, uint256(output.txns[i].gasLimit), "Gas limit mismatch");
            assertEq(keccak256(readTxns[i].data), keccak256(output.txns[i].data), "Data mismatch");
        }
    }

    /// @notice Tests that the harness build path returns the same fork name as the script run path.
    function test_buildOutput_setsFork_succeeds() public {
        GenerateNUTBundle.Output memory output = script.buildOutput();

        assertEq(output.fork, script.upgradeName(), "fork mismatch");
    }

    /// @notice Tests that transactions have correct structure.
    function test_run_transactionStructure_succeeds() public {
        GenerateNUTBundle.Output memory output = script.run();

        // Should include:
        // 1. All implementation deployments (StorageSetter + predeploys)
        // 2. L2ContractsManager deployment
        // 3. Upgrade execution

        // Verify implementation deployments
        string[] memory implementationsToUpgrade = UpgradeUtils.getImplementationsNamesToUpgrade();
        for (uint256 i = 0; i < implementationsToUpgrade.length; i++) {
            assertEq(
                output.txns[i].intent,
                string.concat("Deploy ", implementationsToUpgrade[i], " Implementation"),
                string.concat("Transaction should be ", implementationsToUpgrade[i], " deployment")
            );
        }

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
        assertEq(_output1.fork, _output2.fork, "Fork should match");
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

    /// @notice Tests that the number of implementations in the deployment list matches the number of fields in the
    /// Implementations struct.
    function test_implementationCount_matchesStructFields_succeeds() public pure {
        L2ContractsManagerTypes.Implementations memory emptyImpl;
        uint256 structFieldCount = abi.encode(emptyImpl).length / 32;
        string[] memory names = UpgradeUtils.getImplementationsNamesToUpgrade();
        assertEq(names.length, structFieldCount, "Deployment list must equal Implementations struct field count");
    }

    /// @notice Tests that a bundle with an incorrect number of transactions is rejected.
    /// @dev Builds a valid bundle, then mutates the array length to trigger the assertion.
    function testFuzz_assertValidOutput_transactionCountMismatch_reverts(uint256 _newLength) public {
        GenerateNUTBundle.Output memory output = script.buildOutput();

        _newLength = bound(_newLength, 0, output.txns.length - 1);
        NetworkUpgradeTxns.NetworkUpgradeTxn[] memory txns = output.txns;
        assembly {
            mstore(txns, _newLength)
        }

        vm.expectRevert("GenerateNUTBundle: invalid transaction count");
        script.assertValidOutput(output);
    }

    /// @notice Tests that a transaction with empty data is rejected.
    /// @dev Builds a valid bundle, then mutates one transaction to trigger the assertion.
    function testFuzz_assertValidOutput_emptyData_reverts(uint256 _index) public {
        GenerateNUTBundle.Output memory output = script.buildOutput();

        _index = bound(_index, 0, output.txns.length - 1);
        output.txns[_index].data = new bytes(0);

        vm.expectRevert("GenerateNUTBundle: invalid transaction data");
        script.assertValidOutput(output);
    }

    /// @notice Tests that a transaction with an empty intent is rejected.
    /// @dev Builds a valid bundle, then mutates one transaction to trigger the assertion.
    function testFuzz_assertValidOutput_emptyIntent_reverts(uint256 _index) public {
        GenerateNUTBundle.Output memory output = script.buildOutput();

        _index = bound(_index, 0, output.txns.length - 1);
        output.txns[_index].intent = "";

        vm.expectRevert("GenerateNUTBundle: invalid transaction intent");
        script.assertValidOutput(output);
    }

    /// @notice Tests that a transaction with a zero destination address is rejected.
    /// @dev Builds a valid bundle, then mutates one transaction to trigger the assertion.
    function testFuzz_assertValidOutput_zeroTo_reverts(uint256 _index) public {
        GenerateNUTBundle.Output memory output = script.buildOutput();

        _index = bound(_index, 0, output.txns.length - 1);
        output.txns[_index].to = address(0);

        vm.expectRevert("GenerateNUTBundle: invalid transaction to");
        script.assertValidOutput(output);
    }

    /// @notice Tests that a transaction exceeding the EIP-7825 per-tx gas limit cap is rejected.
    /// @dev Builds a valid bundle, then mutates one transaction to trigger the assertion.
    function testFuzz_assertValidOutput_gasLimitExceedsMax_reverts(uint256 _index, uint64 _gasLimit) public {
        GenerateNUTBundle.Output memory output = script.buildOutput();

        _index = bound(_index, 0, output.txns.length - 1);
        _gasLimit = uint64(bound(_gasLimit, script.MAX_TX_GAS_LIMIT() + 1, type(uint64).max));
        output.txns[_index].gasLimit = _gasLimit;

        vm.expectRevert(
            bytes(
                string.concat(
                    "GenerateNUTBundle: gasLimit outside [EIP-7623 floor, EIP-7825 cap] for ",
                    output.txns[_index].intent
                )
            )
        );
        script.assertValidOutput(output);
    }

    /// @notice Tests that a transaction with a zero gas limit is rejected by the EIP-7623 floor.
    /// @dev Builds a valid bundle, then mutates one transaction to trigger the assertion.
    function testFuzz_assertValidOutput_zeroGasLimit_reverts(uint256 _index) public {
        GenerateNUTBundle.Output memory output = script.buildOutput();

        _index = bound(_index, 0, output.txns.length - 1);
        output.txns[_index].gasLimit = 0;

        vm.expectRevert(
            bytes(
                string.concat(
                    "GenerateNUTBundle: gasLimit outside [EIP-7623 floor, EIP-7825 cap] for ",
                    output.txns[_index].intent
                )
            )
        );
        script.assertValidOutput(output);
    }

    /// @notice Tests that a transaction whose gasLimit is one below the EIP-7623 floor is rejected.
    /// @dev Builds a valid bundle, then mutates one transaction to trigger the assertion.
    function testFuzz_assertValidOutput_gasLimitBelowFloor_reverts(uint256 _index) public {
        GenerateNUTBundle.Output memory output = script.buildOutput();

        _index = bound(_index, 0, output.txns.length - 1);
        uint64 floor = UpgradeUtils.computeFloorDataGas(output.txns[_index].data);
        output.txns[_index].gasLimit = floor - 1;

        vm.expectRevert(
            bytes(
                string.concat(
                    "GenerateNUTBundle: gasLimit outside [EIP-7623 floor, EIP-7825 cap] for ",
                    output.txns[_index].intent
                )
            )
        );
        script.assertValidOutput(output);
    }

    /// @notice Tests that a transaction with a zero sender and a non-privileged destination is rejected.
    /// @dev Builds a valid bundle, then mutates one transaction to trigger the assertion.
    function testFuzz_assertValidOutput_zeroFromNonPrivilegedTo_reverts(uint256 _index, address _to) public {
        GenerateNUTBundle.Output memory output = script.buildOutput();

        vm.assume(_to != address(0));
        vm.assume(_to != Predeploys.PROXY_ADMIN && _to != Predeploys.CONDITIONAL_DEPLOYER);
        _index = bound(_index, 0, output.txns.length - 1);
        output.txns[_index].from = address(0);
        output.txns[_index].to = _to;

        vm.expectRevert("GenerateNUTBundle: invalid transaction from");
        script.assertValidOutput(output);
    }
}
