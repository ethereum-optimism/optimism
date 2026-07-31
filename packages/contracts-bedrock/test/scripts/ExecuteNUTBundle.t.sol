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
    uint256 public lastValue;

    function record(uint256 _value) external {
        records.push(_value);
        lastSender = msg.sender;
    }

    function observePayable() external payable {
        lastSender = msg.sender;
        lastValue = msg.value;
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

    /// @dev Mirrors the literal in `ExecuteNUTBundle_Target.revertWithReason` for reverts.
    string internal constant TARGET_REVERT_REASON = "Target: failed";

    /// @dev High fixture gas for dispatch tests; executor subtracts intrinsic gas before forwarding.
    uint64 internal constant FIXTURE_GAS_LIMIT = 1_000_000;

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
            gasLimit: FIXTURE_GAS_LIMIT,
            intent: "Record Value 1",
            to: TARGET
        });
        txns[1] = NetworkUpgradeTxns.NetworkUpgradeTxn({
            data: abi.encodeCall(ExecuteNUTBundle_Target.record, (2)),
            from: bob,
            gasLimit: FIXTURE_GAS_LIMIT,
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
            gasLimit: FIXTURE_GAS_LIMIT,
            intent: "Revert",
            to: TARGET
        });

        vm.expectRevert(bytes(string.concat("ExecuteNUTBundle: Transaction failed - Revert - ", TARGET_REVERT_REASON)));
        script.executeAll(txns);
    }

    /// @notice Tests that executeAll reports the failing transaction's intent (not a prior or
    ///         later one) when an intermediate transaction reverts.
    function test_executeAll_stopsOnFirstFailure_reverts() public {
        NetworkUpgradeTxns.NetworkUpgradeTxn[] memory txns = new NetworkUpgradeTxns.NetworkUpgradeTxn[](3);
        txns[0] = NetworkUpgradeTxns.NetworkUpgradeTxn({
            data: abi.encodeCall(ExecuteNUTBundle_Target.record, (1)),
            from: alice,
            gasLimit: FIXTURE_GAS_LIMIT,
            intent: "Record Value 1",
            to: TARGET
        });
        txns[1] = NetworkUpgradeTxns.NetworkUpgradeTxn({
            data: abi.encodeCall(ExecuteNUTBundle_Target.revertWithReason, ()),
            from: bob,
            gasLimit: FIXTURE_GAS_LIMIT,
            intent: "Revert",
            to: TARGET
        });
        txns[2] = NetworkUpgradeTxns.NetworkUpgradeTxn({
            data: abi.encodeCall(ExecuteNUTBundle_Target.record, (99)),
            from: alice,
            gasLimit: FIXTURE_GAS_LIMIT,
            intent: "Record Value 99",
            to: TARGET
        });

        vm.expectRevert(bytes(string.concat("ExecuteNUTBundle: Transaction failed - Revert - ", TARGET_REVERT_REASON)));
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

    /// @notice Tests that executeWrapper can mint to the sender and forward value from that sender.
    function testFuzz_executeWrapper_valueAndMint_succeeds(
        uint128 _startingBalance,
        uint128 _mint,
        uint128 _value
    )
        public
    {
        _startingBalance = uint128(bound(_startingBalance, 1, type(uint128).max));
        uint256 available = uint256(_startingBalance) + uint256(_mint);
        uint256 maxValue = available > type(uint128).max ? type(uint128).max : available;
        _value = uint128(bound(_value, 0, maxValue));

        vm.deal(alice, _startingBalance);

        ExecuteNUTBundle.PostWrapperTxn memory wrapper = ExecuteNUTBundle.PostWrapperTxn({
            from: alice,
            to: TARGET,
            data: abi.encodeCall(ExecuteNUTBundle_Target.observePayable, ()),
            gasLimit: FIXTURE_GAS_LIMIT,
            mint: _mint,
            value: _value,
            intent: "Wrapper Value"
        });

        script.executeWrapper(wrapper);

        assertEq(target.lastSender(), alice, "lastSender");
        assertEq(target.lastValue(), uint256(_value), "lastValue");
        assertEq(address(target).balance, uint256(_value), "target balance");
        assertEq(alice.balance, available - uint256(_value), "sender balance");
    }

    /// @notice Tests that executeWrapper reverts with the wrapper intent when gasLimit is below
    ///         intrinsic gas for the provided calldata.
    function test_executeWrapper_gasLimitBelowIntrinsic_reverts() public {
        bytes memory data = abi.encodeCall(ExecuteNUTBundle_Target.record, (1));
        uint64 intrinsicGas = UpgradeUtils.computeIntrinsicGas(data);
        ExecuteNUTBundle.PostWrapperTxn memory wrapper = ExecuteNUTBundle.PostWrapperTxn({
            from: alice,
            to: TARGET,
            data: data,
            gasLimit: intrinsicGas - 1,
            mint: 0,
            value: 0,
            intent: "Wrapper Gas"
        });

        vm.expectRevert("ExecuteNUTBundle: wrapper gasLimit < intrinsicGas for Wrapper Gas");
        script.executeWrapper(wrapper);
    }

    /// @notice Tests that executeWrapper reverts including the wrapper intent and decoded reason.
    function test_executeWrapper_revertIncludesIntent_reverts() public {
        ExecuteNUTBundle.PostWrapperTxn memory wrapper = ExecuteNUTBundle.PostWrapperTxn({
            from: alice,
            to: TARGET,
            data: abi.encodeCall(ExecuteNUTBundle_Target.revertWithReason, ()),
            gasLimit: FIXTURE_GAS_LIMIT,
            mint: 0,
            value: 0,
            intent: "Wrapper Revert"
        });

        vm.expectRevert(
            bytes(string.concat("ExecuteNUTBundle: wrapper failed - Wrapper Revert - ", TARGET_REVERT_REASON))
        );
        script.executeWrapper(wrapper);
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
