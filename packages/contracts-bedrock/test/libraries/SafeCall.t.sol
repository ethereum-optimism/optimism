// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { Test } from "test/setup/Test.sol";
import { StdCheatsSafe } from "forge-std/StdCheats.sol";

// Scripts
import { Config } from "scripts/libraries/Config.sol";

// Libraries
import { SafeCall } from "src/libraries/SafeCall.sol";

contract SimpleSafeCaller {
    uint256 public a;

    function makeSafeCall(uint64 gas, uint64 minGas) external returns (bool) {
        return SafeCall.call(address(this), gas, 0, abi.encodeCall(this.makeSafeCallMinGas, (minGas)));
    }

    function makeSafeCallMinGas(uint64 minGas) external returns (bool) {
        return SafeCall.callWithMinGas(address(this), minGas, 0, abi.encodeCall(this.setA, (1)));
    }

    function setA(uint256 _a) external {
        a = _a;
    }
}

/// @notice Accepts value.
contract ValueReceiver {
    receive() external payable { }
}

/// @notice Calls `SafeCall.callWithMinGas` with its own balance as the value.
contract ValueSafeCaller {
    function callWithMinGas(address _target, uint256 _minGas) external returns (bool) {
        return SafeCall.callWithMinGas(_target, _minGas, address(this).balance, hex"");
    }
}

/// @title SafeCall_TestInit
/// @notice Reusable test initialization for `SafeCall` tests.
abstract contract SafeCall_TestInit is Test {
    /// @notice Helper function to deduplicate code. Makes all assumptions required for these
    ///         tests.
    function assumeNot(address _addr) internal {
        vm.deal(_addr, 0);
        vm.assume(_addr != address(this));
        assumeAddressIsNot(_addr, StdCheatsSafe.AddressType.ForgeAddress, StdCheatsSafe.AddressType.Precompile);
        // Accounts with code may revert, such as the EIP-2935 history contract on Prague and later.
        vm.assume(_addr.code.length == 0);
    }

    /// @notice Internal helper function for `send` tests
    function sendTest(address _from, address _to, uint64 _gas, uint256 _value) internal {
        assumeNot(_from);
        assumeNot(_to);

        assertEq(_from.balance, 0, "from balance is 0");
        vm.deal(_from, _value);
        assertEq(_from.balance, _value, "from balance not dealt");

        uint256[2] memory balancesBefore = [_from.balance, _to.balance];

        vm.expectCall(_to, _value, bytes(""));
        vm.prank(_from);
        bool success;
        if (_gas == 0) {
            success = SafeCall.send({ _target: _to, _value: _value });
        } else {
            success = SafeCall.send({ _target: _to, _gas: _gas, _value: _value });
        }

        assertTrue(success, "send not successful");
        if (_from == _to) {
            assertEq(_from.balance, balancesBefore[0], "Self-send did not change balance");
        } else {
            assertEq(_from.balance, balancesBefore[0] - _value, "from balance not drained");
            assertEq(_to.balance, balancesBefore[1] + _value, "to balance received");
        }
    }
}

/// @title SafeCall_Send_Test
/// @notice Tests the `send` function of the `SafeCall` contract.
contract SafeCall_Send_Test is SafeCall_TestInit {
    /// @notice Tests that the `send` function succeeds.
    function testFuzz_send_succeeds(address _from, address _to, uint256 _value) external {
        sendTest({ _from: _from, _to: _to, _gas: 0, _value: _value });
    }

    /// @notice Tests that the `send` function with value succeeds.
    ///         Not isolated: isolation treats the gas limit as a transaction's, which must cover
    ///         intrinsic gas. SafeCall makes internal calls, whose gas limit never has to.
    /// forge-config: default.isolate = false
    function testFuzz_send_withGas_succeeds(address _from, address _to, uint64 _gas, uint256 _value) external {
        _gas = uint64(bound(_gas, 1, type(uint64).max));
        sendTest({ _from: _from, _to: _to, _gas: _gas, _value: _value });
    }
}

/// @title SafeCall_Call_Test
/// @notice Tests the `call` function of the `SafeCall` contract.
contract SafeCall_Call_Test is SafeCall_TestInit {
    /// @notice Tests that `call` succeeds.
    ///         Not isolated: isolation treats the gas limit as a transaction's, which must cover
    ///         intrinsic gas. SafeCall makes internal calls, whose gas limit never has to.
    /// forge-config: default.isolate = false
    function testFuzz_call_succeeds(address from, address to, uint256 gas, uint64 value, bytes memory data) external {
        assumeNot(from);
        assumeNot(to);

        assertEq(from.balance, 0, "from balance is 0");
        vm.deal(from, value);
        assertEq(from.balance, value, "from balance not dealt");

        uint256[2] memory balancesBefore = [from.balance, to.balance];

        vm.expectCall(to, value, data);
        vm.prank(from);
        bool success = SafeCall.call(to, gas, value, data);

        assertTrue(success, "call not successful");
        if (from == to) {
            assertEq(from.balance, balancesBefore[0], "Self-send did not change balance");
        } else {
            assertEq(from.balance, balancesBefore[0] - value, "from balance not drained");
            assertEq(to.balance, balancesBefore[1] + value, "to balance received");
        }
    }
}

/// @title SafeCall_CallWithMinGas_Test
/// @notice Tests the `callWithMinGas` function of the `SafeCall` contract.
contract SafeCall_CallWithMinGas_Test is SafeCall_TestInit {
    /// @notice Tests that `callWithMinGas` succeeds with enough gas.
    function testFuzz_callWithMinGas_hasEnough_succeeds(
        address from,
        address to,
        uint64 minGas,
        uint64 value,
        bytes memory data
    )
        external
    {
        assumeNot(from);
        assumeNot(to);

        assertEq(from.balance, 0, "from balance is 0");
        vm.deal(from, value);
        assertEq(from.balance, value, "from balance not dealt");

        // Bound minGas to [0, l1_block_gas_limit]
        minGas = uint64(bound(minGas, 0, 30_000_000));

        uint256[2] memory balancesBefore = [from.balance, to.balance];

        vm.expectCallMinGas(to, value, minGas, data);
        vm.prank(from);
        bool success = SafeCall.callWithMinGas(to, minGas, value, data);

        assertTrue(success, "call not successful");
        if (from == to) {
            assertEq(from.balance, balancesBefore[0], "Self-send did not change balance");
        } else {
            assertEq(from.balance, balancesBefore[0] - value, "from balance not drained");
            assertEq(to.balance, balancesBefore[1] + value, "to balance received");
        }
    }

    /// @notice Tests that `callWithMinGas` reverts or supplies the minimum gas when sending value to
    ///         a new account, whose EIP-8037 state gas the CALL charges before the 63/64 split.
    ///         Not isolated: isolation treats the gas limit as a transaction's, which must cover
    ///         intrinsic gas. SafeCall makes internal calls, whose gas limit never has to.
    /// forge-config: default.isolate = false
    function testFuzz_callWithMinGas_valueToNewAccount_succeeds(
        address _target,
        uint256 _minGas,
        uint256 _gas
    )
        external
    {
        assumeNot(_target);
        ValueSafeCaller caller = new ValueSafeCaller();
        vm.assume(_target != address(caller));
        vm.deal(address(caller), 1);

        _minGas = bound(_minGas, 2300, 30_000_000);
        _gas = bound(_gas, 0, 2 * _minGas + 1_000_000);

        // expectCallMinGas adds the 2,300-gas stipend for nonzero-value calls.
        vm.expectCallMinGas(_target, 1, uint64(_minGas - 2300), hex"");
        (bool success,) = address(caller).call{ gas: _gas }(abi.encodeCall(caller.callWithMinGas, (_target, _minGas)));

        // Reverts satisfy the invariant; only calls that pass the check must meet the expectation.
        vm.assume(success);
        assertEq(_target.balance, 1, "target not funded");
    }

    /// @notice Calls `callWithMinGas` sending `_value` to `_target` with gas above the base
    ///         allowance but well below `NEW_ACCOUNT_STATE_GAS`. Callers set `isolate = false`; see
    ///         `testFuzz_callWithMinGas_valueToNewAccount_succeeds`.
    /// @return success_    Whether the call succeeded.
    /// @return returndata_ The call's returndata.
    function _callWithMinGasBelowNewAccountGas(
        address _target,
        uint256 _value
    )
        internal
        returns (bool success_, bytes memory returndata_)
    {
        uint256 minGas = 100_000;
        ValueSafeCaller caller = new ValueSafeCaller();
        vm.deal(address(caller), _value);
        uint256 gas = minGas * 64 / 63 + 40_000 + 50_000;
        (success_, returndata_) =
            address(caller).call{ gas: gas }(abi.encodeCall(caller.callWithMinGas, (_target, minGas)));
    }

    /// @notice Tests that `callWithMinGas` reserves `NEW_ACCOUNT_STATE_GAS` for value to an empty
    ///         account.
    /// forge-config: default.isolate = false
    function test_callWithMinGas_valueToEmptyAccount_reverts() external {
        (bool success, bytes memory returndata) = _callWithMinGasBelowNewAccountGas(makeAddr("empty"), 1);
        assertFalse(success);
        assertEq(returndata, abi.encodeWithSignature("Error(string)", "SafeCall: Not enough gas"));
    }

    /// @notice Tests that `callWithMinGas` does not reserve `NEW_ACCOUNT_STATE_GAS` for a call to an
    ///         empty account without value.
    /// forge-config: default.isolate = false
    function test_callWithMinGas_noValueToEmptyAccount_succeeds() external {
        address target = makeAddr("empty");
        vm.expectCallMinGas(target, 0, 100_000, hex"");
        (bool success,) = _callWithMinGasBelowNewAccountGas(target, 0);
        assertTrue(success);
    }

    /// @notice Tests that `callWithMinGas` does not reserve `NEW_ACCOUNT_STATE_GAS` for value to an
    ///         account with a balance.
    /// forge-config: default.isolate = false
    function test_callWithMinGas_valueToFundedAccount_succeeds() external {
        address target = makeAddr("funded");
        vm.deal(target, 1);
        vm.expectCallMinGas(target, 1, 100_000 - 2300, hex"");
        (bool success,) = _callWithMinGasBelowNewAccountGas(target, 1);
        assertTrue(success);
        assertEq(target.balance, 2);
    }

    /// @notice Tests that `callWithMinGas` does not reserve `NEW_ACCOUNT_STATE_GAS` for value to an
    ///         account with code.
    /// forge-config: default.isolate = false
    function test_callWithMinGas_valueToContract_succeeds() external {
        address target = address(new ValueReceiver());
        vm.expectCallMinGas(target, 1, 100_000 - 2300, hex"");
        (bool success,) = _callWithMinGasBelowNewAccountGas(target, 1);
        assertTrue(success);
        assertEq(target.balance, 1);
    }

    /// @notice Tests that `callWithMinGas` succeeds for the lower gas bounds.
    function test_callWithMinGas_noLeakageLow_succeeds() external {
        SimpleSafeCaller caller = new SimpleSafeCaller();

        for (uint64 i = 40_000; i < 100_000; i++) {
            uint256 snapshot = vm.snapshotState();

            // The values below are best gotten by setting the value to a high number and running
            // the test with a verbosity of `-vvv` then setting the value to the value (gas arg) of
            // the failed assertion. A faster way to do this for forge coverage cases, is to
            // comment out the optimizer and optimizer runs in the foundry.toml file and then run
            // forge test. This is faster because forge test only compiles modified contracts
            // unlike forge coverage.
            uint256 expected;

            // Because forge coverage always runs with the optimizer disabled, if forge coverage is
            // run before testing this with forge test or forge snapshot, forge clean should be run
            // first so that it recompiles the contracts using the foundry.toml optimizer settings.
            if (Config.isUnoptimized()) {
                // 66_378 is the exact amount of gas required with the optimizer disabled.
                expected = 66_378;
            } else {
                // 66_007 is the exact amount of gas required with optimizer enabled.
                expected = 66_007;
            }

            if (i < expected) {
                assertFalse(caller.makeSafeCall(i, 25_000));
            } else {
                vm.expectCallMinGas(address(caller), 0, 25_000, abi.encodeCall(caller.setA, (1)));
                assertTrue(caller.makeSafeCall(i, 25_000));
            }

            assertTrue(vm.revertToState(snapshot));
        }
    }

    /// @notice Tests that `callWithMinGas` succeeds on the upper gas bounds.
    function test_callWithMinGas_noLeakageHigh_succeeds() external {
        SimpleSafeCaller caller = new SimpleSafeCaller();

        for (uint64 i = 15_200_000; i < 15_300_000; i++) {
            uint256 snapshot = vm.snapshotState();

            // The values below are best gotten by setting the value to a high number and running
            // the test with a verbosity of `-vvv` then setting the value to the value (gas arg) of
            // the failed assertion. A faster way to do this for forge coverage cases, is to
            // comment out the optimizer and optimizer runs in the foundry.toml file and then run
            // forge test. This is faster because forge test only compiles modified contracts
            // unlike forge coverage.
            uint256 expected;

            // Because forge coverage always runs with the optimizer disabled, if forge coverage is
            // run before testing this with forge test or forge snapshot, forge clean should be run
            // first so that it recompiles the contracts using the foundry.toml optimizer settings.
            if (Config.isUnoptimized()) {
                // 15_279_077 is the exact amount of gas required with the optimizer disabled.
                expected = 15_279_077;
            } else {
                // 15_278_706 is the exact amount of gas required with optimizer enabled.
                expected = 15_278_706;
            }

            if (i < expected) {
                assertFalse(caller.makeSafeCall(i, 15_000_000));
            } else {
                vm.expectCallMinGas(address(caller), 0, 15_000_000, abi.encodeCall(caller.setA, (1)));
                assertTrue(caller.makeSafeCall(i, 15_000_000));
            }

            assertTrue(vm.revertToState(snapshot));
        }
    }
}
