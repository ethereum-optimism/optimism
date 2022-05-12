// SPDX-License-Identifier: MIT
pragma solidity ^0.8.9;

// solhint-disable max-line-length

/* Library Imports */
import {
    Lib_DefaultValues
} from "@eth-optimism/contracts/libraries/constants/Lib_DefaultValues.sol";
import { CrossDomainHashing } from "../libraries/Lib_CrossDomainHashing.sol";

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
import { ExcessivelySafeCall } from "../libraries/ExcessivelySafeCall.sol";

// solhint-enable max-line-length

/**
 * @title CrossDomainMessenger
 * @dev The CrossDomainMessenger contract delivers messages between two layers.
 */
abstract contract CrossDomainMessenger is
    OwnableUpgradeable,
    PausableUpgradeable,
    ReentrancyGuardUpgradeable
{
    /**********
     * Events *
     **********/

    event SentMessage(
        address indexed target,
        address sender,
        bytes message,
        uint256 messageNonce,
        uint256 gasLimit
    );

    event RelayedMessage(bytes32 indexed msgHash);

    event FailedRelayedMessage(bytes32 indexed msgHash);

    /*************
     * Constants *
     *************/

    uint16 public constant MESSAGE_VERSION = 1;

    /*************
     * Variables *
     *************/

    // blockedMessages in old L1CrossDomainMessenger
    bytes32 internal REMOVED_VARIABLE_SPACER_1;

    // relayedMessages in old L1CrossDomainMessenger
    bytes32 internal REMOVED_VARIABLE_SPACER_2;

    /// @notice Mapping of message hash to boolean success value.
    mapping(bytes32 => bool) public successfulMessages;

    /// @notice Current x-domain message sender.
    address internal xDomainMsgSender;

    /// @notice Nonce for the next message to be sent.
    uint256 internal msgNonce;

    /// @notice Address of the CrossDomainMessenger on the other chain.
    address public otherMessenger;

    /// @notice Mapping of message hash to boolean receipt value.
    mapping(bytes32 => bool) public receivedMessages;

    /// @notice Blocked system addresses that cannot be called (for security reasons).
    mapping(address => bool) public blockedSystemAddresses;

    /********************
     * Public Functions *
     ********************/

    /**
     * Pause relaying.
     */
    function pause() external onlyOwner {
        _pause();
    }

    /**
     * Retrieves the address of the x-domain message sender. Will throw an error
     * if the sender is not currently set (equal to the default sender).
     * This function is meant to be called on the remote side of a cross domain
     * message so that the account that initiated the call can be known.
     *
     * @return Address of the x-domain message sender.
     */
    function xDomainMessageSender() external view returns (address) {
        require(
            xDomainMsgSender != Lib_DefaultValues.DEFAULT_XDOMAIN_SENDER,
            "xDomainMessageSender is not set"
        );

        return xDomainMsgSender;
    }

    /**
     * Retrieves the next message nonce. Adds the hash version to the nonce.
     *
     * @return Next message nonce with added hash version.
     */
    function messageNonce() public view returns (uint256) {
        return CrossDomainHashing.addVersionToNonce(msgNonce, MESSAGE_VERSION);
    }

    /**
     * @param _target Target contract address.
     * @param _message Message to send to the target.
     * @param _minGasLimit Gas limit for the provided message.
     */
    function sendMessage(
        address _target,
        bytes memory _message,
        uint32 _minGasLimit
    ) external payable {
        // TODO: Enforce minimum gas limit.

        _sendMessage(
            otherMessenger,
            _minGasLimit, // TODO: Pad this value.
            msg.value,
            CrossDomainHashing.getVersionedEncoding(
                messageNonce(),
                msg.sender,
                _target,
                msg.value,
                _minGasLimit,
                _message
            )
        );

        emit SentMessage(_target, msg.sender, _message, messageNonce(), _minGasLimit);

        unchecked {
            ++msgNonce;
        }
    }

    function relayMessage(
        uint256 _nonce,
        address _sender,
        address _target,
        uint256 _value,
        uint256 _minGasLimit,
        bytes calldata _message
    ) external payable nonReentrant whenNotPaused {
        bytes32 versionedHash = CrossDomainHashing.getVersionedHash(
            _nonce,
            _sender,
            _target,
            _value,
            _minGasLimit,
            _message
        );

        if (_isSystemMessageSender()) {
            // Should never happen.
            require(msg.value == _value, "Mismatched message value.");
        } else {
            // TODO(tynes): could require that msg.value == 0 here
            // to prevent eth from getting stuck
            require(receivedMessages[versionedHash], "Message cannot be replayed.");
        }

        // TODO: Should blocking happen on sending or receiving side?
        // TODO: Should this just return with an event instead of reverting?
        require(
            blockedSystemAddresses[_target] == false,
            "Cannot send message to blocked system address."
        );

        require(successfulMessages[versionedHash] == false, "Message has already been relayed.");

        // TODO: Make sure this will always give us enough gas.
        require(gasleft() >= _minGasLimit + 45000, "Insufficient gas to relay message.");

        xDomainMsgSender = _sender;
        (bool success, ) = ExcessivelySafeCall.excessivelySafeCall(
            _target,
            gasleft() - 40000,
            _value,
            0,
            _message
        );
        xDomainMsgSender = Lib_DefaultValues.DEFAULT_XDOMAIN_SENDER;

        if (success == true) {
            successfulMessages[versionedHash] = true;
            emit RelayedMessage(versionedHash);
        } else {
            receivedMessages[versionedHash] = true;
            emit FailedRelayedMessage(versionedHash);
        }
    }

    /**********************
     * Internal Functions *
     **********************/

    function _isSystemMessageSender() internal view virtual returns (bool);

    function _sendMessage(
        address _to,
        uint64 _gasLimit,
        uint256 _value,
        bytes memory _data
    ) internal virtual;

    /**
     * Initializes the contract.
     */
    function _initialize(address _otherMessenger, address[] memory _blockedSystemAddresses)
        internal
        initializer
    {
        xDomainMsgSender = Lib_DefaultValues.DEFAULT_XDOMAIN_SENDER;
        otherMessenger = _otherMessenger;

        for (uint256 i = 0; i < _blockedSystemAddresses.length; i++) {
            blockedSystemAddresses[_blockedSystemAddresses[i]] = true;
        }

        // TODO: ensure we know what these are doing and why they are here
        // Initialize upgradable OZ contracts
        __Context_init_unchained();
        __Ownable_init_unchained();
        __Pausable_init_unchained();
        __ReentrancyGuard_init_unchained();
    }
}
