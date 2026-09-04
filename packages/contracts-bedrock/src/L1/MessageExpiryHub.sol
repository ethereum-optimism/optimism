// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Interfaces
import { ISemver } from "interfaces/universal/ISemver.sol";
import { ICrossDomainMessenger } from "interfaces/universal/ICrossDomainMessenger.sol";
import { IL1CrossDomainMessenger } from "interfaces/L1/IL1CrossDomainMessenger.sol";
import { ISystemConfig } from "interfaces/L1/ISystemConfig.sol";
import { IOptimismPortal2 } from "interfaces/L1/IOptimismPortal2.sol";
import { IETHLockbox } from "interfaces/L1/IETHLockbox.sol";
import { IMessageExpiryRelay } from "interfaces/L2/IMessageExpiryRelay.sol";

/// @title MessageExpiryHub
/// @notice The MessageExpiryHub routes interop message expiry attestations between the chains of an
///         interop cluster on L1. A destination chain's MessageExpiryRelay attests through the
///         withdrawal path that a message was never delivered there; the hub records the notice and
///         later forwards it to the message's source chain through the deposit path, where the
///         source chain's MessageExpiryRelay verifies expiry and notifies the sending application.
///
///         The hub is an ownerless singleton shared by all clusters. Cluster membership is derived
///         on chain per call rather than configured: a chain is identified by its SystemConfig
///         (bound bidirectionally to its L1CrossDomainMessenger AND to its OptimismPortal), and two
///         chains belong to the same cluster when their portals are authorized by the same shared
///         ETHLockbox and share the same AnchorStateRegistry. Notices are namespaced by the
///         attestor's lockbox so distinct clusters can never interfere, even with colliding chain
///         IDs.
///
///         Receiving and forwarding are deliberately separate steps: forwarding initiates a deposit
///         whose resource metering burns an unbounded amount of L1 gas, which must not sit inside
///         the fixed gas frame of a relayed withdrawal. Forwarding is permissionless and repeatable;
///         consumption on the source chain is idempotent.
/// @dev TODO(security): a guardian-controlled pause on the hub was considered as an incident
///      kill-switch but is deliberately omitted here. It would introduce a new trust role on an
///      otherwise ownerless singleton, which is a governance decision left to the owner. The
///      available remediation without it is re-pointing chains' relays at a fixed hub via
///      MessageExpiryRelay.setHub.
contract MessageExpiryHub is ISemver {
    /// @notice A recorded expiry notice from a destination chain. The attestor's shared ETHLockbox
    ///         (its cluster identity), the attestor chain ID, and the message's source chain ID are
    ///         the notice's mapping keys rather than fields.
    /// @custom:field anchorStateRegistry Shared AnchorStateRegistry of the attestor's cluster.
    /// @custom:field attestedAt          Timestamp of the attestation on the attestor chain.
    struct ExpiryNotice {
        address anchorStateRegistry;
        uint64 attestedAt;
    }

    /// @notice Thrown when a SystemConfig and its L1CrossDomainMessenger are not bound to each
    ///         other, or the messenger is not set.
    error MessageExpiryHub_InvalidMessenger();

    /// @notice Thrown when a chain's portal is not authorized by a shared ETHLockbox, or the portal
    ///         is not bound back to the SystemConfig that named it.
    error MessageExpiryHub_UnauthorizedPortal();

    /// @notice Thrown when a chain reports a zero chain ID.
    error MessageExpiryHub_InvalidChainId();

    /// @notice Thrown when an expiry notice is received from any cross-domain sender other than the
    ///         MessageExpiryRelay predeploy.
    error MessageExpiryHub_InvalidCrossDomainSender();

    /// @notice Thrown when an attestation names its own chain as the message source.
    error MessageExpiryHub_InvalidSourceChain();

    /// @notice Thrown when an attestation timestamp is in the future, i.e. greater than the current
    ///         L1 block timestamp. Honest attestations are always at least the withdrawal finality
    ///         delay old, so this never rejects a legitimate notice.
    error MessageExpiryHub_InvalidTimestamp();

    /// @notice Thrown when an attestation is not newer than the notice already recorded.
    error MessageExpiryHub_StaleNotice();

    /// @notice Thrown when forwarding a notice that does not exist.
    error MessageExpiryHub_NoticeNotFound();

    /// @notice Thrown when forwarding to a source chain that has not been registered.
    error MessageExpiryHub_ChainNotRegistered();

    /// @notice Thrown when the source chain's cluster no longer matches the notice's cluster.
    error MessageExpiryHub_ClusterMismatch();

    /// @notice Thrown when registering a chain would overwrite an existing registration for the same
    ///         cluster and chain ID with a different SystemConfig.
    error MessageExpiryHub_AlreadyRegistered();

    /// @notice Emitted when a chain is registered for notice forwarding.
    /// @param ethLockbox   Shared ETHLockbox identifying the chain's cluster.
    /// @param chainId      Chain ID of the registered chain.
    /// @param systemConfig SystemConfig of the registered chain.
    event ChainRegistered(address indexed ethLockbox, uint256 indexed chainId, address systemConfig);

    /// @notice Emitted when an expiry notice is received from a destination chain.
    /// @param ethLockbox      Shared ETHLockbox identifying the attestor's cluster.
    /// @param attestorChainId Chain ID of the attesting (destination) chain.
    /// @param msgHash         Hash of the attested message.
    /// @param sourceChainId   Chain ID the message was sent from.
    /// @param attestedAt      Timestamp of the attestation.
    event ExpiryNoticeReceived(
        address indexed ethLockbox,
        uint256 indexed attestorChainId,
        bytes32 indexed msgHash,
        uint256 sourceChainId,
        uint256 attestedAt
    );

    /// @notice Emitted when an expiry notice is forwarded to its source chain.
    /// @param ethLockbox      Shared ETHLockbox identifying the cluster.
    /// @param attestorChainId Chain ID of the attesting (destination) chain.
    /// @param msgHash         Hash of the attested message.
    /// @param sourceChainId   Chain ID the notice was forwarded to.
    event ExpiryNoticeForwarded(
        address indexed ethLockbox, uint256 indexed attestorChainId, bytes32 indexed msgHash, uint256 sourceChainId
    );

    /// @notice Address of the MessageExpiryRelay predeploy, identical on every chain. Mirrors
    ///         Predeploys.MESSAGE_EXPIRY_RELAY.
    address internal constant MESSAGE_EXPIRY_RELAY = 0x420000000000000000000000000000000000002E;

    /// @notice Semantic version.
    /// @custom:semver 1.0.0
    string public constant version = "1.0.0";

    /// @notice Recorded expiry notices, keyed by the attestor's shared ETHLockbox (its cluster
    ///         identity), the attestor chain ID, the message's source chain ID, and the message
    ///         hash. The lockbox key prevents clusters with colliding chain IDs from interfering
    ///         with each other's notices; the source chain ID key prevents an attestation naming a
    ///         bogus source from displacing a pending legitimate notice for the same message.
    mapping(address => mapping(uint256 => mapping(uint256 => mapping(bytes32 => ExpiryNotice)))) public notices;

    /// @notice Registered chains available as forwarding targets, keyed by their cluster's shared
    ///         ETHLockbox and their chain ID.
    mapping(address => mapping(uint256 => ISystemConfig)) public registeredChains;

    /// @notice Registers a chain as a forwarding target. Permissionless: all facts are verified on
    ///         chain against the chain's own contracts, and entries are namespaced by the shared
    ///         ETHLockbox that authorizes the chain's portal, so a registration can never shadow a
    ///         chain of a different cluster. Re-registration with the same SystemConfig is allowed
    ///         and simply re-validates; overwriting an existing registration with a different
    ///         SystemConfig is rejected.
    /// @param _systemConfig SystemConfig of the chain to register.
    function registerChain(ISystemConfig _systemConfig) external {
        address messenger = _systemConfig.l1CrossDomainMessenger();
        if (
            messenger == address(0)
                || address(IL1CrossDomainMessenger(messenger).systemConfig()) != address(_systemConfig)
        ) {
            revert MessageExpiryHub_InvalidMessenger();
        }

        (IETHLockbox lockbox,) = _validateClusterMembership(_systemConfig);

        uint256 chainId = _systemConfig.l2ChainId();
        if (chainId == 0) revert MessageExpiryHub_InvalidChainId();

        // Defense in depth: never let a registration flip an already-recorded (lockbox, chainId)
        // entry to a different SystemConfig. The reverse portal binding already prevents a forged
        // SystemConfig from validating, but pinning the entry removes the overwrite lever entirely.
        // Re-registering the same SystemConfig stays idempotent.
        ISystemConfig existing = registeredChains[address(lockbox)][chainId];
        if (address(existing) != address(0) && address(existing) != address(_systemConfig)) {
            revert MessageExpiryHub_AlreadyRegistered();
        }

        registeredChains[address(lockbox)][chainId] = _systemConfig;

        emit ChainRegistered(address(lockbox), chainId, address(_systemConfig));
    }

    /// @notice Receives an expiry notice from a destination chain's MessageExpiryRelay, relayed
    ///         through that chain's L1CrossDomainMessenger. The attestor's identity and cluster are
    ///         derived from the calling messenger, never from calldata. Notices for the same
    ///         (message, source chain) may only be replaced by strictly newer attestations, so a
    ///         premature attestation (taken before expiry) can always be superseded.
    /// @param _msgHash       Hash of the attested message.
    /// @param _sourceChainId Chain ID the message was sent from, used to route the forward.
    /// @param _attestedAt    Timestamp of the attestation on the attestor chain.
    function receiveExpiryNotice(bytes32 _msgHash, uint256 _sourceChainId, uint256 _attestedAt) external {
        ISystemConfig systemConfig = IL1CrossDomainMessenger(msg.sender).systemConfig();
        if (systemConfig.l1CrossDomainMessenger() != msg.sender) revert MessageExpiryHub_InvalidMessenger();
        if (ICrossDomainMessenger(msg.sender).xDomainMessageSender() != MESSAGE_EXPIRY_RELAY) {
            revert MessageExpiryHub_InvalidCrossDomainSender();
        }

        (IETHLockbox lockbox, address anchorStateRegistry) = _validateClusterMembership(systemConfig);

        uint256 attestorChainId = systemConfig.l2ChainId();
        if (attestorChainId == 0) revert MessageExpiryHub_InvalidChainId();
        if (_sourceChainId == attestorChainId) revert MessageExpiryHub_InvalidSourceChain();
        // Reject future-dated attestations. This bounds the stored timestamp by the current L1 time
        // (removing the unbounded-future lever) and, since block.timestamp fits in a uint64, keeps
        // the cast below safe.
        if (_attestedAt > block.timestamp) revert MessageExpiryHub_InvalidTimestamp();
        if (_attestedAt <= notices[address(lockbox)][attestorChainId][_sourceChainId][_msgHash].attestedAt) {
            revert MessageExpiryHub_StaleNotice();
        }

        notices[address(lockbox)][attestorChainId][_sourceChainId][_msgHash] =
            ExpiryNotice({ anchorStateRegistry: anchorStateRegistry, attestedAt: uint64(_attestedAt) });

        emit ExpiryNoticeReceived(address(lockbox), attestorChainId, _msgHash, _sourceChainId, _attestedAt);
    }

    /// @notice Forwards a recorded expiry notice to its source chain's MessageExpiryRelay through
    ///         that chain's L1CrossDomainMessenger (a deposit transaction, so the source chain
    ///         cannot censor it). Permissionless and repeatable — the caller pays the deposit's
    ///         resource metering gas burn, and consumption on the source chain is idempotent. The
    ///         source chain must be registered and must verifiably belong to the same cluster as
    ///         the attestor at forwarding time.
    /// @param _ethLockbox      Shared ETHLockbox identifying the cluster the notice belongs to.
    /// @param _attestorChainId Chain ID of the attesting (destination) chain.
    /// @param _sourceChainId   Chain ID the message was sent from, i.e. the forwarding target.
    /// @param _msgHash         Hash of the attested message.
    /// @param _minGasLimit     Minimum gas limit for the MessageExpiryRelay call on the source
    ///                         chain. Must cover expiry verification plus the application callback;
    ///                         an undergassed relay lands in the source chain's failed messages and
    ///                         remains permissionlessly replayable there.
    function forwardExpiryNotice(
        address _ethLockbox,
        uint256 _attestorChainId,
        uint256 _sourceChainId,
        bytes32 _msgHash,
        uint32 _minGasLimit
    )
        external
    {
        ExpiryNotice memory notice = notices[_ethLockbox][_attestorChainId][_sourceChainId][_msgHash];
        if (notice.attestedAt == 0) revert MessageExpiryHub_NoticeNotFound();

        ISystemConfig systemConfig = registeredChains[_ethLockbox][_sourceChainId];
        if (address(systemConfig) == address(0)) revert MessageExpiryHub_ChainNotRegistered();

        address messenger = systemConfig.l1CrossDomainMessenger();
        if (
            messenger == address(0)
                || address(IL1CrossDomainMessenger(messenger).systemConfig()) != address(systemConfig)
        ) {
            revert MessageExpiryHub_InvalidMessenger();
        }

        (IETHLockbox lockbox, address anchorStateRegistry) = _validateClusterMembership(systemConfig);
        if (address(lockbox) != _ethLockbox || anchorStateRegistry != notice.anchorStateRegistry) {
            revert MessageExpiryHub_ClusterMismatch();
        }

        ICrossDomainMessenger(messenger).sendMessage({
            _target: MESSAGE_EXPIRY_RELAY,
            _message: abi.encodeCall(
                IMessageExpiryRelay.receiveExpiry, (_msgHash, _attestorChainId, uint256(notice.attestedAt))
            ),
            _minGasLimit: _minGasLimit
        });

        emit ExpiryNoticeForwarded(_ethLockbox, _attestorChainId, _msgHash, _sourceChainId);
    }

    /// @notice Validates that a chain belongs to a shared-lockbox cluster: its SystemConfig and
    ///         OptimismPortal must be bound to each other, and its portal must be authorized by its
    ///         own ETHLockbox, which is the cluster identity.
    /// @param _systemConfig SystemConfig of the chain to validate.
    /// @return lockbox_             The chain's shared ETHLockbox.
    /// @return anchorStateRegistry_ The chain's AnchorStateRegistry.
    function _validateClusterMembership(ISystemConfig _systemConfig)
        internal
        view
        returns (IETHLockbox lockbox_, address anchorStateRegistry_)
    {
        IOptimismPortal2 portal = IOptimismPortal2(payable(_systemConfig.optimismPortal()));
        if (address(portal) == address(0)) revert MessageExpiryHub_UnauthorizedPortal();

        // Reverse-bind the portal to the SystemConfig. Without this, a forged SystemConfig could
        // name a real, cluster-authorized portal as its own; the lockbox check below would then
        // pass and a forged notice would be recorded under a real cluster's lockbox key. A real
        // portal's systemConfig() points back only at the SystemConfig it was initialized with, so
        // a borrowed portal fails here, and a fake portal cannot be in a real lockbox's authorized
        // set.
        if (address(portal.systemConfig()) != address(_systemConfig)) {
            revert MessageExpiryHub_UnauthorizedPortal();
        }

        lockbox_ = portal.ethLockbox();
        if (address(lockbox_) == address(0) || !lockbox_.authorizedPortals(portal)) {
            revert MessageExpiryHub_UnauthorizedPortal();
        }

        anchorStateRegistry_ = address(portal.anchorStateRegistry());
    }
}
