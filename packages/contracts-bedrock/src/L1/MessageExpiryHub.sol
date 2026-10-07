// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Libraries
import { Predeploys } from "src/libraries/Predeploys.sol";

// Interfaces
import { ISemver } from "interfaces/universal/ISemver.sol";
import { ICrossDomainMessenger } from "interfaces/universal/ICrossDomainMessenger.sol";
import { IL1CrossDomainMessenger } from "interfaces/L1/IL1CrossDomainMessenger.sol";
import { ISystemConfig } from "interfaces/L1/ISystemConfig.sol";
import { IOptimismPortal2 } from "interfaces/L1/IOptimismPortal2.sol";
import { IETHLockbox } from "interfaces/L1/IETHLockbox.sol";
import { IL2ToL2CrossDomainMessenger } from "interfaces/L2/IL2ToL2CrossDomainMessenger.sol";

/// @title MessageExpiryHub
/// @notice Carries "this message was never relayed" facts between the chains of an interop cluster.
///         A message's destination chain exports the fact through its withdrawal path; the hub
///         records it; anyone then forwards it to the message's source chain through that chain's
///         deposit path, where the L2ToL2CrossDomainMessenger marks the message expired if the fact
///         is dated after the message expiry window. Both legs are censorship resistant.
///
///         The hub is an ownerless singleton. It is configured by nothing: a chain is identified by
///         its SystemConfig, which must be bound both ways to its L1CrossDomainMessenger and to its
///         OptimismPortal, and a cluster is identified by the ETHLockbox that authorizes the portal
///         together with the portal's AnchorStateRegistry. A fact is only ever forwarded within the
///         cluster it came from.
///
///         Recording and forwarding are separate calls: a forward is a deposit, whose gas burn has
///         no place inside the fixed gas of a relayed withdrawal.
contract MessageExpiryHub is ISemver {
    /// @notice Thrown when a SystemConfig is not bound both ways to its messenger and portal, or
    ///         its portal is not authorized by its ETHLockbox.
    error MessageExpiryHub_InvalidChain();

    /// @notice Thrown when a fact is not received from the L2ToL2CrossDomainMessenger of a chain.
    error MessageExpiryHub_InvalidSender();

    /// @notice Thrown when forwarding a fact the hub has not recorded for the source's cluster.
    error MessageExpiryHub_UnknownFact();

    /// @notice Emitted when a fact is recorded.
    /// @param cluster       The cluster the fact came from (see `cluster`).
    /// @param messageHash   Hash of the message that was not relayed.
    /// @param source        Chain ID of the message's source chain.
    /// @param undeliveredAt Destination timestamp at which the message had not been relayed.
    event UndeliveredMessageRecorded(
        bytes32 indexed cluster, bytes32 indexed messageHash, uint256 indexed source, uint256 undeliveredAt
    );

    /// @notice Emitted when a fact is forwarded to its source chain.
    /// @param cluster       The cluster the fact came from.
    /// @param messageHash   Hash of the message that was not relayed.
    /// @param source        Chain ID of the message's source chain.
    /// @param undeliveredAt Destination timestamp at which the message had not been relayed.
    event UndeliveredMessageForwarded(
        bytes32 indexed cluster, bytes32 indexed messageHash, uint256 indexed source, uint256 undeliveredAt
    );

    /// @notice Semantic version.
    /// @custom:semver 1.0.0
    string public constant version = "1.0.0";

    /// @notice Recorded facts, by `factId`.
    mapping(bytes32 => bool) public facts;

    /// @notice Records that a message was not relayed on its destination chain by `_undeliveredAt`.
    ///         Callable only by a chain's L1CrossDomainMessenger relaying a withdrawal from that
    ///         chain's L2ToL2CrossDomainMessenger, which computed `_messageHash` with its own chain
    ///         as the destination.
    /// @param _messageHash   Hash of the message that was not relayed.
    /// @param _source        Chain ID of the message's source chain.
    /// @param _undeliveredAt Destination timestamp at which the message had not been relayed.
    function receiveUndeliveredMessage(bytes32 _messageHash, uint256 _source, uint256 _undeliveredAt) external {
        (address messenger, bytes32 cluster_) = _chain(IL1CrossDomainMessenger(msg.sender).systemConfig());
        if (
            messenger != msg.sender
                || ICrossDomainMessenger(messenger).xDomainMessageSender() != Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER
        ) {
            revert MessageExpiryHub_InvalidSender();
        }

        facts[factId(cluster_, _messageHash, _source, _undeliveredAt)] = true;

        emit UndeliveredMessageRecorded(cluster_, _messageHash, _source, _undeliveredAt);
    }

    /// @notice Forwards a recorded fact to the message's source chain, as a deposit to its
    ///         L2ToL2CrossDomainMessenger. Permissionless and repeatable: the caller pays the
    ///         deposit's gas, and marking a message expired twice changes nothing.
    /// @param _source        SystemConfig of the message's source chain. It must be in the cluster
    ///                       the fact came from.
    /// @param _messageHash   Hash of the message that was not relayed.
    /// @param _undeliveredAt Destination timestamp at which the message had not been relayed.
    /// @param _minGasLimit   Minimum gas limit for the call on the source chain. An undergassed call
    ///                       lands in the source chain's failed messages and can be replayed there.
    function forwardUndeliveredMessage(
        ISystemConfig _source,
        bytes32 _messageHash,
        uint256 _undeliveredAt,
        uint32 _minGasLimit
    )
        external
    {
        (address messenger, bytes32 cluster_) = _chain(_source);
        uint256 sourceChainId = _source.l2ChainId();
        if (!facts[factId(cluster_, _messageHash, sourceChainId, _undeliveredAt)]) {
            revert MessageExpiryHub_UnknownFact();
        }

        ICrossDomainMessenger(messenger).sendMessage({
            _target: Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER,
            _message: abi.encodeCall(IL2ToL2CrossDomainMessenger.expireMessage, (_messageHash, _undeliveredAt)),
            _minGasLimit: _minGasLimit
        });

        emit UndeliveredMessageForwarded(cluster_, _messageHash, sourceChainId, _undeliveredAt);
    }

    /// @notice Returns the key a fact is recorded under.
    /// @param _cluster       The cluster the fact came from.
    /// @param _messageHash   Hash of the message that was not relayed.
    /// @param _source        Chain ID of the message's source chain.
    /// @param _undeliveredAt Destination timestamp at which the message had not been relayed.
    function factId(
        bytes32 _cluster,
        bytes32 _messageHash,
        uint256 _source,
        uint256 _undeliveredAt
    )
        public
        pure
        returns (bytes32)
    {
        return keccak256(abi.encode(_cluster, _messageHash, _source, _undeliveredAt));
    }

    /// @notice Returns a chain's cluster: the ETHLockbox that authorizes its portal and its
    ///         AnchorStateRegistry. Two chains are in the same cluster only if both match.
    /// @param _systemConfig SystemConfig of the chain.
    function cluster(ISystemConfig _systemConfig) external view returns (bytes32 cluster_) {
        (, cluster_) = _chain(_systemConfig);
    }

    /// @notice Checks that a SystemConfig is bound both ways to its L1CrossDomainMessenger and its
    ///         OptimismPortal, and that the portal is authorized by its ETHLockbox. A forged
    ///         SystemConfig cannot pass: the real messenger and portal point back only at the real
    ///         SystemConfig, and a fake portal is not in a real lockbox.
    /// @param _systemConfig SystemConfig of the chain.
    /// @return messenger_ The chain's L1CrossDomainMessenger.
    /// @return cluster_   The chain's cluster.
    function _chain(ISystemConfig _systemConfig) internal view returns (address messenger_, bytes32 cluster_) {
        messenger_ = _systemConfig.l1CrossDomainMessenger();
        if (messenger_ == address(0) || IL1CrossDomainMessenger(messenger_).systemConfig() != _systemConfig) {
            revert MessageExpiryHub_InvalidChain();
        }

        IOptimismPortal2 portal = IOptimismPortal2(payable(_systemConfig.optimismPortal()));
        if (address(portal) == address(0) || portal.systemConfig() != _systemConfig) {
            revert MessageExpiryHub_InvalidChain();
        }

        IETHLockbox lockbox = portal.ethLockbox();
        if (address(lockbox) == address(0) || !lockbox.authorizedPortals(portal)) {
            revert MessageExpiryHub_InvalidChain();
        }

        cluster_ = keccak256(abi.encode(lockbox, portal.anchorStateRegistry()));
    }
}
