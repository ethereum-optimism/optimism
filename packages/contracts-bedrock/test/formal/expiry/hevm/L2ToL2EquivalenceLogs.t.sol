// SPDX-License-Identifier: MIT
pragma solidity 0.8.25;

// Concrete differential fuzz (forge test), complementing the hevm proofs in L2ToL2Equivalence.t.sol on the one
// observable hevm cannot assert on: emitted events. Same setup (develop vs current runtime bytecode at OLD/NEW,
// same mocks, same exclusions E1-E5); for random inputs it checks that both emit the same logs (same count, topics
// and data; the emitter differs by construction, OLD vs NEW) and, as a cross-check of the proofs, the same
// success flag and return data. This is testing, not proof: it samples inputs, it does not cover them all.

import { Test, Vm } from "forge-std/Test.sol";
import { L2ToL2Bytecodes } from "./L2ToL2Bytecodes.sol";
import { HevmIdentifier, IHevmMessenger, HevmInboxMock, HevmTargetMock } from "./L2ToL2Equivalence.t.sol";

contract L2ToL2EquivalenceLogs_Test is Test {
    address internal constant OLD = address(0x0000000000000000000000000000000000010000);
    address internal constant NEW = address(0x0000000000000000000000000000000000020000);
    address internal constant TARGET = address(0x0000000000000000000000000000000000030000);
    address internal constant EOA = address(0x0000000000000000000000000000000000040000);
    address internal constant L2CDM = 0x4200000000000000000000000000000000000007;
    address internal constant PASSER = 0x4200000000000000000000000000000000000016;
    address internal constant INBOX = 0x4200000000000000000000000000000000000022;
    bytes32 internal constant SENT_MESSAGE_SELECTOR =
        0x382409ac69001e11931a28435afef442cbfd20d9891907e8fa373ba7d351f320;

    function setUp() public {
        vm.etch(OLD, L2ToL2Bytecodes.DEVELOP);
        vm.etch(NEW, L2ToL2Bytecodes.CURRENT);
        vm.etch(INBOX, address(new HevmInboxMock()).code);
        vm.etch(TARGET, address(new HevmTargetMock()).code);
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

    function _assertSameLogs(Vm.Log[] memory _a, Vm.Log[] memory _b) internal pure {
        assertEq(_a.length, _b.length, "log count");
        for (uint256 i = 0; i < _a.length; i++) {
            assertEq(_a[i].topics.length, _b[i].topics.length, "topic count");
            for (uint256 j = 0; j < _a[i].topics.length; j++) {
                assertEq(_a[i].topics[j], _b[i].topics[j], "topic");
            }
            assertEq(_a[i].data, _b[i].data, "data");
            // The messenger's own logs come from OLD/NEW respectively; anything else from the same contract.
            bool messengerLog = _a[i].emitter == OLD && _b[i].emitter == NEW;
            assertTrue(messengerLog || _a[i].emitter == _b[i].emitter, "emitter");
        }
    }

    /// @notice Calls OLD and NEW with the same calldata and asserts the same outcome, return data and logs.
    function _diff(address _caller, uint256 _value, bytes memory _cd) internal {
        (bool okO, bytes memory rO, Vm.Log[] memory lO) = _callAndRecord(OLD, _caller, _value, _cd);
        (bool okN, bytes memory rN, Vm.Log[] memory lN) = _callAndRecord(NEW, _caller, _value, _cd);
        assertEq(okO, okN, "outcome");
        assertEq(rO, rN, "return data");
        _assertSameLogs(lO, lN);
    }

    /// @notice sendMessage: same SentMessage event (and outcome), for any target except E1 (0x..07, 0x..16).
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
        vm.assume(_target != L2CDM && _target != PASSER);
        vm.warp(_ts);
        vm.store(OLD, bytes32(uint256(1)), _nonceWord);
        vm.store(NEW, bytes32(uint256(1)), _nonceWord);
        _diff(_sender, 0, abi.encodeCall(IHevmMessenger.sendMessage, (_dest, _target, _message)));
    }

    struct RelayFuzz {
        HevmIdentifier id;
        bool useEoa;
        uint96 targetHigh;
        uint256 nonce;
        address sender;
        bytes message;
        uint8 flags;
        bytes32 w0;
        uint128 value;
    }

    function _payload(RelayFuzz memory _f) internal view returns (bytes memory) {
        // Mostly well-formed payloads that reach the target; flag pairs flip the failure paths now and then.
        bytes32 selWord = (_f.flags & 0x81) == 0x81 ? bytes32(0) : SENT_MESSAGE_SELECTOR;
        uint256 dest = (_f.flags & 0x42) == 0x42 ? block.chainid + 1 : block.chainid;
        bytes32 targetWord = bytes32(uint256(uint160(_f.useEoa ? EOA : TARGET)));
        if ((_f.flags & 0x60) == 0x60) targetWord |= bytes32(uint256(_f.targetHigh) << 160);
        return abi.encodePacked(abi.encode(selWord, dest, targetWord, _f.nonce), abi.encode(_f.sender, _f.message));
    }

    /// @notice relayMessage: same RelayedMessage event (and outcome), for TargetMock / codeless targets.
    function testFuzz_relayMessage_sameLogs_succeeds(RelayFuzz memory _f) external {
        vm.store(INBOX, bytes32(uint256(0)), bytes32(uint256((_f.flags & 0x14) == 0x14 ? 1 : 0)));
        vm.store(TARGET, bytes32(uint256(0)), bytes32(uint256((_f.flags & 0x08) != 0 ? 1 : 0)));
        vm.store(TARGET, bytes32(uint256(1)), _f.w0);
        vm.store(TARGET, bytes32(uint256(3)), bytes32(uint256((_f.flags & 0x10) != 0 ? 1 : 0)));

        vm.deal(address(0xCA11E5), uint256(_f.value) * 2);
        _diff(address(0xCA11E5), _f.value, abi.encodeCall(IHevmMessenger.relayMessage, (_f.id, _payload(_f))));
    }

    /// @notice Non-vacuity of the event comparison: the relay success path (which emits RelayedMessage) is hit.
    function test_relayMessage_emitsRelayedMessage_succeeds() external {
        HevmIdentifier memory id = HevmIdentifier(address(0x4200000000000000000000000000000000000023), 1, 2, 3, 901);
        bytes memory payload = abi.encodePacked(
            abi.encode(SENT_MESSAGE_SELECTOR, block.chainid, bytes32(uint256(uint160(TARGET))), uint256(7)),
            abi.encode(address(0xBEEF), hex"c0ffee")
        );
        bytes memory cd = abi.encodeCall(IHevmMessenger.relayMessage, (id, payload));
        (bool okO,, Vm.Log[] memory lO) = _callAndRecord(OLD, address(this), 0, cd);
        (bool okN,, Vm.Log[] memory lN) = _callAndRecord(NEW, address(this), 0, cd);
        assertTrue(okO && okN, "relay succeeded");
        assertEq(lN.length, 1, "one log");
        assertEq(lN[0].topics[0], keccak256("RelayedMessage(uint256,uint256,bytes32,bytes32)"), "RelayedMessage");
        _assertSameLogs(lO, lN);
    }
}
