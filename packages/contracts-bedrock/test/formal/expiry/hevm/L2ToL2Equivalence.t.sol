// SPDX-License-Identifier: MIT
pragma solidity 0.8.25;

// Equivalence harness: develop's L2ToL2CrossDomainMessenger (c9b441bac5) vs the current one, on
// the shared surface (sendMessage, relayMessage). The shared getters are checked at bytecode level
// with `hevm equivalence` (see run.sh). README.md states the result, the bounds and the review log.
//
// WHAT IS RUN. Both runtime bytecodes, built with the repo foundry.toml default profile
// (L2ToL2Bytecodes.sol, from gen-bytecodes.sh), are etched at two addresses, OLD and NEW. Each
// check calls both with the same symbolic inputs, from the same symbolic pre-state, and asserts:
//   (O) same success/revert, and byte-identical return data or revert data (modulo E6);
//   (X) relay only, same external interaction: same ETH left in the messenger and moved to the
//       target; same committed calls to the CrossL2Inbox (count, exact arguments) and to the
//       target (count, exact calldata); the inbox and the target revert on a predicate of their
//       own arguments (an uninterpreted keccak bit) and put their argument hash (target: calldata
//       hash and msg.value) in their revert and return data, so their arguments are also compared
//       through (O) on paths that revert; the target reads crossDomainMessageContext() back
//       during the call and attempts a reentrant relayMessage, and returns both results; after
//       the call both are back to "not entered" (the getters are onlyEntered, so this compares
//       only the entered flag, not the stored sender/source);
//   (S) same storage, in two forms:
//       (S-map) Halmos, solidity storage layout: for a symbolic key q, equal msgNonce (slot 1),
//               successfulMessages[q] (slot 0) and sentMessages[q] (slot 2); the new-only
//               state is pinned exactly in NEW (sentMessageTimestamps[q], slot 3;
//               expiredMessages[q], slot 4; expiryPeriod, slot 5) and zero in OLD. Slots 0-5 are
//               every storage variable of both versions (the cross-domain context is transient;
//               the current version's Initializable word is never read or written by send or
//               relay). This is the only check of MAPPING ENTRIES: one symbolic key per mapping,
//               so any one entry.
//       (S-all) a symbolic raw slot s outside the pinned new-only slots (E3, E7) is equal, and
//               NEW's expiryPeriod (slot 5) is unchanged: hevm (empty
//               message) and Halmos with the generic storage layout (every length;
//               check_allSlots_*). Under Halmos 0.3.3 this covers only NON-HASH-DERIVED slots:
//               its generic layout keeps keccak-derived slots (mapping entries) in separate
//               arrays, so s never aliases a mapping entry and a changed entry is invisible to it
//               (run.sh mutants M5/M6, caught by S-map). It catches stray writes to plain slots
//               (mutants M1/M2). hevm's storage model for prove_sendMessage_len0 is not shown to
//               have this limitation, nor shown free of it.
// Engines: L2ToL2CrossDomainMessenger_EquivalenceHevm (prove_*, hevm test) and
// L2ToL2CrossDomainMessenger_EquivalenceHalmos (check_*, halmos). hevm 0.58 cannot reason about
// develop's sendMessage once the message is non-empty ("CopySlice with a symbolically sized region
// not currently implemented", inside develop's code) and its relayMessage run exceeds 16 GB even
// with an empty message, so hevm proves sendMessage with the empty message and Halmos proves
// sendMessage and relayMessage for every listed length.
//
// NOT COMPARED:
//   - Events. Neither engine can assert on logs (hevm: no recordLogs, and `hevm equivalence`
//     ignores logs, see run.sh's log-blindness witness; halmos 0.3.3: no recordLogs/expectEmit).
//     L2ToL2EquivalenceLogs.t.sol is a concrete differential fuzz of the logs instead.
//   - Gas. Calls get the engine's gas; gas use, gas-dependent behaviour and out-of-gas outcomes are
//     excluded (NEW's extra SSTORE in sendMessage costs more, so with a tight gas limit a send can
//     succeed in OLD and run out of gas in NEW; relay forwards gas to the target).
//
// EXCLUDED (intentional differences, by design of the expiry change):
//   E1 target in {L2CrossDomainMessenger 0x..07, L2ToL1MessagePasser 0x..16}: NEW reverts on send
//      and relay (MessageTargetUnsafe). Precondition of every equivalence check.
//   E2 sentMessageTimestamps[H]: written by NEW's sendMessage only. Pinned to block.timestamp.
//   E3 the new-only mappings (slot 3 sentMessageTimestamps, slot 4 expiredMessages): NEW starts
//      with one arbitrary entry in each (sentMessageTimestamps[y] = uy, expiredMessages[z] = uz),
//      to show send/relay do not read them; pinned (unchanged, except E2), not compared with OLD.
//   E4 selectors outside the shared surface are not called: resendMessage (removed),
//      expireMessage, sentMessageTimestamps, expiredMessages, expiryPeriod, initialize,
//      proxyAdmin, proxyAdminOwner (added), version (changed).
//   E5 relay targets are the target mock and an address with no code. A relay to the messenger
//      itself (0x..23) is not modelled; there the two DO differ (the self-call reaches E4, e.g.
//      develop's resendMessage). It needs a SentMessage log from 0x..23 with target 0x..23, which
//      neither version emits: both sendMessage versions reject target 0x..23, and develop's
//      resendMessage only re-emits a stored hash, which commits to the target (keccak collision
//      resistance). So it is unreachable through a valid identifier.
//   E6 errors the current messenger renamed with its contract-name prefix: revert data is compared
//      with its selector mapped to develop's (mapRenamedError, generated by gen-bytecodes.sh from
//      both sources; empty while nothing is renamed). Only top-level revert data is mapped:
//      revert data nested in the target mock's output (its reentrant relay attempt) is compared
//      raw, which holds while the reentrancy error is not renamed.
//   E7 the new-only expiryPeriod (slot 5): NEW holds the production period
//      (Constants.L2_TO_L2_MESSAGE_EXPIRY_PERIOD) there, as a proxy initialized by
//      L2ContractsManager does; OLD has no such variable. Like E3, it is pinned (send and relay
//      leave it unchanged) and not compared with OLD.
//
// DOMAIN (beyond E1-E6):
//   - Message lengths are concrete per check: send 0/4/37/100/128 bytes, relay 0/4/37/100 bytes
//     (contents fully symbolic, built from four free words). The relay payload is canonical
//     (offsets, lengths and padding as SentMessage encodes them) except for a free selector word
//     and free (possibly dirty) target and sender words. check_relayMessage_rawPayload_* adds
//     payloads that are arbitrary byte strings of 0/100/128 bytes (truncated: every byte free);
//     check_relayMessage_badEncoding_* adds 288-byte payloads whose `bytes` offset/length words
//     are concrete non-canonical values (offset 0x20/0x60/2^64, length 33/65/2^64), every other
//     byte free. Halmos cannot decode with a symbolic offset ("symbolic CALLDATALOAD offset"),
//     so longer payloads with free offset/length words are not covered. In all of these the
//     target word's low 160 bits are held to the mock or the code-less address.
//   - The outer relayMessage calldata is ABI-encoded by the harness (not malformed).
//   - Return and revert data come from the mocks: structured, not arbitrary byte strings.
//   - The relay caller is one fixed funded account (engines cannot vm.deal a symbolic address);
//     relayMessage does not read msg.sender in either version. msg.value < 2^120 (halmos caps
//     balances at 2^128). block.chainid is the engine's concrete default; destination and
//     id.chainId are symbolic. Transient storage starts zero (start of a transaction).
//   - Pre-state: storage zero, then symbolic msgNonce (slot 1, all 256 bits), sentMessages[n0]
//     (n0 = the nonce the send uses), successfulMessages[q0] for a free q0 and, for relay,
//     successfulMessages[H] (H = the hash the relay computes); NEW also gets E3 and E7. hevm
//     seeds slot 1 only (and E7): with the current messenger's bytecode it runs out of 16 GB as
//     soon as storage has an entry at a symbolic key, so its S-all statement is from that
//     narrower pre-state; the Halmos checks cover the others.
//   - Caveat (hevm): only the bytecode-level getter checks read transient storage under hevm, from
//     the same initial store for both codes.
// The code under test has no loops. Solvers: hevm with z3 4.13.3; halmos 0.3.3 with yices.

// Testing
import { Test } from "test/setup/Test.sol";

// Libraries
import { L2ToL2Bytecodes } from "./L2ToL2Bytecodes.sol";
import { Constants } from "src/libraries/Constants.sol";

/// @notice Identifier of a SentMessage log, as relayMessage takes it.
struct EquivalenceIdentifier {
    address origin;
    uint256 blockNumber;
    uint256 logIndex;
    uint256 timestamp;
    uint256 chainId;
}

/// @notice The shared surface of both messenger versions.
interface IEquivalenceMessenger {
    function sendMessage(
        uint256 _destination,
        address _target,
        bytes calldata _message
    )
        external
        returns (bytes32 messageHash_);

    function relayMessage(
        EquivalenceIdentifier calldata _id,
        bytes calldata _sentMessage
    )
        external
        payable
        returns (bytes memory returnData_);

    function crossDomainMessageContext() external view returns (address sender_, uint256 source_);
}

/// @notice CrossL2Inbox stand-in. Only validateMessage is callable. It reverts iff the low bit of
///         keccak(salt, argsHash) is set (an uninterpreted predicate of its arguments), with
///         argsHash as the revert data, and otherwise records argsHash per caller.
///         Storage (set with vm.store / read with vm.load): slot 0 salt, slot 1 mapping calls,
///         slot 2 mapping argsHash.
contract EquivalenceInboxMock {
    bytes32 internal salt;
    mapping(address => uint256) internal calls;
    mapping(address => bytes32) internal argsHashOf;

    /// @notice Validates (or rejects) a message.
    ///
    /// @param _id      Identifier of the log.
    /// @param _msgHash Hash of the log payload.
    function validateMessage(EquivalenceIdentifier calldata _id, bytes32 _msgHash) external {
        bytes32 argsHash = keccak256(abi.encode(_id, _msgHash));
        if (uint256(keccak256(abi.encode(salt, argsHash))) & 1 == 1) {
            assembly {
                mstore(0, argsHash)
                revert(0, 32)
            }
        }
        calls[msg.sender] += 1;
        argsHashOf[msg.sender] = argsHash;
    }
}

/// @notice Relay target stand-in. Fallback only, so no message can reconfigure it between the two
///         runs. It reads the cross-domain context back, optionally attempts a reentrant relay,
///         and hands back abi.encode(calldata hash, msg.value, w0, w1, context result, reentry
///         result): as revert data iff the low bit of keccak(salt, calldata hash, value) is set,
///         else as return data.
///         Storage: slot 0 salt, slot 1/2 free words, slot 3 reentry flag, slot 4 mapping calls,
///         slot 5 mapping calldata hash.
contract EquivalenceTargetMock {
    bytes32 internal salt;
    bytes32 internal w0;
    bytes32 internal w1;
    uint256 internal reenterFlag;
    mapping(address => uint256) internal calls;
    mapping(address => bytes32) internal dataHash;

    fallback(bytes calldata _data) external payable returns (bytes memory) {
        bytes32 h = keccak256(_data);
        calls[msg.sender] += 1;
        dataHash[msg.sender] = h;

        (bool ctxOk, bytes memory ctxRet) =
            msg.sender.staticcall(abi.encodeCall(IEquivalenceMessenger.crossDomainMessageContext, ()));

        bool reOk;
        bytes memory reRet;
        if (reenterFlag != 0) {
            EquivalenceIdentifier memory id;
            (reOk, reRet) = msg.sender.call(abi.encodeCall(IEquivalenceMessenger.relayMessage, (id, bytes(""))));
        }

        bytes memory out = abi.encode(h, msg.value, w0, w1, ctxOk, ctxRet, reOk, reRet);
        if (uint256(keccak256(abi.encode(salt, h, msg.value))) & 1 == 1) {
            assembly {
                revert(add(out, 32), mload(out))
            }
        }
        return out;
    }
}

/// @notice Setup, inputs and checks shared by both engines.
abstract contract L2ToL2CrossDomainMessenger_EquivalenceBase is Test {
    address internal constant OLD = address(0x0000000000000000000000000000000000010000);
    address internal constant NEW = address(0x0000000000000000000000000000000000020000);
    address internal constant TARGET = address(0x0000000000000000000000000000000000030000);
    address internal constant EOA = address(0x0000000000000000000000000000000000040000);
    address internal constant CALLER = address(uint160(0xCA11E));
    address internal constant L2CDM = 0x4200000000000000000000000000000000000007;
    address internal constant PASSER = 0x4200000000000000000000000000000000000016;
    address internal constant INBOX = 0x4200000000000000000000000000000000000022;
    bytes32 internal constant SENT_MESSAGE_SELECTOR =
        0x382409ac69001e11931a28435afef442cbfd20d9891907e8fa373ba7d351f320;
    /// @notice Storage slot of NEW's expiryPeriod (E7).
    bytes32 internal constant EXPIRY_PERIOD_SLOT = bytes32(uint256(5));

    /// @notice Etches both messengers and the mocks; NEW gets the production period in storage (E7).
    function setUp() public virtual {
        vm.etch(OLD, L2ToL2Bytecodes.DEVELOP);
        vm.etch(NEW, _newCode());
        _storePeriod();
        vm.etch(INBOX, address(new EquivalenceInboxMock()).code);
        vm.etch(TARGET, address(new EquivalenceTargetMock()).code);
    }

    /// @notice NEW's expiryPeriod, as a proxy initialized with the production period holds it (E7).
    function _storePeriod() internal {
        vm.store(NEW, EXPIRY_PERIOD_SLOT, bytes32(Constants.L2_TO_L2_MESSAGE_EXPIRY_PERIOD));
    }

    /// @notice E7 pin: NEW's period is unchanged, and OLD (which has no such variable) has `_oldWord` there.
    function _checkPeriodPinned(bytes32 _oldWord) internal view {
        assert(vm.load(NEW, EXPIRY_PERIOD_SLOT) == bytes32(Constants.L2_TO_L2_MESSAGE_EXPIRY_PERIOD));
        assert(vm.load(OLD, EXPIRY_PERIOD_SLOT) == _oldWord);
    }

    /// @notice The code under test as NEW (overridden by run.sh's mutants).
    function _newCode() internal virtual returns (bytes memory code_) {
        code_ = L2ToL2Bytecodes.CURRENT;
    }

    // ---------------------------------------------------------------------------------------
    // Inputs
    // ---------------------------------------------------------------------------------------

    struct SendIn {
        uint256 dest;
        address target;
        address sender;
        uint256 ts;
        bytes32 m0;
        bytes32 m1;
        bytes32 m2;
        bytes32 m3;
    }

    struct RelayIn {
        EquivalenceIdentifier id;
        bytes32 selWord; // free: the wrong-selector revert is explored too
        uint256 dest;
        bytes32 targetWord; // free upper 96 bits; low 160 bits: TARGET or EOA
        uint256 nonce;
        bytes32 senderWord; // free 256 bits (dirty-address revert explored)
        bytes32 m0;
        bytes32 m1;
        bytes32 m2;
        bytes32 m3;
        uint256 value;
    }

    struct RawIn {
        EquivalenceIdentifier id;
        bytes32[9] words; // the payload's bytes, all free except the target word's low 160 bits
        uint256 value;
    }

    struct Env {
        bytes32 inboxSalt;
        bytes32 targetSalt;
        bytes32 w0;
        bytes32 w1;
        uint256 reenter;
        bytes32 successfulWord; // pre-state successfulMessages[H], both
    }

    struct Pre {
        bytes32 nonceWord; // slot 1 (msgNonce), all 256 bits free
        bytes32 preSent; // sentMessages[n0], n0 = the nonce the send uses
        bytes32 q0; // successfulMessages[q0] = s0
        bytes32 s0;
        bytes32 y; // NEW only: sentMessageTimestamps[y] = uy
        bytes32 uy;
        bytes32 z; // NEW only: expiredMessages[z] = uz
        bytes32 uz;
    }

    struct Bal {
        uint256 t0;
        uint256 t1;
        uint256 t2;
        uint256 e0;
        uint256 e1;
        uint256 e2;
    }

    // ---------------------------------------------------------------------------------------
    // Helpers
    // ---------------------------------------------------------------------------------------

    /// @notice Whether a target is outside E1.
    function _safe(address _target) internal pure returns (bool safe_) {
        safe_ = _target != L2CDM && _target != PASSER;
    }

    /// @notice Message of concrete length `_len` (<= 128) with fully symbolic contents.
    function _msg(uint256 _len, bytes32 _a, bytes32 _b, bytes32 _c, bytes32 _d)
        internal
        pure
        returns (bytes memory m_)
    {
        m_ = abi.encodePacked(_a, _b, _c, _d);
        assembly {
            mstore(m_, _len)
        }
    }

    /// @notice Storage slot of `mapping[_key]` for a mapping at `_slot`.
    function _mapSlot(bytes32 _key, uint256 _slot) internal pure returns (bytes32 slot_) {
        slot_ = keccak256(abi.encode(_key, _slot));
    }

    /// @notice The nonce key sendMessage uses: msgNonce (low 240 bits of slot 1), version 0.
    function _n0(Pre memory _p) internal pure returns (bytes32 n0_) {
        n0_ = bytes32(uint256(_p.nonceWord) & type(uint240).max);
    }

    /// @notice Mapping-shaped pre-state (S-map checks), both; E3 in NEW.
    function _seed(Pre memory _p) internal {
        vm.store(OLD, bytes32(uint256(1)), _p.nonceWord);
        vm.store(NEW, bytes32(uint256(1)), _p.nonceWord);
        vm.store(OLD, _mapSlot(_n0(_p), 2), _p.preSent);
        vm.store(NEW, _mapSlot(_n0(_p), 2), _p.preSent);
        vm.store(OLD, _mapSlot(_p.q0, 0), _p.s0);
        vm.store(NEW, _mapSlot(_p.q0, 0), _p.s0);
        vm.store(NEW, _mapSlot(_p.y, 3), _p.uy);
        vm.store(NEW, _mapSlot(_p.z, 4), _p.uz);
        _storePeriod();
    }

    /// @notice Configures the mocks.
    function _configure(Env memory _e) internal {
        vm.store(INBOX, bytes32(uint256(0)), _e.inboxSalt);
        vm.store(TARGET, bytes32(uint256(0)), _e.targetSalt);
        vm.store(TARGET, bytes32(uint256(1)), _e.w0);
        vm.store(TARGET, bytes32(uint256(2)), _e.w1);
        vm.store(TARGET, bytes32(uint256(3)), bytes32(_e.reenter));
    }

    /// @notice A mock's per-caller record (mapping at `_mappingSlot`).
    function _perCaller(address _who, address _caller, uint256 _mappingSlot) internal view returns (bytes32 v_) {
        v_ = vm.load(_who, keccak256(abi.encode(_caller, _mappingSlot)));
    }

    /// @notice Whether a target word's address is the mock or the code-less address.
    function _targetOk(bytes32 _targetWord) internal pure returns (bool ok_) {
        address t = address(uint160(uint256(_targetWord)));
        ok_ = t == TARGET || t == EOA;
    }

    /// @notice (S-all): the symbolic raw slot `_s`, if outside the pinned new-only slots (E3, E7),
    ///         is equal, and NEW's period (E7) is unchanged. Under Halmos's generic layout `_s`
    ///         ranges over non-hash-derived slots only (see the header); mapping entries are
    ///         compared by S-map.
    function _checkAllSlots(Pre memory _p, bool _sent, bytes32 _h, bytes32 _s) internal view {
        bool pinned = _s == _mapSlot(_p.y, 3) || _s == _mapSlot(_p.z, 4) || (_sent && _s == _mapSlot(_h, 3))
            || _s == EXPIRY_PERIOD_SLOT;
        if (!pinned) assert(vm.load(OLD, _s) == vm.load(NEW, _s));
        assert(vm.load(NEW, EXPIRY_PERIOD_SLOT) == bytes32(Constants.L2_TO_L2_MESSAGE_EXPIRY_PERIOD));
    }

    /// @notice What (O) compares: the return data, or the revert data with its selector passed
    ///         through mapRenamedError (E6), on both sides.
    function _outKey(bool _ok, bytes memory _data) internal pure returns (bytes32 key_) {
        if (_ok || _data.length < 4) return keccak256(abi.encode(_ok, _data));
        bytes4 selector;
        bytes32 rest;
        assembly {
            selector := mload(add(_data, 32))
            rest := keccak256(add(_data, 36), sub(mload(_data), 4))
        }
        key_ = keccak256(abi.encode(false, L2ToL2Bytecodes.mapRenamedError(selector), rest, _data.length));
    }

    // ---------------------------------------------------------------------------------------
    // sendMessage
    // ---------------------------------------------------------------------------------------

    /// @notice Runs sendMessage on both from the seeded state; checks (O).
    function _send(uint256 _len, SendIn memory _in) internal returns (bool ok_, bytes32 h_) {
        vm.warp(_in.ts);
        bytes memory m = _msg(_len, _in.m0, _in.m1, _in.m2, _in.m3);
        bytes memory cd = abi.encodeCall(IEquivalenceMessenger.sendMessage, (_in.dest, _in.target, m));

        vm.prank(_in.sender);
        (bool okO, bytes memory rO) = _sendCall(OLD, cd);
        vm.prank(_in.sender);
        (bool okN, bytes memory rN) = _sendCall(NEW, cd);

        assert(okO == okN);
        assert(_outKey(okO, rO) == _outKey(okN, rN));

        ok_ = okN;
        if (okN) h_ = abi.decode(rN, (bytes32));
    }

    /// @notice One sendMessage call; overridden by the hevm engine.
    function _sendCall(address _to, bytes memory _cd) internal virtual returns (bool ok_, bytes memory ret_) {
        (ok_, ret_) = _to.call(_cd);
    }

    /// @notice (S-map) after a send, for the symbolic key `_q`.
    function _checkSendMappings(SendIn memory _in, Pre memory _p, bool _ok, bytes32 _h, bytes32 _q) internal view {
        assert(vm.load(OLD, bytes32(uint256(1))) == vm.load(NEW, bytes32(uint256(1))));
        assert(vm.load(OLD, _mapSlot(_q, 0)) == vm.load(NEW, _mapSlot(_q, 0)));
        assert(vm.load(OLD, _mapSlot(_q, 2)) == vm.load(NEW, _mapSlot(_q, 2)));
        bytes32 ts = _ok && _q == _h ? bytes32(_in.ts) : (_q == _p.y ? _p.uy : bytes32(0));
        assert(vm.load(NEW, _mapSlot(_q, 3)) == ts);
        assert(vm.load(NEW, _mapSlot(_q, 4)) == (_q == _p.z ? _p.uz : bytes32(0)));
        assert(vm.load(OLD, _mapSlot(_q, 3)) == bytes32(0));
        assert(vm.load(OLD, _mapSlot(_q, 4)) == bytes32(0));
        _checkPeriodPinned(bytes32(0));
    }

    // ---------------------------------------------------------------------------------------
    // relayMessage
    // ---------------------------------------------------------------------------------------

    /// @notice Canonical relay calldata; seeds successfulMessages[H] for the H the relay computes.
    function _relayCalldata(uint256 _len, RelayIn memory _in, Env memory _e) internal returns (bytes memory cd_) {
        bytes memory m = _msg(_len, _in.m0, _in.m1, _in.m2, _in.m3);
        bytes32 h = keccak256(
            abi.encode(
                _in.dest,
                _in.id.chainId,
                _in.nonce,
                address(uint160(uint256(_in.senderWord))),
                address(uint160(uint256(_in.targetWord))),
                m
            )
        );
        vm.store(OLD, _mapSlot(h, 0), _e.successfulWord);
        vm.store(NEW, _mapSlot(h, 0), _e.successfulWord);

        bytes memory payload = abi.encodePacked(
            abi.encode(_in.selWord, _in.dest, _in.targetWord, _in.nonce), abi.encode(_in.senderWord, m)
        );
        cd_ = abi.encodeCall(IEquivalenceMessenger.relayMessage, (_in.id, payload));
    }

    /// @notice Relay calldata whose payload is `_len` arbitrary bytes.
    function _rawCalldata(uint256 _len, RawIn memory _in) internal pure returns (bytes memory cd_) {
        bytes memory payload = abi.encodePacked(_in.words);
        assembly {
            mstore(payload, _len)
        }
        cd_ = abi.encodeCall(IEquivalenceMessenger.relayMessage, (_in.id, payload));
    }

    /// @notice Runs relayMessage on both with `_cd`; checks (O) and (X).
    function _relay(bytes memory _cd, uint256 _value, Env memory _e) internal {
        vm.assume(_value <= type(uint120).max);
        _configure(_e);

        vm.deal(CALLER, uint256(type(uint120).max) * 2);
        Bal memory b;
        b.t0 = TARGET.balance;
        b.e0 = EOA.balance;
        vm.prank(CALLER);
        (bool okO, bytes memory rO) = OLD.call{ value: _value }(_cd);
        b.t1 = TARGET.balance;
        b.e1 = EOA.balance;
        vm.prank(CALLER);
        (bool okN, bytes memory rN) = NEW.call{ value: _value }(_cd);
        b.t2 = TARGET.balance;
        b.e2 = EOA.balance;

        assert(okO == okN);
        assert(_outKey(okO, rO) == _outKey(okN, rN));
        _checkRelayEffects(b);
    }

    /// @notice (X), committed part.
    function _checkRelayEffects(Bal memory _b) internal view {
        assert(OLD.balance == NEW.balance);
        assert(_b.t1 - _b.t0 == _b.t2 - _b.t1);
        assert(_b.e1 - _b.e0 == _b.e2 - _b.e1);

        assert(_perCaller(INBOX, OLD, 1) == _perCaller(INBOX, NEW, 1));
        assert(_perCaller(INBOX, OLD, 2) == _perCaller(INBOX, NEW, 2));
        assert(_perCaller(TARGET, OLD, 4) == _perCaller(TARGET, NEW, 4));
        assert(_perCaller(TARGET, OLD, 5) == _perCaller(TARGET, NEW, 5));

        // Entered flag cleared identically (the getter is onlyEntered).
        (bool cO, bytes memory xO) = OLD.staticcall(abi.encodeCall(IEquivalenceMessenger.crossDomainMessageContext, ()));
        (bool cN, bytes memory xN) = NEW.staticcall(abi.encodeCall(IEquivalenceMessenger.crossDomainMessageContext, ()));
        assert(cO == cN);
        assert(keccak256(xO) == keccak256(xN));
    }

    /// @notice (S-map) after a relay, for the symbolic key `_q`: no E2 write at all.
    function _checkRelayMappings(Pre memory _p, bytes32 _q) internal view {
        assert(vm.load(OLD, bytes32(uint256(1))) == vm.load(NEW, bytes32(uint256(1))));
        assert(vm.load(OLD, _mapSlot(_q, 0)) == vm.load(NEW, _mapSlot(_q, 0)));
        assert(vm.load(OLD, _mapSlot(_q, 2)) == vm.load(NEW, _mapSlot(_q, 2)));
        assert(vm.load(NEW, _mapSlot(_q, 3)) == (_q == _p.y ? _p.uy : bytes32(0)));
        assert(vm.load(NEW, _mapSlot(_q, 4)) == (_q == _p.z ? _p.uz : bytes32(0)));
        assert(vm.load(OLD, _mapSlot(_q, 3)) == bytes32(0));
        assert(vm.load(OLD, _mapSlot(_q, 4)) == bytes32(0));
        _checkPeriodPinned(bytes32(0));
    }

    /// @notice A relay through NEW to the target mock; whether it succeeded with the right context.
    function _relaySeesContext(uint256 _len, RelayIn memory _in, Env memory _e) internal returns (bool seen_) {
        _configure(_e);
        (bool ok, bytes memory ret) = NEW.call(_relayCalldata(_len, _in, _e));
        if (!ok) return false;
        bytes memory inner = abi.decode(ret, (bytes));
        (,,,, bool ctxOk, bytes memory ctxRet,,) =
            abi.decode(inner, (bytes32, uint256, bytes32, bytes32, bool, bytes, bool, bytes));
        if (!ctxOk) return false;
        (address s, uint256 src) = abi.decode(ctxRet, (address, uint256));
        seen_ = s == address(uint160(uint256(_in.senderWord))) && src == _in.id.chainId;
    }
}

/// @notice hevm engine (hevm test, prove_*): sendMessage with the empty message, (O) and (S-all).
contract L2ToL2CrossDomainMessenger_EquivalenceHevm is L2ToL2CrossDomainMessenger_EquivalenceBase {
    /// @notice hevm's pre-state: msgNonce (both) and NEW's period (E7). With the current messenger's
    ///         bytecode, hevm 0.58 runs out of 16 GB on prove_sendMessage_len0 as soon as either
    ///         code has any storage entry at a symbolic key (the arbitrary raw slot k := v, or the
    ///         E3 entries); the Halmos checks keep those entries.
    function _seedHevm(Pre memory _p) internal {
        vm.store(OLD, bytes32(uint256(1)), _p.nonceWord);
        vm.store(NEW, bytes32(uint256(1)), _p.nonceWord);
        _storePeriod();
    }

    /// @notice sendMessage's outputs, copied with a concrete size: 32 bytes on success, 0, 4 or
    ///         36 bytes on revert (any other size is a counterexample). hevm 0.58 cannot copy a
    ///         returndata region whose size it holds only symbolically, which the current
    ///         messenger's revert paths produce once its storage has a symbolic-key entry.
    function _sendCall(address _to, bytes memory _cd) internal override returns (bool ok_, bytes memory ret_) {
        assembly {
            ok_ := call(gas(), _to, 0, add(_cd, 32), mload(_cd), 0, 0)
        }
        uint256 n;
        assembly {
            n := returndatasize()
        }
        if (n == 0) ret_ = new bytes(0);
        else if (n == 4) ret_ = new bytes(4);
        else if (n == 32) ret_ = new bytes(32);
        else if (n == 36) ret_ = new bytes(36);
        else assert(false);
        assembly {
            returndatacopy(add(ret_, 32), 0, mload(ret_))
        }
    }

    /// @notice (O) and (S-all), plus the E2/E7 pins.
    function prove_sendMessage_len0(SendIn memory _in, Pre memory _p, bytes32 _s) public {
        vm.assume(_safe(_in.target));
        _seedHevm(_p);
        (bool ok, bytes32 h) = _send(0, _in);
        _checkAllSlots(_p, ok, h, _s);
        if (ok) assert(vm.load(NEW, _mapSlot(h, 3)) == bytes32(_in.ts));
    }

    // Non-vacuity witnesses: each must produce a validated counterexample (run.sh checks that).

    /// @notice E1 is real: an unsafe target separates the two.
    function prove_nonvacuity_sendMessage_unsafeTarget(SendIn memory _in, Pre memory _p) public {
        vm.assume(!_safe(_in.target));
        _seedHevm(_p);
        _send(0, _in);
    }

    /// @notice E2 is real: OLD does not hold the timestamp NEW writes.
    function prove_nonvacuity_sendMessage_timestampSlot(SendIn memory _in, Pre memory _p) public {
        vm.assume(_safe(_in.target));
        _seedHevm(_p);
        (bool ok, bytes32 h) = _send(0, _in);
        if (ok) assert(vm.load(OLD, _mapSlot(h, 3)) == vm.load(NEW, _mapSlot(h, 3)));
    }

    /// @notice Reachability: a successful send is explored.
    function prove_nonvacuity_sendMessage_successReachable(SendIn memory _in, Pre memory _p) public {
        vm.assume(_safe(_in.target));
        _seedHevm(_p);
        (bool ok,) = _send(0, _in);
        assert(!ok);
    }
}

/// @notice Halmos engine (halmos, check_*). check_allSlots_* run with --storage-layout generic,
///         the others with the solidity layout (run.sh).
contract L2ToL2CrossDomainMessenger_EquivalenceHalmos is L2ToL2CrossDomainMessenger_EquivalenceBase {
    function _sendCheck(uint256 _len, SendIn memory _in, Pre memory _p, bytes32 _q) internal {
        vm.assume(_safe(_in.target));
        _seed(_p);
        (bool ok, bytes32 h) = _send(_len, _in);
        _checkSendMappings(_in, _p, ok, h, _q);
    }

    function _sendAll(uint256 _len, SendIn memory _in, Pre memory _p, bytes32 _s) internal {
        vm.assume(_safe(_in.target));
        _seed(_p);
        (bool ok, bytes32 h) = _send(_len, _in);
        _checkAllSlots(_p, ok, h, _s);
    }

    function _relayCheck(uint256 _len, RelayIn memory _in, Env memory _e, Pre memory _p, bytes32 _q) internal {
        vm.assume(_targetOk(_in.targetWord));
        _seed(_p);
        _relay(_relayCalldata(_len, _in, _e), _in.value, _e);
        _checkRelayMappings(_p, _q);
    }

    function _relayAll(uint256 _len, RelayIn memory _in, Env memory _e, Pre memory _p, bytes32 _s) internal {
        vm.assume(_targetOk(_in.targetWord));
        _seed(_p);
        _relay(_relayCalldata(_len, _in, _e), _in.value, _e);
        _checkAllSlots(_p, false, bytes32(0), _s);
    }

    function _rawCheck(uint256 _len, RawIn memory _in, Env memory _e, Pre memory _p, bytes32 _q) internal {
        vm.assume(_targetOk(_in.words[2]));
        _seed(_p);
        _relay(_rawCalldata(_len, _in), _in.value, _e);
        _checkRelayMappings(_p, _q);
    }

    // sendMessage, (O) + (S-map)
    function check_sendMessage_len0(SendIn memory _in, Pre memory _p, bytes32 _q) public {
        _sendCheck(0, _in, _p, _q);
    }

    function check_sendMessage_len4(SendIn memory _in, Pre memory _p, bytes32 _q) public {
        _sendCheck(4, _in, _p, _q);
    }

    function check_sendMessage_len37(SendIn memory _in, Pre memory _p, bytes32 _q) public {
        _sendCheck(37, _in, _p, _q);
    }

    function check_sendMessage_len100(SendIn memory _in, Pre memory _p, bytes32 _q) public {
        _sendCheck(100, _in, _p, _q);
    }

    function check_sendMessage_len128(SendIn memory _in, Pre memory _p, bytes32 _q) public {
        _sendCheck(128, _in, _p, _q);
    }

    // sendMessage, (O) + (S-all), generic storage layout
    function check_allSlots_sendMessage_len0(SendIn memory _in, Pre memory _p, bytes32 _s) public {
        _sendAll(0, _in, _p, _s);
    }

    function check_allSlots_sendMessage_len4(SendIn memory _in, Pre memory _p, bytes32 _s) public {
        _sendAll(4, _in, _p, _s);
    }

    function check_allSlots_sendMessage_len37(SendIn memory _in, Pre memory _p, bytes32 _s) public {
        _sendAll(37, _in, _p, _s);
    }

    function check_allSlots_sendMessage_len100(SendIn memory _in, Pre memory _p, bytes32 _s) public {
        _sendAll(100, _in, _p, _s);
    }

    function check_allSlots_sendMessage_len128(SendIn memory _in, Pre memory _p, bytes32 _s) public {
        _sendAll(128, _in, _p, _s);
    }

    // relayMessage, (O) + (X) + (S-map)
    function check_relayMessage_len0(RelayIn memory _in, Env memory _e, Pre memory _p, bytes32 _q) public {
        _relayCheck(0, _in, _e, _p, _q);
    }

    function check_relayMessage_len4(RelayIn memory _in, Env memory _e, Pre memory _p, bytes32 _q) public {
        _relayCheck(4, _in, _e, _p, _q);
    }

    function check_relayMessage_len37(RelayIn memory _in, Env memory _e, Pre memory _p, bytes32 _q) public {
        _relayCheck(37, _in, _e, _p, _q);
    }

    function check_relayMessage_len100(RelayIn memory _in, Env memory _e, Pre memory _p, bytes32 _q) public {
        _relayCheck(100, _in, _e, _p, _q);
    }

    // relayMessage, (O) + (X) + (S-all), generic storage layout
    function check_allSlots_relayMessage_len0(RelayIn memory _in, Env memory _e, Pre memory _p, bytes32 _s) public {
        _relayAll(0, _in, _e, _p, _s);
    }

    function check_allSlots_relayMessage_len4(RelayIn memory _in, Env memory _e, Pre memory _p, bytes32 _s) public {
        _relayAll(4, _in, _e, _p, _s);
    }

    function check_allSlots_relayMessage_len37(RelayIn memory _in, Env memory _e, Pre memory _p, bytes32 _s) public {
        _relayAll(37, _in, _e, _p, _s);
    }

    function check_allSlots_relayMessage_len100(RelayIn memory _in, Env memory _e, Pre memory _p, bytes32 _s) public {
        _relayAll(100, _in, _e, _p, _s);
    }

    // relayMessage with an arbitrary payload byte string, (O) + (X) + (S-map)
    function check_relayMessage_rawPayload_len0(RawIn memory _in, Env memory _e, Pre memory _p, bytes32 _q) public {
        _rawCheck(0, _in, _e, _p, _q);
    }

    function check_relayMessage_rawPayload_len100(RawIn memory _in, Env memory _e, Pre memory _p, bytes32 _q) public {
        _rawCheck(100, _in, _e, _p, _q);
    }

    function check_relayMessage_rawPayload_len128(RawIn memory _in, Env memory _e, Pre memory _p, bytes32 _q) public {
        _rawCheck(128, _in, _e, _p, _q);
    }

    // relayMessage with a 288-byte payload whose `bytes` offset/length words are concrete and
    // non-canonical (Halmos needs them concrete); every other byte free, (O) + (X) + (S-map)
    function _badCheck(
        uint256 _offset,
        uint256 _len,
        RawIn memory _in,
        Env memory _e,
        Pre memory _p,
        bytes32 _q
    )
        internal
    {
        _in.words[5] = bytes32(_offset);
        _in.words[6] = bytes32(_len);
        _rawCheck(288, _in, _e, _p, _q);
    }

    function check_relayMessage_badEncoding_len33(RawIn memory _in, Env memory _e, Pre memory _p, bytes32 _q) public {
        _badCheck(0x40, 33, _in, _e, _p, _q);
    }

    function check_relayMessage_badEncoding_len65(RawIn memory _in, Env memory _e, Pre memory _p, bytes32 _q) public {
        _badCheck(0x40, 65, _in, _e, _p, _q);
    }

    function check_relayMessage_badEncoding_hugeLen(RawIn memory _in, Env memory _e, Pre memory _p, bytes32 _q) public {
        _badCheck(0x40, 1 << 64, _in, _e, _p, _q);
    }

    function check_relayMessage_badEncoding_offset20(
        RawIn memory _in,
        Env memory _e,
        Pre memory _p,
        bytes32 _q
    )
        public
    {
        _badCheck(0x20, 32, _in, _e, _p, _q);
    }

    function check_relayMessage_badEncoding_offset60(
        RawIn memory _in,
        Env memory _e,
        Pre memory _p,
        bytes32 _q
    )
        public
    {
        _in.words[7] = bytes32(0); // the length word an offset of 0x60 points at: empty bytes
        _badCheck(0x60, 0, _in, _e, _p, _q);
    }

    function check_relayMessage_badEncoding_hugeOffset(
        RawIn memory _in,
        Env memory _e,
        Pre memory _p,
        bytes32 _q
    )
        public
    {
        _badCheck(1 << 64, 0, _in, _e, _p, _q);
    }

    // Non-vacuity witnesses: each must produce a valid counterexample (run.sh checks that).

    /// @notice E1 is real for send.
    function check_nonvacuity_sendMessage_unsafeTarget(SendIn memory _in, Pre memory _p) public {
        vm.assume(!_safe(_in.target));
        _seed(_p);
        _send(37, _in);
    }

    /// @notice E2 is real: OLD does not hold the timestamp NEW writes.
    function check_nonvacuity_sendMessage_timestampSlot(SendIn memory _in, Pre memory _p) public {
        vm.assume(_safe(_in.target));
        _seed(_p);
        (bool ok, bytes32 h) = _send(37, _in);
        if (ok) assert(vm.load(OLD, _mapSlot(h, 3)) == vm.load(NEW, _mapSlot(h, 3)));
    }

    /// @notice Reachability: a successful send is explored.
    function check_nonvacuity_sendMessage_successReachable(SendIn memory _in, Pre memory _p) public {
        vm.assume(_safe(_in.target));
        _seed(_p);
        (bool ok,) = _send(37, _in);
        assert(!ok);
    }

    /// @notice E1 is real for relay: the target mock vs an unsafe target (no code here).
    function check_nonvacuity_relayMessage_unsafeTarget(RelayIn memory _in, Env memory _e, Pre memory _p) public {
        address t = address(uint160(uint256(_in.targetWord)));
        vm.assume(t == TARGET || !_safe(t));
        _seed(_p);
        _relay(_relayCalldata(37, _in, _e), _in.value, _e);
    }

    /// @notice Reachability: a relay reaches the target mock, which sees the right context.
    function check_nonvacuity_relayMessage_successReachable(RelayIn memory _in, Env memory _e, Pre memory _p) public {
        vm.assume(address(uint160(uint256(_in.targetWord))) == TARGET);
        _seed(_p);
        assert(!_relaySeesContext(37, _in, _e));
    }

    /// @notice Reachability: a 288-byte payload with offset 0x40 and length 32 (the other words
    ///         free) decodes and reaches the target mock.
    function check_nonvacuity_relayMessage_badEncodingReachesTarget(RawIn memory _in, Env memory _e) public {
        vm.assume(address(uint160(uint256(_in.words[2]))) == TARGET);
        _in.words[5] = bytes32(uint256(0x40));
        _in.words[6] = bytes32(uint256(32));
        _configure(_e);
        (bool ok,) = NEW.call(_rawCalldata(288, _in));
        assert(!ok || _perCaller(TARGET, NEW, 4) == bytes32(0));
    }
}
