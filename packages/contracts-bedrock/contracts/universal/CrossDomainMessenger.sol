// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import {
    OwnableUpgradeable
} from "@openzeppelin/contracts-upgradeable/access/OwnableUpgradeable.sol";
import {
    PausableUpgradeable
} from "@openzeppelin/contracts-upgradeable/security/PausableUpgradeable.sol";
import {
    ReentrancyGuardUpgradeable
} from "@openzeppelin/contracts-upgradeable/security/ReentrancyGuardUpgradeable.sol";
import { SafeCall } from "../libraries/SafeCall.sol";
import { Hashing } from "../libraries/Hashing.sol";
import { Encoding } from "../libraries/Encoding.sol";

/**
 * @custom:legacy
 * @title CrossDomainMessengerLegacySpacer
 * @notice Contract only exists to add a spacer to the CrossDomainMessenger where the
 *         libAddressManager variable used to exist. Must be the first contract in the inheritance
 *         tree of the CrossDomainMessenger
 */
contract CrossDomainMessengerLegacySpacer {
    /**
     * @custom:legacy
     * @custom:spacer libAddressManager
     * @notice Spacer for backwards compatibility.
     */
    address private spacer_0_0_20;
}

/**
 * @custom:upgradeable
 * @title CrossDomainMessenger
 * @notice CrossDomainMessenger is a base contract that provides the core logic for the L1 and L2
 *         cross-chain messenger contracts. It's designed to be a universal interface that only
 *         needs to be extended slightly to provide low-level message passing functionality on each
 *         chain it's deployed on. Currently only designed for message passing between two paired
 *         chains and does not support one-to-many interactions.
 */
abstract contract CrossDomainMessenger is
    CrossDomainMessengerLegacySpacer,
    OwnableUpgradeable,
    PausableUpgradeable,
    ReentrancyGuardUpgradeable
{
    /**
     * @notice Current message version identifier.
     */
    uint16 public constant MESSAGE_VERSION = 1;

    /**
     * @notice Constant overhead added to the base gas for a message.
     */
    uint64 public constant MIN_GAS_CONSTANT_OVERHEAD = 200_000;

    /**
     * @notice Numerator for dynamic overhead added to the base gas for a message.
     */
    uint64 public constant MIN_GAS_DYNAMIC_OVERHEAD_NUMERATOR = 1016;

    /**
     * @notice Denominator for dynamic overhead added to the base gas for a message.
     */
    uint64 public constant MIN_GAS_DYNAMIC_OVERHEAD_DENOMINATOR = 1000;

    /**
     * @notice Extra gas added to base gas for each byte of calldata in a message.
     */
    uint64 public constant MIN_GAS_CALLDATA_OVERHEAD = 16;

    /**
     * @notice Minimum amount of gas required to relay a message.
     */
    uint256 internal constant RELAY_GAS_REQUIRED = 45_000;

    /**
     * @notice Amount of gas held in reserve to guarantee that relay execution completes.
     */
    uint256 internal constant RELAY_GAS_BUFFER = RELAY_GAS_REQUIRED - 5000;

    /**
     * @notice Initial value for the xDomainMsgSender variable. We set this to a non-zero value
     *         because performing an SSTORE on a non-zero value is significantly cheaper than on a
     *         zero value.
     */
    address internal constant DEFAULT_XDOMAIN_SENDER = 0x000000000000000000000000000000000000dEaD;

    /**
     * @notice Address of the paired CrossDomainMessenger contract on the other chain.
     */
    address public immutable OTHER_MESSENGER;

    /**
     * @custom:legacy
     * @custom:spacer blockedMessages
     * @notice Spacer for backwards compatibility.
     */
    mapping(bytes32 => bool) private spacer_201_0_32;

    /**
     * @custom:legacy
     * @custom:spacer relayedMessages
     * @notice Spacer for backwards compatibility.
     */
    mapping(bytes32 => bool) private spacer_202_0_32;

    /**
     * @notice Mapping of message hashes to boolean receipt values. Note that a message will only
     *         be present in this mapping if it has successfully been relayed on this chain, and
     *         can therefore not be relayed again.
     */
    mapping(bytes32 => bool) public successfulMessages;

    /**
     * @notice Address of the sender of the currently executing message on the other chain. If the
     *         value of this variable is the default value (0x00000000...dead) then no message is
     *         currently being executed. Use the xDomainMessageSender getter which will throw an
     *         error if this is the case.
     */
    address internal xDomainMsgSender;

    /**
     * @notice Nonce for the next message to be sent, without the message version applied. Use the
     *         messageNonce getter which will insert the message version into the nonce to give you
     *         the actual nonce to be used for the message.
     */
    uint240 internal msgNonce;

    /**
     * @notice Mapping of message hashes to boolean receipt values. Note that a message will only
     *         be present in this mapping if it failed to be relayed on this chain at least once.
     *         If a message is successfully relayed on the first attempt, then it will only be
     *         present within the successfulMessages mapping.
     */
    mapping(bytes32 => bool) public receivedMessages;

    /**
     * @notice Reserve extra slots in the storage layout for future upgrades.
     *         A gap size of 41 was chosen here, so that the first slot used in a child contract
     *         would be a multiple of 50.
     */
    uint256[42] private __gap;

    /**
     * @notice Emitted whenever a message is sent to the other chain.
     *
     * @param target       Address of the recipient of the message.
     * @param sender       Address of the sender of the message.
     * @param message      Message to trigger the recipient address with.
     * @param messageNonce Unique nonce attached to the message.
     * @param gasLimit     Minimum gas limit that the message can be executed with.
     */
    event SentMessage(
        address indexed target,
        address sender,
        bytes message,
        uint256 messageNonce,
        uint256 gasLimit
    );

    /**
     * @notice Additional event data to emit, required as of Bedrock. Cannot be merged with the
     *         SentMessage event without breaking the ABI of this contract, this is good enough.
     *
     * @param sender Address of the sender of the message.
     * @param value  ETH value sent along with the message to the recipient.
     */
    event SentMessageExtension1(address indexed sender, uint256 value);

    /**
     * @notice Emitted whenever a message is successfully relayed on this chain.
     *
     * @param msgHash Hash of the message that was relayed.
     */
    event RelayedMessage(bytes32 indexed msgHash);

    /**
     * @notice Emitted whenever a message fails to be relayed on this chain.
     *
     * @param msgHash Hash of the message that failed to be relayed.
     */
    event FailedRelayedMessage(bytes32 indexed msgHash);

    /**
     * @param _otherMessenger Address of the messenger on the paired chain.
     */
    constructor(address _otherMessenger) {
        OTHER_MESSENGER = _otherMessenger;
    }

    /**
     * @notice Allows the owner of this contract to temporarily pause message relaying. Backup
     *         security mechanism just in case. Owner should be the same as the upgrade wallet to
     *         maintain the security model of the system as a whole.
     */
    function pause() external onlyOwner {
        _pause();
    }

    /**
     * @notice Allows the owner of this contract to resume message relaying once paused.
     */
    function unpause() external onlyOwner {
        _unpause();
    }

    /**
     * @notice Sends a message to some target address on the other chain. Note that if the call
     *         always reverts, then the message will be unrelayable, and any ETH sent will be
     *         permanently locked. The same will occur if the target on the other chain is
     *         considered unsafe (see the _isUnsafeTarget() function).
     *
     * @param _target      Target contract or wallet address.
     * @param _message     Message to trigger the target address with.
     * @param _minGasLimit Minimum gas limit that the message can be executed with.
     */
    function sendMessage(
        address _target,
        bytes calldata _message,
        uint32 _minGasLimit
    ) external payable {
        // Triggers a message to the other messenger. Note that the amount of gas provided to the
        // message is the amount of gas requested by the user PLUS the base gas value. We want to
        // guarantee the property that the call to the target contract will always have at least
        // the minimum gas limit specified by the user.
        _sendMessage(
            OTHER_MESSENGER,
            baseGas(_message, _minGasLimit),
            msg.value,
            abi.encodeWithSelector(
                this.relayMessage.selector,
                messageNonce(),
                msg.sender,
                _target,
                msg.value,
                _minGasLimit,
                _message
            )
        );

        emit SentMessage(_target, msg.sender, _message, messageNonce(), _minGasLimit);
        emit SentMessageExtension1(msg.sender, msg.value);

        unchecked {
            ++msgNonce;
        }
    }

    /**
     * @notice Relays a message that was sent by the other CrossDomainMessenger contract. Can only
     *         be executed via cross-chain call from the other messenger OR if the message was
     *         already received once and is currently being replayed.
     *
     * @param _nonce       Nonce of the message being relayed.
     * @param _sender      Address of the user who sent the message.
     * @param _target      Address that the message is targeted at.
     * @param _value       ETH value to send with the message.
     * @param _minGasLimit Minimum amount of gas that the message can be executed with.
     * @param _message     Message to send to the target.
     */
    function relayMessage(
        uint256 _nonce,
        address _sender,
        address _target,
        uint256 _value,
        uint256 _minGasLimit,
        bytes calldata _message
    ) external payable nonReentrant whenNotPaused {
        (, uint16 version) = Encoding.decodeVersionedNonce(_nonce);

        // Block any messages that aren't version 1. All version 0 messages have been guaranteed to
        // be relayed OR have been migrated to version 1 messages. Version 0 messages do not commit
        // to the value or minGasLimit fields, which can create unexpected issues for end-users.
        require(
            version == 1,
            "CrossDomainMessenger: only version 1 messages are supported after the Bedrock upgrade"
        );

        bytes32 versionedHash = Hashing.hashCrossDomainMessageV1(
            _nonce,
            _sender,
            _target,
            _value,
            _minGasLimit,
            _message
        );

        if (_isOtherMessenger()) {
            // These properties should always hold when the message is first submitted (as
            // opposed to being replayed).
            assert(msg.value == _value);
            assert(!receivedMessages[versionedHash]);
        } else {
            require(
                msg.value == 0,
                "CrossDomainMessenger: value must be zero unless message is from a system address"
            );

            require(
                receivedMessages[versionedHash],
                "CrossDomainMessenger: message cannot be replayed"
            );
        }

        require(
            _isUnsafeTarget(_target) == false,
            "CrossDomainMessenger: cannot send message to blocked system address"
        );

        require(
            successfulMessages[versionedHash] == false,
            "CrossDomainMessenger: message has already been relayed"
        );

        require(
            gasleft() >= _minGasLimit + RELAY_GAS_REQUIRED,
            "CrossDomainMessenger: insufficient gas to relay message"
        );

        xDomainMsgSender = _sender;
        bool success = SafeCall.call(_target, gasleft() - RELAY_GAS_BUFFER, _value, _message);
        xDomainMsgSender = DEFAULT_XDOMAIN_SENDER;

        if (success == true) {
            successfulMessages[versionedHash] = true;
            emit RelayedMessage(versionedHash);
        } else {
            receivedMessages[versionedHash] = true;
            emit FailedRelayedMessage(versionedHash);
        }
    }

    /**
     * @notice Retrieves the address of the contract or wallet that initiated the currently
     *         executing message on the other chain. Will throw an error if there is no message
     *         currently being executed. Allows the recipient of a call to see who triggered it.
     *
     * @return Address of the sender of the currently executing message on the other chain.
     */
    function xDomainMessageSender() external view returns (address) {
        require(
            xDomainMsgSender != DEFAULT_XDOMAIN_SENDER,
            "CrossDomainMessenger: xDomainMessageSender is not set"
        );

        return xDomainMsgSender;
    }

    /**
     * @notice Retrieves the next message nonce. Message version will be added to the upper two
     *         bytes of the message nonce. Message version allows us to treat messages as having
     *         different structures.
     *
     * @return Nonce of the next message to be sent, with added message version.
     */
    function messageNonce() public view returns (uint256) {
        return Encoding.encodeVersionedNonce(msgNonce, MESSAGE_VERSION);
    }

    /**
     * @notice Computes the amount of gas required to guarantee that a given message will be
     *         received on the other chain without running out of gas. Guaranteeing that a message
     *         will not run out of gas is important because this ensures that a message can always
     *         be replayed on the other chain if it fails to execute completely.
     *
     * @param _message     Message to compute the amount of required gas for.
     * @param _minGasLimit Minimum desired gas limit when message goes to target.
     *
     * @return Amount of gas required to guarantee message receipt.
     */
    function baseGas(bytes calldata _message, uint32 _minGasLimit) public pure returns (uint64) {
        // We peform the following math on uint64s to avoid overflow errors. Multiplying the
        //  by MIN_GAS_DYNAMIC_OVERHEAD_NUMERATOR would otherwise limit the _mingasLimit to
        // approximately 4.2 MM.
        return
            // Dynamic overhead
            ((uint64(_minGasLimit) * MIN_GAS_DYNAMIC_OVERHEAD_NUMERATOR) /
                MIN_GAS_DYNAMIC_OVERHEAD_DENOMINATOR) +
            // Calldata overhead
            (uint64(_message.length) * MIN_GAS_CALLDATA_OVERHEAD) +
            // Constant overhead
            MIN_GAS_CONSTANT_OVERHEAD;
    }

    /**
     * @notice Intializer.
     */
    // solhint-disable-next-line func-name-mixedcase
    function __CrossDomainMessenger_init() internal onlyInitializing {
        xDomainMsgSender = DEFAULT_XDOMAIN_SENDER;
        __Context_init_unchained();
        __Ownable_init_unchained();
        __Pausable_init_unchained();
        __ReentrancyGuard_init_unchained();
    }

    /**
     * @notice Sends a low-level message to the other messenger. Needs to be implemented by child
     *         contracts because the logic for this depends on the network where the messenger is
     *         being deployed.
     *
     * @param _to       Recipient of the message on the other chain.
     * @param _gasLimit Minimum gas limit the message can be executed with.
     * @param _value    Amount of ETH to send with the message.
     * @param _data     Message data.
     */
    function _sendMessage(
        address _to,
        uint64 _gasLimit,
        uint256 _value,
        bytes memory _data
    ) internal virtual;

    /**
     * @notice Checks whether the message is coming from the other messenger. Implemented by child
     *         contracts because the logic for this depends on the network where the messenger is
     *         being deployed.
     *
     * @return Whether the message is coming from the other messenger.
     */
    function _isOtherMessenger() internal view virtual returns (bool);

    /**
     * @notice Checks whether a given call target is a system address that could cause the
     *         messenger to peform an unsafe action. This is NOT a mechanism for blocking user
     *         addresses. This is ONLY used to prevent the execution of messages to specific
     *         system addresses that could cause security issues, e.g., having the
     *         CrossDomainMessenger send messages to itself.
     *
     * @param _target Address of the contract to check.
     *
     * @return Whether or not the address is an unsafe system address.
     */
    function _isUnsafeTarget(address _target) internal view virtual returns (bool);
}
