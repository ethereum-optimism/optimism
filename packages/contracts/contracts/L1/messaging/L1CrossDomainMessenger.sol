// SPDX-License-Identifier: MIT
pragma solidity ^0.8.9;

/* Library Imports */
import { AddressAliasHelper } from "../../standards/AddressAliasHelper.sol";
import { Lib_AddressResolver } from "../../libraries/resolver/Lib_AddressResolver.sol";
import { Lib_OVMCodec } from "../../libraries/codec/Lib_OVMCodec.sol";
import { Lib_AddressManager } from "../../libraries/resolver/Lib_AddressManager.sol";
import { Lib_SecureMerkleTrie } from "../../libraries/trie/Lib_SecureMerkleTrie.sol";
import { Lib_DefaultValues } from "../../libraries/constants/Lib_DefaultValues.sol";
import { Lib_PredeployAddresses } from "../../libraries/constants/Lib_PredeployAddresses.sol";
import { Lib_CrossDomainUtils } from "../../libraries/bridge/Lib_CrossDomainUtils.sol";

/* Interface Imports */
import { IL1CrossDomainMessenger } from "./IL1CrossDomainMessenger.sol";
import { ICanonicalTransactionChain } from "../rollup/ICanonicalTransactionChain.sol";
import { IStateCommitmentChain } from "../rollup/IStateCommitmentChain.sol";

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

/**
 * @title L1CrossDomainMessenger
 * @dev The L1 Cross Domain Messenger contract sends messages from L1 to L2, and relays messages
 * from L2 onto L1. In the event that a message sent from L1 to L2 is rejected for exceeding the L2
 * epoch gas limit, it can be resubmitted via this contract's replay function.
 *
 */
contract L1CrossDomainMessenger is
    IL1CrossDomainMessenger,
    Lib_AddressResolver,
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

    mapping(bytes32 => bool) public blockedMessages;
    mapping(bytes32 => bool) public relayedMessages;
    mapping(bytes32 => bool) public successfulMessages;

    address internal xDomainMsgSender = Lib_DefaultValues.DEFAULT_XDOMAIN_SENDER;

    /***************
     * Constructor *
     ***************/

    /**
     * This contract is intended to be behind a delegate proxy.
     * We pass the zero address to the address resolver just to satisfy the constructor.
     * We still need to set this value in initialize().
     */
    constructor() Lib_AddressResolver(address(0)) {}

    /********************
     * Public Functions *
     ********************/

    /**
     * @param _libAddressManager Address of the Address Manager.
     */
    // slither-disable-next-line external-function
    function initialize(address _libAddressManager) public initializer {
        require(
            address(libAddressManager) == address(0),
            "L1CrossDomainMessenger already intialized."
        );
        libAddressManager = Lib_AddressManager(_libAddressManager);
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

    // slither-disable-next-line external-function
    function xDomainMessageSender() public view returns (address) {
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
    // slither-disable-next-line external-function
    function sendMessage(
        address _target,
        bytes memory _message,
        uint32 _gasLimit
    ) public {
        address ovmCanonicalTransactionChain = resolve("CanonicalTransactionChain");
        // Use the CTC queue length as nonce
        uint40 nonce = ICanonicalTransactionChain(ovmCanonicalTransactionChain).getQueueLength();

        bytes memory xDomainCalldata = Lib_CrossDomainUtils.encodeXDomainCalldata(
            _target,
            msg.sender,
            _message,
            nonce
        );

        // slither-disable-next-line reentrancy-events
        _sendXDomainMessage(ovmCanonicalTransactionChain, xDomainCalldata, _gasLimit);

        // slither-disable-next-line reentrancy-events
        emit SentMessage(_target, msg.sender, _message, nonce, _gasLimit);
    }

    /**
     * Relays a cross domain message to a contract.
     * @inheritdoc IL1CrossDomainMessenger
     */
    // slither-disable-next-line external-function
    function relayMessage(
        address _target,
        address _sender,
        bytes memory _message,
        uint256 _messageNonce,
        L2MessageInclusionProof memory _proof
    ) public nonReentrant whenNotPaused {
        bytes memory xDomainCalldata = Lib_CrossDomainUtils.encodeXDomainCalldata(
            _target,
            _sender,
            _message,
            _messageNonce
        );

        require(
            _verifyXDomainMessage(xDomainCalldata, _proof) == true,
            "Provided message could not be verified."
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
            _target != resolve("CanonicalTransactionChain"),
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

    /**
     * Replays a cross domain message to the target messenger.
     * @inheritdoc IL1CrossDomainMessenger
     */
    // slither-disable-next-line external-function
    function replayMessage(
        address _target,
        address _sender,
        bytes memory _message,
        uint256 _queueIndex,
        uint32 _oldGasLimit,
        uint32 _newGasLimit
    ) public {
        // Verify that the message is in the queue:
        address canonicalTransactionChain = resolve("CanonicalTransactionChain");
        Lib_OVMCodec.QueueElement memory element = ICanonicalTransactionChain(
            canonicalTransactionChain
        ).getQueueElement(_queueIndex);

        // Compute the calldata that was originally used to send the message.
        bytes memory xDomainCalldata = Lib_CrossDomainUtils.encodeXDomainCalldata(
            _target,
            _sender,
            _message,
            _queueIndex
        );

        // Compute the transactionHash
        bytes32 transactionHash = keccak256(
            abi.encode(
                AddressAliasHelper.applyL1ToL2Alias(address(this)),
                Lib_PredeployAddresses.L2_CROSS_DOMAIN_MESSENGER,
                _oldGasLimit,
                xDomainCalldata
            )
        );

        // Now check that the provided message data matches the one in the queue element.
        require(
            transactionHash == element.transactionHash,
            "Provided message has not been enqueued."
        );

        // Send the same message but with the new gas limit.
        _sendXDomainMessage(canonicalTransactionChain, xDomainCalldata, _newGasLimit);
    }

    /**********************
     * Internal Functions *
     **********************/

    /**
     * Verifies that the given message is valid.
     * @param _xDomainCalldata Calldata to verify.
     * @param _proof Inclusion proof for the message.
     * @return Whether or not the provided message is valid.
     */
    function _verifyXDomainMessage(
        bytes memory _xDomainCalldata,
        L2MessageInclusionProof memory _proof
    ) internal view returns (bool) {
        return (_verifyStateRootProof(_proof) && _verifyStorageProof(_xDomainCalldata, _proof));
    }

    /**
     * Verifies that the state root within an inclusion proof is valid.
     * @param _proof Message inclusion proof.
     * @return Whether or not the provided proof is valid.
     */
    function _verifyStateRootProof(L2MessageInclusionProof memory _proof)
        internal
        view
        returns (bool)
    {
        IStateCommitmentChain ovmStateCommitmentChain = IStateCommitmentChain(
            resolve("StateCommitmentChain")
        );

        return (ovmStateCommitmentChain.insideFraudProofWindow(_proof.stateRootBatchHeader) ==
            false &&
            ovmStateCommitmentChain.verifyStateCommitment(
                _proof.stateRoot,
                _proof.stateRootBatchHeader,
                _proof.stateRootProof
            ));
    }

    /**
     * Verifies that the storage proof within an inclusion proof is valid.
     * @param _xDomainCalldata Encoded message calldata.
     * @param _proof Message inclusion proof.
     * @return Whether or not the provided proof is valid.
     */
    function _verifyStorageProof(
        bytes memory _xDomainCalldata,
        L2MessageInclusionProof memory _proof
    ) internal view returns (bool) {
        bytes32 storageKey = keccak256(
            abi.encodePacked(
                keccak256(
                    abi.encodePacked(
                        _xDomainCalldata,
                        Lib_PredeployAddresses.L2_CROSS_DOMAIN_MESSENGER
                    )
                ),
                uint256(0)
            )
        );

        (bool exists, bytes memory encodedMessagePassingAccount) = Lib_SecureMerkleTrie.get(
            abi.encodePacked(Lib_PredeployAddresses.L2_TO_L1_MESSAGE_PASSER),
            _proof.stateTrieWitness,
            _proof.stateRoot
        );

        require(
            exists == true,
            "Message passing predeploy has not been initialized or invalid proof provided."
        );

        Lib_OVMCodec.EVMAccount memory account = Lib_OVMCodec.decodeEVMAccount(
            encodedMessagePassingAccount
        );

        return
            Lib_SecureMerkleTrie.verifyInclusionProof(
                abi.encodePacked(storageKey),
                abi.encodePacked(uint8(1)),
                _proof.storageTrieWitness,
                account.storageRoot
            );
    }

    /**
     * Sends a cross domain message.
     * @param _canonicalTransactionChain Address of the CanonicalTransactionChain instance.
     * @param _message Message to send.
     * @param _gasLimit OVM gas limit for the message.
     */
    function _sendXDomainMessage(
        address _canonicalTransactionChain,
        bytes memory _message,
        uint256 _gasLimit
    ) internal {
        // slither-disable-next-line reentrancy-events
        ICanonicalTransactionChain(_canonicalTransactionChain).enqueue(
            Lib_PredeployAddresses.L2_CROSS_DOMAIN_MESSENGER,
            _gasLimit,
            _message
        );
    }
}
