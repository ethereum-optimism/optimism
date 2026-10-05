// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { MessengerBrickingTestInit } from "test/setup/MessengerBrickingTestInit.sol";
import { GasBurner } from "test/mocks/GasBurner.sol";

// Libraries
import { Encoding } from "src/libraries/Encoding.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";
import { Types } from "src/libraries/Types.sol";

// Interfaces
import { IOptimismPortal2 } from "interfaces/L1/IOptimismPortal2.sol";

/// @notice Models an ETH withdrawal through the messengers whose target consumes almost all of the
///         gas it receives. Under Amsterdam, recording a fresh messenger status slot costs more
///         than the messenger reserves after the call. With too little gas, the messenger reverts
///         without recording the message. The Portal must then not finalize the withdrawal, or the
///         withdrawal is lost.
/// forge-config: default.evm_version = "amsterdam"
contract MessengerETHBricking_Integration_Test is MessengerBrickingTestInit {
    uint32 internal constant MIN_GAS_LIMIT = 100_000;
    bytes internal constant MESSAGE = hex"abcd";

    /// @notice Tests that finalization with too little gas for the Portal reverts without recording
    ///         the withdrawal.
    function test_finalizeWithdrawalTransaction_belowMinPortalGas_reverts() external {
        (Types.WithdrawalTransaction memory withdrawal, bytes32 withdrawalHash, address target) = _prepareRelay();
        (uint256 minPortalGas,) = _gasThresholds(withdrawal);

        vm.expectRevert("SafeCall: Not enough gas");
        optimismPortal2.finalizeWithdrawalTransaction{ gas: minPortalGas - 1 }(withdrawal);

        _assertNothingRecorded(withdrawal, withdrawalHash, target);
    }

    /// @notice Tests that finalization with enough gas for the Portal but too little for the
    ///         messenger to record the message reverts instead of bricking the withdrawal.
    function test_finalizeWithdrawalTransaction_belowMinMessengerGas_reverts() external {
        (Types.WithdrawalTransaction memory withdrawal, bytes32 withdrawalHash, address target) = _prepareRelay();
        (uint256 minPortalGas,) = _gasThresholds(withdrawal);

        vm.expectRevert(IOptimismPortal2.OptimismPortal_L1CDMCallFailed.selector);
        optimismPortal2.finalizeWithdrawalTransaction{ gas: minPortalGas }(withdrawal);

        _assertNothingRecorded(withdrawal, withdrawalHash, target);
    }

    /// @notice Tests that a withdrawal whose first finalization attempt had too little gas for the
    ///         messenger can be finalized again with enough gas.
    function test_finalizeWithdrawalTransaction_retryAfterMessengerFailure_succeeds() external {
        (Types.WithdrawalTransaction memory withdrawal, bytes32 withdrawalHash, address target) = _prepareRelay();
        (uint256 minPortalGas, uint256 minMessengerGas) = _gasThresholds(withdrawal);

        (bool success,) = address(optimismPortal2).call{ gas: minPortalGas }(
            abi.encodeCall(optimismPortal2.finalizeWithdrawalTransaction, (withdrawal))
        );
        assertFalse(success);

        optimismPortal2.finalizeWithdrawalTransaction{ gas: minMessengerGas + 50_000 }(withdrawal);

        _assertDelivered(withdrawal, withdrawalHash, target);
    }

    /// @notice Tests that finalization with enough gas for the messenger delivers the ETH.
    function test_finalizeWithdrawalTransaction_minMessengerGas_succeeds() external {
        (Types.WithdrawalTransaction memory withdrawal, bytes32 withdrawalHash, address target) = _prepareRelay();
        (, uint256 minMessengerGas) = _gasThresholds(withdrawal);

        optimismPortal2.finalizeWithdrawalTransaction{ gas: minMessengerGas }(withdrawal);

        _assertDelivered(withdrawal, withdrawalHash, target);
    }

    /// @notice Tests that for any gas the Portal forwards, up to the transaction gas cap, the
    ///         Portal either does not finalize the withdrawal or the messenger records it as
    ///         successful or failed.
    function testFuzz_finalizeWithdrawalTransaction_gasWindow_succeeds(uint256 _gas) external {
        (Types.WithdrawalTransaction memory withdrawal, bytes32 withdrawalHash, address target) = _prepareRelay();
        uint256 minPortalGas = _leastGas(withdrawal, _portalForwards);
        _gas = bound(_gas, minPortalGas, MAX_TX_GAS);

        _finalizeAndAssertOutcome(withdrawal, withdrawalHash, target, _gas);
    }

    /// @notice Tests the same property for a target that burns all the gas it receives, so the
    ///         messenger is left with only its reserve and the gas EIP-150 withholds.
    function testFuzz_finalizeWithdrawalTransaction_burnAllTargetGasWindow_succeeds(uint256 _gas) external {
        (Types.WithdrawalTransaction memory withdrawal, bytes32 withdrawalHash, address target) =
            _prepareRelay(type(uint256).max);
        uint256 minPortalGas = _leastGas(withdrawal, _portalForwards);
        _gas = bound(_gas, minPortalGas, MAX_TX_GAS);

        _finalizeAndAssertOutcome(withdrawal, withdrawalHash, target, _gas);
    }

    /// @notice Tests that a withdrawal whose target burns all the gas it receives can still be
    ///         finalized within the transaction gas cap, with the message recorded as failed so it
    ///         can be replayed.
    function test_finalizeWithdrawalTransaction_burnAllTargetAtGasCap_succeeds() external {
        (Types.WithdrawalTransaction memory withdrawal, bytes32 withdrawalHash,) = _prepareRelay(type(uint256).max);

        optimismPortal2.finalizeWithdrawalTransaction{ gas: MAX_TX_GAS }(withdrawal);

        bytes32 messageHash = keccak256(withdrawal.data);
        assertTrue(optimismPortal2.finalizedWithdrawals(withdrawalHash));
        assertFalse(l1CrossDomainMessenger.successfulMessages(messageHash));
        assertTrue(l1CrossDomainMessenger.failedMessages(messageHash));
    }

    function _finalizeAndAssertOutcome(
        Types.WithdrawalTransaction memory _withdrawal,
        bytes32 _withdrawalHash,
        address _target,
        uint256 _gas
    )
        internal
    {
        (bool success,) = address(optimismPortal2).call{ gas: _gas }(
            abi.encodeCall(optimismPortal2.finalizeWithdrawalTransaction, (_withdrawal))
        );

        if (success) {
            bytes32 messageHash = keccak256(_withdrawal.data);
            assertTrue(optimismPortal2.finalizedWithdrawals(_withdrawalHash));
            assertTrue(
                l1CrossDomainMessenger.successfulMessages(messageHash)
                    || l1CrossDomainMessenger.failedMessages(messageHash),
                "withdrawal finalized without a messenger outcome"
            );
        } else {
            _assertNothingRecorded(_withdrawal, _withdrawalHash, _target);
        }
    }

    function _assertNothingRecorded(
        Types.WithdrawalTransaction memory _withdrawal,
        bytes32 _withdrawalHash,
        address _target
    )
        internal
        view
    {
        bytes32 messageHash = keccak256(_withdrawal.data);
        assertFalse(optimismPortal2.finalizedWithdrawals(_withdrawalHash));
        assertFalse(l1CrossDomainMessenger.successfulMessages(messageHash));
        assertFalse(l1CrossDomainMessenger.failedMessages(messageHash));
        assertEq(_target.balance, 0);
    }

    function _assertDelivered(
        Types.WithdrawalTransaction memory _withdrawal,
        bytes32 _withdrawalHash,
        address _target
    )
        internal
        view
    {
        bytes32 messageHash = keccak256(_withdrawal.data);
        assertTrue(optimismPortal2.finalizedWithdrawals(_withdrawalHash));
        assertTrue(l1CrossDomainMessenger.successfulMessages(messageHash));
        assertFalse(l1CrossDomainMessenger.failedMessages(messageHash));
        assertEq(_target.balance, _withdrawal.value);
    }

    function _prepareRelay()
        internal
        returns (Types.WithdrawalTransaction memory withdrawal_, bytes32 withdrawalHash_, address target_)
    {
        return _prepareRelay(450_000);
    }

    function _prepareRelay(uint256 _gasToBurn)
        internal
        returns (Types.WithdrawalTransaction memory withdrawal_, bytes32 withdrawalHash_, address target_)
    {
        target_ = address(new GasBurner(_gasToBurn));
        withdrawal_ = _defaultTx;
        withdrawal_.sender = Predeploys.L2_CROSS_DOMAIN_MESSENGER;
        withdrawal_.target = address(l1CrossDomainMessenger);
        withdrawal_.gasLimit = l1CrossDomainMessenger.baseGas(MESSAGE, MIN_GAS_LIMIT);
        withdrawal_.data = Encoding.encodeCrossDomainMessageV1(
            Encoding.encodeVersionedNonce(0, 1), alice, target_, withdrawal_.value, MIN_GAS_LIMIT, MESSAGE
        );
        withdrawalHash_ = _prepareWithdrawalForFinalization(withdrawal_);
    }
}
