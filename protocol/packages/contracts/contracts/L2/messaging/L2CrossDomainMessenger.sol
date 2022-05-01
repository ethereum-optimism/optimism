// SPDX-License-Identifier: MIT
pragma solidity ^0.8.9;

// solhint-disable max-line-length
/* Library Imports */
import { AddressAliasHelper } from "@eth-optimism/contracts/standards/AddressAliasHelper.sol";
import {
    Lib_CrossDomainUtils
} from "@eth-optimism/contracts/libraries/bridge/Lib_CrossDomainUtils.sol";
import {
    Lib_DefaultValues
} from "@eth-optimism/contracts/libraries/constants/Lib_DefaultValues.sol";
import { Lib_BedrockPredeployAddresses } from "../../libraries/Lib_BedrockPredeployAddresses.sol";

/* Interface Imports */
import {
    IL2CrossDomainMessenger
} from "@eth-optimism/contracts/L2/messaging/IL2CrossDomainMessenger.sol";
import { Withdrawer } from "../Withdrawer.sol";

// solhint-enable max-line-length

/**
 * @title L2CrossDomainMessenger
 * @dev The L2 Cross Domain Messenger contract sends messages from L2 to L1, and is the entry point
 * for L2 messages sent via the L1 Cross Domain Messenger.
 *
 */
contract L2CrossDomainMessenger is IL2CrossDomainMessenger {
    /*************
     * Variables *
     *************/

    mapping(bytes32 => bool) public relayedMessages;
    mapping(bytes32 => bool) public successfulMessages;
    mapping(bytes32 => bool) public sentMessages;
    uint256 public messageNonce;
    address internal xDomainMsgSender = Lib_DefaultValues.DEFAULT_XDOMAIN_SENDER;
    address public l1CrossDomainMessenger;

    /***************
     * Constructor *
     ***************/

    constructor(address _l1CrossDomainMessenger) {
        l1CrossDomainMessenger = _l1CrossDomainMessenger;
    }

    /********************
     * Public Functions *
     ********************/

    function xDomainMessageSender() external view returns (address) {
        require(
            xDomainMsgSender != Lib_DefaultValues.DEFAULT_XDOMAIN_SENDER,
            "xDomainMessageSender is not set"
        );
        return xDomainMsgSender;
    }

    /**
     * Sends a cross domain message to the target messenger.
     * @param _target Target contract address.
     * @param _message Message to send to the target.
     * @param _gasLimit Gas limit for the provided message.
     */
    function sendMessage(
        address _target,
        bytes memory _message,
        uint32 _gasLimit
    ) external {
        // Temp note: I think we might be able to remove xDomainCallData from this contract
        // entirely.
        // Possibly also from the L1xDM as well, but needs considerations.
        // Rationale: the proof no longer occurs in the L1 messenger, but rather in the
        // WithdrawalRelayer logic (which is inherited into the Portal). This proof relies on
        // the value returned by WithdrawalVerifier._deriveWithdrawalHash(), which takes in
        // more arguments.
        bytes memory xDomainCalldata = Lib_CrossDomainUtils.encodeXDomainCalldata(
            _target,
            msg.sender,
            _message,
            messageNonce
        );

        // Temp note: Further to my notes above, afaict this mapping is not used for anything.
        // It'd be great if we can save an SSTORE by removing it.
        sentMessages[keccak256(xDomainCalldata)] = true;

        // Emit an event before we bump the nonce or the nonce will be off by one.
        emit SentMessage(_target, msg.sender, _message, messageNonce, _gasLimit);
        unchecked {
            ++messageNonce;
        }

        // Actually send the message.
        Withdrawer(Lib_BedrockPredeployAddresses.WITHDRAWER).initiateWithdrawal(
            l1CrossDomainMessenger,
            _gasLimit,
            _message
        );
    }

    /**
     * Relays a cross domain message to a contract.
     * @inheritdoc IL2CrossDomainMessenger
     */
    function relayMessage(
        address _target,
        address _sender,
        bytes memory _message,
        uint256 _messageNonce
    ) external {
        // Since it is impossible to deploy a contract to an address on L2 which matches
        // the alias of the L1CrossDomainMessenger, this check can only pass when it is called in
        // the first call frame of a deposit transaction. Thus reentrancy is prevented here.
        require(
            AddressAliasHelper.undoL1ToL2Alias(msg.sender) == l1CrossDomainMessenger,
            "Provided message could not be verified."
        );

        // Temp note: Here we would need to keep the xDomainCalldata hashing and
        // storage, because it allows replays when the call fails, and prevents
        // them when it succeeds.
        bytes memory xDomainCalldata = Lib_CrossDomainUtils.encodeXDomainCalldata(
            _target,
            _sender,
            _message,
            _messageNonce
        );

        bytes32 xDomainCalldataHash = keccak256(xDomainCalldata);

        require(
            successfulMessages[xDomainCalldataHash] == false,
            "Provided message has already been received."
        );

        // Prevent calls to WITHDRAWER, which would enable
        // an attacker to maliciously craft the _message to spoof
        // a call from any L2 account.
        // Todo: evaluate if this attack is still relevant
        if (_target == Lib_BedrockPredeployAddresses.WITHDRAWER) {
            // Write to the successfulMessages mapping and return immediately.
            successfulMessages[xDomainCalldataHash] = true;
            return;
        }

        xDomainMsgSender = _sender;
        // slither-disable-next-line reentrancy-no-eth, reentrancy-events, reentrancy-benign
        (bool success, ) = _target.call(_message);
        // slither-disable-next-line reentrancy-benign
        xDomainMsgSender = Lib_DefaultValues.DEFAULT_XDOMAIN_SENDER;

        // Mark the message as received if the call was successful. Ensures that a message can be
        // relayed multiple times in the case that the call reverted.
        if (success == true) {
            // slither-disable-next-line reentrancy-no-eth
            successfulMessages[xDomainCalldataHash] = true;
            // slither-disable-next-line reentrancy-events
            emit RelayedMessage(xDomainCalldataHash);
        } else {
            // slither-disable-next-line reentrancy-events
            emit FailedRelayedMessage(xDomainCalldataHash);
        }

        // Store an identifier that can be used to prove that the given message was relayed by some
        // user. Gives us an easy way to pay relayers for their work.
        // TODO: consider unaliasing `msg.sender` here
        bytes32 relayId = keccak256(abi.encodePacked(xDomainCalldata, msg.sender, block.number));

        // slither-disable-next-line reentrancy-benign
        relayedMessages[relayId] = true;
    }
}
