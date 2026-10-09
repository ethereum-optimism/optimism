// SPDX-License-Identifier: MIT
pragma solidity 0.8.25;

// Halmos symbolic checks on the REAL L2ToL2CrossDomainMessenger (src/L2/L2ToL2CrossDomainMessenger.sol) at
// the exporter design, etched at its predeploy address 0x4200..0023, with the REAL
// UndeliveredMessageExporter at Predeploys.UNDELIVERED_MESSAGE_EXPORTER. Exact statements, assumptions and bounds:
// README.md in this directory.
//   (1) UnsafeTargetRule      sendMessage / relayMessage never succeed for target 0x..07 or 0x..16 (and 0x..23 on
// send). (2) OnlyExportReachesL1   0x..23 never calls 0x..07 or 0x..16 (send, relay, re-entrant relay); an export made
//                             re-entrantly during a relay carries exactly the export payload for an unrelayed hash.
//   (5) expireMessage         auth + exact boundary at EXPIRY_PERIOD, as an iff, plus full storage frame.
//   (+) Storage effects/frames of sendMessage and relayMessage; relay delivery, value, context, failure.
//   The exporter itself (group 3) is checked in ExporterExpiryHalmos.t.sol.
//
// Symbolic inputs: every check_ parameter (block.chainid / block.timestamp via vm.chainId / vm.warp). `bytes`
// parameters take each length in --default-bytes-lengths (run.sh: 0,1,32,33,100,132,260).

import { Test } from "test/setup/Test.sol";
import { DeployUtils } from "scripts/libraries/DeployUtils.sol";
import { L2ToL2CrossDomainMessenger } from "src/L2/L2ToL2CrossDomainMessenger.sol";
import { Constants } from "src/libraries/Constants.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";
import { Identifier } from "interfaces/L2/ICrossL2Inbox.sol";
import { ICrossDomainMessenger } from "interfaces/universal/ICrossDomainMessenger.sol";
import { IL1CrossDomainMessenger } from "interfaces/L1/IL1CrossDomainMessenger.sol";
import {
    SVM,
    SVM_ADDRESS,
    MockCrossL2Inbox,
    Recorder,
    MockL2CDMGetters,
    ReentrantTarget,
    RelayProbe
} from "./HalmosMocks.sol";

contract L2ToL2ExpiryHalmos is Test {
    SVM internal constant svm = SVM(SVM_ADDRESS);

    address internal constant L2_TO_L2 = Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER; // 0x..23
    address internal constant L2CDM = Predeploys.L2_CROSS_DOMAIN_MESSENGER; // 0x..07
    address internal constant PASSER = Predeploys.L2_TO_L1_MESSAGE_PASSER; // 0x..16
    address internal constant INBOX = Predeploys.CROSS_L2_INBOX; // 0x..22

    L2ToL2CrossDomainMessenger internal m = L2ToL2CrossDomainMessenger(L2_TO_L2);

    /// @notice Template deployments whose code is etched at the predeploys. They (and the test contract and the
    /// cheatcode addresses) are harness accounts, not chain accounts, so symbolic relay targets are assumed to avoid
    /// them.
    address[5] internal templates;
    address internal constant EXPORTER = Predeploys.UNDELIVERED_MESSAGE_EXPORTER;

    function setUp() public {
        templates[0] = address(new L2ToL2CrossDomainMessenger(Constants.L2_TO_L2_MESSAGE_EXPIRY_PERIOD));
        templates[1] = address(new MockCrossL2Inbox());
        templates[2] = address(new Recorder());
        templates[3] = address(new Recorder());
        // The real UndeliveredMessageExporter (solc 0.8.15, compiled by ExporterExpiryHalmos.t.sol).
        bytes memory code = DeployUtils.getCode(
            "test/formal/expiry/halmos/out/UndeliveredMessageExporter.sol/UndeliveredMessageExporter.json"
        );
        address exporter;
        assembly {
            exporter := create(0, add(code, 32), mload(code))
        }
        templates[4] = exporter;
        vm.etch(L2_TO_L2, templates[0].code);
        vm.etch(INBOX, templates[1].code);
        vm.etch(L2CDM, templates[2].code);
        vm.etch(PASSER, templates[3].code);
        vm.etch(EXPORTER, templates[4].code);
    }

    /// @notice ASSUMPTION for symbolic relay targets: the target is not a harness account. It may still be any of the
    ///         predeploys above (0x..22, 0x..07, 0x..16) or any account without code. 0x..23 is excluded separately
    ///         (see check_UnsafeTargetRule_relay).
    function _assumeNotHarness(address _target) internal view {
        vm.assume(_target != address(this));
        vm.assume(_target != address(vm));
        vm.assume(_target != SVM_ADDRESS);
        vm.assume(_target != 0x000000000000000000636F6e736F6c652e6c6f67); // console.log
        vm.assume(_target != CREATE2_FACTORY);
        for (uint256 i = 0; i < 5; i++) {
            vm.assume(_target != templates[i]);
        }
        // The exporter ignores msg.sender, so a relay that calls it is an export with caller 0x..23, covered by
        // ExporterExpiryHalmos (symbolic caller) and by check_OnlyExportReachesL1_relay_reentrant. Calling it here with
        // symbolic calldata would only make halmos stuck on symbolic ABI offsets.
        vm.assume(_target != EXPORTER);
    }

    // ---------------------------------------------------------------- helpers

    function _env(uint256 _chainId, uint256 _ts) internal {
        vm.chainId(_chainId);
        vm.warp(_ts);
    }

    function _callsFrom23(address _recorder) internal view returns (uint256) {
        return uint256(vm.load(_recorder, bytes32(uint256(0))));
    }

    function _lastHashFrom23(address _recorder) internal view returns (bytes32) {
        return vm.load(_recorder, bytes32(uint256(2)));
    }

    function _callsFromExporter(address _recorder) internal view returns (uint256) {
        return uint256(vm.load(_recorder, bytes32(uint256(4))));
    }

    function _lastHashFromExporter(address _recorder) internal view returns (bytes32) {
        return vm.load(_recorder, bytes32(uint256(5)));
    }

    /// @notice Symbolic probe keys for storage-frame assertions: `k` is an arbitrary message hash, `j` an arbitrary
    /// nonce. Because the messenger storage is fully symbolic and k, j are unconstrained (except where a check excludes
    ///         the slot it legitimately writes), "unchanged at k / j" means unchanged at EVERY key.
    struct FrameKeys {
        bytes32 k;
        uint256 j;
    }

    /// @notice Everything in messenger storage, observed at the probe keys (mappings) plus the nonce.
    struct Snap {
        uint256 nonce;
        bytes32 sentJ;
        uint256 tsK;
        bool succK;
        bool expK;
    }

    function _snap(FrameKeys memory _fk) internal view returns (Snap memory s_) {
        s_.nonce = m.messageNonce();
        s_.sentJ = m.sentMessages(_fk.j);
        s_.tsK = m.sentMessageTimestamps(_fk.k);
        s_.succK = m.successfulMessages(_fk.k);
        s_.expK = m.expiredMessages(_fk.k);
    }

    function _sameExceptNothing(Snap memory _a, Snap memory _b) internal pure {
        assert(_a.nonce == _b.nonce);
        assert(_a.sentJ == _b.sentJ);
        assert(_a.tsK == _b.tsK);
        assert(_a.succK == _b.succK);
        assert(_a.expK == _b.expK);
    }

    function _hash(
        uint256 _destination,
        uint256 _source,
        uint256 _nonce,
        address _sender,
        address _target,
        bytes memory _message
    )
        internal
        pure
        returns (bytes32)
    {
        // Written out here (not via Hashing) so the check states the formula independently.
        return keccak256(abi.encode(_destination, _source, _nonce, _sender, _target, _message));
    }

    /// @notice A well-formed SentMessage payload: abi.encode(selector, destination, target, nonce) ++
    ///         abi.encode(sender, message). Malformed payloads are not explored (the decoder reverts on them).
    function _payload(
        uint256 _destination,
        address _target,
        uint256 _nonce,
        address _sender,
        bytes memory _message
    )
        internal
        pure
        returns (bytes memory)
    {
        return abi.encodePacked(
            abi.encode(L2ToL2CrossDomainMessenger.SentMessage.selector, _destination, _target, _nonce),
            abi.encode(_sender, _message)
        );
    }

    function _send(
        address _caller,
        uint256 _destination,
        address _target,
        bytes memory _message
    )
        internal
        returns (bool ok_, bytes memory ret_)
    {
        vm.prank(_caller);
        (ok_, ret_) = L2_TO_L2.call(abi.encodeCall(m.sendMessage, (_destination, _target, _message)));
    }

    function _relay(
        address _caller,
        Identifier memory _id,
        uint256 _destination,
        address _target,
        uint256 _nonce,
        address _sender,
        bytes memory _message
    )
        internal
        returns (bool ok_)
    {
        bytes memory payload = _payload(_destination, _target, _nonce, _sender, _message);
        vm.prank(_caller);
        (ok_,) = L2_TO_L2.call(abi.encodeCall(m.relayMessage, (_id, payload)));
    }

    // ================================================================ (1) UnsafeTargetRule

    /// @notice sendMessage succeeds only if target is none of 0x..23, 0x..07, 0x..16 and destination != chainid.
    ///         Symbolic: caller, destination, target, message, chainid, timestamp, all messenger storage.
    function check_UnsafeTargetRule_send(
        address _caller,
        uint256 _chainId,
        uint256 _ts,
        uint256 _destination,
        address _target,
        bytes calldata _message
    )
        public
    {
        _env(_chainId, _ts);
        svm.enableSymbolicStorage(L2_TO_L2);
        (bool ok,) = _send(_caller, _destination, _target, _message);
        if (ok) {
            assert(_target != L2_TO_L2);
            assert(_target != L2CDM);
            assert(_target != PASSER);
            assert(_destination != _chainId);
        }
    }

    /// @notice Rule in the exporter design (pending in the earlier design): sendMessage to the L2ToL1MessagePasser
    /// reverts.
    function check_UnsafeTargetRule_send_passer(
        address _caller,
        uint256 _chainId,
        uint256 _destination,
        bytes calldata _message
    )
        public
    {
        vm.chainId(_chainId);
        (bool ok,) = _send(_caller, _destination, PASSER, _message);
        assert(!ok);
    }

    /// @notice NON-VACUITY (expected FAIL): sendMessage never succeeds. The counterexample is a successful send.
    function check_FALSE_send_neverSucceeds(
        address _caller,
        uint256 _chainId,
        uint256 _destination,
        address _target,
        bytes calldata _message
    )
        public
    {
        vm.chainId(_chainId);
        (bool ok,) = _send(_caller, _destination, _target, _message);
        assert(!ok);
    }

    /// @notice relayMessage succeeds only if target != 0x..07, the identifier origin is 0x..23 and the payload's
    ///         destination is this chain. Symbolic: caller, identifier, destination, target, nonce, sender, message,
    ///         chainid. Target 0x..23 is excluded here only because a call into 0x..23 with symbolic calldata
    ///         dispatches into every messenger function; sendMessage on every chain rejects target 0x..23, so no
    ///         valid SentMessage payload carries it (assumption: every source chain runs the real messenger).
    function check_UnsafeTargetRule_relay(
        address _caller,
        uint256 _chainId,
        Identifier memory _id,
        uint256 _destination,
        address _target,
        uint256 _nonce,
        address _sender,
        bytes calldata _message
    )
        public
    {
        vm.assume(_target != L2_TO_L2);
        _assumeNotHarness(_target);
        vm.chainId(_chainId);
        svm.enableSymbolicStorage(L2_TO_L2);
        bool ok = _relay(_caller, _id, _destination, _target, _nonce, _sender, _message);
        if (ok) {
            assert(_target != L2CDM);
            assert(_target != PASSER);
            assert(_id.origin == L2_TO_L2);
            assert(_destination == _chainId);
        }
    }

    /// @notice Same as above for target == 0x..07 exactly, with fully symbolic messenger storage: always reverts.
    function check_UnsafeTargetRule_relay_l2cdm(
        address _caller,
        uint256 _chainId,
        Identifier memory _id,
        uint256 _nonce,
        address _sender,
        bytes calldata _message
    )
        public
    {
        vm.chainId(_chainId);
        svm.enableSymbolicStorage(L2_TO_L2);
        bool ok = _relay(_caller, _id, _chainId, L2CDM, _nonce, _sender, _message);
        assert(!ok);
    }

    /// @notice Rule in the exporter design (pending in the earlier design): relayMessage to the L2ToL1MessagePasser
    /// reverts.
    function check_UnsafeTargetRule_relay_passer(
        address _caller,
        uint256 _chainId,
        Identifier memory _id,
        uint256 _nonce,
        address _sender,
        bytes calldata _message
    )
        public
    {
        vm.chainId(_chainId);
        bool ok = _relay(_caller, _id, _chainId, PASSER, _nonce, _sender, _message);
        assert(!ok);
    }

    /// @notice NON-VACUITY (expected FAIL): relayMessage never succeeds.
    function check_FALSE_relay_neverSucceeds(
        address _caller,
        uint256 _chainId,
        Identifier memory _id,
        uint256 _destination,
        address _target,
        uint256 _nonce,
        address _sender,
        bytes calldata _message
    )
        public
    {
        vm.assume(_target != L2_TO_L2);
        _assumeNotHarness(_target);
        vm.chainId(_chainId);
        bool ok = _relay(_caller, _id, _destination, _target, _nonce, _sender, _message);
        assert(!ok);
    }

    // ================================================================ (2) OnlyExportReachesL1

    /// @notice sendMessage (any outcome) makes 0x..23 call neither 0x..07 nor 0x..16.
    function check_OnlyExportReachesL1_send(
        address _caller,
        uint256 _chainId,
        uint256 _destination,
        address _target,
        bytes calldata _message
    )
        public
    {
        vm.chainId(_chainId);
        svm.enableSymbolicStorage(L2_TO_L2);
        _send(_caller, _destination, _target, _message);
        assert(_callsFrom23(L2CDM) == 0);
        assert(_callsFrom23(PASSER) == 0);
    }

    /// @notice relayMessage (any outcome) to a NON-re-entering target (any account without code or a predeploy mock;
    ///         not 0x..23, see check_UnsafeTargetRule_relay) makes 0x..23 call the L2CrossDomainMessenger zero times.
    ///         Re-entering targets: see check_OnlyExportReachesL1_relay_reentrant.
    function check_OnlyExportReachesL1_relay_l2cdm(
        address _caller,
        uint256 _chainId,
        Identifier memory _id,
        uint256 _destination,
        address _target,
        uint256 _nonce,
        address _sender,
        bytes calldata _message
    )
        public
    {
        vm.assume(_target != L2_TO_L2);
        _assumeNotHarness(_target);
        vm.chainId(_chainId);
        svm.enableSymbolicStorage(L2_TO_L2);
        _relay(_caller, _id, _destination, _target, _nonce, _sender, _message);
        assert(_callsFrom23(L2CDM) == 0);
    }

    /// @notice Rule in the exporter design (pending in the earlier design): relayMessage never makes 0x..23 call the
    ///         L2ToL1MessagePasser.
    function check_OnlyExportReachesL1_relay_passer(
        address _caller,
        uint256 _chainId,
        Identifier memory _id,
        uint256 _destination,
        address _target,
        uint256 _nonce,
        address _sender,
        bytes calldata _message
    )
        public
    {
        vm.assume(_target != L2_TO_L2);
        _assumeNotHarness(_target);
        vm.chainId(_chainId);
        _relay(_caller, _id, _destination, _target, _nonce, _sender, _message);
        assert(_callsFrom23(PASSER) == 0);
    }

    // ================================================================ export arguments (the exporter is checked in
    // ExporterExpiryHalmos)

    /// @notice Symbolic arguments of an exportUndeliveredMessage call (used by the re-entrant relay target).
    struct ExportArgs {
        address sourceMessenger;
        uint256 source;
        uint256 nonce;
        address sender;
        address target;
        uint32 minGas;
    }

    // ================================================================ storage effects and frames (send, relay)

    /// @notice sendMessage, from fully symbolic storage. On success, with n = messageNonce() before the call and
    ///         H = keccak256(abi.encode(destination, block.chainid, n, msg.sender, target, message)):
    ///           returns H; sentMessageTimestamps[H] == block.timestamp; sentMessages[n] == H; messageNonce() == n + 1;
    ///           and for every k != H and j != n: sentMessageTimestamps[k], sentMessages[j] unchanged; for every k:
    ///           successfulMessages[k], expiredMessages[k] unchanged.
    ///         On revert: nothing changes (all observations at H, n, k, j).
    function check_send_effects_and_frame(
        address _caller,
        uint256 _chainId,
        uint256 _ts,
        uint256 _destination,
        address _target,
        bytes calldata _message,
        FrameKeys memory _fk
    )
        public
    {
        _env(_chainId, _ts);
        svm.enableSymbolicStorage(L2_TO_L2);
        uint256 n = m.messageNonce();
        bytes32 h = _hash(_destination, _chainId, n, _caller, _target, _message);
        Snap memory atH = _snap(FrameKeys(h, n));
        Snap memory before = _snap(_fk);

        (bool ok, bytes memory ret) = _send(_caller, _destination, _target, _message);

        Snap memory after_ = _snap(_fk);
        if (ok) {
            assert(abi.decode(ret, (bytes32)) == h);
            assert(m.sentMessageTimestamps(h) == _ts);
            assert(m.sentMessages(n) == h);
            assert(after_.nonce == n + 1);
            assert(after_.succK == before.succK);
            assert(after_.expK == before.expK);
            assert(m.successfulMessages(h) == atH.succK);
            assert(m.expiredMessages(h) == atH.expK);
            if (_fk.k != h) assert(after_.tsK == before.tsK);
            if (_fk.j != n) assert(after_.sentJ == before.sentJ);
        } else {
            _sameExceptNothing(before, after_);
            _sameExceptNothing(atH, _snap(FrameKeys(h, n)));
        }
    }

    /// @notice relayMessage, from fully symbolic storage, to a target that does not re-enter 0x..23 (any account
    ///         without code, or the predeploy mocks). On success, with H = keccak256(abi.encode(block.chainid,
    ///         id.chainId, nonce, sender, target, message)): successfulMessages[H] becomes true (and was false before);
    ///         for every k != H successfulMessages[k] is unchanged; for every k, j: nonce, sentMessages[j],
    ///         sentMessageTimestamps[k], expiredMessages[k] unchanged. On revert: nothing changes.
    ///         Re-entering targets are covered by check_OnlyExportReachesL1_relay_reentrant.
    function check_relay_effects_and_frame(
        address _caller,
        uint256 _chainId,
        Identifier memory _id,
        address _target,
        uint256 _nonce,
        address _sender,
        bytes calldata _message,
        FrameKeys memory _fk
    )
        public
    {
        vm.assume(_target != L2_TO_L2);
        _assumeNotHarness(_target);
        vm.chainId(_chainId);
        svm.enableSymbolicStorage(L2_TO_L2);
        bytes32 h = _hash(_chainId, _id.chainId, _nonce, _sender, _target, _message);
        bool succHBefore = m.successfulMessages(h);
        Snap memory before = _snap(_fk);

        bool ok = _relay(_caller, _id, _chainId, _target, _nonce, _sender, _message);

        Snap memory after_ = _snap(_fk);
        assert(after_.nonce == before.nonce);
        assert(after_.sentJ == before.sentJ);
        assert(after_.tsK == before.tsK);
        assert(after_.expK == before.expK);
        if (ok) {
            assert(!succHBefore);
            assert(m.successfulMessages(h));
            if (_fk.k != h) assert(after_.succK == before.succK);
        } else {
            assert(after_.succK == before.succK);
            assert(m.successfulMessages(h) == succHBefore);
        }
    }

    /// @notice keccak256 of the exact calldata exportUndeliveredMessage must send to the L2CrossDomainMessenger.
    function _exportPayloadHash(address _sm, bytes32 _h, uint256 _ts, uint32 _minGas) internal pure returns (bytes32) {
        return keccak256(
            abi.encodeCall(
                ICrossDomainMessenger.sendMessage,
                (_sm, abi.encodeCall(IL1CrossDomainMessenger.relayUndeliveredMessage, (_h, _ts)), _minGas)
            )
        );
    }

    /// @notice Relays (from this contract) message `_message` to `_t` with sender = low 160 bits of _rel.k and nonce
    ///         _rel.j, destination = block.chainid; returns the hash of the relayed message.
    function _relayTo(
        Identifier memory _id,
        FrameKeys memory _rel,
        address _t,
        bytes calldata _message
    )
        internal
        returns (bytes32 h_)
    {
        address sender = address(uint160(uint256(_rel.k)));
        h_ = _hash(block.chainid, _id.chainId, _rel.j, sender, _t, _message);
        _relay(address(this), _id, block.chainid, _t, _rel.j, sender, _message);
    }

    function _reentryArgs(uint256 _mode, ExportArgs memory _a2) internal pure returns (ReentrantTarget.Args memory) {
        return ReentrantTarget.Args({
            mode: _mode,
            sourceMessenger: _a2.sourceMessenger,
            source: _a2.source,
            nonce: _a2.nonce,
            sender: _a2.sender,
            target: _a2.target,
            minGas: _a2.minGas,
            destination: _a2.source
        });
    }

    /// @notice Deploys a ReentrantTarget armed with the given symbolic export/send arguments; returns it and the hash
    /// an export with those arguments computes (destination = block.chainid).
    function _armReentrant(
        uint256 _mode,
        ExportArgs memory _a2,
        bytes calldata _message2
    )
        internal
        returns (address t_, bytes32 h2_)
    {
        t_ = address(new ReentrantTarget(_reentryArgs(_mode, _a2), _message2));
        h2_ = _hash(block.chainid, _a2.source, _a2.nonce, _a2.sender, _a2.target, _message2);
    }

    /// @notice OnlyExportReachesL1 under RE-ENTRY: relay to a target with code that, during the relayed call, calls
    ///         exportUndeliveredMessage on the exporter (mode 1), sendMessage on 0x..23 (mode 2), or a NESTED
    /// relayMessage of another well-formed message (mode 3; it must never succeed: the reentrancy guard), with symbolic
    ///         arguments. Every call
    ///         0x..23 makes to 0x..07 in this whole execution (here: at most one) has exactly the export payload
    ///         sendMessage(sm', relayUndeliveredMessage(H', block.timestamp), g') for H' = keccak256(abi.encode(
    ///         block.chainid, src', nonce', sender', target', msg')), and H' was not relayed at that time
    ///         (H' != the hash being relayed, and successfulMessages[H'] was false before the relay). No call to
    ///         0x..16. (Committed calls only: a call followed by a revert leaves no trace in the recorder; such a call
    ///         has no effect either.)
    function check_OnlyExportReachesL1_relay_reentrant(
        uint256 _chainId,
        uint256 _ts,
        Identifier memory _id,
        FrameKeys memory _rel, // relayed message: k = sender (low 160 bits), j = nonce
        bytes calldata _message,
        uint8 _modeSel,
        ExportArgs memory _a2,
        bytes calldata _message2
    )
        public
    {
        _env(_chainId, _ts);
        svm.enableSymbolicStorage(L2_TO_L2);
        // mode: 1 export, 2 send, 3 nested relay
        (address t, bytes32 h2) = _armReentrant(1 + uint256(_modeSel) % 3, _a2, _message2);
        bool succH2Before = m.successfulMessages(h2);

        bool h2IsRelayed = h2 == _relayTo(_id, _rel, t, _message);

        assert(_callsFrom23(L2CDM) == 0); // 0x..23 itself never calls the L2CrossDomainMessenger
        assert(_callsFromExporter(L2CDM) <= 1);
        if (_callsFromExporter(L2CDM) == 1) {
            assert(uint256(_modeSel) % 3 == 0); // mode 1 (export)
            assert(!h2IsRelayed && !succH2Before);
            assert(_lastHashFromExporter(L2CDM) == _exportPayloadHash(_a2.sourceMessenger, h2, _ts, _a2.minGas));
        }
        assert(_callsFrom23(PASSER) == 0 && _callsFromExporter(PASSER) == 0);
        assert(uint256(vm.load(t, bytes32(0))) == 0); // a nested relayMessage (mode 3) never succeeds
    }

    /// @notice NON-VACUITY (expected FAIL): a re-entrant export during a relay never reaches 0x..07. The
    ///         counterexample is a successful relay whose target exports another, unrelayed hash.
    function check_FALSE_relay_reentrantExportNeverReachesL2CDM(
        uint256 _chainId,
        Identifier memory _id,
        uint256 _nonce,
        address _sender,
        ExportArgs memory _a2
    )
        public
    {
        vm.chainId(_chainId);
        ReentrantTarget t = new ReentrantTarget(_reentryArgs(1, _a2), "");
        _relay(address(this), _id, _chainId, address(t), _nonce, _sender, "");
        assert(_callsFromExporter(L2CDM) == 0);
    }

    // ================================================================ relay delivery, value, context, failure

    function _probe(address _p, uint256 _slot) internal view returns (uint256) {
        return uint256(vm.load(_p, bytes32(_slot)));
    }

    function _relayWithValue(
        Identifier memory _id,
        FrameKeys memory _rel,
        address _t,
        bytes calldata _message,
        uint256 _v
    )
        internal
        returns (bool ok_, bytes32 h_)
    {
        address sender = address(uint160(uint256(_rel.k)));
        h_ = _hash(block.chainid, _id.chainId, _rel.j, sender, _t, _message);
        bytes memory payload = _payload(block.chainid, _t, _rel.j, sender, _message);
        (ok_,) = L2_TO_L2.call{ value: _v }(abi.encodeCall(m.relayMessage, (_id, payload)));
    }

    /// @notice relayMessage to a target with an observable effect (RelayProbe), with symbolic msg.value, symbolic
    ///         storage, and a target that either returns or reverts:
    ///           - success => the target did not revert, it ran exactly with msg.value, and during the call
    ///             crossDomainMessageContext() returned (sender, id.chainId); successfulMessages[H] is set;
    ///           - target reverts => the relay reverts and successfulMessages[H] is unchanged (nothing consumed);
    ///           - liveness: a non-reverting target, origin 0x..23 and H not yet relayed => the relay succeeds;
    ///           - afterwards the context getter reverts again (entered flag cleared), and a second relay in the same
    ///             transaction is not blocked by the reentrancy guard.
    ///         (The sender/source transient slots are also reset by the contract; that is unobservable through the
    ///         interface, since every context getter is onlyEntered.)
    function check_relay_delivery_value_context_failure(
        uint256 _chainId,
        Identifier memory _id,
        FrameKeys memory _rel, // sender = low 160 bits of k, nonce = j
        bytes calldata _message,
        uint256 _value,
        bool _revertTarget
    )
        public
    {
        vm.chainId(_chainId);
        svm.enableSymbolicStorage(L2_TO_L2);
        vm.assume(_value <= 1 << 128);
        vm.deal(address(this), _value);
        address p = address(new RelayProbe(_revertTarget));
        bytes32 h = _hash(_chainId, _id.chainId, _rel.j, address(uint160(uint256(_rel.k))), p, _message);
        bool before = m.successfulMessages(h);

        (bool ok,) = _relayWithValue(_id, _rel, p, _message, _value);

        if (ok) {
            assert(!_revertTarget);
            assert(_probe(p, 0) == 1);
            assert(_probe(p, 1) == _value && p.balance == _value);
            assert(_probe(p, 2) == 1);
            assert(_probe(p, 3) == uint256(uint160(uint256(_rel.k))));
            assert(_probe(p, 4) == _id.chainId);
            assert(m.successfulMessages(h));
        } else {
            assert(_probe(p, 0) == 0);
            assert(m.successfulMessages(h) == before);
            assert(p.balance == 0);
        }
        if (!_revertTarget && _id.origin == L2_TO_L2 && !before) assert(ok);

        (bool ctxOk,) = L2_TO_L2.staticcall(abi.encodeCall(m.crossDomainMessageContext, ()));
        assert(!ctxOk);
        _assertSecondRelayNotBlocked(_id, _rel, _message);
    }

    /// @notice A second relay in the same transaction (different nonce, codeless target) is not blocked by the guard.
    function _assertSecondRelayNotBlocked(
        Identifier memory _id,
        FrameKeys memory _rel,
        bytes calldata _message
    )
        internal
    {
        FrameKeys memory rel2 = FrameKeys(_rel.k, _rel.j ^ 1);
        address t2 = address(0xC0DE);
        bytes32 h2 = _hash(block.chainid, _id.chainId, rel2.j, address(uint160(uint256(_rel.k))), t2, _message);
        if (_id.origin == L2_TO_L2 && !m.successfulMessages(h2)) {
            (bool ok2,) = _relayWithValue(_id, rel2, t2, _message, 0);
            assert(ok2);
        }
    }

    /// @notice NON-VACUITY (expected FAIL): with a valid origin and fresh storage (so the only possible cause of a
    ///         revert is the target), a relay to a reverting target still succeeds.
    function check_FALSE_relay_revertingTargetStillSucceeds(
        uint256 _chainId,
        Identifier memory _id,
        FrameKeys memory _rel,
        bytes calldata _message
    )
        public
    {
        vm.chainId(_chainId);
        vm.assume(_id.origin == L2_TO_L2);
        address p = address(new RelayProbe(true));
        (bool ok,) = _relayWithValue(_id, _rel, p, _message, 0);
        assert(ok);
    }

    /// @notice NON-VACUITY (expected FAIL): a relay to a non-reverting observing target never runs it. Witness for
    ///         the success side of check_relay_delivery_value_context_failure.
    function check_FALSE_relay_probeNeverRuns(
        uint256 _chainId,
        Identifier memory _id,
        FrameKeys memory _rel,
        bytes calldata _message
    )
        public
    {
        vm.chainId(_chainId);
        address p = address(new RelayProbe(false));
        _relayWithValue(_id, _rel, p, _message, 0);
        assert(_probe(p, 0) == 0);
    }

    // ================================================================ (5) expireMessage

    function _setupExpire(address _xSender, address _other) internal {
        vm.etch(L2CDM, address(new MockL2CDMGetters()).code);
        MockL2CDMGetters(L2CDM).setGetters(_xSender, _other);
        svm.enableSymbolicStorage(L2_TO_L2);
    }

    function _expire(address _caller, bytes32 _h, uint256 _t) internal returns (bool ok_) {
        vm.prank(_caller);
        (ok_,) = L2_TO_L2.call(abi.encodeCall(m.expireMessage, (_h, _t)));
    }

    /// @notice expireMessage succeeds iff
    ///           msg.sender == 0x..07 && L2CDM.xDomainMessageSender() == L2CDM.otherMessenger()
    ///           && (expiredMessages[H] || (sentAt != 0 && t > sentAt + W))
    ///         where sentAt = sentMessageTimestamps[H] and W = EXPIRY_PERIOD read from the contract (not
    ///         hardcoded). For an already-expired H (the early return) the raw storage word of
    ///         expiredMessages[H] is unchanged.
    ///         ASSUMPTION: sentAt <= 2^64 - 1 (a block timestamp). Without it see check_expire_iff_unbounded.
    ///         Frame: on success expiredMessages[H] becomes true; on revert it is unchanged; everything else
    ///         (sentMessageTimestamps[H], successfulMessages[H], nonce, sentMessages[j], and all five observations at
    ///         every other hash H2 != H) is unchanged.
    function check_expire_iff(
        address _caller,
        address _xSender,
        address _other,
        bytes32 _h,
        bytes32 _h2,
        uint256 _t,
        uint256 _j
    )
        public
    {
        _setupExpire(_xSender, _other);
        uint256 w = m.expiryPeriod();
        uint256 sentAt = m.sentMessageTimestamps(_h);
        vm.assume(sentAt <= type(uint64).max);
        bool expiredBefore = m.expiredMessages(_h);
        vm.assume(_h2 != _h);
        Snap memory before = _snap(FrameKeys(_h2, _j));
        bool succBefore = m.successfulMessages(_h);
        bytes32 rawBefore = vm.load(L2_TO_L2, keccak256(abi.encode(_h, uint256(4))));

        bool ok = _expire(_caller, _h, _t);

        bool expected = _caller == L2CDM && _xSender == _other && (expiredBefore || (sentAt != 0 && _t > sentAt + w));
        assert(ok == expected);
        assert(m.expiredMessages(_h) == (ok ? true : expiredBefore));
        if (expiredBefore) assert(vm.load(L2_TO_L2, keccak256(abi.encode(_h, uint256(4)))) == rawBefore);
        assert(m.sentMessageTimestamps(_h) == sentAt);
        assert(m.successfulMessages(_h) == succBefore);
        _sameExceptNothing(before, _snap(FrameKeys(_h2, _j)));
    }

    /// @notice Same iff with NO bound on sentAt: the extra conjunct is that sentAt + W does not overflow (checked
    ///         arithmetic reverts otherwise).
    function check_expire_iff_unbounded(
        address _caller,
        address _xSender,
        address _other,
        bytes32 _h,
        uint256 _t
    )
        public
    {
        _setupExpire(_xSender, _other);
        uint256 w = m.expiryPeriod();
        uint256 sentAt = m.sentMessageTimestamps(_h);
        bool expiredBefore = m.expiredMessages(_h);

        bool ok = _expire(_caller, _h, _t);

        bool expected = _caller == L2CDM && _xSender == _other
            && (expiredBefore || (sentAt != 0 && sentAt <= type(uint256).max - w && _t > sentAt + w));
        assert(ok == expected);
    }

    /// @notice Exact boundary of a first expiry, authorized call, symbolic sentAt in (0, 2^64):
    ///         t == sentAt + W reverts, t == sentAt + W + 1 succeeds.
    function check_expire_boundary(address _l1Messenger, bytes32 _h) public {
        _setupExpire(_l1Messenger, _l1Messenger);
        uint256 w = m.expiryPeriod();
        uint256 sentAt = m.sentMessageTimestamps(_h);
        vm.assume(sentAt != 0 && sentAt <= type(uint64).max);
        vm.assume(!m.expiredMessages(_h));

        assert(!_expire(L2CDM, _h, sentAt + w));
        assert(_expire(L2CDM, _h, sentAt + w + 1));
        assert(m.expiredMessages(_h));
    }

    /// @notice NON-VACUITY (expected FAIL): the window check is `>=` (expiry at exactly sentAt + W). The
    ///         counterexample is t == sentAt + W.
    function check_FALSE_expire_windowIsGte(address _l1Messenger, bytes32 _h, uint256 _t) public {
        _setupExpire(_l1Messenger, _l1Messenger);
        uint256 w = m.expiryPeriod();
        uint256 sentAt = m.sentMessageTimestamps(_h);
        vm.assume(sentAt != 0 && sentAt <= type(uint64).max);

        bool ok = _expire(L2CDM, _h, _t);
        assert(ok == (_t >= sentAt + w));
    }

    /// @notice NON-VACUITY (expected FAIL): the early return is reachable: an authorized call for an
    ///         already-expired H with no send timestamp and t = 0 succeeds.
    function check_FALSE_expire_alreadyExpiredRejected(address _l1Messenger, bytes32 _h) public {
        _setupExpire(_l1Messenger, _l1Messenger);
        vm.assume(m.expiredMessages(_h) && m.sentMessageTimestamps(_h) == 0);
        assert(!_expire(L2CDM, _h, 0));
    }

    /// @notice NON-VACUITY (expected FAIL): expireMessage never succeeds (same symbolic world as check_expire_iff).
    function check_FALSE_expire_neverSucceeds(
        address _caller,
        address _xSender,
        address _other,
        bytes32 _h,
        uint256 _t
    )
        public
    {
        _setupExpire(_xSender, _other);
        assert(!_expire(_caller, _h, _t));
    }

    /// @notice NON-VACUITY (expected FAIL): the auth check ignores xDomainMessageSender.
    function check_FALSE_expire_ignoresXDomainSender(address _xSender, address _other, bytes32 _h, uint256 _t) public {
        _setupExpire(_xSender, _other);
        uint256 w = m.expiryPeriod();
        uint256 sentAt = m.sentMessageTimestamps(_h);
        vm.assume(sentAt != 0 && sentAt <= type(uint64).max);

        bool ok = _expire(L2CDM, _h, _t);
        assert(ok == (_t > sentAt + w));
    }

    // ================================================================ parameters / documented facts

    /// @notice P_contract >= W_protocol: the contract's expiry period covers the protocol window, which op-core/kona
    ///         config parsing caps at 7 days. Holds for the current constant (7 days) and the planned one (8 days).
    function check_contractWindowCoversProtocolCap() public view {
        assert(m.expiryPeriod() >= 7 days);
    }

    /// @notice The expiry period is the protocol cap (7 days) plus the 1-day margin.
    function check_expiryPeriodIsCapPlusMargin() public view {
        assert(m.expiryPeriod() == 7 days + 1 days);
    }

    /// @notice NON-VACUITY (expected FAIL): the expiry period equals the bare protocol cap (it has the 1-day margin).
    function check_FALSE_expiryPeriodIsProtocolCap() public view {
        assert(m.expiryPeriod() == 7 days);
    }

    /// @notice INFO (expected FAIL, documents an assumption): relayMessage ITSELF does not reject target 0x..23; the
    ///         relay checks above exclude it because sendMessage on every source chain rejects it (and the identifier
    ///         origin must be 0x..23). The counterexample is a relay to 0x..23 calling messageNonce().
    function check_INFO_relayDoesNotRejectTarget23(
        uint256 _chainId,
        Identifier memory _id,
        uint256 _nonce,
        address _sender
    )
        public
    {
        vm.chainId(_chainId);
        bool ok = _relay(address(this), _id, _chainId, L2_TO_L2, _nonce, _sender, abi.encodeCall(m.messageNonce, ()));
        assert(!ok);
    }
}
