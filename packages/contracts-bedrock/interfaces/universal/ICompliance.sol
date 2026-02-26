// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { IProxyAdmin } from "interfaces/universal/IProxyAdmin.sol";

/// @title ICompliance
/// @notice Interface for the Compliance contract that provides optional compliance screening
///         for cross-chain transactions on OptimismPortal2 (deposits) and L2ToL1MessagePasser
///         (withdrawals).
interface ICompliance {
    /// @notice The status of a transaction in the compliance system.
    enum Status {
        Approved,
        Pending,
        Rejected,
        Refunded
    }

    /// @notice Emitted when a transaction is flagged as pending by compliance rules.
    event Pending(
        bytes32 indexed id,
        address indexed from,
        address indexed to,
        uint256 value,
        uint256 mint,
        uint64 gasLimit,
        uint256 nonce,
        bytes data
    );

    /// @notice Emitted when a transaction is rejected — either automatically during check()
    ///         or when the owner calls reject().
    event Rejected(bytes32 indexed id);

    /// @notice Emitted when a transaction is approved (either automatically or by the owner).
    event Approved(bytes32 indexed id);

    /// @notice Emitted when a rejected transaction's ETH is refunded to the sender.
    event Refunded(bytes32 indexed id);

    // Solady Ownable events
    event OwnershipTransferred(address indexed oldOwner, address indexed newOwner);
    event OwnershipHandoverRequested(address indexed pendingOwner);
    event OwnershipHandoverCanceled(address indexed pendingOwner);

    // ReinitializableBase event
    event Initialized(uint8 version);

    /// @notice Thrown when the caller is not the bridge contract.
    error Compliance_OnlyBridge();

    /// @notice Thrown when attempting to settle a transaction that is not pending.
    error Compliance_NotPending();

    /// @notice Thrown when an ETH transfer fails.
    error Compliance_TransferFailed();

    /// @notice Thrown when attempting to add a rule that already exists.
    error Compliance_DuplicateRule();

    /// @notice Thrown when attempting to remove a rule that does not exist.
    error Compliance_RuleNotFound();

    /// @notice Thrown when a rule returns an invalid status (Refunded).
    error Compliance_InvalidRuleStatus();

    // Solady Ownable errors
    error Unauthorized();
    error NewOwnerIsZeroAddress();
    error NoHandoverRequest();
    error AlreadyInitialized();
    error Reentrancy();

    // ProxyAdminOwnedBase errors
    error ProxyAdminOwnedBase_NotSharedProxyAdminOwner();
    error ProxyAdminOwnedBase_NotProxyAdminOwner();
    error ProxyAdminOwnedBase_NotProxyAdmin();
    error ProxyAdminOwnedBase_NotProxyAdminOrProxyAdminOwner();
    error ProxyAdminOwnedBase_ProxyAdminNotFound();
    error ProxyAdminOwnedBase_NotResolvedDelegateProxy();

    // ReinitializableBase errors
    error ReinitializableBase_ZeroInitVersion();

    /// @notice Checks a transaction against all configured compliance rules.
    function check(
        address _from,
        address _to,
        uint256 _value,
        uint64 _gasLimit,
        bool _isCreation,
        bytes calldata _data,
        uint256 _nonce
    )
        external
        payable
        returns (bool allowed_);

    /// @notice Returns the Status of a cross chain message and whether it is final.
    function status(bytes32 _id) external view returns (bool isFinal_, Status status_);

    /// @notice Owner approves a pending transaction.
    function approve(bytes32 _id) external;

    /// @notice Owner rejects a pending transaction.
    function reject(bytes32 _id) external;

    /// @notice Settles a pending transaction.
    function settle(
        address _from,
        address _to,
        uint256 _value,
        uint256 _mint,
        uint64 _gasLimit,
        bool _isCreation,
        bytes calldata _data,
        uint256 _nonce
    )
        external;

    /// @notice Adds a new compliance rule to the set.
    function addRule(address _rule) external;

    /// @notice Removes a compliance rule from the set.
    function removeRule(address _rule) external;

    /// @notice Returns the bridge contract address.
    function bridge() external view returns (address payable);

    /// @notice Returns all rule addresses.
    function rules() external view returns (address[] memory);

    /// @notice Returns whether the given address is a registered rule.
    function hasRule(address _rule) external view returns (bool);

    /// @notice Initializes the compliance contract.
    function initialize(address _bridge, address _owner) external;

    function version() external pure returns (string memory);

    // Solady Ownable functions
    function owner() external view returns (address result);
    function transferOwnership(address newOwner) external payable;
    function renounceOwnership() external payable;
    function requestOwnershipHandover() external payable;
    function cancelOwnershipHandover() external payable;
    function completeOwnershipHandover(address pendingOwner) external payable;
    function ownershipHandoverExpiresAt(address pendingOwner) external view returns (uint256 result);

    // ProxyAdminOwnedBase functions
    function proxyAdmin() external view returns (IProxyAdmin);
    function proxyAdminOwner() external view returns (address);

    // ReinitializableBase functions
    function initVersion() external view returns (uint8);
}
