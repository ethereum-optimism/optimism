// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { CommonTest } from "test/setup/CommonTest.sol";
import { ExpiryHandler } from "test/formal/expiry/invariants/ExpiryHandler.sol";

// Libraries
import { Predeploys } from "src/libraries/Predeploys.sol";

// Interfaces
import { IL2ToL2CrossDomainMessenger } from "interfaces/L2/IL2ToL2CrossDomainMessenger.sol";

/// @title ExpiryInvariants_TestInit
/// @notice Shared setup for the per-message expiry invariant harness (see ExpiryHandler for the model and its
///         abstractions). Builds the real L2 genesis with interop enabled, then targets the handler only.
///         Configuration:
///         - W_protocol (relay validity window) = env EXPIRY_INV_W_PROTOCOL, default 7 days.
///         - P_contract (expireMessage period) = the messenger's constant, read from the contract.
///         - Assumption checked in setUp: P_contract >= W_protocol (protocol window <= contract window).
///           Only the UnsafeWindow variants drop it (W_protocol = P_contract + 1 day) to show the double spend.
///         - Exports go through the real UndeliveredMessageExporter predeploy, whose withdrawals are the only facts
///           (the sender L1CrossDomainMessenger.relayUndeliveredMessage trusts), except in the legacy-design mutant.
///         - Expected-to-fail variants run only with EXPIRY_INV_EXPECT_FAIL=true (skipped otherwise).
///         Targets karl/message-expiry-refunds at 5992028e08 (UndeliveredMessageExporter, EXPIRY_PERIOD = 8 days).
abstract contract ExpiryInvariants_TestInit is CommonTest {
    /// @notice Messenger replacements (test-only copies under mutants/; the real contracts are not modified).
    uint8 internal constant MUTANT_NONE = 0;
    /// @notice 5992028e08 messenger with _isUnsafeTarget always false (no L2CrossDomainMessenger/passer target rule).
    uint8 internal constant MUTANT_NO_UNSAFE_TARGETS = 1;
    /// @notice Legacy design: the 37b44c48c7 messenger (exports itself; L1 trusted 0x..23) without its target rule.
    uint8 internal constant MUTANT_LEGACY_NO_TARGET_RULE = 2;

    /// @notice EIP-1967 implementation slot.
    bytes32 internal constant IMPL_SLOT = 0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc;

    IL2ToL2CrossDomainMessenger internal immutable messenger =
        IL2ToL2CrossDomainMessenger(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER);

    ExpiryHandler internal handler;

    /// @notice ETHLiquidity balance after setUp.
    uint256 internal initialLiquidity;

    /// @notice Name written to the stats file.
    function _label() internal pure virtual returns (string memory);

    /// @notice W_protocol for this configuration.
    function _wProtocol(uint256) internal view virtual returns (uint256) {
        return vm.envOr("EXPIRY_INV_W_PROTOCOL", uint256(7 days));
    }

    /// @notice Whether this configuration drops the P_contract >= W_protocol assumption.
    function _allowUnsafeWindow() internal pure virtual returns (bool) {
        return false;
    }

    /// @notice Which messenger replacement to etch over the L2ToL2CrossDomainMessenger implementation.
    function _mutant() internal pure virtual returns (uint8) {
        return MUTANT_NONE;
    }

    /// @notice Whether the campaign may relay a payload with no initiating event (relayForgedPayloadToL2CDM). On for
    ///         the real contracts (a stronger adversary than the protocol allows); off for the mutant so that its
    ///         counterexamples use only protocol-valid relays.
    function _includeForgedPayloadAction() internal pure virtual returns (bool) {
        return true;
    }

    /// @notice Whether this is an expected-to-fail configuration (run only with EXPIRY_INV_EXPECT_FAIL=true).
    function _expectFail() internal pure virtual returns (bool) {
        return false;
    }

    /// @notice Whether to register the handler as the invariant target.
    function _isInvariant() internal pure virtual returns (bool) {
        return true;
    }

    function setUp() public virtual override {
        if (_expectFail() && !vm.envOr("EXPIRY_INV_EXPECT_FAIL", false)) vm.skip(true);

        super.enableInterop();
        super.setUp();

        // The real exporter predeploy is in genesis (proxy with an implementation).
        assertGt(Predeploys.UNDELIVERED_MESSAGE_EXPORTER.code.length, 0);
        assertTrue(vm.load(Predeploys.UNDELIVERED_MESSAGE_EXPORTER, IMPL_SLOT) != bytes32(0));

        // The exporter the handler calls and the sender A's L1CrossDomainMessenger trusts.
        address exporter = Predeploys.UNDELIVERED_MESSAGE_EXPORTER;
        if (_mutant() != MUTANT_NONE) {
            address impl = address(uint160(uint256(vm.load(address(messenger), IMPL_SLOT))));
            assertTrue(impl != address(0));
            if (_mutant() == MUTANT_NO_UNSAFE_TARGETS) {
                vm.etch(
                    impl,
                    vm.getDeployedCode(
                        "L2ToL2CrossDomainMessengerNoUnsafeTargets.sol:L2ToL2CrossDomainMessengerNoUnsafeTargets"
                    )
                );
            } else {
                vm.etch(
                    impl,
                    vm.getDeployedCode(
                        "L2ToL2CrossDomainMessengerLegacyNoTargetRule.sol:L2ToL2CrossDomainMessengerLegacyNoTargetRule"
                    )
                );
                exporter = address(messenger);
            }
        }

        uint256 pContract = _contractExpiryPeriod();
        uint256 wProtocol = _wProtocol(pContract);
        if (!_allowUnsafeWindow()) {
            // Assumption: the protocol window never exceeds the contract's expiry period.
            require(pContract >= wProtocol, "ExpiryInvariants: P_contract < W_protocol");
        } else {
            require(pContract < wProtocol, "ExpiryInvariants: unsafe-window config must have P_contract < W_protocol");
        }

        // The L2CrossDomainMessenger's otherMessenger plays A's L1CrossDomainMessenger.
        assertEq(address(l2CrossDomainMessenger.otherMessenger()), address(l1CrossDomainMessenger));
        assertGt(Predeploys.CROSS_L2_INBOX.code.length, 0);

        handler = new ExpiryHandler(address(l1CrossDomainMessenger), exporter, exporter, wProtocol, pContract);
        vm.label(address(handler), "ExpiryHandler");
        initialLiquidity = address(ethLiquidity).balance;

        if (!_isInvariant()) return;
        targetContract(address(handler));
        bytes4[] memory selectors = new bytes4[](_includeForgedPayloadAction() ? 10 : 9);
        selectors[0] = handler.sendETH.selector;
        selectors[1] = handler.warp.selector;
        selectors[2] = handler.relayETH.selector;
        selectors[3] = handler.attackerSend.selector;
        selectors[4] = handler.attackerRelay.selector;
        selectors[5] = handler.exportMessage.selector;
        selectors[6] = handler.deliverFact.selector;
        selectors[7] = handler.forgeExpiry.selector;
        selectors[8] = handler.refund.selector;
        if (_includeForgedPayloadAction()) selectors[9] = handler.relayForgedPayloadToL2CDM.selector;
        targetSelector(FuzzSelector({ addr: address(handler), selectors: selectors }));
    }

    /// @notice Appends this run's handler statistics to env EXPIRY_INV_STATS (a path under .testdata/), if set.
    function afterInvariant() public {
        string memory path = vm.envOr("EXPIRY_INV_STATS", string(""));
        if (bytes(path).length == 0) return;
        string memory line = string.concat(
            _label(),
            ",sends=",
            vm.toString(handler.nSends()),
            ",relays=",
            vm.toString(handler.nRelays()),
            ",relaysBlockedByWindow=",
            vm.toString(handler.nRelaysBlockedByWindow()),
            ",exports=",
            vm.toString(handler.nExports()),
            ",exportsReverted=",
            vm.toString(handler.nExportsReverted()),
            ",facts=",
            vm.toString(handler.factsLength()),
            ",deliveries=",
            vm.toString(handler.nDeliveries())
        );
        line = string.concat(
            line,
            ",expiries=",
            vm.toString(handler.nExpiries()),
            ",refunds=",
            vm.toString(handler.nRefunds()),
            ",refundsReverted=",
            vm.toString(handler.nRefundsReverted()),
            ",attackerSends=",
            vm.toString(handler.nAttackerSends()),
            ",attackerSendsRejected=",
            vm.toString(handler.nAttackerSendsRejected()),
            ",attackerRelays=",
            vm.toString(handler.nAttackerRelays()),
            ",rawWithdrawalsFrom23=",
            vm.toString(handler.rawWithdrawalsFrom23())
        );
        line = string.concat(
            line,
            ",rawWithdrawalsFromTrusted=",
            vm.toString(handler.rawWithdrawalsFromTrusted()),
            ",untrustedWithdrawalsFrom23=",
            vm.toString(handler.untrustedWithdrawalsFrom23()),
            ",forgeAttempts=",
            vm.toString(handler.nForgeAttempts())
        );
        vm.writeLine(path, line);
    }

    /// @notice P_contract. Reads the messenger's expiry constant: EXPIRY_PERIOD at 5992028e08 (8 days), or
    ///         MESSAGE_EXPIRY_WINDOW for the legacy-design mutant (a 37b44c48c7 copy, 7 days).
    function _contractExpiryPeriod() internal view returns (uint256) {
        (bool ok, bytes memory ret) = address(messenger).staticcall(abi.encodeWithSignature("EXPIRY_PERIOD()"));
        if (!ok) (ok, ret) = address(messenger).staticcall(abi.encodeWithSignature("MESSAGE_EXPIRY_WINDOW()"));
        require(ok && ret.length == 32, "ExpiryInvariants: cannot read the contract expiry period");
        return abi.decode(ret, (uint256));
    }

    ////////////////////////////////////////////////////////////////
    //                    Property checks                         //
    ////////////////////////////////////////////////////////////////

    /// @notice NoDoubleSpend: no ETH send is both relayed on its destination and refunded on A.
    function _checkNoDoubleSpend() internal view {
        for (uint256 i = 0; i < handler.ethSendsLength(); i++) {
            (bytes32 h,,,) = handler.ethSend(i);
            assertFalse(messenger.successfulMessages(h) && superchainETHBridge.refunded(h), "NoDoubleSpend");
        }
    }

    /// @notice ETH conservation: ETH minted by relayETH on destinations plus ETH refunded on A never exceeds ETH
    ///         sent from A. Checked on real state (shared ETHLiquidity balance), exact flow accounting, and per send.
    function _checkEthConservation() internal view {
        uint256 bal = address(ethLiquidity).balance;
        assertGe(bal, initialLiquidity, "ETHConservation: liquidity below initial");
        assertEq(
            bal,
            initialLiquidity + handler.ghostSent() - handler.ghostRelayMinted() - handler.ghostRefunded(),
            "ETHConservation: unexplained liquidity flow"
        );
        uint256 out;
        for (uint256 i = 0; i < handler.ethSendsLength(); i++) {
            (bytes32 h,,, uint256 amount) = handler.ethSend(i);
            if (messenger.successfulMessages(h)) out += amount;
            if (superchainETHBridge.refunded(h)) out += amount;
        }
        assertLe(out, handler.ghostSent(), "ETHConservation: minted + refunded > sent");
    }

    /// @notice RefundImpliesExpired.
    function _checkRefundImpliesExpired() internal view {
        assertFalse(handler.refundWithoutExpiry(), "RefundImpliesExpired: refund before expiry");
        assertFalse(handler.refundOfUnknownHash(), "RefundImpliesExpired: refund of a hash never sent");
        for (uint256 i = 0; i < handler.ethSendsLength(); i++) {
            (bytes32 h,,,) = handler.ethSend(i);
            if (superchainETHBridge.refunded(h)) assertTrue(messenger.expiredMessages(h), "RefundImpliesExpired");
        }
    }

    /// @notice ExpiredImpliesNeverRelayable: an expired message was never relayed, and its only initiating event
    ///         was already outside W_protocol at the fact's undeliveredAt and at expiry (so, with a monotone clock,
    ///         no relay can ever be valid again).
    function _checkExpiredImpliesNeverRelayable() internal view {
        assertFalse(handler.relayableAfterExpiry(), "ExpiredImpliesNeverRelayable: relay valid after expiry");
        assertFalse(handler.relayedWhileExpired(), "ExpiredImpliesNeverRelayable: relayed after expiry");
        uint256 w = handler.W_PROTOCOL();
        for (uint256 i = 0; i < handler.ethSendsLength(); i++) {
            (bytes32 h,, uint256 initTs,) = handler.ethSend(i);
            _checkExpired(h, initTs, w);
        }
        for (uint256 j = 0; j < handler.attackerMsgsLength(); j++) {
            (bytes32 h, uint256 initTs) = handler.attackerMsg(j);
            _checkExpired(h, initTs, w);
        }
    }

    function _checkExpired(bytes32 _h, uint256 _initTs, uint256 _w) internal view {
        if (!messenger.expiredMessages(_h)) return;
        assertFalse(messenger.successfulMessages(_h), "ExpiredImpliesNeverRelayable: expired and relayed");
        assertGt(handler.expiredAt(_h), 0, "ExpiredImpliesNeverRelayable: expiry not from a delivered fact");
        assertGt(handler.expiredFactTime(_h), _initTs + _w, "ExpiredImpliesNeverRelayable: fact time within W");
        assertGt(handler.expiredAt(_h), _initTs + _w, "ExpiredImpliesNeverRelayable: expiry within W");
        assertGt(block.timestamp, _initTs + _w, "ExpiredImpliesNeverRelayable: now within W");
    }

    /// @notice AtMostOneRefund: refundETH succeeds at most once per message hash, and pays `from` exactly.
    function _checkAtMostOneRefund() internal view {
        assertFalse(handler.refundPaidWrong(), "AtMostOneRefund: refund paid wrong amount");
        for (uint256 k = 0; k < handler.refundedHashesLength(); k++) {
            assertLe(handler.refundCount(handler.refundedHash(k)), 1, "AtMostOneRefund");
        }
    }

    /// @notice OnlyExportReachesL1: every withdrawal in which the trusted sender (0x..23, or the exporter in the
    ///         exporter design) is the L2CrossDomainMessenger sender came from an exportUndeliveredMessage call, and
    ///         says what an export on that chain at that time would (t == now, H unrelayed there, H not a known
    ///         message to another chain); exports never succeed for a relayed hash.
    function _checkOnlyExportReachesL1() internal view {
        assertEq(handler.nonExportFacts(), 0, "OnlyExportReachesL1: trusted-sender withdrawal not from export");
        assertFalse(handler.dishonestFactCaptured(), "OnlyExportReachesL1: trusted-sender withdrawal not an export's");
        assertFalse(handler.exportAfterRelay(), "OnlyExportReachesL1: export of a relayed message");
        assertFalse(handler.exportHashMismatch(), "OnlyExportReachesL1: export hash mismatch");
    }

    /// @notice NoForgedFact / OnlyDestinationCanExport: every expiry came from a fact exported on the message's
    ///         destination; no other path into expireMessage succeeds.
    function _checkNoForgedFact() internal view {
        assertFalse(handler.forgedFactAccepted(), "NoForgedFact: expiry from a non-destination or non-export fact");
        assertFalse(handler.adversarialExpiryAccepted(), "NoForgedFact: expiry by another path");
    }

    /// @notice UnsafeTargetRule: no message targeting the L2CrossDomainMessenger is ever sent or relayed.
    function _checkUnsafeTargetRule() internal view {
        assertFalse(handler.unsafeTargetAccepted(), "UnsafeTargetRule");
    }

    /// @notice Bookkeeping: sentMessageTimestamps holds each ETH send's initiating timestamp (never rewritten).
    function _checkSentTimestamps() internal view {
        for (uint256 i = 0; i < handler.ethSendsLength(); i++) {
            (bytes32 h,, uint256 initTs,) = handler.ethSend(i);
            assertEq(messenger.sentMessageTimestamps(h), initTs, "SentTimestamps");
        }
    }

    /// @notice OnlyExportInitiatesWithdrawal (trusted sender): the trusted sender never calls the passer directly.
    function _checkNoRawTrustedWithdrawal() internal view {
        assertEq(handler.rawWithdrawalsFromTrusted(), 0, "OnlyExportInitiatesWithdrawal (trusted sender)");
    }

    /// @notice PasserTargetRule: 0x..23 never initiates a raw withdrawal (it never calls the L2ToL1MessagePasser).
    function _checkNoRawWithdrawalFrom23() internal view {
        assertEq(handler.rawWithdrawalsFrom23(), 0, "PasserTargetRule: raw withdrawal from 0x..23");
    }

    function _checkAll() internal view {
        _checkNoDoubleSpend();
        _checkEthConservation();
        _checkRefundImpliesExpired();
        _checkExpiredImpliesNeverRelayable();
        _checkAtMostOneRefund();
        _checkOnlyExportReachesL1();
        _checkNoForgedFact();
        _checkUnsafeTargetRule();
        _checkSentTimestamps();
    }
}

/// @title ExpiryInvariants_Safety_Invariant
/// @notice The safety properties on the real contracts, at W_protocol (default 7 days) and the contract's
///         P_contract (EXPIRY_PERIOD, 8 days).
contract ExpiryInvariants_Safety_Invariant is ExpiryInvariants_TestInit {
    function _label() internal pure override returns (string memory) {
        return "Safety";
    }

    /// @custom:invariant NoDoubleSpend
    function invariant_noDoubleSpend() public view {
        _checkNoDoubleSpend();
    }

    /// @custom:invariant ETH conservation (minted on destinations + refunded on A <= sent from A)
    function invariant_ethConservation() public view {
        _checkEthConservation();
    }

    /// @custom:invariant RefundImpliesExpired
    function invariant_refundImpliesExpired() public view {
        _checkRefundImpliesExpired();
    }

    /// @custom:invariant ExpiredImpliesNeverRelayable
    function invariant_expiredImpliesNeverRelayable() public view {
        _checkExpiredImpliesNeverRelayable();
    }

    /// @custom:invariant AtMostOneRefund
    function invariant_atMostOneRefund() public view {
        _checkAtMostOneRefund();
    }

    /// @custom:invariant OnlyExportReachesL1
    function invariant_onlyExportReachesL1() public view {
        _checkOnlyExportReachesL1();
    }

    /// @custom:invariant NoForgedFact / OnlyDestinationCanExport
    function invariant_noForgedFact() public view {
        _checkNoForgedFact();
    }

    /// @custom:invariant UnsafeTargetRule (L2CrossDomainMessenger and L2ToL1MessagePasser targets)
    function invariant_unsafeTargetRule() public view {
        _checkUnsafeTargetRule();
        _checkNoRawWithdrawalFrom23();
    }

    /// @custom:invariant OnlyExportInitiatesWithdrawal: the exporter never initiates a raw withdrawal.
    function invariant_onlyExportInitiatesWithdrawal() public view {
        _checkNoRawTrustedWithdrawal();
    }

    /// @custom:invariant sentMessageTimestamps is written once per message
    function invariant_sentTimestamps() public view {
        _checkSentTimestamps();
    }
}

/// @title ExpiryInvariants_TightWindow_Invariant
/// @notice All safety properties at the boundary W_protocol == P_contract (8 days), i.e. without the 1-day margin.
contract ExpiryInvariants_TightWindow_Invariant is ExpiryInvariants_TestInit {
    function _label() internal pure override returns (string memory) {
        return "TightWindow";
    }

    function _wProtocol(uint256 _pContract) internal pure override returns (uint256) {
        return _pContract;
    }

    /// @custom:invariant All safety properties at W_protocol == P_contract.
    function invariant_allSafetyProperties() public view {
        _checkAll();
        _checkNoRawWithdrawalFrom23();
        _checkNoRawTrustedWithdrawal();
    }
}

/// @title ExpiryInvariants_NoUnsafeTargetRule_Invariant
/// @notice Expected to PASS. The messenger's unsafe-target rule is REMOVED (test-only mutant
///         mutants/L2ToL2CrossDomainMessengerNoUnsafeTargets.sol): relayed messages may call the
///         L2CrossDomainMessenger and the L2ToL1MessagePasser as 0x..23. With exports in the UndeliveredMessageExporter
///         that A's L1CrossDomainMessenger trusts, no relayed message can make the trusted sender send anything, so
///         safety does not depend on the rule (the legacy 0x..23-trusted design did:
///         ExpiryInvariants_LegacyNoTargetRule_Invariant). UnsafeTargetRule and PasserTargetRule are not checked
///         here (the mutant violates them by construction).
contract ExpiryInvariants_NoUnsafeTargetRule_Invariant is ExpiryInvariants_TestInit {
    function _label() internal pure override returns (string memory) {
        return "NoUnsafeTargetRule";
    }

    function _mutant() internal pure override returns (uint8) {
        return MUTANT_NO_UNSAFE_TARGETS;
    }

    /// @custom:invariant Safety properties (all but the target rules) without the unsafe-target rule.
    function invariant_safetyWithoutTargetRule() public view {
        _checkNoDoubleSpend();
        _checkEthConservation();
        _checkRefundImpliesExpired();
        _checkExpiredImpliesNeverRelayable();
        _checkAtMostOneRefund();
        _checkOnlyExportReachesL1();
        _checkNoForgedFact();
        _checkSentTimestamps();
        _checkNoRawTrustedWithdrawal();
    }
}

/// @title ExpiryInvariants_NonVacuity_Invariant
/// @notice EXPECTED TO FAIL (run with EXPIRY_INV_EXPECT_FAIL=true). Each property says a step of the honest path
///         never happens; the fuzzer must falsify each, showing the safety campaign reaches relays, exports,
///         expiries and refunds.
contract ExpiryInvariants_NonVacuity_Invariant is ExpiryInvariants_TestInit {
    function _label() internal pure override returns (string memory) {
        return "NonVacuity";
    }

    function _expectFail() internal pure override returns (bool) {
        return true;
    }

    /// @custom:invariant EXPECTED FAIL: a refund is reachable.
    function invariant_refundNeverHappens() public view {
        assertEq(handler.nRefunds(), 0, "witness: refund reached");
    }

    /// @custom:invariant EXPECTED FAIL: an expiry is reachable.
    function invariant_expiryNeverHappens() public view {
        assertEq(handler.nExpiries(), 0, "witness: expiry reached");
    }

    /// @custom:invariant EXPECTED FAIL: an ETH relay is reachable.
    function invariant_relayNeverHappens() public view {
        assertEq(handler.nRelays(), 0, "witness: relay reached");
    }

    /// @custom:invariant EXPECTED FAIL: a refund after a relay attempt was blocked by the window is reachable.
    function invariant_refundAfterBlockedRelayNeverHappens() public view {
        assertFalse(handler.nRefunds() > 0 && handler.nRelaysBlockedByWindow() > 0, "witness: blocked relay + refund");
    }
}

/// @title ExpiryInvariants_UnsafeWindow_Invariant
/// @notice EXPECTED TO FAIL (run with EXPIRY_INV_EXPECT_FAIL=true). Drops the assumption: W_protocol =
///         P_contract + 1 day. A message relayed between P_contract and W_protocol after its send can also be
///         expired and refunded.
contract ExpiryInvariants_UnsafeWindow_Invariant is ExpiryInvariants_TestInit {
    function _label() internal pure override returns (string memory) {
        return "UnsafeWindow";
    }

    function _expectFail() internal pure override returns (bool) {
        return true;
    }

    function _allowUnsafeWindow() internal pure override returns (bool) {
        return true;
    }

    function _wProtocol(uint256 _pContract) internal pure override returns (uint256) {
        return _pContract + 1 days;
    }

    /// @custom:invariant EXPECTED FAIL when P_contract < W_protocol: NoDoubleSpend.
    function invariant_noDoubleSpend() public view {
        _checkNoDoubleSpend();
    }

    /// @custom:invariant EXPECTED FAIL when P_contract < W_protocol: ExpiredImpliesNeverRelayable.
    function invariant_expiredImpliesNeverRelayable() public view {
        _checkExpiredImpliesNeverRelayable();
    }
}

/// @title ExpiryInvariants_LegacyNoTargetRule_Invariant
/// @notice EXPECTED TO FAIL (run with EXPIRY_INV_EXPECT_FAIL=true). Legacy design (37b44c48c7: the messenger exports
///         and A's L1CrossDomainMessenger trusts 0x..23) with its target rule removed (test-only mutant
///         mutants/L2ToL2CrossDomainMessengerLegacyNoTargetRule.sol). A relayed message to the
///         L2CrossDomainMessenger forges a fact. Relays without an initiating event are excluded, so counterexamples
///         use only protocol-valid relays.
contract ExpiryInvariants_LegacyNoTargetRule_Invariant is ExpiryInvariants_TestInit {
    function _label() internal pure override returns (string memory) {
        return "LegacyNoTargetRule";
    }

    function _expectFail() internal pure override returns (bool) {
        return true;
    }

    function _mutant() internal pure override returns (uint8) {
        return MUTANT_LEGACY_NO_TARGET_RULE;
    }

    function _includeForgedPayloadAction() internal pure override returns (bool) {
        return false;
    }

    /// @custom:invariant EXPECTED FAIL under the legacy mutant: OnlyExportReachesL1.
    function invariant_onlyExportReachesL1() public view {
        _checkOnlyExportReachesL1();
    }

    /// @custom:invariant EXPECTED FAIL under the legacy mutant: NoDoubleSpend.
    function invariant_noDoubleSpend() public view {
        _checkNoDoubleSpend();
    }

    /// @custom:invariant EXPECTED FAIL under the legacy mutant: NoForgedFact.
    function invariant_noForgedFact() public view {
        _checkNoForgedFact();
    }
}

/// @title ExpiryInvariants_Witness_Test
/// @notice Deterministic reachability witnesses on the real contracts (CI-friendly non-vacuity): each drives the
///         handler through a specific path and checks both the path's effect and that every property still holds.
contract ExpiryInvariants_Witness_Test is ExpiryInvariants_TestInit {
    function _label() internal pure override returns (string memory) {
        return "Witness";
    }

    function _isInvariant() internal pure override returns (bool) {
        return false;
    }

    function _checkAllReal() internal view {
        _checkAll();
        _checkNoRawWithdrawalFrom23();
        _checkNoRawTrustedWithdrawal();
    }

    /// @notice send -> warp to init + P + 1 -> export on B -> deliver -> refund; repeat refund and late relay fail.
    function test_witness_refundAfterExpiry_succeeds() external {
        handler.sendETH(0, 0, 1 ether, 0);
        (bytes32 h, uint256 dest, uint256 initTs,) = handler.ethSend(0);
        assertEq(dest, handler.CHAIN_B());

        handler.warp(3, 4, 0);
        assertEq(block.timestamp, initTs + handler.P_CONTRACT() + 1);

        handler.exportMessage(0, 0, 1, 1, 200_000, false);
        assertEq(handler.nExports(), 1);
        assertEq(handler.factsLength(), 1);

        handler.deliverFact(0, 2, 0);
        assertTrue(messenger.expiredMessages(h));
        assertEq(handler.nExpiries(), 1);

        handler.refund(0, 0, 0, address(this));
        assertTrue(superchainETHBridge.refunded(h));
        assertEq(handler.nRefunds(), 1);

        handler.refund(0, 0, 0, address(this));
        assertEq(handler.nRefunds(), 1);
        assertEq(handler.nRefundsReverted(), 1);

        handler.relayETH(0);
        assertEq(handler.nRelaysBlockedByWindow(), 1);
        assertFalse(messenger.successfulMessages(h));

        _checkAllReal();
    }

    /// @notice A fact dated at or before init + P (here: honest export at init + P, and a weakened t) cannot expire.
    function test_witness_earlyFactRejected_succeeds() external {
        handler.sendETH(0, 0, 1 ether, 0);
        (bytes32 h,,,) = handler.ethSend(0);
        handler.warp(3, 3, 0); // init + P
        handler.exportMessage(0, 0, 1, 1, 200_000, false);
        handler.deliverFact(0, 2, 0);
        assertFalse(messenger.expiredMessages(h));
        handler.warp(3, 5, 0); // init + P + 2
        handler.exportMessage(0, 0, 1, 1, 200_000, false);
        handler.deliverFact(1, 1, 0); // weakened to t' = 0
        assertFalse(messenger.expiredMessages(h));
        handler.deliverFact(1, 2, 0); // honest t
        assertTrue(messenger.expiredMessages(h));
        _checkAllReal();
    }

    /// @notice A relayed message cannot be exported, and no other path expires it.
    function test_witness_relayBlocksExport_succeeds() external {
        handler.sendETH(0, 0, 1 ether, 0);
        (bytes32 h,,,) = handler.ethSend(0);
        handler.relayETH(0);
        assertTrue(messenger.successfulMessages(h));
        assertEq(handler.nRelays(), 1);

        handler.warp(3, 4, 0);
        handler.exportMessage(0, 0, 1, 1, 200_000, false);
        assertEq(handler.nExportsReverted(), 1);
        assertEq(handler.factsLength(), 0);

        for (uint8 mode = 0; mode < 4; mode++) {
            handler.forgeExpiry(mode, 0, 0, address(0xBAD), address(0));
        }
        assertFalse(messenger.expiredMessages(h));
        _checkAllReal();
    }

    /// @notice Exports from a non-destination chain, with a wrong source, or to a wrong L1 target cannot expire.
    function test_witness_wrongExportRejected_succeeds() external {
        handler.sendETH(0, 0, 1 ether, 0);
        (bytes32 h,,,) = handler.ethSend(0);
        handler.warp(3, 4, 0);
        handler.exportMessage(0, 6, 1, 1, 200_000, false); // on C (dest is B)
        handler.exportMessage(0, 7, 1, 1, 200_000, false); // on A
        handler.exportMessage(0, 0, 1, 0, 200_000, false); // wrong source
        handler.exportMessage(0, 0, 0, 1, 200_000, false); // wrong L1 target
        assertEq(handler.factsLength(), 4);
        for (uint256 k = 0; k < 4; k++) {
            handler.deliverFact(k, 2, 0);
        }
        assertFalse(messenger.expiredMessages(h));
        assertEq(handler.nUndeliverable(), 1);
        _checkAllReal();
    }

    /// @notice The unsafe-target rule rejects attacker messages to the L2CrossDomainMessenger and the
    ///         L2ToL1MessagePasser on send, and a relay to the L2CrossDomainMessenger of a payload with no initiating
    ///         event.
    function test_witness_targetRuleRejects_succeeds() external {
        handler.sendETH(0, 0, 1 ether, 0);
        handler.attackerSend(0, 0, 0, 0, 200_000, "", true); // L2CrossDomainMessenger
        handler.attackerSend(1, 1, 0, 0, 200_000, "", true); // L2ToL1MessagePasser
        assertEq(handler.nAttackerSendsRejected(), 2);
        handler.relayForgedPayloadToL2CDM(0, 0, 0, 200_000, "");
        assertEq(handler.factsLength(), 0);
        assertEq(handler.rawWithdrawalsFrom23(), 0);
        _checkAllReal();
    }

    /// @notice A relayed message whose target is the exporter, calling exportUndeliveredMessage, is an export: its
    ///         fact is honest and expires an unrelayed send past its expiry period.
    function test_witness_relayedExportCall_succeeds() external {
        handler.sendETH(0, 0, 1 ether, 0);
        (bytes32 h,,,) = handler.ethSend(0);
        handler.warp(3, 4, 0);
        handler.attackerSend(7, 5, 0, 0, 200_000, "", true);
        assertEq(handler.nAttackerRelays(), 1);
        assertEq(handler.factsLength(), 1);
        assertEq(handler.nonExportFacts(), 0);
        handler.deliverFact(0, 2, 0);
        assertTrue(messenger.expiredMessages(h));
        handler.refund(0, 0, 0, address(this));
        assertTrue(superchainETHBridge.refunded(h));
        _checkAllReal();
    }
}

/// @title ExpiryInvariants_UnsafeWindowWitness_Test
/// @notice Deterministic counterexample with W_protocol = P_contract + 1 day: relay at init + P + 1 (valid), after an
///         export at init + P + 1 (also valid), then expiry and refund: a double spend.
contract ExpiryInvariants_UnsafeWindowWitness_Test is ExpiryInvariants_TestInit {
    function _label() internal pure override returns (string memory) {
        return "UnsafeWindowWitness";
    }

    function _isInvariant() internal pure override returns (bool) {
        return false;
    }

    function _allowUnsafeWindow() internal pure override returns (bool) {
        return true;
    }

    function _wProtocol(uint256 _pContract) internal pure override returns (uint256) {
        return _pContract + 1 days;
    }

    function test_witness_unsafeWindowDoubleSpend_succeeds() external {
        handler.sendETH(0, 0, 1 ether, 0);
        (bytes32 h,,,) = handler.ethSend(0);
        handler.warp(3, 4, 0); // init + P + 1
        handler.exportMessage(0, 0, 1, 1, 200_000, false);
        handler.relayETH(0);
        assertTrue(messenger.successfulMessages(h));
        handler.deliverFact(0, 2, 0);
        handler.refund(0, 0, 0, address(this));
        assertTrue(superchainETHBridge.refunded(h));
        // NoDoubleSpend is violated: both relayed and refunded.
        assertTrue(messenger.successfulMessages(h) && superchainETHBridge.refunded(h));
    }
}

/// @title ExpiryInvariants_NoUnsafeTargetRuleWitness_Test
/// @notice Without the unsafe-target rule (test-only mutant), the attack that breaks the legacy design goes through
///         on L2 but its withdrawal's sender is 0x..23, which A's L1CrossDomainMessenger does not trust: no fact.
contract ExpiryInvariants_NoUnsafeTargetRuleWitness_Test is ExpiryInvariants_TestInit {
    function _label() internal pure override returns (string memory) {
        return "NoUnsafeTargetRuleWitness";
    }

    function _isInvariant() internal pure override returns (bool) {
        return false;
    }

    function _mutant() internal pure override returns (uint8) {
        return MUTANT_NO_UNSAFE_TARGETS;
    }

    function test_witness_exporterDesignBlocksForgery_succeeds() external {
        handler.sendETH(0, 0, 1 ether, 0);
        (bytes32 h,,,) = handler.ethSend(0);
        handler.relayETH(0);

        // target L2CrossDomainMessenger, calldata sendMessage(A's L1CDM, relayUndeliveredMessage(h, init + P + 1)).
        handler.attackerSend(0, 0, 0, 0, 200_000, "", true);
        assertTrue(handler.unsafeTargetAccepted());
        assertEq(handler.untrustedWithdrawalsFrom23(), 1);
        // target L2ToL1MessagePasser, calldata initiateWithdrawal(A's L1CDM, gas, relayUndeliveredMessage(...)).
        handler.attackerSend(1, 1, 0, 0, 200_000, "", true);
        assertEq(handler.rawWithdrawalsFrom23(), 1);

        assertEq(handler.factsLength(), 0);
        assertFalse(messenger.expiredMessages(h));
        _checkNoDoubleSpend();
        _checkOnlyExportReachesL1();
        _checkNoForgedFact();
        _checkNoRawTrustedWithdrawal();
    }
}

/// @title ExpiryInvariants_LegacyNoTargetRuleWitness_Test
/// @notice Deterministic counterexample for the legacy design (37b44c48c7 messenger trusted as 0x..23) without its
///         target rule: an attacker message to the L2CrossDomainMessenger forges a fact for a relayed send, which then
///         expires and is refunded.
contract ExpiryInvariants_LegacyNoTargetRuleWitness_Test is ExpiryInvariants_TestInit {
    function _label() internal pure override returns (string memory) {
        return "LegacyNoTargetRuleWitness";
    }

    function _isInvariant() internal pure override returns (bool) {
        return false;
    }

    function _mutant() internal pure override returns (uint8) {
        return MUTANT_LEGACY_NO_TARGET_RULE;
    }

    function test_witness_legacyForgedFact_succeeds() external {
        handler.sendETH(0, 0, 1 ether, 0);
        (bytes32 h,,,) = handler.ethSend(0);
        handler.relayETH(0);
        assertTrue(messenger.successfulMessages(h));

        // target L2CrossDomainMessenger, calldata sendMessage(A's L1CDM, relayUndeliveredMessage(h, init + P + 1)).
        handler.attackerSend(0, 0, 0, 0, 200_000, "", true);
        assertTrue(handler.unsafeTargetAccepted());
        assertEq(handler.nonExportFacts(), 1);

        handler.deliverFact(0, 2, 0);
        assertTrue(messenger.expiredMessages(h));
        assertTrue(handler.forgedFactAccepted());

        handler.refund(0, 0, 0, address(this));
        assertTrue(messenger.successfulMessages(h) && superchainETHBridge.refunded(h));
    }
}

/// @title ExpiryInvariants_GovernanceAssumptionWitness_Test
/// @notice Shows the governance assumption is load-bearing: if a cluster chain's L2 governance (its L2 ProxyAdmin
/// owner) upgrades its own UndeliveredMessageExporter, the exporter can say a relayed message was not relayed, and the
///         send is both relayed and refunded. The harness assumes every cluster chain runs the real exporter; this is
///         the same trust as the shared ETHLockbox, whose portals must share the proxy admin owner.
contract ExpiryInvariants_GovernanceAssumptionWitness_Test is ExpiryInvariants_TestInit {
    function _label() internal pure override returns (string memory) {
        return "GovernanceAssumptionWitness";
    }

    function _isInvariant() internal pure override returns (bool) {
        return false;
    }

    function test_witness_upgradedExporterForgesFact_succeeds() external {
        handler.sendETH(0, 0, 1 ether, 0);
        (bytes32 h,, uint256 initTs,) = handler.ethSend(0);
        handler.relayETH(0);
        assertTrue(messenger.successfulMessages(h));

        handler.forgeAsUpgradedExporter(0, initTs + handler.P_CONTRACT() + 1);
        assertEq(handler.nonExportFacts(), 1);
        assertTrue(handler.dishonestFactCaptured());

        handler.deliverFact(0, 2, 0);
        assertTrue(messenger.expiredMessages(h));
        handler.refund(0, 0, 0, address(this));
        assertTrue(messenger.successfulMessages(h) && superchainETHBridge.refunded(h));
    }
}
