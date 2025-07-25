// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing utilities
import { Test } from "forge-std/Test.sol";
import { CommonTest } from "test/setup/CommonTest.sol";
import { VmSafe } from "forge-std/Vm.sol";

// Interfaces
import { ICrossL2Inbox, Identifier } from "interfaces/L2/ICrossL2Inbox.sol";

/// @title ValidateMessageRelayer
/// @notice For test contract used to validate multiple messages in a single tx.
contract ValidateMessageRelayer is Test {
    ICrossL2Inbox public immutable CROSS_L2_INBOX;

    constructor(address _crossL2Inbox) {
        CROSS_L2_INBOX = ICrossL2Inbox(_crossL2Inbox);
    }

    /// @notice Validates a message and retries it after it reverts.
    function validateAndRetry(Identifier memory _id, bytes32 _messageHash) external {
        try CROSS_L2_INBOX.validateMessage(_id, _messageHash) {
            // It should always revert
            assertFalse(true);
        } catch {
            // It should revert with NotInAccessList when called a second time without any access
            // list
            vm.expectRevert(ICrossL2Inbox.NotInAccessList.selector);
            CROSS_L2_INBOX.validateMessage(_id, _messageHash);
        }
    }

    /// @notice Validates multiple messages in a single tx.
    function validateMessages(Identifier[20] memory _ids, bytes32[20] memory _messageHashes) external {
        for (uint256 i; i < _ids.length; i++) {
            CROSS_L2_INBOX.validateMessage(_ids[i], _messageHashes[i]);
        }
    }
}

/// @title CrossL2Inbox_Test_Init
/// @notice Reusable test initialization for `CrossL2Inbox` tests.
contract CrossL2Inbox_TestInit is CommonTest {
    event ExecutingMessage(bytes32 indexed msgHash, Identifier id);

    ValidateMessageRelayer public validateMessageRelayer;

    mapping(bytes32 => bool) public relayedMessages;
    mapping(bytes32 => bool) public warmedSlots;

    function setUp() public override {
        useInteropOverride = true;
        super.setUp();
        validateMessageRelayer = new ValidateMessageRelayer(address(crossL2Inbox));
    }
}

/// @title CrossL2Inbox_ValidateMessage_Test
/// @notice Tests the `validateMessage` function of the `CrossL2Inbox` contract.
contract CrossL2Inbox_ValidateMessage_Test is CrossL2Inbox_TestInit {
    /// @notice Test that `validateMessage` reverts when the slot is not warm.
    function testFuzz_validateMessage_accessList_reverts(Identifier memory _id, bytes32 _messageHash) external {
        // Bound values types to ensure they are not too large
        _id.blockNumber = bound(_id.blockNumber, 0, type(uint64).max);
        _id.logIndex = bound(_id.logIndex, 0, type(uint32).max);
        _id.timestamp = bound(_id.timestamp, 0, type(uint64).max);

        // Cold all the slots
        vm.cool(address(crossL2Inbox));

        // Expect revert
        vm.expectRevert(ICrossL2Inbox.NotInAccessList.selector);
        crossL2Inbox.validateMessage(_id, _messageHash);
    }

    /// @notice Test that `validateMessage` succeeds when the slot for the message checksum is
    ///         warm.
    /// forge-config: default.isolate = true
    function testFuzz_validateMessage_succeeds(
        Identifier memory _id,
        bytes32 _messageHash
    )
        public
        returns (bytes32 slot_)
    {
        // Bound values types to ensure they are not too large
        _id.blockNumber = bound(_id.blockNumber, 0, type(uint64).max);
        _id.logIndex = bound(_id.logIndex, 0, type(uint32).max);
        _id.timestamp = bound(_id.timestamp, 0, type(uint64).max);

        // Prepare the access list to be sent with the next call
        slot_ = crossL2Inbox.calculateChecksum(_id, _messageHash);
        bytes32[] memory slots = new bytes32[](1);
        slots[0] = slot_;
        VmSafe.AccessListItem[] memory accessList = new VmSafe.AccessListItem[](1);
        accessList[0] = VmSafe.AccessListItem({ target: address(crossL2Inbox), storageKeys: slots });

        // Expect `ExecutingMessage` event to be emitted
        vm.expectEmit(address(crossL2Inbox));
        emit ExecutingMessage(_messageHash, _id);

        // Validate the message
        vm.accessList(accessList);
        crossL2Inbox.validateMessage(_id, _messageHash);
    }

    /// @notice Test that multiple calls to `validateMessage` with different access lists don't
    ///         collide and succeeds.
    /// @dev This tests that the way we encode and hash the checksum slot is unique enough to avoid
    ///      collisions.
    /// forge-config: default.isolate = true
    function testFuzz_validateMessage_multipleAccessLists_succeeds(
        Identifier[20] memory _ids,
        bytes32[20] calldata _messageHash
    )
        external
    {
        // Send batches of calls with different access lists and check they never collide and
        // always succeed
        for (uint256 i; i < _ids.length; i++) {
            // Make sure we're not re-validating the same message
            bytes32 msgToValidate = keccak256(abi.encode(_ids[i], _messageHash[i]));
            vm.assume(relayedMessages[msgToValidate] == false);
            relayedMessages[msgToValidate] = true;

            // Call validateMessage and get the slot
            bytes32 slot_ = testFuzz_validateMessage_succeeds(_ids[i], _messageHash[i]);
            // Check that the slot doesn't match a previously warmed slot
            assertEq(warmedSlots[slot_], false);
            // Mark the slot as warmed
            warmedSlots[slot_] = true;

            // Remove the access list
            vm.noAccessList();
        }
    }

    /// @notice Test that an invalid tx calling `validateMessage` doesn't warm the slot for the
    ///         next one.
    /// forge-config: default.isolate = true
    function test_validateMessage_revertDoesntWarm_reverts(
        Identifier memory _idOne,
        Identifier memory _idTwo,
        bytes32 _messageHashOne,
        bytes32 _messageHashTwo
    )
        external
    {
        // Bound values types to ensure they are not too large
        _idOne.blockNumber = bound(_idOne.blockNumber, 0, type(uint64).max);
        _idOne.logIndex = bound(_idOne.logIndex, 0, type(uint32).max);
        _idOne.timestamp = bound(_idOne.timestamp, 0, type(uint64).max);
        _idTwo.blockNumber = bound(_idTwo.blockNumber, 0, type(uint64).max);
        _idTwo.logIndex = bound(_idTwo.logIndex, 0, type(uint32).max);
        _idTwo.timestamp = bound(_idTwo.timestamp, 0, type(uint64).max);

        // Make sure the first message is valid
        bytes32 slotTwo = crossL2Inbox.calculateChecksum(_idTwo, _messageHashTwo);
        bytes32[] memory slots = new bytes32[](1);
        slots[0] = slotTwo;
        VmSafe.AccessListItem[] memory accessList = new VmSafe.AccessListItem[](1);
        accessList[0] = VmSafe.AccessListItem({ target: address(crossL2Inbox), storageKeys: slots });

        // Expect a revert on the tx1 warming the slot two
        vm.expectRevert(ICrossL2Inbox.NotInAccessList.selector);
        vm.accessList(accessList);
        crossL2Inbox.validateMessage(_idOne, _messageHashOne);

        // Send the tx2 but without any access list and check that it reverts since the slot should not be warmed
        vm.expectRevert(ICrossL2Inbox.NotInAccessList.selector);
        crossL2Inbox.validateMessage(_idTwo, _messageHashTwo);
    }

    /// @notice Test that a valid tx calling `validateMessage` doesn't warm the slot for the next
    ///         one.
    /// forge-config: default.isolate = true
    function test_validateMessage_validDoesntWarm_reverts(Identifier memory _id, bytes32 _messageHash) external {
        // Bound values types to ensure they are not too large
        _id.blockNumber = bound(_id.blockNumber, 0, type(uint64).max);
        _id.logIndex = bound(_id.logIndex, 0, type(uint32).max);
        _id.timestamp = bound(_id.timestamp, 0, type(uint64).max);

        // Make sure the first message is valid
        bytes32 slotOne = crossL2Inbox.calculateChecksum(_id, _messageHash);
        bytes32[] memory slots = new bytes32[](1);
        slots[0] = slotOne;
        VmSafe.AccessListItem[] memory accessList = new VmSafe.AccessListItem[](1);
        accessList[0] = VmSafe.AccessListItem({ target: address(crossL2Inbox), storageKeys: slots });

        // Expect `ExecutingMessage` event to be emitted
        vm.expectEmit(address(crossL2Inbox));
        emit ExecutingMessage(_messageHash, _id);

        // Validate the message
        vm.accessList(accessList);
        crossL2Inbox.validateMessage(_id, _messageHash);

        // Send the same msg but without any access list and check that it reverts since the
        // slot should not be warmed
        vm.expectRevert(ICrossL2Inbox.NotInAccessList.selector);
        crossL2Inbox.validateMessage(_id, _messageHash);
    }

    /// @notice Test that an invalid message without access list does not succeed warm the slot and
    ///         fails the second time.
    function test_validateMessage_sameMsgWithoutAccessListTwice_reverts(
        Identifier memory _id,
        bytes32 _messageHash
    )
        public
    {
        // Make sure the Identifier is valid
        _id.blockNumber = bound(_id.blockNumber, 0, type(uint64).max);
        _id.logIndex = bound(_id.logIndex, 0, type(uint32).max);
        _id.timestamp = bound(_id.timestamp, 0, type(uint64).max);

        // Try and retry the message without any access list
        vm.expectCall(address(crossL2Inbox), abi.encodeCall(ICrossL2Inbox.validateMessage, (_id, _messageHash)), 2);
        validateMessageRelayer.validateAndRetry(_id, _messageHash);
    }

    /// @notice Test that multiple calls to `validateMessage` with multiple storage keys succeeds
    ///         on the same tx.
    /// forge-config: default.isolate = true
    function test_validateMessage_multipleStorageKeys_succeeds(
        Identifier[20] memory _ids,
        bytes32[20] memory _messageHashes
    )
        public
    {
        bytes32[] memory slots = new bytes32[](_ids.length);
        for (uint256 i; i < _ids.length; i++) {
            // Make sure the Identifier is valid
            _ids[i].blockNumber = bound(_ids[i].blockNumber, 0, type(uint64).max);
            _ids[i].logIndex = bound(_ids[i].logIndex, 0, type(uint32).max);
            _ids[i].timestamp = bound(_ids[i].timestamp, 0, type(uint64).max);

            // Calculate the checksum for the message and add it to the storage keys
            bytes32 slot = crossL2Inbox.calculateChecksum(_ids[i], _messageHashes[i]);
            slots[i] = slot;
        }

        // Prepare the access list to be sent with the next txs
        VmSafe.AccessListItem[] memory accessList = new VmSafe.AccessListItem[](1);
        accessList[0] = VmSafe.AccessListItem({ target: address(crossL2Inbox), storageKeys: slots });

        // Expect `ExecutingMessage` events to be emitted
        for (uint256 i; i < _ids.length; i++) {
            vm.expectEmit(address(crossL2Inbox));
            emit ExecutingMessage(_messageHashes[i], _ids[i]);
        }

        // Validate the message
        vm.accessList(accessList);
        validateMessageRelayer.validateMessages(_ids, _messageHashes);
    }
}

/// @title CrossL2Inbox_CalculateChecksum_Test
/// @notice Tests the `calculateChecksum` function of the `CrossL2Inbox` contract.
contract CrossL2Inbox_CalculateChecksum_Test is CrossL2Inbox_TestInit {
    /// @notice Test that `calculateChecksum` reverts when the block number is greater than 2^64.
    function testFuzz_calculateChecksum_withTooLargeBlockNumber_reverts(
        Identifier memory _id,
        bytes32 _messageHash
    )
        external
    {
        // Set to the 2**64 + 1
        _id.blockNumber = 18446744073709551615 + 1;
        vm.expectRevert(ICrossL2Inbox.BlockNumberTooHigh.selector);
        crossL2Inbox.calculateChecksum(_id, _messageHash);
    }

    /// @notice Test that `calculateChecksum` reverts when the log index is greater than 2^32.
    function testFuzz_calculateChecksum_withTooLargeLogIndex_reverts(
        Identifier memory _id,
        bytes32 _messageHash
    )
        external
    {
        _id.blockNumber = bound(_id.blockNumber, 0, type(uint64).max);

        // Set to the 2**32 + 1
        _id.logIndex = 4294967295 + 1;
        vm.expectRevert(ICrossL2Inbox.LogIndexTooHigh.selector);
        crossL2Inbox.calculateChecksum(_id, _messageHash);
    }

    /// @notice Test that `calculateChecksum` reverts when the timestamp is greater than 2^64.
    function testFuzz_calculateChecksum_withTooLargeTimestamp_reverts(
        Identifier memory _id,
        bytes32 _messageHash
    )
        external
    {
        _id.blockNumber = bound(_id.blockNumber, 0, type(uint64).max);
        _id.logIndex = bound(_id.logIndex, 0, type(uint32).max);

        // Set to the 2**64 + 1
        _id.timestamp = 18446744073709551615 + 1;
        vm.expectRevert(ICrossL2Inbox.TimestampTooHigh.selector);
        crossL2Inbox.calculateChecksum(_id, _messageHash);
    }

    /// @notice Test that `calculateChecksum` succeeds matching the expected calculated checksum.
    /// @dev Using a hardcoded checksum manually calculated and verified.
    function test_calculateChecksum_succeeds() external view {
        Identifier memory id = Identifier(
            address(0),
            uint64(0xa1a2a3a4a5a6a7a8),
            uint32(0xb1b2b3b4),
            uint64(0xc1c2c3c4c5c6c7c8),
            uint256(0xd1d2d3d4d5d6d7d8)
        );

        // Calculate the expected checksum.
        bytes32 messageHash = 0x8017559a85b12c04b14a1a425d53486d1015f833714a09bd62f04152a7e2ae9b;
        bytes32 checksum = crossL2Inbox.calculateChecksum(id, messageHash);
        bytes32 expectedChecksum = 0x03139ddd21106abad4bb82800fedfa3a103f53f242c2d5b7615b0baad8379531;

        // Expect it to match
        assertEq(checksum, expectedChecksum);
    }
}
