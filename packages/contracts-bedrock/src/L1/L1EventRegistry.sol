// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Libraries
import { Predeploys } from "src/libraries/Predeploys.sol";

// Interfaces
import { IETHLockbox } from "interfaces/L1/IETHLockbox.sol";
import { IOptimismPortal2 } from "interfaces/L1/IOptimismPortal2.sol";
import { ICrossL2Inbox, Identifier } from "interfaces/L2/ICrossL2Inbox.sol";
import { ISemver } from "interfaces/universal/ISemver.sol";

/// @title L1EventRegistry
/// @notice Records recent L2 events finalized through standard withdrawal proofs and relays their
///         certificates to other chains in the same ETHLockbox cluster.
/// @dev This contract is intentionally immutable. A new registry requires updating CrossL2Inbox
///      through the L2 ProxyAdmin on each participating chain.
contract L1EventRegistry is ISemver {
    /// @notice Thrown when constructed with a zero ETHLockbox address.
    error L1EventRegistry_InvalidLockbox();

    /// @notice Thrown when a source or destination portal is not in the registry's cluster.
    error L1EventRegistry_UnauthorizedPortal();

    /// @notice Thrown when a source withdrawal was not initiated by CrossL2Inbox.
    error L1EventRegistry_UnauthorizedL2Sender();

    /// @notice Thrown when an identifier's chain ID does not match the source portal's chain.
    error L1EventRegistry_WrongSourceChain();

    /// @notice Thrown when attempting to relay an event that has not been registered.
    error L1EventRegistry_EventNotRegistered();

    /// @notice Emitted when an event is registered on L1.
    event EventRegistered(bytes32 indexed certificate, bytes32 indexed payloadHash, Identifier id);

    /// @notice Emitted when an event certificate is relayed to a destination portal.
    event EventRelayed(bytes32 indexed certificate, address indexed portal, bool executeMessage);

    /// @notice Shared ETHLockbox defining the set of source and destination portals.
    IETHLockbox public immutable ethLockbox;

    /// @notice Registered event certificates.
    mapping(bytes32 => bool) public registeredEvents;

    /// @notice Semantic version.
    /// @custom:semver 1.0.0
    string public constant version = "1.0.0";

    /// @param _ethLockbox Shared ETHLockbox for the interop cluster.
    constructor(IETHLockbox _ethLockbox) {
        if (address(_ethLockbox) == address(0)) revert L1EventRegistry_InvalidLockbox();
        ethLockbox = _ethLockbox;
    }

    /// @notice Records an event exported from an authorized L2 CrossL2Inbox.
    /// @dev This function is the target of a zero-value L2 withdrawal. The portal authenticates
    ///      the source CrossL2Inbox through l2Sender while the lockbox authenticates cluster membership.
    function registerEvent(Identifier calldata _id, bytes32 _payloadHash) external {
        IOptimismPortal2 portal = IOptimismPortal2(payable(msg.sender));
        _assertAuthorizedPortal(portal);
        if (portal.l2Sender() != Predeploys.CROSS_L2_INBOX) {
            revert L1EventRegistry_UnauthorizedL2Sender();
        }
        if (portal.systemConfig().l2ChainId() != _id.chainId) revert L1EventRegistry_WrongSourceChain();

        bytes32 certificate = calculateCertificate(_id, _payloadHash);
        registeredEvents[certificate] = true;
        emit EventRegistered(certificate, _payloadHash, _id);
    }

    /// @notice Relays a generic event certificate to another chain in the cluster.
    function relayEvent(
        IOptimismPortal2 _destinationPortal,
        Identifier calldata _id,
        bytes32 _payloadHash,
        uint64 _gasLimit
    )
        external
    {
        bytes32 certificate = _assertRegistered(_id, _payloadHash);
        _assertAuthorizedPortal(_destinationPortal);

        bytes memory data = abi.encodeCall(ICrossL2Inbox.importEvent, (_id, _payloadHash));
        _destinationPortal.depositTransaction(Predeploys.CROSS_L2_INBOX, 0, _gasLimit, false, data);
        emit EventRelayed(certificate, address(_destinationPortal), false);
    }

    /// @notice Relays a certified SentMessage event and executes it in one destination deposit.
    function relayMessage(
        IOptimismPortal2 _destinationPortal,
        Identifier calldata _id,
        bytes calldata _sentMessage,
        uint64 _gasLimit
    )
        external
    {
        bytes32 certificate = _assertRegistered(_id, keccak256(_sentMessage));
        _assertAuthorizedPortal(_destinationPortal);

        bytes memory data = abi.encodeCall(ICrossL2Inbox.importAndExecute, (_id, _sentMessage));
        _destinationPortal.depositTransaction(Predeploys.CROSS_L2_INBOX, 0, _gasLimit, false, data);
        emit EventRelayed(certificate, address(_destinationPortal), true);
    }

    /// @notice Calculates the canonical certificate for an event and payload hash.
    function calculateCertificate(Identifier memory _id, bytes32 _payloadHash) public pure returns (bytes32) {
        return keccak256(abi.encode(_id, _payloadHash));
    }

    /// @notice Verifies that a portal is a current member of this registry's ETHLockbox cluster.
    function _assertAuthorizedPortal(IOptimismPortal2 _portal) internal view {
        if (!_portalIsAuthorized(_portal)) revert L1EventRegistry_UnauthorizedPortal();
    }

    /// @notice Returns whether a portal is a current member of this registry's ETHLockbox cluster.
    function _portalIsAuthorized(IOptimismPortal2 _portal) internal view returns (bool) {
        return ethLockbox.authorizedPortals(_portal) && _portal.ethLockbox() == ethLockbox;
    }

    /// @notice Verifies and returns the certificate for an event and payload hash.
    function _assertRegistered(
        Identifier calldata _id,
        bytes32 _payloadHash
    )
        internal
        view
        returns (bytes32 certificate_)
    {
        certificate_ = calculateCertificate(_id, _payloadHash);
        if (!registeredEvents[certificate_]) revert L1EventRegistry_EventNotRegistered();
    }
}
