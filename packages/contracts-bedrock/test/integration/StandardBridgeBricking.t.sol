// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { MessengerBrickingTestInit } from "test/setup/MessengerBrickingTestInit.sol";
import { GasBurningERC20 } from "test/mocks/GasBurningERC20.sol";

// Libraries
import { Encoding } from "src/libraries/Encoding.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";
import { Types } from "src/libraries/Types.sol";
import { ERC20 } from "@rari-capital/solmate/src/tokens/ERC20.sol";

/// @notice Models an ERC20 withdrawal through the Standard Bridge whose token transfer consumes
///         almost all of the gas it receives. Under Amsterdam, recording a fresh messenger status
///         slot costs EIP-8037 state gas, which the messenger must reserve after the call. With
///         too little gas for the messenger to finish, the Portal must not finalize the withdrawal,
///         or the tokens are locked in the bridge forever.
/// forge-config: default.evm_version = "amsterdam"
contract StandardBridgeBricking_Integration_Test is MessengerBrickingTestInit {
    uint32 internal constant MIN_GAS_LIMIT = 100_000;
    uint32 internal constant ERC20_MIN_GAS_LIMIT = 0;
    uint256 internal constant AMOUNT = 100;

    ERC20 internal token;
    address internal recipient;

    function setUp() public override {
        super.setUp();
        recipient = makeAddr("ERC20 withdrawal recipient");
        token = new GasBurningERC20(300_000);
    }

    /// @notice Tests that finalization with too little gas for the Portal reverts without recording
    ///         the withdrawal.
    function test_finalizeWithdrawalTransaction_belowMinPortalGas_reverts() external {
        (Types.WithdrawalTransaction memory withdrawal, bytes32 withdrawalHash) = _prepareERC20Relay();
        (uint256 minPortalGas,) = _gasThresholds(withdrawal);

        vm.expectRevert("SafeCall: Not enough gas");
        optimismPortal2.finalizeWithdrawalTransaction{ gas: minPortalGas - 1 }(withdrawal);

        _assertNothingRecorded(withdrawal, withdrawalHash);
    }

    /// @notice Tests that finalization with enough gas for the Portal but too little for the token
    ///         transfer finalizes the withdrawal with the message recorded as failed, keeping the
    ///         tokens in the bridge.
    function test_finalizeWithdrawalTransaction_belowMinMessengerGas_succeeds() external {
        (Types.WithdrawalTransaction memory withdrawal, bytes32 withdrawalHash) = _prepareERC20Relay();
        (uint256 minPortalGas,) = _gasThresholds(withdrawal);

        optimismPortal2.finalizeWithdrawalTransaction{ gas: minPortalGas }(withdrawal);

        _assertRecordedFailure(withdrawal, withdrawalHash);
    }

    /// @notice Tests that a message recorded as failed can be replayed through the messenger to
    ///         deliver the tokens.
    function test_finalizeWithdrawalTransaction_replayAfterMessengerFailure_succeeds() external {
        (Types.WithdrawalTransaction memory withdrawal, bytes32 withdrawalHash) = _prepareERC20Relay();
        (uint256 minPortalGas, uint256 minMessengerGas) = _gasThresholds(withdrawal);

        optimismPortal2.finalizeWithdrawalTransaction{ gas: minPortalGas }(withdrawal);
        _assertRecordedFailure(withdrawal, withdrawalHash);

        l1CrossDomainMessenger.relayMessage{ gas: minMessengerGas + 50_000 }(
            Encoding.encodeVersionedNonce(0, 1),
            Predeploys.L2_STANDARD_BRIDGE,
            address(l1StandardBridge),
            0,
            ERC20_MIN_GAS_LIMIT,
            _finalizeMessage()
        );

        assertTrue(l1CrossDomainMessenger.successfulMessages(keccak256(withdrawal.data)));
        assertEq(token.balanceOf(recipient), AMOUNT);
        assertEq(token.balanceOf(address(l1StandardBridge)), 0);
        assertEq(l1StandardBridge.deposits(address(token), address(L2Token)), 0);
    }

    /// @notice Tests that finalization with enough gas for the messenger delivers the tokens.
    function test_finalizeWithdrawalTransaction_minMessengerGas_succeeds() external {
        (Types.WithdrawalTransaction memory withdrawal, bytes32 withdrawalHash) = _prepareERC20Relay();
        (, uint256 minMessengerGas) = _gasThresholds(withdrawal);

        optimismPortal2.finalizeWithdrawalTransaction{ gas: minMessengerGas }(withdrawal);

        _assertDelivered(withdrawal, withdrawalHash);
    }

    /// @notice Tests that for any gas the Portal forwards, up to the transaction gas cap, the
    ///         Portal either does not finalize the withdrawal or the messenger records it as
    ///         successful or failed.
    function testFuzz_finalizeWithdrawalTransaction_gasWindow_succeeds(uint256 _gas) external {
        (Types.WithdrawalTransaction memory withdrawal, bytes32 withdrawalHash) = _prepareERC20Relay();
        uint256 minPortalGas = _leastGas(withdrawal, _portalForwards);
        _gas = bound(_gas, minPortalGas, MAX_TX_GAS);

        (bool success,) = address(optimismPortal2).call{ gas: _gas }(
            abi.encodeCall(optimismPortal2.finalizeWithdrawalTransaction, (withdrawal))
        );

        if (success) {
            bytes32 messageHash = keccak256(withdrawal.data);
            assertTrue(optimismPortal2.finalizedWithdrawals(withdrawalHash));
            assertTrue(
                l1CrossDomainMessenger.successfulMessages(messageHash)
                    || l1CrossDomainMessenger.failedMessages(messageHash),
                "withdrawal finalized without a messenger outcome"
            );
        } else {
            _assertNothingRecorded(withdrawal, withdrawalHash);
        }
    }

    function _assertNothingRecorded(
        Types.WithdrawalTransaction memory _withdrawal,
        bytes32 _withdrawalHash
    )
        internal
        view
    {
        _assertStatus(_withdrawal, _withdrawalHash, false, false, false);
        assertEq(token.balanceOf(recipient), 0);
        assertEq(token.balanceOf(address(l1StandardBridge)), AMOUNT);
        assertEq(l1StandardBridge.deposits(address(token), address(L2Token)), AMOUNT);
    }

    function _assertRecordedFailure(
        Types.WithdrawalTransaction memory _withdrawal,
        bytes32 _withdrawalHash
    )
        internal
        view
    {
        _assertStatus(_withdrawal, _withdrawalHash, true, false, true);
        assertEq(token.balanceOf(recipient), 0);
        assertEq(token.balanceOf(address(l1StandardBridge)), AMOUNT);
        assertEq(l1StandardBridge.deposits(address(token), address(L2Token)), AMOUNT);
    }

    function _assertDelivered(Types.WithdrawalTransaction memory _withdrawal, bytes32 _withdrawalHash) internal view {
        _assertStatus(_withdrawal, _withdrawalHash, true, true, false);
        assertEq(token.balanceOf(recipient), AMOUNT);
        assertEq(token.balanceOf(address(l1StandardBridge)), 0);
        assertEq(l1StandardBridge.deposits(address(token), address(L2Token)), 0);
    }

    function _prepareERC20Relay()
        internal
        returns (Types.WithdrawalTransaction memory withdrawal_, bytes32 withdrawalHash_)
    {
        deal(address(token), alice, AMOUNT, true);
        vm.startPrank(alice, alice);
        token.approve(address(l1StandardBridge), AMOUNT);
        l1StandardBridge.bridgeERC20(address(token), address(L2Token), AMOUNT, MIN_GAS_LIMIT, hex"");
        vm.stopPrank();
        assertEq(l1StandardBridge.deposits(address(token), address(L2Token)), AMOUNT);

        bytes memory message = _finalizeMessage();
        withdrawal_ = _defaultTx;
        withdrawal_.sender = Predeploys.L2_CROSS_DOMAIN_MESSENGER;
        withdrawal_.target = address(l1CrossDomainMessenger);
        withdrawal_.value = 0;
        withdrawal_.gasLimit = l1CrossDomainMessenger.baseGas(message, ERC20_MIN_GAS_LIMIT);
        withdrawal_.data = Encoding.encodeCrossDomainMessageV1(
            Encoding.encodeVersionedNonce(0, 1),
            Predeploys.L2_STANDARD_BRIDGE,
            address(l1StandardBridge),
            0,
            ERC20_MIN_GAS_LIMIT,
            message
        );
        withdrawalHash_ = _prepareWithdrawalForFinalization(withdrawal_);
    }

    function _finalizeMessage() internal view returns (bytes memory) {
        return abi.encodeCall(
            l1StandardBridge.finalizeBridgeERC20, (address(token), address(L2Token), alice, recipient, AMOUNT, hex"")
        );
    }
}
