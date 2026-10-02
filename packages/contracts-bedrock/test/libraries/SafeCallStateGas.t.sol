// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { CommonTest } from "test/setup/CommonTest.sol";

/// @title SafeCall_MinGasOverhead_Harness
/// @notice Performs the exact worst-case `CALL` that `SafeCall.hasMinGas`'s flat buffer is sized to
///         cover: a cold account access, a positive value transfer, and the creation of an account
///         that did not previously exist. The target has no code, so it consumes nothing of its own
///         and the entire measurement is `CALL` overhead.
contract SafeCall_MinGasOverhead_Harness {
    /// @notice Sends 1 wei to `_target` and reports the gas the `CALL` consumed.
    /// @param _target Account to call. Must not exist yet, or no creation charge is levied.
    /// @return used_ Gas consumed by the call.
    function measure(address _target) external payable returns (uint256 used_) {
        uint256 start = gasleft();
        (bool success,) = _target.call{ value: 1 wei }("");
        used_ = start - gasleft();
        require(success, "SafeCall_MinGasOverhead_Harness: call failed");
    }
}

/// @title SafeCall_StateGas_Test
/// @notice Sizes the Portal-side half of the Amsterdam repricing problem: `callWithMinGas` promises
///         a withdrawal's target at least `tx.gasLimit`, and guards that promise with a flat gas
///         buffer that EIP-8037 invalidates.
///
///         `OptimismPortal2.finalizeWithdrawalTransaction` marks a withdrawal finalized before it
///         calls the target, and ignores the call's return value by design. The minimum-gas promise
///         is therefore the only thing standing between an underforwarded target and a permanently
///         finalized withdrawal whose message never executed.
///
///         Run under both `--evm-version cancun` and `--evm-version amsterdam`. The cancun run is
///         the control: the buffer is correctly sized today, so a test that measures it honestly
///         must pass there.
contract SafeCall_StateGas_Test is CommonTest {
    /// @notice The flat buffer `SafeCall.hasMinGas` adds on top of `_minGas`, mirrored from
    ///         `src/libraries/SafeCall.sol`. It is a literal inside an assembly block rather than a
    ///         named constant, so it cannot be read from the contract and has to be restated here.
    ///
    ///         Its own natspec sizes it as the worst case of the `CALL` opcode's
    ///         `address_access_cost` (2,600) + `positive_value_cost` (9,000 less the 2,300 stipend)
    ///         + `value_to_empty_account_cost` (25,000) = 34,300, plus 5,700 of slack.
    uint256 internal constant SAFE_CALL_BUFFER = 40_000;

    /// @notice Asserts the buffer covers the worst-case `CALL` overhead it is sized against.
    ///
    ///         Under EIP-8037 `value_to_empty_account_cost` stops being a flat 25,000 and becomes a
    ///         priced state creation, so the buffer is no longer an upper bound. When that happens
    ///         `hasMinGas` can pass while the overhead consumes more than the caller budgeted,
    ///         leaving the child frame with less than the `_minGas` it was promised — without any
    ///         revert to signal it.
    ///
    ///         The measurement is a lower bound on the real shortfall: it omits the memory expansion
    ///         and calldata costs that a withdrawal with a non-empty payload also incurs, which
    ///         `SafeCall`'s natspec already warns the buffer does not cover.
    function test_hasMinGasBuffer_coversWorstCaseCallOverhead_succeeds() external {
        skipIfForkTest("SafeCall_StateGas_Test: measures local EVM pricing");

        SafeCall_MinGasOverhead_Harness probe = new SafeCall_MinGasOverhead_Harness();
        vm.deal(address(probe), 1 ether);

        // A never-touched address: cold, zero balance, zero nonce, no code. Deliberately not read
        // before the call, since reading it would warm it and drop the cold-access cost from the
        // measurement.
        address fresh = address(uint160(uint256(keccak256("SafeCall_StateGas_Test.fresh"))));

        uint256 overhead = probe.measure(fresh);

        // Confirms the call really did create the account, so the creation charge was in fact
        // levied and the measurement reflects the worst case rather than a cheaper warm path.
        assertEq(fresh.balance, 1, "SafeCall_StateGas_Test: target was not created by the call");

        assertLe(
            overhead,
            SAFE_CALL_BUFFER,
            "hasMinGas's buffer does not cover worst-case CALL overhead; targets can be underforwarded"
        );
    }
}
