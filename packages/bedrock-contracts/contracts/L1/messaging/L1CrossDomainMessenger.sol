// SPDX-License-Identifier: MIT
pragma solidity ^0.8.9;

// solhint-disable max-line-length
/* Library Imports */
import { AddressAliasHelper } from "@eth-optimism/contracts/standards/AddressAliasHelper.sol";
import { Lib_OVMCodec } from "@eth-optimism/contracts/libraries/codec/Lib_OVMCodec.sol";
import {
    Lib_SecureMerkleTrie
} from "@eth-optimism/contracts/libraries/trie/Lib_SecureMerkleTrie.sol";
import {
    Lib_DefaultValues
} from "@eth-optimism/contracts/libraries/constants/Lib_DefaultValues.sol";
import {
    Lib_PredeployAddresses
} from "@eth-optimism/contracts/libraries/constants/Lib_PredeployAddresses.sol";
import {
    Lib_CrossDomainUtils
} from "@eth-optimism/contracts/libraries/bridge/Lib_CrossDomainUtils.sol";

/* Interface Imports */
import { IL1CrossDomainMessenger } from "./IL1CrossDomainMessenger.sol";
import { OptimismPortal } from "../OptimismPortal.sol";

/* External Imports */
import {
    OwnableUpgradeable
} from "@openzeppelin/contracts-upgradeable/access/OwnableUpgradeable.sol";
import {
    PausableUpgradeable
} from "@openzeppelin/contracts-upgradeable/security/PausableUpgradeable.sol";
import {
    ReentrancyGuardUpgradeable
} from "@openzeppelin/contracts-upgradeable/security/ReentrancyGuardUpgradeable.sol";

// solhint-enable max-line-length

/**
 * @title L1CrossDomainMessenger
 * @dev The L1 Cross Domain Messenger contract sends messages from L1 to L2, and relays messages
 * from L2 onto L1. In the event that a message sent from L1 to L2 is rejected for exceeding the L2
 * epoch gas limit, it can be resubmitted via this contract's replay function.
 *
 */
contract L1CrossDomainMessenger is
    IL1CrossDomainMessenger,
    OwnableUpgradeable,
    PausableUpgradeable,
    ReentrancyGuardUpgradeable
{
    /**********
     * Events *
     **********/

    event MessageBlocked(bytes32 indexed _xDomainCalldataHash);

    event MessageAllowed(bytes32 indexed _xDomainCalldataHash);

    /**********************
     * Contract Variables *
     **********************/
    OptimismPortal public optimismPortal;
    address public l2CrossDomainMessenger;

    // Bedrock upgrade note: the nonce must be initialized to greater than the last value of
    // CanonicalTransactionChain.queueElements.length. Otherwise it will be possible to have
    // messages which cannot be relayed on L2.
    uint256 public messageNonce;

    mapping(bytes32 => bool) public blockedMessages;
    mapping(bytes32 => bool) public relayedMessages;
    mapping(bytes32 => bool) public successfulMessages;

    address internal xDomainMsgSender = Lib_DefaultValues.DEFAULT_XDOMAIN_SENDER;

    /***************
     * Constructor *
     ***************/

    /**
     * This contract is intended to be behind a delegate proxy.
     * We still need to set this value in initialize().
     */
    constructor() {}

    /********************
     * Public Functions *
     ********************/

    /**
     * @param _optimismPortal Address of the OptimismPortal.
     */
    function initialize(OptimismPortal _optimismPortal, address _l2CrossDomainMessenger)
        external
        initializer
    {
        require(
            address(optimismPortal) == address(0),
            "L1CrossDomainMessenger already intialized."
        );
        optimismPortal = _optimismPortal;
        l2CrossDomainMessenger = _l2CrossDomainMessenger;
        xDomainMsgSender = Lib_DefaultValues.DEFAULT_XDOMAIN_SENDER;

        // Initialize upgradable OZ contracts
        __Context_init_unchained(); // Context is a dependency for both Ownable and Pausable
        __Ownable_init_unchained();
        __Pausable_init_unchained();
        __ReentrancyGuard_init_unchained();
    }

    /**
     * Pause relaying.
     */
    function pause() external onlyOwner {
        _pause();
    }

    /**
     * Block a message.
     * @param _xDomainCalldataHash Hash of the message to block.
     */
    function blockMessage(bytes32 _xDomainCalldataHash) external onlyOwner {
        blockedMessages[_xDomainCalldataHash] = true;
        emit MessageBlocked(_xDomainCalldataHash);
    }

    /**
     * Allow a message.
     * @param _xDomainCalldataHash Hash of the message to block.
     */
    function allowMessage(bytes32 _xDomainCalldataHash) external onlyOwner {
        blockedMessages[_xDomainCalldataHash] = false;
        emit MessageAllowed(_xDomainCalldataHash);
    }

    /**
     * Get the xDomainMessageSender
     */
    function xDomainMessageSender() external view returns (address) {
        require(
            xDomainMsgSender != Lib_DefaultValues.DEFAULT_XDOMAIN_SENDER,
            "xDomainMessageSender is not set"
        );
        return xDomainMsgSender;
    }

    /**
     * Sends a cross domain message to the target messenger.
     * This function faciliates L1 to L2 communication.
     * @param _target Target contract address.
     * @param _message Message to send to the target.
     * @param _gasLimit Gas limit for the provided message.
     */
    function sendMessage(
        address _target,
        bytes memory _message,
        uint32 _gasLimit
    ) external {
        bytes memory xDomainCalldata = Lib_CrossDomainUtils.encodeXDomainCalldata(
            _target,
            msg.sender,
            _message,
            messageNonce
        );

        emit SentMessage(_target, msg.sender, _message, messageNonce, _gasLimit);

        _sendXDomainMessage(xDomainCalldata, _gasLimit);
    }

    /**
     * Relays a cross domain message to a contract.
     * This function faciliates L2 to L1 communication.
     * Calls WithdrawalsRelay.finalizeWithdrawalTransaction
     * @inheritdoc IL1CrossDomainMessenger
     */
    function relayMessage(
        address _target,
        address _sender,
        bytes memory _message,
        uint256 _messageNonce
    ) external nonReentrant whenNotPaused {
        require(
            msg.sender == address(optimismPortal),
            "Messages must be relayed by first calling the Optimism Portal"
        );
        require(
            optimismPortal.l2Sender() == l2CrossDomainMessenger,
            "Message must be sent from the L2 Cross Domain Messenger"
        );

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

        require(
            blockedMessages[xDomainCalldataHash] == false,
            "Provided message has been blocked."
        );

        require(
            _target != address(optimismPortal),
            "Cannot send L2->L1 messages to L1 system contracts."
        );

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
        bytes32 relayId = keccak256(abi.encodePacked(xDomainCalldata, msg.sender, block.number));
        // slither-disable-next-line reentrancy-benign
        relayedMessages[relayId] = true;
    }

    /**********************
     * Internal Functions *
     **********************/

    /**
     * Sends a cross domain message. To preserve backwards compatibility,
     * the gasLimit is a uint256 so that the function selector stays the same.
     * TODO: revert on overflow when converting to uint64?
     * @param _message Message to send.
     * @param _gasLimit L2 gas limit for the message.
     */
    function _sendXDomainMessage(bytes memory _message, uint256 _gasLimit) internal {
        optimismPortal.depositTransaction(
            Lib_PredeployAddresses.L2_CROSS_DOMAIN_MESSENGER,
            0,
            uint64(_gasLimit),
            false,
            _message
        );
    }
}
