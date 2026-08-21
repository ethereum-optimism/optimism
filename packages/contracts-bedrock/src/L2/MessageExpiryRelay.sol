// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Contracts
import { ProxyAdminOwnedBase } from "src/universal/ProxyAdminOwnedBase.sol";

// Libraries
import { Initializable } from "@openzeppelin/contracts-upgradeable/proxy/utils/Initializable.sol";
import { Hashing } from "src/libraries/Hashing.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";

// Interfaces
import { ISemver } from "interfaces/universal/ISemver.sol";
import { ICrossDomainMessenger } from "interfaces/universal/ICrossDomainMessenger.sol";
import { IL2ToL2CrossDomainMessenger } from "interfaces/L2/IL2ToL2CrossDomainMessenger.sol";
import { IMessageExpiryHandler } from "interfaces/L2/IMessageExpiryHandler.sol";
import { IMessageExpiryHub } from "interfaces/L1/IMessageExpiryHub.sol";

/// @custom:proxied true
/// @custom:predeploy 0x420000000000000000000000000000000000002E
/// @title MessageExpiryRelay
/// @notice The MessageExpiryRelay provides a censorship-resistant escape hatch for interop messages
///         that were never relayed on their destination chain. Executing messages cannot be force
///         included via deposit transactions (the CrossL2Inbox rejects deposits), so a censoring
///         destination sequencer can permanently prevent delivery. The protocol's message expiry
///         window (enforced at cross-safety and in the fault proof) makes non-delivery PERMANENT
///         once the window has passed, which this contract turns into an actionable fact:
///
///         1. On the destination chain, `attestUndelivered` checks the local
///            L2ToL2CrossDomainMessenger delivery state and exports an attestation to the
///            MessageExpiryHub on L1 through the ordinary (censorship-resistant) withdrawal path.
///         2. The hub validates cluster membership and forwards the attestation to this contract on
///            the source chain via a deposit (also censorship-resistant).
///         3. `receiveExpiry` verifies the attestation against the recorded send and notifies the
///            sending application, which can then compensate (e.g. refund) exactly once.
///
///         Applications opt in by calling `recordSentMessage` when they send a message. The
///         recording is preimage-verified so only the actual message sender can register itself as
///         the handler for a message hash.
contract MessageExpiryRelay is ProxyAdminOwnedBase, Initializable, ISemver {
    /// @notice Record of a sent message that opted into expiry handling.
    /// @custom:field app         Application that sent the message and receives the expiry callback.
    /// @custom:field recordedAt  Timestamp at which the message was recorded (>= send time).
    /// @custom:field destination Chain ID of the destination chain the message was sent to.
    struct SentMessageRecord {
        address app;
        uint96 recordedAt;
        uint256 destination;
    }

    /// @notice Thrown when initializing with an expiry window that does not fit in a uint64.
    error MessageExpiryRelay_InvalidExpiryWindow();

    /// @notice Thrown when the message being recorded does not match a message sent by the caller.
    error MessageExpiryRelay_MessageNotSent();

    /// @notice Thrown when attempting to record a message that has already been recorded.
    error MessageExpiryRelay_AlreadyRecorded();

    /// @notice Thrown when attempting to attest a message that has been delivered on this chain.
    error MessageExpiryRelay_MessageDelivered();

    /// @notice Thrown when attesting with a source chain equal to this chain.
    error MessageExpiryRelay_InvalidSourceChain();

    /// @notice Thrown when the contract has not been initialized with a hub and expiry window.
    error MessageExpiryRelay_NotInitialized();

    /// @notice Thrown when an expiry is received by any caller other than the
    ///         L2CrossDomainMessenger relaying a message from the MessageExpiryHub.
    error MessageExpiryRelay_InvalidExpirySender();

    /// @notice Thrown when an expiry is received for a message hash that was never recorded.
    error MessageExpiryRelay_MessageNotRecorded();

    /// @notice Thrown when an expiry attestation comes from a chain other than the recorded
    ///         destination of the message.
    error MessageExpiryRelay_WrongAttestor();

    /// @notice Thrown when the attestation timestamp is not beyond the recorded send time plus the
    ///         expiry window, meaning delivery may still be possible.
    error MessageExpiryRelay_MessageNotExpired();

    /// @notice Emitted when a sent message is recorded for expiry handling.
    /// @param msgHash     Hash of the recorded message.
    /// @param app         Application that sent the message.
    /// @param destination Chain ID of the destination chain.
    /// @param recordedAt  Timestamp at which the message was recorded.
    event SentMessageRecorded(bytes32 indexed msgHash, address indexed app, uint256 destination, uint256 recordedAt);

    /// @notice Emitted when an undelivered message attestation is sent to the hub on L1.
    /// @param msgHash       Hash of the attested message.
    /// @param sourceChainId Chain ID of the chain the message was sent from.
    /// @param attestedAt    Timestamp of the attestation on this chain.
    event UndeliveredMessageAttested(bytes32 indexed msgHash, uint256 indexed sourceChainId, uint256 attestedAt);

    /// @notice Emitted when an expired message is consumed and its application notified.
    /// @param msgHash         Hash of the expired message.
    /// @param app             Application that was notified.
    /// @param attestorChainId Chain ID of the destination chain that attested non-delivery.
    /// @param attestedAt      Timestamp of the attestation on the destination chain.
    event ExpiredMessageRelayed(
        bytes32 indexed msgHash, address indexed app, uint256 attestorChainId, uint256 attestedAt
    );

    /// @notice Semantic version.
    /// @custom:semver 1.0.0
    string public constant version = "1.0.0";

    /// @notice Mapping of message hashes to their sent message records.
    mapping(bytes32 => SentMessageRecord) public sentMessageRecords;

    /// @notice Address of the MessageExpiryHub on L1.
    address public hub;

    /// @notice Number of seconds after which an undelivered message is considered expired. MUST be
    ///         greater than or equal to the protocol's interop message expiry window for the
    ///         dependency set this chain belongs to. If it were smaller, an expiry could be
    ///         consumed while delivery on the destination chain is still possible, breaking the
    ///         mutual exclusion between delivery and expiry handling.
    uint256 public expiryWindow;

    /// @notice Constructs the MessageExpiryRelay contract.
    constructor() {
        _disableInitializers();
    }

    /// @notice Initializer. Zero values are permitted so that etched genesis implementations can be
    ///         initialization-locked; the contract fails closed until both values are set on the
    ///         proxy (`attestUndelivered` requires a hub, `receiveExpiry` requires a window).
    /// @param _hub          Address of the MessageExpiryHub on L1.
    /// @param _expiryWindow Expiry window in seconds. Must be >= the protocol's interop message
    ///                      expiry window (see `expiryWindow`).
    function initialize(address _hub, uint256 _expiryWindow) external initializer {
        _assertOnlyProxyAdminOrProxyAdminOwner();
        // Bound the window so `recordedAt + expiryWindow` can never overflow in receiveExpiry,
        // which would otherwise permanently brick expiry consumption until an upgrade.
        if (_expiryWindow > type(uint64).max) revert MessageExpiryRelay_InvalidExpiryWindow();
        hub = _hub;
        expiryWindow = _expiryWindow;
    }

    /// @notice Records a message previously sent by the caller through the
    ///         L2ToL2CrossDomainMessenger, registering the caller as the handler to be notified if
    ///         the message expires undelivered. The full message preimage is required so that the
    ///         recorded hash is verifiably bound to `msg.sender` — no other party can register
    ///         itself as the handler for a message it did not send. Intended to be called in the
    ///         same transaction as the send; recording later only delays expiry eligibility.
    /// @param _destination Chain ID of the destination chain the message was sent to.
    /// @param _nonce       Nonce of the sent message.
    /// @param _target      Target contract of the sent message on the destination chain.
    /// @param _message     Message payload of the sent message.
    /// @return msgHash_ Hash of the recorded message.
    function recordSentMessage(
        uint256 _destination,
        uint256 _nonce,
        address _target,
        bytes calldata _message
    )
        external
        returns (bytes32 msgHash_)
    {
        msgHash_ = Hashing.hashL2toL2CrossDomainMessage({
            _destination: _destination,
            _source: block.chainid,
            _nonce: _nonce,
            _sender: msg.sender,
            _target: _target,
            _message: _message
        });

        if (IL2ToL2CrossDomainMessenger(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER).sentMessages(_nonce) != msgHash_) {
            revert MessageExpiryRelay_MessageNotSent();
        }
        if (sentMessageRecords[msgHash_].app != address(0)) revert MessageExpiryRelay_AlreadyRecorded();

        sentMessageRecords[msgHash_] =
            SentMessageRecord({ app: msg.sender, recordedAt: uint96(block.timestamp), destination: _destination });

        emit SentMessageRecorded(msgHash_, msg.sender, _destination, block.timestamp);
    }

    /// @notice Attests that a message destined for this chain has not been delivered here, and
    ///         exports the attestation to the MessageExpiryHub on L1 through the withdrawal path.
    ///         Permissionless: the attestation only states that the message was undelivered at the
    ///         current timestamp, which the source chain combines with its recorded send time and
    ///         expiry window before acting, so attesting a live message has no effect. This
    ///         function is NOT an interop executing message, so if this chain's sequencer censors
    ///         it, it can be force included as a deposit transaction.
    /// @param _msgHash       Hash of the message to attest, as computed by the
    ///                       L2ToL2CrossDomainMessenger.
    /// @param _sourceChainId Chain ID of the chain the message was sent from, used by the hub to
    ///                       route the attestation. An incorrect value routes the attestation to a
    ///                       chain with no record of the message, where it has no effect.
    /// @param _minGasLimit   Minimum gas limit for the hub call on L1. If relaying on L1 fails due
    ///                       to insufficient gas, the message can be permissionlessly replayed
    ///                       through the L1CrossDomainMessenger.
    function attestUndelivered(bytes32 _msgHash, uint256 _sourceChainId, uint32 _minGasLimit) external {
        if (hub == address(0)) revert MessageExpiryRelay_NotInitialized();
        if (_sourceChainId == block.chainid) revert MessageExpiryRelay_InvalidSourceChain();
        if (IL2ToL2CrossDomainMessenger(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER).successfulMessages(_msgHash)) {
            revert MessageExpiryRelay_MessageDelivered();
        }

        ICrossDomainMessenger(Predeploys.L2_CROSS_DOMAIN_MESSENGER).sendMessage({
            _target: hub,
            _message: abi.encodeCall(IMessageExpiryHub.receiveExpiryNotice, (_msgHash, _sourceChainId, block.timestamp)),
            _minGasLimit: _minGasLimit
        });

        emit UndeliveredMessageAttested(_msgHash, _sourceChainId, block.timestamp);
    }

    /// @notice Receives an expiry attestation from the MessageExpiryHub on L1, verifies it against
    ///         the recorded send, and notifies the recording application exactly once. Reverts if
    ///         verification fails or the application callback reverts, in which case the message
    ///         lands in the L2CrossDomainMessenger's failed messages and remains permissionlessly
    ///         replayable — a not-yet-expired attestation can never succeed later since its
    ///         attestation timestamp is fixed.
    /// @param _msgHash         Hash of the expired message.
    /// @param _attestorChainId Chain ID of the chain that attested non-delivery. Must equal the
    ///                         recorded destination of the message.
    /// @param _attestedAt      Timestamp of the attestation on the destination chain. Must exceed
    ///                         the recorded send time plus the expiry window.
    function receiveExpiry(bytes32 _msgHash, uint256 _attestorChainId, uint256 _attestedAt) external {
        if (
            msg.sender != Predeploys.L2_CROSS_DOMAIN_MESSENGER
                || ICrossDomainMessenger(Predeploys.L2_CROSS_DOMAIN_MESSENGER).xDomainMessageSender() != hub
        ) {
            revert MessageExpiryRelay_InvalidExpirySender();
        }
        // Fail closed if the expiry window was never configured: with a zero window any
        // attestation would satisfy the expiry check below while delivery might still be possible.
        if (expiryWindow == 0) revert MessageExpiryRelay_NotInitialized();

        SentMessageRecord memory record = sentMessageRecords[_msgHash];
        if (record.app == address(0)) revert MessageExpiryRelay_MessageNotRecorded();
        if (record.destination != _attestorChainId) revert MessageExpiryRelay_WrongAttestor();
        if (_attestedAt <= uint256(record.recordedAt) + expiryWindow) {
            revert MessageExpiryRelay_MessageNotExpired();
        }

        delete sentMessageRecords[_msgHash];

        IMessageExpiryHandler(record.app).onMessageExpired(_msgHash, _attestorChainId, _attestedAt);

        emit ExpiredMessageRelayed(_msgHash, record.app, _attestorChainId, _attestedAt);
    }
}
