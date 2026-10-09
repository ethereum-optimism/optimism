// SPDX-License-Identifier: MIT
pragma solidity 0.8.25;

// Concrete tests (forge test) next to the symbolic checks in L2ToL2Equivalence.t.sol:
//   - L2ToL2CrossDomainMessenger_EquivalenceLogs_Test: differential fuzz of the one observable the
//     symbolic checks cannot assert on, emitted events. Same setup (develop vs current runtime
//     bytecode at OLD/NEW, same mocks, same exclusions E1-E6). Both must emit the same logs
//     (count, topics, data; the emitter differs by construction) and, as a cross-check of the
//     proofs, return the same outcome and data. This samples inputs; it is testing, not proof.
//   - L2ToL2CrossDomainMessenger_EquivalenceWitness_Test: one concrete input per non-vacuity
//     check that makes the symbolic check's assertion fail, so each witness is an executable
//     counterexample (Halmos marks models that mention keccak "potentially invalid").
// run.sh runs both; they also run in the repo's forge test.

// Testing
import { Vm } from "forge-std/Vm.sol";
import { stdError } from "forge-std/StdError.sol";
import {
    EquivalenceIdentifier,
    IEquivalenceMessenger,
    L2ToL2CrossDomainMessenger_EquivalenceBase,
    L2ToL2CrossDomainMessenger_EquivalenceHalmos
} from "./L2ToL2Equivalence.t.sol";

/// @notice Smallest salt >= `_start` whose keccak bit (as the mocks compute it) is clear.
function quietSalt(bytes32 _start, bytes memory _rest) pure returns (bytes32 salt_) {
    salt_ = _start;
    while (uint256(keccak256(abi.encodePacked(salt_, _rest))) & 1 == 1) {
        unchecked {
            salt_ = bytes32(uint256(salt_) + 1);
        }
    }
}

contract L2ToL2CrossDomainMessenger_EquivalenceLogs_Test is L2ToL2CrossDomainMessenger_EquivalenceBase {
    address internal constant MESSENGER = 0x4200000000000000000000000000000000000023;
    bytes32 internal constant RELAYED_MESSAGE_TOPIC = keccak256("RelayedMessage(uint256,uint256,bytes32,bytes32)");

    struct RelayFuzz {
        EquivalenceIdentifier id;
        bool useEoa;
        uint96 targetHigh;
        uint256 nonce;
        address sender;
        bytes message;
        uint8 flags;
        bytes32 inboxSalt;
        bytes32 targetSalt;
        bytes32 w0;
        uint128 value;
    }

    /// @notice Calls `_who` with `_cd`, returning the outcome and the logs it emitted.
    function _callAndRecord(
        address _who,
        address _caller,
        uint256 _value,
        bytes memory _cd
    )
        internal
        returns (bool ok_, bytes memory ret_, Vm.Log[] memory logs_)
    {
        vm.recordLogs();
        vm.prank(_caller);
        (ok_, ret_) = _who.call{ value: _value }(_cd);
        logs_ = vm.getRecordedLogs();
    }

    /// @notice Asserts both log lists are the same up to the emitter (OLD vs NEW).
    function _assertSameLogs(Vm.Log[] memory _a, Vm.Log[] memory _b) internal pure {
        assertEq(_a.length, _b.length, "log count");
        for (uint256 i = 0; i < _a.length; i++) {
            assertEq(_a[i].topics.length, _b[i].topics.length, "topic count");
            for (uint256 j = 0; j < _a[i].topics.length; j++) {
                assertEq(_a[i].topics[j], _b[i].topics[j], "topic");
            }
            assertEq(_a[i].data, _b[i].data, "data");
            bool messengerLog = _a[i].emitter == OLD && _b[i].emitter == NEW;
            assertTrue(messengerLog || _a[i].emitter == _b[i].emitter, "emitter");
        }
    }

    /// @notice Calls OLD and NEW with the same calldata; asserts the same outcome, data and logs.
    ///         Returns NEW's outcome and logs.
    function _diff(
        address _caller,
        uint256 _value,
        bytes memory _cd
    )
        internal
        returns (bool ok_, Vm.Log[] memory logs_)
    {
        (bool okO, bytes memory rO, Vm.Log[] memory lO) = _callAndRecord(OLD, _caller, _value, _cd);
        (bool okN, bytes memory rN, Vm.Log[] memory lN) = _callAndRecord(NEW, _caller, _value, _cd);
        assertEq(okO, okN, "outcome");
        assertEq(_outKey(okO, rO), _outKey(okN, rN), "return data (revert data modulo E6)");
        _assertSameLogs(lO, lN);
        ok_ = okN;
        logs_ = lN;
    }

    /// @notice Payload: well-formed unless a flag pair asks for a wrong selector, destination or a
    ///         dirty target word.
    function _payload(RelayFuzz memory _f) internal view returns (bytes memory payload_) {
        bytes32 selWord = (_f.flags & 0x81) == 0x81 ? bytes32(0) : SENT_MESSAGE_SELECTOR;
        uint256 dest = (_f.flags & 0x42) == 0x42 ? block.chainid + 1 : block.chainid;
        bytes32 targetWord = bytes32(uint256(uint160(_f.useEoa ? EOA : TARGET)));
        if ((_f.flags & 0x60) == 0x60) targetWord |= bytes32(uint256(_f.targetHigh) << 160);
        payload_ = abi.encodePacked(abi.encode(selWord, dest, targetWord, _f.nonce), abi.encode(_f.sender, _f.message));
    }

    function _configureFuzz(RelayFuzz memory _f) internal {
        _configure(
            Env({
                inboxSalt: _f.inboxSalt,
                targetSalt: _f.targetSalt,
                w0: _f.w0,
                w1: bytes32(0),
                reenter: (_f.flags & 0x10) != 0 ? 1 : 0,
                successfulWord: bytes32(0)
            })
        );
        vm.deal(CALLER, uint256(_f.value) * 2);
    }

    /// forge-config: ciheavy.fuzz.runs = 1000
    /// @notice sendMessage: same SentMessage event (and outcome), any target except E1.
    function testFuzz_sendMessage_sameLogs_succeeds(
        uint256 _dest,
        address _target,
        address _sender,
        bytes32 _nonceWord,
        uint64 _ts,
        bytes calldata _message
    )
        external
    {
        vm.assume(_safe(_target));
        vm.warp(_ts);
        vm.store(OLD, bytes32(uint256(1)), _nonceWord);
        vm.store(NEW, bytes32(uint256(1)), _nonceWord);
        _diff(_sender, 0, abi.encodeCall(IEquivalenceMessenger.sendMessage, (_dest, _target, _message)));
    }

    /// forge-config: ciheavy.fuzz.runs = 1000
    /// @notice relayMessage from the real origin (0x..23), any mock behaviour and payload flags.
    function testFuzz_relayMessage_sameLogs_succeeds(RelayFuzz memory _f) external {
        _f.id.origin = MESSENGER;
        _configureFuzz(_f);
        _diff(CALLER, _f.value, abi.encodeCall(IEquivalenceMessenger.relayMessage, (_f.id, _payload(_f))));
    }

    /// forge-config: ciheavy.fuzz.runs = 1000
    /// @notice relayMessage on the success path every run: real origin, well-formed payload to the
    ///         target mock, and mock salts chosen so neither mock reverts. Both must succeed and emit
    ///         the same single RelayedMessage.
    function testFuzz_relayMessage_successPath_succeeds(RelayFuzz memory _f) external {
        _f.id.origin = MESSENGER;
        _f.flags = 0;
        _f.useEoa = false;
        bytes memory payload = _payload(_f);
        bytes memory cd = abi.encodeCall(IEquivalenceMessenger.relayMessage, (_f.id, payload));

        bytes32 argsHash = keccak256(abi.encode(_f.id, keccak256(payload)));
        _f.inboxSalt = quietSalt(_f.inboxSalt, abi.encode(argsHash));
        bytes32 dataHash = keccak256(_f.message);
        _f.targetSalt = quietSalt(_f.targetSalt, abi.encode(dataHash, uint256(_f.value)));
        // abi.encode(salt, x, ...) == abi.encodePacked(salt, abi.encode(x, ...)) for static words.
        _configureFuzz(_f);

        (bool ok, Vm.Log[] memory logs) = _diff(CALLER, _f.value, cd);
        assertTrue(ok, "relay succeeded");
        assertEq(logs.length, 1, "one log");
        assertEq(logs[0].topics[0], RELAYED_MESSAGE_TOPIC, "RelayedMessage");
    }

    /// forge-config: ciheavy.fuzz.runs = 1000
    /// @notice relayMessage with any other origin: both reverts must match (no logs).
    function testFuzz_relayMessage_invalidOrigin_succeeds(RelayFuzz memory _f) external {
        vm.assume(_f.id.origin != MESSENGER);
        _configureFuzz(_f);
        (bool ok, Vm.Log[] memory logs) =
            _diff(CALLER, _f.value, abi.encodeCall(IEquivalenceMessenger.relayMessage, (_f.id, _payload(_f))));
        assertFalse(ok, "reverted");
        assertEq(logs.length, 0, "no logs");
    }
}

/// @notice Concrete counterexamples for the Halmos non-vacuity checks: each call must hit the
///         check's assertion (Panic 0x01).
contract L2ToL2CrossDomainMessenger_EquivalenceWitness_Test is L2ToL2CrossDomainMessenger_EquivalenceHalmos {
    address internal constant MESSENGER = 0x4200000000000000000000000000000000000023;

    function _sendIn(address _target) internal view returns (SendIn memory in_) {
        in_ = SendIn({
            dest: block.chainid + 1,
            target: _target,
            sender: address(0xBEEF),
            ts: 1000,
            m0: bytes32(0),
            m1: bytes32(0),
            m2: bytes32(0),
            m3: bytes32(0)
        });
    }

    function _relayIn(address _target) internal view returns (RelayIn memory in_) {
        in_.id = EquivalenceIdentifier(MESSENGER, 1, 2, 3, 901);
        in_.selWord = SENT_MESSAGE_SELECTOR;
        in_.dest = block.chainid;
        in_.targetWord = bytes32(uint256(uint160(_target)));
        in_.nonce = 7;
        in_.senderWord = bytes32(uint256(0xBEEF));
    }

    /// @notice Salts under which neither mock reverts for this relay (37-byte zero message).
    function _quietEnv(RelayIn memory _in) internal pure returns (Env memory e_) {
        bytes memory m = _msg(37, _in.m0, _in.m1, _in.m2, _in.m3);
        bytes memory payload = abi.encodePacked(
            abi.encode(_in.selWord, _in.dest, _in.targetWord, _in.nonce), abi.encode(_in.senderWord, m)
        );
        e_.inboxSalt = quietSalt(bytes32(0), abi.encode(keccak256(abi.encode(_in.id, keccak256(payload)))));
        e_.targetSalt = quietSalt(bytes32(0), abi.encode(keccak256(m), uint256(0)));
    }

    function test_sendMessage_unsafeTargetWitness_succeeds() external {
        Pre memory p;
        vm.expectRevert(stdError.assertionError);
        this.check_nonvacuity_sendMessage_unsafeTarget(_sendIn(L2CDM), p);
    }

    function test_sendMessage_timestampSlotWitness_succeeds() external {
        Pre memory p;
        vm.expectRevert(stdError.assertionError);
        this.check_nonvacuity_sendMessage_timestampSlot(_sendIn(address(0xCAFE)), p);
    }

    function test_sendMessage_successReachableWitness_succeeds() external {
        Pre memory p;
        vm.expectRevert(stdError.assertionError);
        this.check_nonvacuity_sendMessage_successReachable(_sendIn(address(0xCAFE)), p);
    }

    function test_relayMessage_unsafeTargetWitness_succeeds() external {
        RelayIn memory r = _relayIn(L2CDM);
        Pre memory p;
        vm.expectRevert(stdError.assertionError);
        this.check_nonvacuity_relayMessage_unsafeTarget(r, _quietEnv(r), p);
    }

    function test_relayMessage_successReachableWitness_succeeds() external {
        RelayIn memory r = _relayIn(TARGET);
        Pre memory p;
        vm.expectRevert(stdError.assertionError);
        this.check_nonvacuity_relayMessage_successReachable(r, _quietEnv(r), p);
    }

    function test_relayMessage_badEncodingWitness_succeeds() external {
        RawIn memory raw;
        raw.id = EquivalenceIdentifier(MESSENGER, 1, 2, 3, 901);
        raw.words[0] = SENT_MESSAGE_SELECTOR;
        raw.words[1] = bytes32(block.chainid);
        raw.words[2] = bytes32(uint256(uint160(TARGET)));
        raw.words[3] = bytes32(uint256(7));
        raw.words[4] = bytes32(uint256(0xBEEF));
        raw.words[5] = bytes32(uint256(0x40));
        raw.words[6] = bytes32(uint256(32));
        Env memory e;
        bytes memory payload = abi.encodePacked(raw.words);
        e.inboxSalt = quietSalt(bytes32(0), abi.encode(keccak256(abi.encode(raw.id, keccak256(payload)))));
        e.targetSalt = quietSalt(bytes32(0), abi.encode(keccak256(abi.encodePacked(bytes32(0))), uint256(0)));
        vm.expectRevert(stdError.assertionError);
        this.check_nonvacuity_relayMessage_badEncodingReachesTarget(raw, e);
    }
}
