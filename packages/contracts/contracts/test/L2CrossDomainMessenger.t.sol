//SPDX-License-Identifier: MIT
pragma solidity 0.8.10;

import { CommonTest } from "./CommonTest.t.sol";
import { L2OutputOracle_Initializer } from "./L2OutputOracle.t.sol";

import {
    Lib_PredeployAddresses
} from "@eth-optimism/contracts/libraries/constants/Lib_PredeployAddresses.sol";
import {
    Lib_CrossDomainUtils
} from "@eth-optimism/contracts/libraries/bridge/Lib_CrossDomainUtils.sol";
import { AddressAliasHelper } from "@eth-optimism/contracts/standards/AddressAliasHelper.sol";

import { L2OutputOracle } from "../L1/L2OutputOracle.sol";
import { OptimismPortal } from "../L1/OptimismPortal.sol";
import { L2CrossDomainMessenger } from "../L2/messaging/L2CrossDomainMessenger.sol";
import { L1CrossDomainMessenger } from "../L1/messaging/L1CrossDomainMessenger.sol";
import { Withdrawer } from "../L2/Withdrawer.sol";
import { IWithdrawer } from "../L2/IWithdrawer.sol";
import { Lib_BedrockPredeployAddresses } from "../libraries/Lib_BedrockPredeployAddresses.sol";

import {
    Lib_DefaultValues
} from "@eth-optimism/contracts/libraries/constants/Lib_DefaultValues.sol";

import { console } from "forge-std/console.sol";

contract L2CrossDomainMessenger_Test is CommonTest, L2OutputOracle_Initializer {

    // Dependencies
    OptimismPortal op;

    IWithdrawer W;
    L1CrossDomainMessenger L1Messenger;
    L2CrossDomainMessenger L2Messenger;

    event SentMessage(
        address indexed target,
        address sender,
        bytes message,
        uint256 messageNonce,
        uint256 gasLimit
    );

    event WithdrawalInitiated(
        uint256 indexed nonce,
        address indexed sender,
        address indexed target,
        uint256 value,
        uint256 gasLimit,
        bytes data
    );

    function setUp() external {
        op = new OptimismPortal(oracle, 100);
        L1Messenger = new L1CrossDomainMessenger();
        L1Messenger.initialize(op, Lib_PredeployAddresses.L2_CROSS_DOMAIN_MESSENGER);

        L2Messenger = new L2CrossDomainMessenger(address(L1Messenger));

        // Deploy the Withdrawer and then get its code to set at the
        // correct address
        Withdrawer w = new Withdrawer();
        bytes memory code = address(w).code;
        vm.etch(Lib_BedrockPredeployAddresses.WITHDRAWER, code);
        W = IWithdrawer(Lib_BedrockPredeployAddresses.WITHDRAWER);
    }

    // xDomainMessageSender: should return correct L1Messenger address
    function test_L2MessengerCorrectL1Messenger() external {
        address l1 = L2Messenger.l1CrossDomainMessenger();
        assertEq(l1, address(L1Messenger));
    }

    // xDomainMessageSender: should return the xDomainMsgSender address
    function test_L2MessengerxDomainMsgSender() external {
        vm.expectRevert("xDomainMessageSender is not set");
        L2Messenger.xDomainMessageSender();

        bytes32 slot = vm.load(address(L2Messenger), bytes32(uint256(4)));
        assertEq(address(uint160(uint256(slot))), Lib_DefaultValues.DEFAULT_XDOMAIN_SENDER);
    }

    // sendMessage: should be able to send a single message
    function test_L2MessengerSendMessage() external {
        address target = address(0);
        bytes memory message = hex"";
        uint32 gasLimit = 1000;

        uint256 nonce = W.nonce();
        address sender = address(L2Messenger);

        vm.expectEmit(true, true, true, true);
        emit SentMessage(target, address(this), message, nonce, gasLimit);
        vm.expectEmit(true, true, true, true);
        emit WithdrawalInitiated(nonce, sender, address(L1Messenger), 0, gasLimit, message);

        L2Messenger.sendMessage(target, message, gasLimit);
    }

    // sendMessage: should be able to send the same message twice
    function test_L2MessengerSendSameMessageTwice() external {
        L2Messenger.sendMessage(address(0), hex"", 1000);
        L2Messenger.sendMessage(address(0), hex"", 1000);
        // TODO: assertion on events, nonce increments
    }

    // relayMessage: should revert if the L1 message sender is not the L1CrossDomainMessenger
    function test_L2MessengerRevertInvalidL1XDomainMessenger() external {
        vm.expectRevert("Provided message could not be verified.");
        vm.prank(address(0));
        L2Messenger.relayMessage(
            address(0),
            address(0),
            hex"",
            0
        );
    }

    // relayMessage: should send a call to the target contract
    function test_L2MessengerCallsTarget() external {
        address target = address(4);

        vm.expectCall(target, hex"ff");
        vm.prank(AddressAliasHelper.applyL1ToL2Alias(address(L1Messenger)));
        L2Messenger.relayMessage(
            target,
            address(this),
            hex"ff",
            1000
        );
    }

    // relayMessage: the xDomainMessageSender is reset to the original value
    function test_L2MessengerXDomainMessageSenderReset() external {
        vm.expectRevert("xDomainMessageSender is not set");
        L2Messenger.xDomainMessageSender();

        vm.expectCall(address(4), hex"ff");
        vm.prank(AddressAliasHelper.applyL1ToL2Alias(address(L1Messenger)));
        L2Messenger.relayMessage(
            address(4),
            address(this),
            hex"ff",
            1000
        );

        vm.expectRevert("xDomainMessageSender is not set");
        L2Messenger.xDomainMessageSender();
        bytes32 slot = vm.load(address(L2Messenger), bytes32(uint256(4)));
        assertEq(address(uint160(uint256(slot))), Lib_DefaultValues.DEFAULT_XDOMAIN_SENDER);
    }

    // relayMessage: should revert if trying to send the same message twice
    function test_L2MessengerCannotRelaySameMessageTwice() external {
        vm.expectCall(address(4), hex"ff");
        vm.prank(AddressAliasHelper.applyL1ToL2Alias(address(L1Messenger)));
        L2Messenger.relayMessage(
            address(4),
            address(this),
            hex"ff",
            1000
        );

        vm.expectRevert("Provided message has already been received.");
        vm.prank(AddressAliasHelper.applyL1ToL2Alias(address(L1Messenger)));
        L2Messenger.relayMessage(
            address(4),
            address(this),
            hex"ff",
            1000
        );
    }

    // relayMessage: should not make a call if the target is the L2 MessagePasser
    function test_L2MessengerCannotCallL2MessagePasser() external {
        address target = Lib_BedrockPredeployAddresses.WITHDRAWER;

        vm.prank(AddressAliasHelper.applyL1ToL2Alias(address(L1Messenger)));
        L2Messenger.relayMessage(
            target,
            address(this),
            hex"ff",
            1000
        );

        bytes memory xDomainCalldata = Lib_CrossDomainUtils.encodeXDomainCalldata(
            target,
            address(this),
            hex"ff",
            1000
        );
        bytes32 hash = keccak256(xDomainCalldata);
        assert(L2Messenger.successfulMessages(hash) == true);

        bytes32 relayId = keccak256(abi.encodePacked(
            xDomainCalldata,
            AddressAliasHelper.applyL1ToL2Alias(address(L1Messenger)),
            block.number
        ));

        assert(L2Messenger.relayedMessages(relayId) == false);
    }
}
