// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Types } from "src/libraries/Types.sol";
import { Constants } from "src/libraries/Constants.sol";
import { SafeCall } from "src/libraries/SafeCall.sol";
import { Hashing } from "src/libraries/Hashing.sol";
import { ISemver } from "src/universal/ISemver.sol";

/// @notice Entry to post to the inbox
/// @custom:field chain        Chain identifier.
/// @custom:field output       Output-root of the chain.
/// @custom:field blockNumber  Block Number of the output
/// @custom:field messageRoots List of messages to deliver
struct InboxMessages {
    bytes32 chain;
    bytes32 output;
    uint256 blockNumber;
    bytes32[] messageRoots;
}

/// @notice Chain State
/// @custom:field output       Output-root of the chain.
/// @custom:field blockNumber  Block Number of the output
struct ChainState {
    bytes32 output;
    uint256 blockNumber;
}

/// @custom:proxied
/// @title CrossL2Inbox
/// @notice The CrossL2Inbox receives messages & output-roots of any chain
contract CrossL2Inbox is ISemver {
    /// @notice The system address that is allowed to post into the inbox.
    address internal immutable INBOX_POSTIE_ADDRESS;

    /// @custom:semver 0.0.1
    string public constant version = "0.0.1";

    /// @notice The collection of output roots, by chain.
    /// source chain ID => output root => bool.
    mapping(bytes32 => mapping(bytes32 => bool)) public roots;

    /// @notice The collection of delivered messages that have yet to be consumed
    mapping(bytes32 => bool) public unconsumedMessages;

    /// @notice The latest state recorded per chain
    mapping(bytes32 => ChainState) public chainState;

    /// @notice Address of the cross L2 account which initiated a call in this cross L2 message. If
    ///         the of this variable is the default L2 sender address, then we are NOT inside of a call
    address public crossL2Sender = Constants.DEFAULT_L2_SENDER;

    /// @notice Source chain identifier from where the cross L2 call originated. Empty if not in a call.
    bytes32 public messageSourceChain;

    /// @notice Emitted when a cross L2 message has been relayed.
    /// @param messageRoot Root of the cross L2 message.
    /// @param success     Whether the cross L2 message call was successful.
    event CrossL2MessageRelayed(bytes32 indexed messageRoot, bool success);

    /// @notice Initialize the inbox.
    /// @param _postie_address System address that will be allowed to deliver to the inbox.
    constructor(address _postie_address) {
        INBOX_POSTIE_ADDRESS = _postie_address;
    }
    
    /// @notice The inbox receives mail from the postie of deliverd messages 
    function deliverMessages(InboxMessages[] calldata mail) external payable {
        require(msg.sender == INBOX_POSTIE_ADDRESS, "CrossL2Inbox: only postie can deliver mail");

        for (uint256 i = 0; i < mail.length; i++) {
            require(mail[i].blockNumber > chainState[mail[i].chain].blockNumber, "CrossL2Inbox: blockNumber must be increasing");

            chainState[mail[i].chain] = ChainState(mail[i].output, mail[i].blockNumber);
            roots[mail[i].chain][mail[i].output] = true;

            for (uint256 j = 0; j < mail[i].messageRoots.length; j++) {
                unconsumedMessages[mail[i].messageRoots[j]] = true;
            }
        }
    }

    /// @notice Executes delivered but unconsumed messages waiting to be consumed
    function consumeMessage(Types.SuperchainMessage memory _msg) external {
        require(_msg.targetChain == bytes32(block.chainid), "CrossL2Inbox: target chain does not match");

        // Message validity
        bytes32 messageRoot = Hashing.superchainMessageRoot(_msg);
        require(unconsumedMessages[messageRoot], "CrossL2Inbox: unknown message");

        // Make sure that the crossL2Sender has not yet been set.
        require(crossL2Sender == Constants.DEFAULT_L2_SENDER, "CrossL2Inbox: can only trigger one call per message");

        // Set message origination info
        crossL2Sender = _msg.from;
        messageSourceChain = _msg.sourceChain;

        bool success = SafeCall.callWithMinGas(_msg.to, _msg.gasLimit, _msg.value, _msg.data);

        // Reset message origination info
        crossL2Sender = Constants.DEFAULT_L2_SENDER;
        messageSourceChain = bytes32(0);

        // Delete message
        delete unconsumedMessages[messageRoot];

        emit CrossL2MessageRelayed(messageRoot, success);
    }
}
