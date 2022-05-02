//SPDX-License-Identifier: MIT
pragma solidity 0.8.10;

/* Testing utilities */
import { CommonTest } from "./CommonTest.t.sol";
import { L2OutputOracle_Initializer } from "./L2OutputOracle.t.sol";

/* Libraries */
import {
    Lib_DefaultValues
} from "@eth-optimism/contracts/libraries/constants/Lib_DefaultValues.sol";
import {
    Lib_PredeployAddresses
} from "@eth-optimism/contracts/libraries/constants/Lib_PredeployAddresses.sol";
import {
    Lib_CrossDomainUtils
} from "@eth-optimism/contracts/libraries/bridge/Lib_CrossDomainUtils.sol";
import { WithdrawalVerifier } from "../libraries/Lib_WithdrawalVerifier.sol";

/* Target contract dependencies */
import { L2OutputOracle } from "../L1/L2OutputOracle.sol";
import { OptimismPortal } from "../L1/OptimismPortal.sol";

/* Target contract */
import { L1CrossDomainMessenger } from "../L1/messaging/L1CrossDomainMessenger.sol";
import { IDepositFeed } from "../L1/abstracts/IDepositFeed.sol";

import {
    ICrossDomainMessenger
} from "@eth-optimism/contracts/libraries/bridge/ICrossDomainMessenger.sol";

contract L1CrossDomainMessenger_Test is CommonTest, L2OutputOracle_Initializer {
    // Dependencies
    OptimismPortal op;
    // 'L2OutputOracle oracle' is declared in L2OutputOracle_Initializer

    event SentMessage(
        address indexed target,
        address sender,
        bytes message,
        uint256 messageNonce,
        uint256 gasLimit
    );
    event RelayedMessage(bytes32 indexed msgHash);

    // Contract under test
    L1CrossDomainMessenger messenger;

    // Receiver address for testing
    address recipient = address(0xabbaacdc);

    function setUp() external {
        // new portal with small finalization window
        op = new OptimismPortal(oracle, 100);
        messenger = new L1CrossDomainMessenger();
        messenger.initialize(op, Lib_PredeployAddresses.L2_CROSS_DOMAIN_MESSENGER);
    }

    // pause: should pause the contract when called by the current owner
    function test_pause() external {
        messenger.pause();
        assert(messenger.paused());
    }

    // pause: should not pause the contract when called by account other than the owner
    function testCannot_pause() external {
        vm.expectRevert("Ownable: caller is not the owner");
        vm.prank(address(0xABBA));
        messenger.pause();
    }

    // sendMessage: should be able to send a single message
    function test_sendMessage() external {
        uint256 messageNonce = messenger.messageNonce();
        bytes memory xDomainCalldata = Lib_CrossDomainUtils.encodeXDomainCalldata(
            recipient,
            address(this),
            NON_ZERO_DATA,
            messageNonce
        );
        vm.expectCall(
            address(op),
            abi.encodeWithSelector(
                IDepositFeed.depositTransaction.selector,
                Lib_PredeployAddresses.L2_CROSS_DOMAIN_MESSENGER,
                0,
                NON_ZERO_GASLIMIT,
                false,
                xDomainCalldata
            )
        );
        messenger.sendMessage(recipient, NON_ZERO_DATA, uint32(NON_ZERO_GASLIMIT));
    }

    // sendMessage: should be able to send the same message twice
    function test_sendMessageTwice() external {
        messenger.sendMessage(recipient, NON_ZERO_DATA, uint32(NON_ZERO_GASLIMIT));
        messenger.sendMessage(recipient, NON_ZERO_DATA, uint32(NON_ZERO_GASLIMIT));
    }

    // xDomainMessageSender: should return the xDomainMsgSender address
    // TODO: might need a test contract
    // function test_xDomainSenderSetCorrectly() external {}

    // relayMessage: should send a successful call to the target contract
    function test_relayMessageSucceeds() external {
        address target = address(0xabcd);
        address sender = Lib_PredeployAddresses.L2_CROSS_DOMAIN_MESSENGER;
        bytes memory message = hex"1111";
        uint256 messageNonce = 42;
        // The encoding we'll use to verify that the message was successful relayed
        bytes memory xDomainCalldata = Lib_CrossDomainUtils.encodeXDomainCalldata(
            target,
            sender,
            message,
            messageNonce
        );

        // ensure that both the messenger and target receive a call
        vm.expectCall(
            address(messenger),
            abi.encodeWithSelector(
                L1CrossDomainMessenger.relayMessage.selector,
                target,
                sender,
                message,
                messageNonce
            )
        );
        vm.expectCall(address(0xabcd), hex"1111");
        // set the value of op.l2Sender() to be the L2 Cross Domain Messenger.
        vm.store(address(op), 0, bytes32(abi.encode(sender)));
        vm.prank(address(op));
        vm.expectEmit(true, true, true, true);
        emit RelayedMessage(keccak256(xDomainCalldata));
        messenger.relayMessage(target, sender, message, messageNonce);

        // Ensure the hash of the xDomainCalldata was stored in the successfulMessages mapping.
        bytes32 messageHash = keccak256(xDomainCalldata);
        assert(messenger.successfulMessages(messageHash));
    }


    // relayMessage: should revert if still inside the fraud proof window
    function test_relayMessageInsideFraudProofWindow() external {
        bytes memory cd = abi.encodeWithSelector(
            L1CrossDomainMessenger.relayMessage.selector,
            address(42),
            address(this),
            hex"1111",
            0
        );

         WithdrawalVerifier.OutputRootProof memory outputRootProof = WithdrawalVerifier.OutputRootProof({
            version: bytes32(0),
            stateRoot: bytes32(0),
            withdrawerStorageRoot: bytes32(0),
            latestBlockhash:bytes32(0)
        });

        bytes memory withdrawProof = bytes(hex"");

        // get the finalization window
        uint256 window = op.FINALIZATION_PERIOD();
        assert(window != 0);
        // set block.timestamp to be one less than the finalization window.
        // the timestamp 0 is passed into `finalizeWithdrawalTransaction`
        vm.warp(window - 1);

        // The OptimismPortal is responsible for keeping track
        // of the finalization window
        vm.expectRevert(abi.encodeWithSignature("NotYetFinal()"));
        op.finalizeWithdrawalTransaction(
            0,               // nonce
            address(this),   // sender
            address(42),     // target
            0,               // value
            100000,          // gasLimit
            cd,              // calldata
            0,               // timestamp
            outputRootProof, // outputRootProof
            withdrawProof    // withdrawProof
        );
    }

    // relayMessage: should revert if attempting to relay a message sent to an L1 system contract
    function test_relayMessageToSystemContract() external {
        // set the target to be the OptimismPortal
        address target = address(op);
        address sender = Lib_PredeployAddresses.L2_CROSS_DOMAIN_MESSENGER;
        bytes memory message = hex"1111";
        uint256 messageNonce = 42;

        // set the value of op.l2Sender() to be the L2 Cross Domain Messenger.
        vm.store(address(op), 0, bytes32(abi.encode(sender)));
        vm.prank(address(op));
        vm.expectRevert("Cannot send L2->L1 messages to L1 system contracts.");
        messenger.relayMessage(target, sender, message, messageNonce);
    }

    // relayMessage: should revert if provided an invalid output root proof
    function test_revertOnInvalidOutputRootProof() external {
        // create an invalid output root proof
        WithdrawalVerifier.OutputRootProof memory outputRootProof = WithdrawalVerifier.OutputRootProof({
            version: bytes32(0),
            stateRoot: bytes32(0),
            withdrawerStorageRoot: bytes32(0),
            latestBlockhash:bytes32(0)
        });
        bytes memory withdrawProof = bytes(hex"");

        vm.expectRevert(abi.encodeWithSignature("InvalidOutputRootProof()"));
        op.finalizeWithdrawalTransaction(
            0,               // nonce
            address(this),   // sender
            address(42),     // target
            0,               // value
            100000,          // gasLimit
            bytes(""),       // calldata
            0,               // timestamp
            outputRootProof, // outputRootProof
            withdrawProof    // withdrawProof
        );
    }

    // relayMessage: the xDomainMessageSender is reset to the original value
    function test_xDomainMessageSenderResets() external {
        vm.expectRevert("xDomainMessageSender is not set");
        messenger.xDomainMessageSender();

        address sender = Lib_PredeployAddresses.L2_CROSS_DOMAIN_MESSENGER;
        bytes memory message = hex"1111";
        uint256 messageNonce = 42;

        vm.store(address(op), 0, bytes32(abi.encode(sender)));
        vm.prank(address(op));
        messenger.relayMessage(address(0), sender, message, messageNonce);

        vm.expectRevert("xDomainMessageSender is not set");
        messenger.xDomainMessageSender();
    }

    // relayMessage: should revert if trying to send the same message twice
    function test_relayShouldRevertSendingSameMessageTwice() external {
        // TODO: this is a test on the L2CrossDomainMessenger
    }

    // relayMessage: should revert if paused
    function test_relayShouldRevertIfPaused() external {
        vm.prank(messenger.owner());
        messenger.pause();

        vm.expectRevert("Pausable: paused");
        messenger.relayMessage(address(0), address(0), hex"", 0);
    }

    // blockMessage and allowMessage: should revert if called by an account other than the owner
    function test_relayMessageBlockingAuth() external {
        bytes32 msgHash = bytes32(hex"ff");

        vm.prank(address(0));
        vm.expectRevert("Ownable: caller is not the owner");
        messenger.blockMessage(msgHash);
        assert(messenger.blockedMessages(msgHash) == false);

        vm.prank(address(0));
        vm.expectRevert("Ownable: caller is not the owner");
        messenger.allowMessage(msgHash);
        assert(messenger.blockedMessages(msgHash) == false);
    }

    // blockMessage and allowMessage: should revert if the message is blocked
    function test_relayRevertOnBlockedMessage() external {
        bytes memory xDomainCalldata = Lib_CrossDomainUtils.encodeXDomainCalldata(
            address(0),
            address(0),
            hex"ff",
            0
        );
        bytes32 msgHash = keccak256(xDomainCalldata);

        vm.prank(messenger.owner());
        messenger.blockMessage(msgHash);

        vm.store(address(op), 0, bytes32(abi.encode(Lib_PredeployAddresses.L2_CROSS_DOMAIN_MESSENGER)));
        vm.prank(address(op));
        vm.expectRevert("Provided message has been blocked.");
        messenger.relayMessage(address(0), address(0), hex"ff", 0);
    }

    // blockMessage and allowMessage: should succeed if the message is blocked, then unblocked
    function test_blockAndUnblockSuccessfulMessage() external {
        bytes memory xDomainCalldata = Lib_CrossDomainUtils.encodeXDomainCalldata(
            address(0),
            address(0),
            hex"ff",
            0
        );
        bytes32 msgHash = keccak256(xDomainCalldata);

        vm.prank(messenger.owner());
        messenger.blockMessage(msgHash);

        vm.store(address(op), 0, bytes32(abi.encode(Lib_PredeployAddresses.L2_CROSS_DOMAIN_MESSENGER)));
        vm.prank(address(op));
        vm.expectRevert("Provided message has been blocked.");
        messenger.relayMessage(address(0), address(0), hex"ff", 0);

        vm.prank(messenger.owner());
        messenger.allowMessage(msgHash);

        vm.prank(address(op));

        vm.expectEmit(true, true, true, true);
        emit RelayedMessage(msgHash);
        messenger.relayMessage(address(0), address(0), hex"ff", 0);
    }
}
