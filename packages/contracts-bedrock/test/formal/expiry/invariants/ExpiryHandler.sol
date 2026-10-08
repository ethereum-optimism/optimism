// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { CommonBase } from "forge-std/Base.sol";
import { StdUtils } from "forge-std/StdUtils.sol";
import { Vm } from "forge-std/Vm.sol";

// Libraries
import { Predeploys } from "src/libraries/Predeploys.sol";
import { Hashing } from "src/libraries/Hashing.sol";
import { Encoding } from "src/libraries/Encoding.sol";
import { AddressAliasHelper } from "src/vendor/AddressAliasHelper.sol";

// Interfaces
import { IL2ToL2CrossDomainMessenger } from "interfaces/L2/IL2ToL2CrossDomainMessenger.sol";
import { ICrossL2Inbox, Identifier } from "interfaces/L2/ICrossL2Inbox.sol";
import { ISuperchainETHBridge } from "interfaces/L2/ISuperchainETHBridge.sol";
import { ICrossDomainMessenger } from "interfaces/universal/ICrossDomainMessenger.sol";
import { IL1CrossDomainMessenger } from "interfaces/L1/IL1CrossDomainMessenger.sol";
import { IL2ToL1MessagePasser } from "interfaces/L2/IL2ToL1MessagePasser.sol";
import { IUndeliveredMessageExporter } from "interfaces/L2/IUndeliveredMessageExporter.sol";

/// @title ExpiryHandler
/// @notice Stateful fuzzing handler for per-message interop expiry. One EVM hosts the REAL
///         L2ToL2CrossDomainMessenger, SuperchainETHBridge, ETHLiquidity, L2CrossDomainMessenger and
///         L2ToL1MessagePasser predeploys (from the repo's L2 genesis), and plays several chains by switching
///         `vm.chainId` before each step:
///         - CHAIN_A is the only source chain: it is the only chain that calls sendMessage / sendETH, receives
///           expiry facts (expireMessage via the L2CrossDomainMessenger) and refunds. So the A-side mappings
///           (sentMessageTimestamps, expiredMessages, SuperchainETHBridge.refunded) are written by A only.
///         - CHAIN_B and CHAIN_C are destinations: they relay (writing successfulMessages[H], whose H commits to
///           the destination) and export. Every chain may call exportUndeliveredMessage.
///         Shared storage across roles is sound because each mapping is written by exactly one role and every
///         hash commits to (destination, source). The ETHLiquidity pool is shared, so its balance measures
///         cluster-wide conservation (minted on destinations + refunded on A <= sent from A).
///
///         Abstractions (stated, not proven):
///         - CrossL2Inbox.validateMessage is mocked, exactly for the (identifier, payload hash) being relayed.
///           The protocol rule is enforced here instead: a relay is attempted only for a payload taken from a
///           SentMessage log the real messenger emitted on CHAIN_A, and only if exec - init <= W_PROTOCOL.
///         - One global, monotone clock (block.timestamp) is shared by all chains.
///         - The L1 hop is not executed. TRUSTED_SENDER is the L2 sender L1CrossDomainMessenger.relayUndeliveredMessage
///           trusts (xDomainMessageSender): the UndeliveredMessageExporter predeploy
/// (Predeploys.UNDELIVERED_MESSAGE_EXPORTER) at c7c51d79e2; 0x..23 in the legacy design (37b44c48c7), which the
/// legacy-mutant configuration models. EXPORTER is the
///           contract whose exportUndeliveredMessage the handler calls (the same address). The L1 interop feature gate
///           (systemConfig INTEROP) is assumed on for A. A
///           withdrawal is captured from the L2ToL1MessagePasser MessagePassed log; one sent by the
///           L2CrossDomainMessenger whose inner sender is TRUSTED_SENDER, whose L1 target is A's
///           L1CrossDomainMessenger and whose calldata is relayUndeliveredMessage(H, t) becomes a "fact". Delivering
///           a fact runs the REAL L2CrossDomainMessenger.relayMessage on CHAIN_A, pranked as the aliased
///           L1CrossDomainMessenger, with sender = A's L1CrossDomainMessenger and target 0x..23, calling
///           expireMessage(H, t) with gas limit EXPIRE_MESSAGE_GAS_LIMIT. This models that
///           L1CrossDomainMessenger.relayUndeliveredMessage accepts any messenger-sent withdrawal from
///           TRUSTED_SENDER of any chain whose portal is in A's lockbox (every chain here is), and nothing else.
///         - Raw withdrawals (MessagePassed whose sender is 0x..23 or TRUSTED_SENDER, i.e. the contract calling the
///           passer directly) are counted but never delivered: on L1 the portal calls their target directly, so
///           relayUndeliveredMessage's caller is a portal (no portal()/xDomainMessageSender), and an
///           L1CrossDomainMessenger rejects them (portal.l2Sender() != L2CrossDomainMessenger). At c7c51d79e2 the
///           messenger rejects the L2ToL1MessagePasser as a target, so none should exist; an invariant checks that.
contract ExpiryHandler is CommonBase, StdUtils {
    /// @notice Thrown when sendETH returns a hash other than the one recomputed from its arguments.
    error ExpiryHandler_SendHashMismatch();

    /// @notice Thrown when a send emits no L2ToL2CrossDomainMessenger SentMessage log.
    error ExpiryHandler_NoSentMessageLog();

    ////////////////////////////////////////////////////////////////
    //                         Constants                          //
    ////////////////////////////////////////////////////////////////

    /// @notice Chain IDs. A is the source; B and C are destinations in A's cluster.
    uint256 public constant CHAIN_A = 901;
    uint256 public constant CHAIN_B = 902;
    uint256 public constant CHAIN_C = 903;

    /// @notice Gas limit L1CrossDomainMessenger.relayUndeliveredMessage uses for expireMessage.
    uint32 public constant EXPIRE_MESSAGE_GAS_LIMIT = 100_000;

    /// @notice Bound on fuzzed ETH amounts and on junk calldata length.
    uint256 internal constant MAX_AMOUNT = 1_000 ether;
    uint256 internal constant MAX_JUNK = 256;

    /// @notice Event signatures (solc 0.8.15 cannot read another contract's event selector). The witnesses assert
    ///         that each one is observed on the real contracts (sends revert without a SentMessage log, exports
    ///         must yield facts, deliveries must count MessageExpired emissions), so a renamed event fails loudly.
    bytes32 internal constant L2TOL2_SENT_MESSAGE_SIG = keccak256("SentMessage(uint256,address,uint256,address,bytes)");
    bytes32 internal constant MESSAGE_EXPIRED_SIG = keccak256("MessageExpired(bytes32,uint256)");
    bytes32 internal constant MESSAGE_PASSED_SIG =
        keccak256("MessagePassed(uint256,address,address,uint256,uint256,bytes,bytes32)");

    /// @notice Actors.
    address internal constant ATTACKER = address(0xA77AC4E7);
    address internal constant RELAYER = address(0x5E1A7E5);

    ////////////////////////////////////////////////////////////////
    //                         Contracts                          //
    ////////////////////////////////////////////////////////////////

    IL2ToL2CrossDomainMessenger internal immutable messenger =
        IL2ToL2CrossDomainMessenger(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER);
    ISuperchainETHBridge internal immutable bridge = ISuperchainETHBridge(payable(Predeploys.SUPERCHAIN_ETH_BRIDGE));
    ICrossDomainMessenger internal immutable l2cdm = ICrossDomainMessenger(Predeploys.L2_CROSS_DOMAIN_MESSENGER);

    /// @notice A's L1CrossDomainMessenger (the L2CrossDomainMessenger's otherMessenger).
    address public immutable aL1Messenger;

    /// @notice The contract whose exportUndeliveredMessage the handler calls (the UndeliveredMessageExporter).
    address public immutable EXPORTER;

    /// @notice The L2 sender L1CrossDomainMessenger.relayUndeliveredMessage trusts (the UndeliveredMessageExporter).
    address public immutable TRUSTED_SENDER;

    /// @notice A decoy L1 address used as a wrong export target ("B's L1CrossDomainMessenger").
    address public constant B_L1_MESSENGER = address(0xB11CD);

    /// @notice W_protocol: relay validity window (exec - init <= W_PROTOCOL).
    uint256 public immutable W_PROTOCOL;

    /// @notice P_contract: the messenger's expiry period, read from the contract.
    uint256 public immutable P_CONTRACT;

    ////////////////////////////////////////////////////////////////
    //                       Ghost state                          //
    ////////////////////////////////////////////////////////////////

    /// @notice A message initiated on CHAIN_A, with the SentMessage payload the real messenger emitted.
    struct Message {
        uint256 destination;
        uint256 nonce;
        address sender;
        address target;
        bytes message;
        bytes32 hash;
        uint256 initTs;
        bytes payload;
        // ETH sends only.
        address from;
        address to;
        uint256 amount;
    }

    /// @notice A withdrawal from TRUSTED_SENDER through the L2CrossDomainMessenger, captured on some chain.
    struct Fact {
        uint256 chain;
        address l1Target;
        bytes l1Message;
        bool fromExport;
        uint256 at;
    }

    Message[] internal ethSends;
    Message[] internal attackerMsgs;
    Fact[] internal facts;

    /// @notice Destination and initiating timestamp of every message initiated on A, by hash (0 = unknown).
    mapping(bytes32 => uint256) public destOf;
    mapping(bytes32 => uint256) public initTsOf;
    mapping(bytes32 => bool) public isEthSend;
    mapping(bytes32 => uint256) internal sendIndexPlusOne;

    /// @notice ETH sends that expired (for biased refund selection).
    bytes32[] internal expiredSends;

    /// @notice First expiry of a hash: A's timestamp, the fact's undeliveredAt, and fact index + 1.
    mapping(bytes32 => uint256) public expiredAt;
    mapping(bytes32 => uint256) public expiredFactTime;
    mapping(bytes32 => uint256) public expiredFactIndexPlusOne;

    /// @notice Successful relays per hash (any message), observed by the handler.
    mapping(bytes32 => uint256) public relayCount;

    /// @notice Whether a relay of the hash was attempted and refused by the protocol window rule.
    mapping(bytes32 => bool) public blockedRelay;

    /// @notice Successful refunds per hash, as computed from the refundETH arguments.
    mapping(bytes32 => uint256) public refundCount;
    bytes32[] internal refundedHashes;

    /// @notice ETH flow ghosts.
    uint256 public ghostSent;
    uint256 public ghostRelayMinted;
    uint256 public ghostRefunded;

    /// @notice Violation flags (each must stay false).
    bool public refundWithoutExpiry;
    bool public refundOfUnknownHash;
    bool public refundPaidWrong;
    bool public relayableAfterExpiry;
    bool public relayedWhileExpired;
    bool public exportAfterRelay;
    bool public exportHashMismatch;
    bool public forgedFactAccepted;
    bool public adversarialExpiryAccepted;
    bool public unsafeTargetAccepted;
    bool public relayPaidWrong;
    bool public bridgeRetainedEth;
    bool public wrongChainRelayAccepted;
    bool public unauthorizedExpiryEmitted;

    /// @notice Non-vacuity witness: some hash had a window-blocked relay attempt and was refunded afterwards.
    bool public refundAfterBlockedRelay;

    /// @notice L2ToL2CrossDomainMessenger SentMessage events emitted outside A's sends (nested sends from a
    ///         destination). The handler filters the only path to them (a relayed bridge.sendETH), so this must stay 0.
    uint256 public nestedSends;

    /// @notice Withdrawals from TRUSTED_SENDER not produced by an exportUndeliveredMessage call (must stay 0).
    uint256 public nonExportFacts;
    /// @notice Withdrawals from TRUSTED_SENDER carrying relayUndeliveredMessage(H, t) that an export could not have
    ///         produced on that chain at that time: t != now, H already relayed there, or H is a known message to
    ///         another chain (must stay false).
    bool public dishonestFactCaptured;
    /// @notice Raw withdrawals (sender == 0x..23 at the passer). Zero with the passer target rule (c7c51d79e2).
    uint256 public rawWithdrawalsFrom23;
    /// @notice Raw withdrawals with sender == TRUSTED_SENDER at the passer (same as above when it is 0x..23).
    uint256 public rawWithdrawalsFromTrusted;
    /// @notice L2CrossDomainMessenger withdrawals whose inner sender is 0x..23 while 0x..23 is NOT trusted (exporter
    ///         design): attacks that would have been facts under the legacy 0x..23-trusted design.
    uint256 public untrustedWithdrawalsFrom23;

    /// @notice Statistics (non-vacuity / coverage).
    uint256 public nSends;
    uint256 public nRelays;
    uint256 public nRelaysBlockedByWindow;
    uint256 public nRelaysReverted;
    uint256 public nExports;
    uint256 public nExportsReverted;
    uint256 public nDeliveries;
    uint256 public nUndeliverable;
    uint256 public nExpiries;
    uint256 public nRefunds;
    uint256 public nRefundsReverted;
    uint256 public nAttackerSends;
    uint256 public nAttackerSendsRejected;
    uint256 public nAttackerRelays;
    uint256 public nForgeAttempts;
    uint256 public nWarps;
    uint256 public nWrongChainRelays;
    uint256 public nExpiredEmissions;
    uint256 public nNestedSendsFiltered;

    /// @notice L1->L2 nonce counter for delivered facts.
    uint240 internal l1Nonce;

    /// @notice Whether attackerSend filters relayed bridge.sendETH calls (nested sends). Only the nestedSends
    ///         failing witness turns it off.
    bool public nestedSendFilter = true;

    constructor(
        address _aL1Messenger,
        address _exporter,
        address _trustedSender,
        uint256 _wProtocol,
        uint256 _pContract
    ) {
        aL1Messenger = _aL1Messenger;
        EXPORTER = _exporter;
        TRUSTED_SENDER = _trustedSender;
        W_PROTOCOL = _wProtocol;
        P_CONTRACT = _pContract;
    }

    ////////////////////////////////////////////////////////////////
    //                         Actions                            //
    ////////////////////////////////////////////////////////////////

    /// @notice A: SuperchainETHBridge.sendETH to B (3/4) or C (1/4).
    function sendETH(uint256 _fromSeed, uint256 _toSeed, uint256 _amount, uint8 _destSel) public {
        vm.chainId(CHAIN_A);
        address from = _actor(_fromSeed);
        address to = _recipient(_toSeed);
        uint256 amount = bound(_amount, 0, MAX_AMOUNT);
        uint256 destination = _destSel % 4 == 3 ? CHAIN_C : CHAIN_B;
        vm.deal(from, amount);

        uint256 nonce = messenger.messageNonce();
        vm.recordLogs();
        vm.prank(from);
        bytes32 h = bridge.sendETH{ value: amount }(to, destination);
        bytes memory payload = _sentMessagePayload(vm.getRecordedLogs());

        bytes memory message = abi.encodeCall(ISuperchainETHBridge.relayETH, (from, to, amount));
        if (
            h
                != Hashing.hashL2toL2CrossDomainMessage(
                    destination, CHAIN_A, nonce, address(bridge), address(bridge), message
                )
        ) revert ExpiryHandler_SendHashMismatch();

        ethSends.push(
            Message({
                destination: destination,
                nonce: nonce,
                sender: address(bridge),
                target: address(bridge),
                message: message,
                hash: h,
                initTs: block.timestamp,
                payload: payload,
                from: from,
                to: to,
                amount: amount
            })
        );
        destOf[h] = destination;
        initTsOf[h] = block.timestamp;
        isEthSend[h] = true;
        sendIndexPlusOne[h] = ethSends.length;
        ghostSent += amount;
        nSends++;
    }

    /// @notice Advance the shared clock. Modes 3-5 jump to a boundary of an ETH send's windows.
    function warp(uint8 _mode, uint256 _amount, uint256 _i) public {
        uint256 mode = _mode % 6;
        uint256 dt;
        if (mode == 0) {
            dt = bound(_amount, 1, 1 hours);
        } else if (mode == 1) {
            dt = bound(_amount, 1, 2 days);
        } else if (mode == 2) {
            dt = bound(_amount, 1, 10 days);
        } else {
            if (ethSends.length == 0) return;
            Message storage m = ethSends[_i % ethSends.length];
            uint256 k = _amount % 7;
            uint256 targetTs;
            if (k < 3) targetTs = m.initTs + W_PROTOCOL + k - 1; // W - 1, W, W + 1
            else targetTs = m.initTs + P_CONTRACT + k - 3; // P, P + 1, P + 2, P + 3
            if (targetTs <= block.timestamp) return;
            dt = targetTs - block.timestamp;
        }
        vm.warp(block.timestamp + dt);
        nWarps++;
    }

    /// @notice Destination: relay ETH send `_i` on its destination, only if the protocol accepts it.
    function relayETH(uint256 _i) public {
        if (ethSends.length == 0) return;
        Message storage m = ethSends[_i % ethSends.length];
        uint256 toBefore = m.to.balance;
        uint256 bridgeBefore = address(bridge).balance;
        bool ok = _relay(m);
        if (address(bridge).balance != bridgeBefore) bridgeRetainedEth = true;
        if (ok) {
            // The destination pays the recipient exactly the amount.
            if (m.to.balance != toBefore + m.amount) relayPaidWrong = true;
            ghostRelayMinted += m.amount;
        }
    }

    /// @notice Relay an authentic, within-window ETH send payload on the OTHER destination chain (B <-> C). The
    ///         inbox mock accepts it (it is a real initiating event), so only the messenger's destination check can
    ///         refuse it; it must.
    function relayOnWrongChain(uint256 _i) public {
        if (ethSends.length == 0) return;
        Message storage m = ethSends[_i % ethSends.length];
        if (block.timestamp - m.initTs > W_PROTOCOL) return;
        uint256 wrong = m.destination == CHAIN_B ? CHAIN_C : CHAIN_B;
        vm.chainId(wrong);
        Identifier memory id = Identifier({
            origin: Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER,
            blockNumber: 1,
            logIndex: 0,
            timestamp: m.initTs,
            chainId: CHAIN_A
        });
        vm.mockCall(
            Predeploys.CROSS_L2_INBOX, abi.encodeCall(ICrossL2Inbox.validateMessage, (id, keccak256(m.payload))), ""
        );
        vm.recordLogs();
        vm.prank(RELAYER);
        try messenger.relayMessage(id, m.payload) {
            wrongChainRelayAccepted = true;
        } catch { }
        vm.clearMockedCalls();
        _capture(vm.getRecordedLogs(), false, wrong);
        nWrongChainRelays++;
    }

    /// @notice Attacker on A sends a message to B with a chosen target and calldata, and optionally relays it at
    ///         once (exec == init). Targets include the L2CrossDomainMessenger (the target rule must reject it) and
    ///         the L2ToL1MessagePasser.
    function attackerSend(
        uint8 _targetSel,
        uint8 _dataSel,
        uint256 _i,
        uint256 _tSeed,
        uint32 _gas,
        bytes calldata _junk,
        bool _relayNow
    )
        public
    {
        address target = _attackTarget(_targetSel, _junk);
        bytes memory data = _attackData(_dataSel, _i, _tSeed, _gas, _junk);
        // Nested traffic: a relayed bridge.sendETH would make a destination send a child message (a second source),
        // which this single-source model does not track. Every hash commits to its source chain, so such a child
        // can never touch A's hashes; the handler filters it, and nestedSends checks the filter is complete.
        (bytes4 dataSel,) = _split(data);
        if (
            nestedSendFilter && target == Predeploys.SUPERCHAIN_ETH_BRIDGE
                && dataSel == ISuperchainETHBridge.sendETH.selector
        ) {
            nNestedSendsFiltered++;
            return;
        }

        vm.chainId(CHAIN_A);
        uint256 nonce = messenger.messageNonce();
        vm.recordLogs();
        vm.prank(ATTACKER);
        try messenger.sendMessage(CHAIN_B, target, data) returns (bytes32 h_) {
            bytes memory payload = _sentMessagePayload(vm.getRecordedLogs());
            if (_isUnsafe(target)) unsafeTargetAccepted = true;
            attackerMsgs.push(
                Message({
                    destination: CHAIN_B,
                    nonce: nonce,
                    sender: ATTACKER,
                    target: target,
                    message: data,
                    hash: h_,
                    initTs: block.timestamp,
                    payload: payload,
                    from: address(0),
                    to: address(0),
                    amount: 0
                })
            );
            destOf[h_] = CHAIN_B;
            initTsOf[h_] = block.timestamp;
            nAttackerSends++;
        } catch {
            vm.getRecordedLogs();
            nAttackerSendsRejected++;
            return;
        }
        if (_relayNow) _relay(attackerMsgs[attackerMsgs.length - 1]);
    }

    /// @notice Destination: relay an earlier attacker message, only if the protocol accepts it.
    function attackerRelay(uint256 _j) public {
        if (attackerMsgs.length == 0) return;
        _relay(attackerMsgs[_j % attackerMsgs.length]);
    }

    /// @notice B: relay a payload whose target is the L2CrossDomainMessenger (even `_targetSel`) or the
    ///         L2ToL1MessagePasser (odd) as if it had an initiating event (which the send-side rule and the activation
    ///         requirement rule out). Checks the relay-side target rule on its own: the relay must revert.
    function relayForgedPayloadToUnsafeTarget(
        uint8 _targetSel,
        uint8 _dataSel,
        uint256 _i,
        uint256 _tSeed,
        uint32 _gas,
        bytes calldata _junk
    )
        public
    {
        bytes memory data = _attackData(_dataSel, _i, _tSeed, _gas, _junk);
        address target = _targetSel % 2 == 0 ? Predeploys.L2_CROSS_DOMAIN_MESSENGER : Predeploys.L2_TO_L1_MESSAGE_PASSER;
        vm.chainId(CHAIN_B);
        bytes memory payload = abi.encodePacked(
            abi.encode(L2TOL2_SENT_MESSAGE_SIG, CHAIN_B, target, uint256(type(uint240).max)), abi.encode(ATTACKER, data)
        );
        Identifier memory id = Identifier({
            origin: Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER,
            blockNumber: 1,
            logIndex: 0,
            timestamp: block.timestamp,
            chainId: CHAIN_A
        });
        vm.mockCall(
            Predeploys.CROSS_L2_INBOX, abi.encodeCall(ICrossL2Inbox.validateMessage, (id, keccak256(payload))), ""
        );
        vm.recordLogs();
        vm.prank(RELAYER);
        try messenger.relayMessage(id, payload) {
            unsafeTargetAccepted = true;
        } catch { }
        vm.clearMockedCalls();
        _capture(vm.getRecordedLogs(), false, CHAIN_B);
        nForgeAttempts++;
    }

    /// @notice Any chain: exportUndeliveredMessage for an ETH send or attacker message, on its destination or on
    ///         another chain, to A's L1CrossDomainMessenger (mostly) or a wrong L1 target, optionally with a wrong
    ///         source chain.
    function exportMessage(
        uint256 _i,
        uint8 _chainSel,
        uint8 _msgrSel,
        uint8 _mutSel,
        uint32 _gas,
        bool _attacker
    )
        public
    {
        bool useAttacker = attackerMsgs.length > 0 && (_attacker || ethSends.length == 0);
        if (!useAttacker && ethSends.length == 0) return;
        // Bias (chainSel >= 128): prefer an ETH send that is past P_contract, unrelayed and unexpired.
        if (!useAttacker && _chainSel >= 128) _i = _findExpirable(_i);
        _export(
            useAttacker ? attackerMsgs[_i % attackerMsgs.length] : ethSends[_i % ethSends.length],
            _chainSel,
            _exportTarget(_msgrSel, _i),
            _mutSel % 8 == 0 ? CHAIN_C : CHAIN_A,
            _gas,
            _actor(_i)
        );
    }

    /// @notice Index of the first ETH send from `_i` (cyclically) that is at or past init + P_contract (the boundary
    ///         is included so boundary mutants are reachable), unrelayed and unexpired, or `_i` if none.
    function _findExpirable(uint256 _i) internal view returns (uint256) {
        uint256 n = ethSends.length;
        _i = _i % n;
        for (uint256 j = 0; j < n; j++) {
            Message storage m = ethSends[(_i + j) % n];
            if (
                block.timestamp >= m.initTs + P_CONTRACT && !messenger.successfulMessages(m.hash)
                    && !messenger.expiredMessages(m.hash)
            ) return (_i + j) % n;
        }
        return _i;
    }

    /// @notice A's L1CrossDomainMessenger (7/8), the decoy, or a pseudo-random L1 address.
    function _exportTarget(uint8 _msgrSel, uint256 _i) internal view returns (address) {
        if (_msgrSel % 8 != 0) return aL1Messenger;
        if (_msgrSel % 16 == 0) return B_L1_MESSENGER;
        return address(uint160(uint256(keccak256(abi.encode(_msgrSel, _i)))));
    }

    function _export(
        Message storage _m,
        uint8 _chainSel,
        address _sourceMessenger,
        uint256 _source,
        uint32 _gas,
        address _caller
    )
        internal
    {
        // The message's destination (5/8), or B, C or A.
        uint256 c = _chainSel % 8;
        uint256 _x = c < 5 ? _m.destination : (c == 5 ? CHAIN_B : (c == 6 ? CHAIN_C : CHAIN_A));
        vm.chainId(_x);
        bytes32 hx = Hashing.hashL2toL2CrossDomainMessage(_x, _source, _m.nonce, _m.sender, _m.target, _m.message);
        bool relayedBefore = messenger.successfulMessages(hx);
        bytes memory data = abi.encodeCall(
            IUndeliveredMessageExporter.exportUndeliveredMessage,
            (_sourceMessenger, _source, _m.nonce, _m.sender, _m.target, _m.message, _gas)
        );

        vm.recordLogs();
        vm.prank(_caller);
        (bool ok, bytes memory ret) = EXPORTER.call(data);
        if (ok) {
            if (abi.decode(ret, (bytes32)) != hx) exportHashMismatch = true;
            if (relayedBefore) exportAfterRelay = true;
            _capture(vm.getRecordedLogs(), true, _x);
            nExports++;
        } else {
            _capture(vm.getRecordedLogs(), false, _x);
            nExportsReverted++;
        }
    }

    /// @notice A: deliver captured fact `_k` (or, for mode >= 128, the latest fact) through the real
    ///         L2CrossDomainMessenger. Modes 0, 1 and 3 (for an export fact) replace undeliveredAt with some t' <= t
    ///         (within 2 days of t, anywhere in [0, t], or exactly init + P_contract): a weaker statement, true
    ///         because successfulMessages only grows, so the destination could have exported it at t' too. Other
    ///         modes deliver t as exported.
    function deliverFact(uint256 _k, uint8 _mode, uint256 _tSeed) public {
        if (facts.length == 0) return;
        uint256 k = _mode >= 128 ? facts.length - 1 : _k % facts.length;
        Fact storage f = facts[k];
        (bool ok, bytes32 h, uint256 t) = _decodeRelayUndelivered(f.l1Target, f.l1Message);
        if (!ok) {
            nUndeliverable++;
            return;
        }
        if (f.fromExport && _mode % 8 == 0) {
            t = bound(_tSeed, t > 2 days ? t - 2 days : 0, t);
        } else if (f.fromExport && _mode % 8 == 1) {
            t = bound(_tSeed, 0, t);
        } else if (f.fromExport && _mode % 8 == 3 && initTsOf[h] != 0 && initTsOf[h] + P_CONTRACT <= t) {
            // Exactly at the expiry boundary (still a weaker, true statement): only `t > sentAt + P` rejects it.
            t = initTsOf[h] + P_CONTRACT;
        }

        vm.chainId(CHAIN_A);
        bool expiredBefore = messenger.expiredMessages(h);
        vm.recordLogs();
        _l2cdmRelay(aL1Messenger, abi.encodeCall(IL2ToL2CrossDomainMessenger.expireMessage, (h, t)));
        Vm.Log[] memory logs = vm.getRecordedLogs();
        nDeliveries++;

        // NoForgedFact / OnlyDestinationCanExport: every successful expireMessage execution (a MessageExpired
        // emission, including repeats after an earlier expiry) must run on a fact exported on the message's
        // destination.
        for (uint256 i = 0; i < logs.length; i++) {
            if (logs[i].emitter != address(messenger) || logs[i].topics[0] != MESSAGE_EXPIRED_SIG) continue;
            if (logs[i].topics[1] != h) unauthorizedExpiryEmitted = true;
            nExpiredEmissions++;
            if (!f.fromExport || f.chain != destOf[h]) forgedFactAccepted = true;
        }

        if (!expiredBefore && messenger.expiredMessages(h)) {
            expiredAt[h] = block.timestamp;
            expiredFactTime[h] = t;
            expiredFactIndexPlusOne[h] = k + 1;
            if (isEthSend[h]) expiredSends.push(h);
            nExpiries++;
        }
    }

    /// @notice A: try to expire a message by paths other than a fact from A's L1CrossDomainMessenger, with an
    ///         arbitrary undeliveredAt. Every path must fail.
    function forgeExpiry(uint8 _mode, uint256 _i, uint256 _tSeed, address _l1Sender, address _caller) public {
        if (ethSends.length == 0) return;
        Message storage m = ethSends[_i % ethSends.length];
        uint256 t = m.initTs + P_CONTRACT + 1 + (_tSeed % 365 days);
        if (_l1Sender == aL1Messenger) _l1Sender = B_L1_MESSENGER;
        bytes memory call_ = abi.encodeCall(IL2ToL2CrossDomainMessenger.expireMessage, (m.hash, t));

        vm.chainId(CHAIN_A);
        bool expiredBefore = messenger.expiredMessages(m.hash);
        vm.recordLogs();
        uint256 mode = _mode % 4;
        if (mode == 0) {
            // A's L1CrossDomainMessenger relays a message whose L1 sender is someone else.
            _l2cdmRelay(_l1Sender, call_);
        } else if (mode == 1) {
            // A deposit from another L1 contract straight to 0x..23.
            vm.prank(AddressAliasHelper.applyL1ToL2Alias(_l1Sender));
            try messenger.expireMessage(m.hash, t) { } catch { }
        } else if (mode == 2) {
            // Any L2 caller, including the L2CrossDomainMessenger outside of a relay.
            address caller = _caller == address(0) ? Predeploys.L2_CROSS_DOMAIN_MESSENGER : _caller;
            vm.prank(caller);
            try messenger.expireMessage(m.hash, t) { } catch { }
        } else {
            // Another L1 contract's deposit into the L2CrossDomainMessenger, claiming A's messenger as sender.
            vm.prank(AddressAliasHelper.applyL1ToL2Alias(_l1Sender));
            try l2cdm.relayMessage(
                Encoding.encodeVersionedNonce(++l1Nonce, 1),
                aL1Messenger,
                Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER,
                0,
                EXPIRE_MESSAGE_GAS_LIMIT,
                call_
            ) { }
                catch { }
        }
        if (!expiredBefore && messenger.expiredMessages(m.hash)) adversarialExpiryAccepted = true;
        // Any successful expireMessage execution on these paths, including a repeat on an expired hash.
        Vm.Log[] memory logs = vm.getRecordedLogs();
        for (uint256 i = 0; i < logs.length; i++) {
            if (logs[i].emitter == address(messenger) && logs[i].topics[0] == MESSAGE_EXPIRED_SIG) {
                adversarialExpiryAccepted = true;
            }
        }
        nForgeAttempts++;
    }

    /// @notice A: SuperchainETHBridge.refundETH for ETH send `_i` (or, for mutSel >= 128, an expired ETH send), with
    ///         the right preimage (6/10) or a wrong one.
    function refund(uint256 _i, uint8 _mutSel, uint256 _junk, address _caller) public {
        uint256 destination;
        uint256 nonce;
        address from;
        address to;
        uint256 amount;
        if (ethSends.length == 0) {
            destination = CHAIN_B;
            nonce = _junk;
            from = _actor(_junk);
            to = _recipient(_junk);
            amount = bound(_junk, 0, MAX_AMOUNT);
        } else {
            if (_mutSel >= 128 && expiredSends.length > 0) {
                _i = sendIndexPlusOne[expiredSends[_i % expiredSends.length]] - 1;
            }
            Message storage m = ethSends[_i % ethSends.length];
            (destination, nonce, from, to, amount) = (m.destination, m.nonce, m.from, m.to, m.amount);
            uint256 mut = _mutSel % 10;
            if (mut == 6) amount = amount + 1;
            else if (mut == 7) from = _actor(uint256(uint160(from)) + 1);
            else if (mut == 8) destination = destination == CHAIN_B ? CHAIN_C : CHAIN_B;
            else if (mut == 9) nonce = nonce + 1 + (_junk % 3);
        }

        vm.chainId(CHAIN_A);
        bytes32 h = Hashing.hashL2toL2CrossDomainMessage(
            destination,
            CHAIN_A,
            nonce,
            address(bridge),
            address(bridge),
            abi.encodeCall(ISuperchainETHBridge.relayETH, (from, to, amount))
        );
        bool expiredBefore = messenger.expiredMessages(h);
        uint256 balanceBefore = from.balance;
        uint256 bridgeBefore = address(bridge).balance;

        vm.prank(_caller);
        try bridge.refundETH(destination, nonce, from, to, amount) {
            if (address(bridge).balance != bridgeBefore) bridgeRetainedEth = true;
            if (blockedRelay[h]) refundAfterBlockedRelay = true;
            if (!expiredBefore) refundWithoutExpiry = true;
            if (!isEthSend[h]) refundOfUnknownHash = true;
            if (from.balance != balanceBefore + amount) refundPaidWrong = true;
            refundCount[h]++;
            refundedHashes.push(h);
            ghostRefunded += amount;
            nRefunds++;
        } catch {
            nRefundsReverted++;
        }
    }

    /// @notice NOT a fuzz target. Governance assumption witness: the destination's L2 governance upgrades its
    ///         exporter, which then sends relayUndeliveredMessage(H, _t) for any H, e.g. a relayed one. Modeled as the
    ///         trusted sender calling the L2CrossDomainMessenger directly on the send's destination.
    function forgeAsUpgradedExporter(uint256 _i, uint256 _t) public {
        if (ethSends.length == 0) return;
        Message storage m = ethSends[_i % ethSends.length];
        vm.chainId(m.destination);
        vm.recordLogs();
        vm.prank(TRUSTED_SENDER);
        l2cdm.sendMessage(
            aL1Messenger, abi.encodeCall(IL1CrossDomainMessenger.relayUndeliveredMessage, (m.hash, _t)), 200_000
        );
        _capture(vm.getRecordedLogs(), false, m.destination);
    }

    /// @notice NOT a fuzz target. Failing-witness helper: turns the nested-send filter off.
    function setNestedSendFilter(bool _on) public {
        nestedSendFilter = _on;
    }

    /// @notice NOT a fuzz target. Governance failing witness: an upgraded exporter calls the L2ToL1MessagePasser
    ///         directly (a raw withdrawal whose L2 sender is the trusted sender).
    function rawWithdrawalAsUpgradedExporter() public {
        vm.chainId(CHAIN_B);
        vm.recordLogs();
        vm.prank(TRUSTED_SENDER);
        IL2ToL1MessagePasser(payable(Predeploys.L2_TO_L1_MESSAGE_PASSER))
            .initiateWithdrawal(
                aL1Messenger, 200_000, abi.encodeCall(IL1CrossDomainMessenger.relayUndeliveredMessage, (bytes32(0), 0))
            );
        _capture(vm.getRecordedLogs(), false, CHAIN_B);
    }

    ////////////////////////////////////////////////////////////////
    //                         Getters                            //
    ////////////////////////////////////////////////////////////////

    function ethSendsLength() external view returns (uint256) {
        return ethSends.length;
    }

    function attackerMsgsLength() external view returns (uint256) {
        return attackerMsgs.length;
    }

    function factsLength() external view returns (uint256) {
        return facts.length;
    }

    function refundedHashesLength() external view returns (uint256) {
        return refundedHashes.length;
    }

    function refundedHash(uint256 _k) external view returns (bytes32) {
        return refundedHashes[_k];
    }

    /// @notice (hash, destination, initTs, amount) of ETH send `_i`.
    function ethSend(uint256 _i) external view returns (bytes32, uint256, uint256, uint256) {
        Message storage m = ethSends[_i];
        return (m.hash, m.destination, m.initTs, m.amount);
    }

    /// @notice (hash, initTs) of attacker message `_j`.
    function attackerMsg(uint256 _j) external view returns (bytes32, uint256) {
        Message storage m = attackerMsgs[_j];
        return (m.hash, m.initTs);
    }

    /// @notice (chain, fromExport, at) of fact `_k`.
    function fact(uint256 _k) external view returns (uint256, bool, uint256) {
        Fact storage f = facts[_k];
        return (f.chain, f.fromExport, f.at);
    }

    ////////////////////////////////////////////////////////////////
    //                         Internals                          //
    ////////////////////////////////////////////////////////////////

    /// @notice Relays `_m` on its destination iff the protocol accepts it (exec - init <= W_PROTOCOL).
    function _relay(Message storage _m) internal returns (bool ok_) {
        if (block.timestamp - _m.initTs > W_PROTOCOL) {
            blockedRelay[_m.hash] = true;
            nRelaysBlockedByWindow++;
            return false;
        }
        // ExpiredImpliesNeverRelayable: the protocol must already reject any relay of an expired message.
        if (messenger.expiredMessages(_m.hash)) relayableAfterExpiry = true;

        vm.chainId(_m.destination);
        Identifier memory id = Identifier({
            origin: Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER,
            blockNumber: 1,
            logIndex: 0,
            timestamp: _m.initTs,
            chainId: CHAIN_A
        });
        vm.mockCall(
            Predeploys.CROSS_L2_INBOX, abi.encodeCall(ICrossL2Inbox.validateMessage, (id, keccak256(_m.payload))), ""
        );
        bool expiredBefore = messenger.expiredMessages(_m.hash);
        vm.recordLogs();
        vm.prank(RELAYER);
        try messenger.relayMessage(id, _m.payload) {
            ok_ = true;
        } catch { }
        vm.clearMockedCalls();
        // A relayed message whose target is the exporter, calling exportUndeliveredMessage, is an export call.
        (bytes4 sel,) = _split(_m.message);
        _capture(
            vm.getRecordedLogs(),
            _m.target == EXPORTER && sel == IUndeliveredMessageExporter.exportUndeliveredMessage.selector,
            _m.destination
        );

        if (ok_) {
            relayCount[_m.hash]++;
            if (expiredBefore) relayedWhileExpired = true;
            if (_isUnsafe(_m.target)) unsafeTargetAccepted = true;
            if (_m.sender == ATTACKER) nAttackerRelays++;
            else nRelays++;
        } else {
            nRelaysReverted++;
        }
    }

    /// @notice Runs the real L2CrossDomainMessenger.relayMessage as a deposit from A's L1CrossDomainMessenger
    ///         (msg.sender = its alias), with L1 sender `_l1Sender`, to 0x..23.
    function _l2cdmRelay(address _l1Sender, bytes memory _message) internal {
        vm.prank(AddressAliasHelper.applyL1ToL2Alias(aL1Messenger));
        l2cdm.relayMessage(
            Encoding.encodeVersionedNonce(++l1Nonce, 1),
            _l1Sender,
            Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER,
            0,
            EXPIRE_MESSAGE_GAS_LIMIT,
            _message
        );
    }

    /// @notice Captures withdrawals from MessagePassed logs. A withdrawal by the L2CrossDomainMessenger whose inner
    ///         (relayMessage) sender is TRUSTED_SENDER becomes a fact; raw withdrawals from 0x..23 or TRUSTED_SENDER
    ///         are counted.
    function _capture(Vm.Log[] memory _logs, bool _fromExport, uint256 _chain) internal {
        for (uint256 i = 0; i < _logs.length; i++) {
            Vm.Log memory l = _logs[i];
            if (l.emitter == Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER && l.topics.length > 0) {
                if (l.topics[0] == L2TOL2_SENT_MESSAGE_SIG) nestedSends++;
                // Expiry happens only through deliverFact, whose logs are not captured here.
                if (l.topics[0] == MESSAGE_EXPIRED_SIG) unauthorizedExpiryEmitted = true;
            }
            if (l.emitter != Predeploys.L2_TO_L1_MESSAGE_PASSER || l.topics.length != 4) continue;
            if (l.topics[0] != MESSAGE_PASSED_SIG) continue;
            address sender = address(uint160(uint256(l.topics[2])));
            if (sender == Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER || sender == TRUSTED_SENDER) {
                if (sender == Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER) rawWithdrawalsFrom23++;
                if (sender == TRUSTED_SENDER) rawWithdrawalsFromTrusted++;
                continue;
            }
            if (sender != Predeploys.L2_CROSS_DOMAIN_MESSENGER) continue;
            (,, bytes memory data,) = abi.decode(l.data, (uint256, uint256, bytes, bytes32));
            (bytes4 sel, bytes memory args) = _split(data);
            if (sel != ICrossDomainMessenger.relayMessage.selector) continue;
            (, address innerSender, address innerTarget,,, bytes memory innerMessage) =
                abi.decode(args, (uint256, address, address, uint256, uint256, bytes));
            if (innerSender != TRUSTED_SENDER) {
                if (innerSender == Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER) untrustedWithdrawalsFrom23++;
                continue;
            }
            _checkHonest(innerMessage, _chain);
            facts.push(
                Fact({
                    chain: _chain,
                    l1Target: innerTarget,
                    l1Message: innerMessage,
                    fromExport: _fromExport,
                    at: block.timestamp
                })
            );
            if (!_fromExport) nonExportFacts++;
        }
    }

    /// @notice Flags a trusted-sender relayUndeliveredMessage(H, t) that no export on `_chain` now could produce.
    function _checkHonest(bytes memory _message, uint256 _chain) internal {
        (bytes4 sel, bytes memory args) = _split(_message);
        if (sel != IL1CrossDomainMessenger.relayUndeliveredMessage.selector || args.length != 64) {
            dishonestFactCaptured = true;
            return;
        }
        (bytes32 h, uint256 t) = abi.decode(args, (bytes32, uint256));
        if (t != block.timestamp || messenger.successfulMessages(h) || (destOf[h] != 0 && destOf[h] != _chain)) {
            dishonestFactCaptured = true;
        }
    }

    /// @notice Decodes relayUndeliveredMessage(H, t) sent to A's L1CrossDomainMessenger; anything else is not a fact
    ///         A's L1CrossDomainMessenger turns into an expireMessage deposit.
    function _decodeRelayUndelivered(
        address _l1Target,
        bytes memory _message
    )
        internal
        view
        returns (bool ok_, bytes32 h_, uint256 t_)
    {
        if (_l1Target != aL1Messenger || _message.length != 68) return (false, 0, 0);
        (bytes4 sel, bytes memory args) = _split(_message);
        if (sel != IL1CrossDomainMessenger.relayUndeliveredMessage.selector) return (false, 0, 0);
        (h_, t_) = abi.decode(args, (bytes32, uint256));
        ok_ = true;
    }

    /// @notice Builds the SentMessage payload (topics ++ data) from the real messenger's log.
    function _sentMessagePayload(Vm.Log[] memory _logs) internal pure returns (bytes memory) {
        for (uint256 i = 0; i < _logs.length; i++) {
            Vm.Log memory l = _logs[i];
            if (l.emitter == Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER && l.topics[0] == L2TOL2_SENT_MESSAGE_SIG) {
                return abi.encodePacked(l.topics[0], l.topics[1], l.topics[2], l.topics[3], l.data);
            }
        }
        revert ExpiryHandler_NoSentMessageLog();
    }

    /// @notice Splits calldata into selector and arguments.
    function _split(bytes memory _data) internal pure returns (bytes4 sel_, bytes memory args_) {
        if (_data.length < 4) return (bytes4(0), "");
        sel_ = bytes4(_data[0]) | (bytes4(_data[1]) >> 8) | (bytes4(_data[2]) >> 16) | (bytes4(_data[3]) >> 24);
        args_ = new bytes(_data.length - 4);
        for (uint256 i = 0; i < args_.length; i++) {
            args_[i] = _data[i + 4];
        }
    }

    /// @notice The messenger's unsafe targets (L2CrossDomainMessenger and L2ToL1MessagePasser).
    function _isUnsafe(address _target) internal pure returns (bool) {
        return _target == Predeploys.L2_CROSS_DOMAIN_MESSENGER || _target == Predeploys.L2_TO_L1_MESSAGE_PASSER;
    }

    function _actor(uint256 _seed) internal pure returns (address) {
        return address(uint160(0x10000 + (_seed % 4)));
    }

    function _recipient(uint256 _seed) internal pure returns (address) {
        if (_seed % 2 == 0) return _actor(_seed / 2);
        return address(uint160(uint256(keccak256(abi.encode("recipient", (_seed / 2) % 16)))));
    }

    function _attackTarget(uint8 _sel, bytes calldata _junk) internal view returns (address) {
        uint256 s = _sel % 8;
        if (s == 7) return EXPORTER;
        if (s == 0) return Predeploys.L2_CROSS_DOMAIN_MESSENGER;
        if (s == 1) return Predeploys.L2_TO_L1_MESSAGE_PASSER;
        if (s == 2) return Predeploys.SUPERCHAIN_ETH_BRIDGE;
        if (s == 3) return Predeploys.ETH_LIQUIDITY;
        if (s == 4) return Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER;
        if (s == 5) return aL1Messenger;
        return address(uint160(uint256(keccak256(_junk))));
    }

    /// @notice Attacker calldata. H is an ETH send's hash (or junk), t is past its expiry or arbitrary.
    function _attackData(
        uint8 _sel,
        uint256 _i,
        uint256 _tSeed,
        uint32 _gas,
        bytes calldata _junk
    )
        internal
        view
        returns (bytes memory)
    {
        bytes32 h = keccak256(_junk);
        uint256 t = _tSeed;
        Message memory m;
        if (ethSends.length > 0) {
            m = ethSends[_i % ethSends.length];
            h = m.hash;
            if (_tSeed % 2 == 0) t = m.initTs + P_CONTRACT + 1 + (_tSeed % 30 days);
        }
        bytes memory fact_ = abi.encodeCall(IL1CrossDomainMessenger.relayUndeliveredMessage, (h, t));
        uint256 s = _sel % 7;
        if (s == 0) return abi.encodeCall(ICrossDomainMessenger.sendMessage, (aL1Messenger, fact_, _gas));
        if (s == 1) return abi.encodeCall(IL2ToL1MessagePasser.initiateWithdrawal, (aL1Messenger, _gas, fact_));
        if (s == 2) {
            bytes memory relay = abi.encodeCall(
                ICrossDomainMessenger.relayMessage,
                (
                    Encoding.encodeVersionedNonce(0, 1),
                    Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER,
                    aL1Messenger,
                    0,
                    _gas,
                    fact_
                )
            );
            return abi.encodeCall(IL2ToL1MessagePasser.initiateWithdrawal, (B_L1_MESSENGER, _gas, relay));
        }
        if (s == 3) {
            return abi.encodeCall(ISuperchainETHBridge.relayETH, (m.from, ATTACKER, m.amount));
        }
        if (s == 4) return abi.encodeCall(IL2ToL2CrossDomainMessenger.expireMessage, (h, t));
        if (s == 5) {
            return abi.encodeCall(
                IUndeliveredMessageExporter.exportUndeliveredMessage,
                (aL1Messenger, CHAIN_A, m.nonce, m.sender, m.target, m.message, _gas)
            );
        }
        return _junk.length > MAX_JUNK ? _junk[:MAX_JUNK] : _junk;
    }
}
