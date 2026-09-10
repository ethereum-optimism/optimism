// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Test } from "test/setup/Test.sol";
import { VmSafe } from "forge-std/Vm.sol";
import { Features } from "src/libraries/Features.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";
import { PrivateProjection } from "src/libraries/PrivateProjection.sol";
import { IL1Block } from "interfaces/L2/IL1Block.sol";
import { ICrossL2Inbox, Identifier } from "interfaces/L2/ICrossL2Inbox.sol";
import { IEventReplayer } from "interfaces/private-interop/IEventReplayer.sol";
import { IL2ToL2CrossDomainMessengerReplay } from "interfaces/private-interop/IL2ToL2CrossDomainMessengerReplay.sol";

/// @notice Exercises immediate-caller authorization through a forwarding contract.
contract PrivateProjection_Forwarder_Harness {
    function forward(address _target, bytes calldata _data) external {
        (bool success, bytes memory result) = _target.call(_data);
        if (!success) {
            assembly {
                revert(add(result, 32), mload(result))
            }
        }
    }
}

/// @notice The projection feature protects protocol calls while leaving ordinary execution intact.
contract PrivateProjection_Uncategorized_Test is Test {
    address internal batcher;
    address internal outsider;

    function setUp() public {
        batcher = makeAddr("batcher");
        outsider = makeAddr("outsider");
        vm.etch(Predeploys.L1_BLOCK_ATTRIBUTES, vm.getDeployedCode("L1Block.sol:L1Block"));
        vm.etch(Predeploys.CROSS_L2_INBOX, vm.getDeployedCode("CrossL2Inbox.sol:CrossL2Inbox"));
        vm.etch(Predeploys.EVENT_REPLAYER, vm.getDeployedCode("EventReplayer.sol:EventReplayer"));
        vm.etch(
            Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER,
            vm.getDeployedCode("L2ToL2CrossDomainMessengerReplay.sol:L2ToL2CrossDomainMessengerReplay")
        );
        _feature(true);
        _batcher(batcher);
    }

    function _feature(bool _enabled) internal {
        vm.store(
            Predeploys.L1_BLOCK_ATTRIBUTES,
            keccak256(abi.encode(Features.PRIVATE_PROJECTION, uint256(9))),
            bytes32(uint256(_enabled ? 1 : 0))
        );
    }

    function _batcher(address _sender) internal {
        vm.prank(0xDeaDDEaDDeAdDeAdDEAdDEaddeAddEAdDEAd0001);
        IL1Block(Predeploys.L1_BLOCK_ATTRIBUTES).setL1BlockValues(
            0, 0, 0, bytes32(0), 0, bytes32(uint256(uint160(_sender))), 0, 0
        );
    }

    function _replay() internal {
        IL2ToL2CrossDomainMessengerReplay(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER).replaySentMessage(
            901, 0, outsider, outsider, hex"1234"
        );
    }

    function test_protocolCalls_unauthorized_reverts() external {
        Identifier memory id;
        address[6] memory targets = [
            Predeploys.EVENT_REPLAYER,
            Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER,
            Predeploys.CROSS_L2_INBOX,
            Predeploys.CROSS_L2_INBOX,
            Predeploys.CROSS_L2_INBOX,
            Predeploys.CROSS_L2_INBOX
        ];
        bytes[6] memory calls = [
            abi.encodeCall(IEventReplayer.replayEvent, (new bytes32[](0), hex"")),
            abi.encodeCall(IL2ToL2CrossDomainMessengerReplay.replaySentMessage, (901, 0, outsider, outsider, hex"")),
            abi.encodeCall(ICrossL2Inbox.validateMessage, (id, bytes32(0))),
            abi.encodeCall(ICrossL2Inbox.exportEvent, (id, bytes32(0))),
            abi.encodeCall(ICrossL2Inbox.importEvent, (id, bytes32(0))),
            abi.encodeCall(ICrossL2Inbox.importAndExecute, (id, hex""))
        ];
        PrivateProjection_Forwarder_Harness forwarder = new PrivateProjection_Forwarder_Harness();
        for (uint256 i; i < targets.length; i++) {
            vm.prank(outsider);
            (bool success, bytes memory result) = targets[i].call(calls[i]);
            assertFalse(success);
            assertEq(result, abi.encodeWithSelector(PrivateProjection.PrivateProjection_NotBatcher.selector));

            vm.expectRevert(PrivateProjection.PrivateProjection_NotBatcher.selector);
            vm.prank(batcher, batcher);
            forwarder.forward(targets[i], calls[i]);
        }
    }

    function test_replay_batcherRotation_succeeds() external {
        vm.prank(batcher);
        _replay();
        vm.prank(batcher);
        IEventReplayer(Predeploys.EVENT_REPLAYER).replayEvent(new bytes32[](0), hex"01");
        _batcher(outsider);
        vm.expectRevert(PrivateProjection.PrivateProjection_NotBatcher.selector);
        vm.prank(batcher);
        _replay();
        vm.prank(outsider);
        _replay();
    }

    function test_flagDisabled_preservesPermissionlessCalls_succeeds() external {
        _feature(false);
        vm.prank(outsider);
        _replay();
        vm.prank(outsider);
        IEventReplayer(Predeploys.EVENT_REPLAYER).replayEvent(new bytes32[](0), hex"01");
    }

    function test_validateMessage_batcherWithoutAccessList_reverts() external {
        _batcher(address(this));
        vm.fee(0);
        vm.txGasPrice(0);
        ICrossL2Inbox inbox = ICrossL2Inbox(Predeploys.CROSS_L2_INBOX);
        Identifier memory id;
        vm.expectRevert(ICrossL2Inbox.NotInAccessList.selector);
        inbox.validateMessage(id, bytes32(0));
    }

    /// @notice An authorized batcher can validate an access-listed message.
    /// forge-config: default.isolate = true
    function test_validateMessage_batcherWithAccessList_succeeds() external {
        _batcher(address(this));
        vm.fee(0);
        vm.txGasPrice(0);
        ICrossL2Inbox inbox = ICrossL2Inbox(Predeploys.CROSS_L2_INBOX);
        Identifier memory id;
        bytes32[] memory slots = new bytes32[](1);
        slots[0] = inbox.calculateChecksum(id, bytes32(0));
        VmSafe.AccessListItem[] memory accessList = new VmSafe.AccessListItem[](1);
        accessList[0] = VmSafe.AccessListItem({ target: address(inbox), storageKeys: slots });
        vm.accessList(accessList);
        inbox.validateMessage(id, bytes32(0));
    }
}
