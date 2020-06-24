pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

/* Internal Imports */
import {DataTypes as dt} from "../utils/DataTypes.sol";

contract L2ToL1MessageReceiver {
    event L2ToL1MessageEnqueued(
        address ovmSender,
        bytes callData,
        uint nonce
    );

    struct EnqueuedL2ToL1Message {
        dt.L2ToL1Message message;
        uint l1BlockEnqueued;
    }

    address public sequencer;
    uint public blocksUntilFinal;
    uint messageNonce = 0;
    mapping (uint => EnqueuedL2ToL1Message) public messages;

    constructor(address _sequencer, uint _blocksUntilFinal) public {
        sequencer = _sequencer;
        blocksUntilFinal = _blocksUntilFinal;
    }

    function enqueueL2ToL1Message(dt.L2ToL1Message memory _message) public {
        require(msg.sender == sequencer, "For now, only our trusted sequencer can enqueue messages to be verified on L1");
        uint blockNum = block.number;
        messages[messageNonce] = EnqueuedL2ToL1Message({
            message: _message,
            l1BlockEnqueued: blockNum
        });
        emit L2ToL1MessageEnqueued(
            _message.ovmSender,
            _message.callData,
            messageNonce
        );
        messageNonce += 1;
    }

    function verifyL2ToL1Message(dt.L2ToL1Message memory _message, uint _nonce) public view returns (bool) {
        // The enqueued message for the given nonce must match the _message being verified
        bytes32 givenMessageHash = getMessageHash(_message);
        bytes32 storedMessageHash = getMessageHash(messages[_nonce].message);
        bool messageWasEnqueued = (storedMessageHash == givenMessageHash);
        // Message must be finalized on L1
        bool messageIsFinalized = (block.number >= messages[_nonce].l1BlockEnqueued + blocksUntilFinal);
        return messageWasEnqueued && messageIsFinalized;
    }

    function getMessageHash(dt.L2ToL1Message memory _message) internal pure returns(bytes32) {
        return keccak256(abi.encode(_message));
    }
}