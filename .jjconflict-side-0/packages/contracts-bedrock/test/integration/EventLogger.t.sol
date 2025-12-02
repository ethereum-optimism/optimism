// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

// Testing
import { Test } from "test/setup/Test.sol";
import { VmSafe } from "forge-std/Vm.sol";

// Contracts
import { EventLogger } from "../../src/integration/EventLogger.sol";
import { CrossL2Inbox } from "src/L2/CrossL2Inbox.sol";

// Libraries
import { Predeploys } from "src/libraries/Predeploys.sol";

// Interfaces
import { Identifier as IfaceIdentifier } from "interfaces/L2/ICrossL2Inbox.sol";
import { ICrossL2Inbox, Identifier as ImplIdentifier } from "interfaces/L2/ICrossL2Inbox.sol";

/// @title EventLogger_TestInit
/// @notice Reusable test initialization for `EventLogger` tests.
abstract contract EventLogger_TestInit is Test {
    event ExecutingMessage(bytes32 indexed msgHash, ImplIdentifier id);

    EventLogger eventLogger;

    function setUp() public {
        // Deploy EventLogger contract
        eventLogger = new EventLogger();
        vm.label(address(eventLogger), "EventLogger");

        vm.etch(Predeploys.CROSS_L2_INBOX, address(new CrossL2Inbox()).code);
        vm.label(Predeploys.CROSS_L2_INBOX, "CrossL2Inbox");
    }
}

/// @title EventLogger_EmitLog_Test
/// @notice Tests the `emitLog` function of the `EventLogger` contract.
contract EventLogger_EmitLog_Test is EventLogger_TestInit {
    /// @notice Test logging
    function test_emitLog_succeeds(
        uint256 topicCount,
        bytes32 t0,
        bytes32 t1,
        bytes32 t2,
        bytes32 t3,
        bytes memory data
    )
        external
    {
        bytes32[] memory topics = new bytes32[](topicCount % 5);
        if (topics.length == 0) {
            vm.expectEmitAnonymous();
            assembly {
                log0(add(data, 32), mload(data))
            }
        } else if (topics.length == 1) {
            topics[0] = t0;
            vm.expectEmit(false, false, false, true);
            assembly {
                log1(add(data, 32), mload(data), t0)
            }
        } else if (topics.length == 2) {
            topics[0] = t0;
            topics[1] = t1;
            vm.expectEmit(true, false, false, true);
            assembly {
                log2(add(data, 32), mload(data), t0, t1)
            }
        } else if (topics.length == 3) {
            topics[0] = t0;
            topics[1] = t1;
            topics[2] = t2;
            vm.expectEmit(true, true, false, true);
            assembly {
                log3(add(data, 32), mload(data), t0, t1, t2)
            }
        } else if (topics.length == 4) {
            topics[0] = t0;
            topics[1] = t1;
            topics[2] = t2;
            topics[3] = t3;
            vm.expectEmit(true, true, true, true);
            assembly {
                log4(add(data, 32), mload(data), t0, t1, t2, t3)
            }
        }
        eventLogger.emitLog(topics, data);
    }

    /// @notice It should revert if called with 5 topics
    function test_emitLog_5topics_reverts() external {
        bytes32[] memory topics = new bytes32[](5); // 5 or more topics: not possible to log
        bytes memory empty = new bytes(0);
        vm.expectRevert(empty);
        eventLogger.emitLog(topics, empty);
    }
}

/// @title EventLogger_ValidateMessage_Test
/// @notice Tests the `validateMessage` function of the `EventLogger` contract.
contract EventLogger_ValidateMessage_Test is EventLogger_TestInit {
    /// @notice It should succeed with any Identifier
    /// forge-config: default.isolate = true
    function test_validateMessage_succeeds(
        address _origin,
        uint64 _blockNumber,
        uint32 _logIndex,
        uint64 _timestamp,
        uint256 _chainId,
        bytes32 _msgHash
    )
        external
    {
        IfaceIdentifier memory idIface = IfaceIdentifier({
            origin: _origin,
            blockNumber: _blockNumber,
            logIndex: _logIndex,
            timestamp: _timestamp,
            chainId: _chainId
        });
        ImplIdentifier memory idImpl = ImplIdentifier({
            origin: _origin,
            blockNumber: _blockNumber,
            logIndex: _logIndex,
            timestamp: _timestamp,
            chainId: _chainId
        });

        address emitter = Predeploys.CROSS_L2_INBOX;

        // Cool the contract's slots
        vm.cool(address(emitter));

        // Calculate the checksum and prepare the access list
        bytes32 checksum = ICrossL2Inbox(emitter).calculateChecksum(idImpl, _msgHash);
        bytes32[] memory slots = new bytes32[](1);
        slots[0] = checksum;
        VmSafe.AccessListItem[] memory accessList = new VmSafe.AccessListItem[](1);
        accessList[0] = VmSafe.AccessListItem({ target: address(emitter), storageKeys: slots });

        vm.expectEmit(false, false, false, true, emitter);
        emit ExecutingMessage(_msgHash, idImpl);

        // Call with access list
        vm.accessList(accessList);
        eventLogger.validateMessage(idIface, _msgHash);
    }
}
