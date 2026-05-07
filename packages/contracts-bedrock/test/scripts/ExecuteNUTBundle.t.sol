// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { Test } from "test/setup/Test.sol";

// Scripts
import { ExecuteNUTBundle } from "scripts/upgrade/ExecuteNUTBundle.s.sol";

// Libraries
import { NetworkUpgradeTxns } from "src/libraries/NetworkUpgradeTxns.sol";
import { UpgradeUtils } from "scripts/libraries/UpgradeUtils.sol";

/// @title ExecuteNUTBundle_Target
/// @notice Minimal target contract used to observe ExecuteNUTBundle dispatch behavior.
///         Etched at a fixed address by the test so that fixture transactions can target it.
contract ExecuteNUTBundle_Target {
    uint256[] public records;
    address public lastSender;

    function record(uint256 _value) external {
        records.push(_value);
        lastSender = msg.sender;
    }

    function recordsLength() external view returns (uint256) {
        return records.length;
    }

    function revertWithReason() external pure {
        revert("Target: failed");
    }
}

/// @title ExecuteNUTBundle_Test
/// @notice Tests that ExecuteNUTBundle correctly dispatches Network Upgrade Transactions from
///         inline arrays and from artifact files.
contract ExecuteNUTBundle_Test is Test {
    ExecuteNUTBundle internal script;
    ExecuteNUTBundle_Target internal target;
    address internal alice;
    address internal bob;

    /// @dev Address derived from `keccak256("ExecuteNUTBundle.fixture.target")` as a private key.
    address internal constant TARGET = 0xe6190d5229f8bC6C82cb42136ae182a941519E65;
    string internal constant FIXTURE_PATH = "test/fixtures/execute-nut-bundle.json";
    uint64 internal constant TXN_GAS_LIMIT = 100_000;

    function setUp() public {
        alice = makeAddr("alice");
        bob = makeAddr("bob");

        script = new ExecuteNUTBundle();
        vm.etch(TARGET, type(ExecuteNUTBundle_Target).runtimeCode);
        target = ExecuteNUTBundle_Target(TARGET);
    }

    /// @notice Tests that executeAll runs transactions in order with each `from` set as msg.sender.
    function test_executeAll_succeeds() public {
        NetworkUpgradeTxns.NetworkUpgradeTxn[] memory txns = new NetworkUpgradeTxns.NetworkUpgradeTxn[](2);
        txns[0] = NetworkUpgradeTxns.NetworkUpgradeTxn({
            data: abi.encodeCall(ExecuteNUTBundle_Target.record, (1)),
            from: alice,
            gasLimit: TXN_GAS_LIMIT,
            intent: "Record Value 1",
            to: TARGET
        });
        txns[1] = NetworkUpgradeTxns.NetworkUpgradeTxn({
            data: abi.encodeCall(ExecuteNUTBundle_Target.record, (2)),
            from: bob,
            gasLimit: TXN_GAS_LIMIT,
            intent: "Record Value 2",
            to: TARGET
        });

        script.executeAll(txns);

        assertEq(target.recordsLength(), 2, "records length");
        assertEq(target.records(0), 1, "records[0]");
        assertEq(target.records(1), 2, "records[1]");
        assertEq(target.lastSender(), bob, "lastSender");
    }

    /// @notice Tests that executeAll reverts including the failing transaction's intent and the
    ///         decoded revert reason.
    function test_executeAll_revertIncludesIntent_reverts() public {
        NetworkUpgradeTxns.NetworkUpgradeTxn[] memory txns = new NetworkUpgradeTxns.NetworkUpgradeTxn[](1);
        txns[0] = NetworkUpgradeTxns.NetworkUpgradeTxn({
            data: abi.encodeCall(ExecuteNUTBundle_Target.revertWithReason, ()),
            from: alice,
            gasLimit: TXN_GAS_LIMIT,
            intent: "Revert",
            to: TARGET
        });

        vm.expectRevert("ExecuteNUTBundle: Transaction failed - Revert - Target: failed");
        script.executeAll(txns);
    }

    /// @notice Tests that executeAll reports the failing transaction's intent (not a prior or
    ///         later one) when an intermediate transaction reverts.
    function test_executeAll_stopsOnFirstFailure_reverts() public {
        NetworkUpgradeTxns.NetworkUpgradeTxn[] memory txns = new NetworkUpgradeTxns.NetworkUpgradeTxn[](3);
        txns[0] = NetworkUpgradeTxns.NetworkUpgradeTxn({
            data: abi.encodeCall(ExecuteNUTBundle_Target.record, (1)),
            from: alice,
            gasLimit: TXN_GAS_LIMIT,
            intent: "Record Value 1",
            to: TARGET
        });
        txns[1] = NetworkUpgradeTxns.NetworkUpgradeTxn({
            data: abi.encodeCall(ExecuteNUTBundle_Target.revertWithReason, ()),
            from: bob,
            gasLimit: TXN_GAS_LIMIT,
            intent: "Revert",
            to: TARGET
        });
        txns[2] = NetworkUpgradeTxns.NetworkUpgradeTxn({
            data: abi.encodeCall(ExecuteNUTBundle_Target.record, (99)),
            from: alice,
            gasLimit: TXN_GAS_LIMIT,
            intent: "Record Value 99",
            to: TARGET
        });

        vm.expectRevert("ExecuteNUTBundle: Transaction failed - Revert - Target: failed");
        script.executeAll(txns);
    }

    /// @notice Tests that executeSingle reverts with the txn intent when gasLimit is below the
    ///         intrinsic gas for the provided calldata.
    function test_executeSingle_gasLimitBelowIntrinsic_reverts() public {
        bytes memory data = abi.encodeCall(ExecuteNUTBundle_Target.record, (1));
        uint64 intrinsicGas = UpgradeUtils.computeIntrinsicGas(data);
        NetworkUpgradeTxns.NetworkUpgradeTxn memory txn = NetworkUpgradeTxns.NetworkUpgradeTxn({
            data: data,
            from: alice,
            gasLimit: intrinsicGas - 1,
            intent: "Deploy StorageSetter Implementation",
            to: TARGET
        });

        vm.expectRevert("ExecuteNUTBundle: gasLimit < intrinsicGas for Deploy StorageSetter Implementation");
        script.executeSingle(txn);
    }

    /// @notice Tests that executePath reads an artifact and dispatches the decoded transactions.
    function test_executePath_succeeds() public {
        script.executePath(FIXTURE_PATH);

        assertEq(target.recordsLength(), 2, "records length");
        assertEq(target.records(0), 1, "records[0]");
        assertEq(target.records(1), 2, "records[1]");
        assertEq(target.lastSender(), bob, "lastSender");
    }
}
