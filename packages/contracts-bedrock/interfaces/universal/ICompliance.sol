// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

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

    /// @notice Emitted when a transaction is rejected by compliance rules.
    event Rejected(
        bytes32 indexed id,
        address indexed from,
        address indexed to,
        uint256 value,
        uint256 mint,
        uint64 gasLimit,
        uint256 nonce,
        bytes data
    );

    /// @notice Emitted when a transaction is approved (either automatically or by the owner).
    event Approved(bytes32 indexed id);

    /// @notice Emitted when a rejected transaction's ETH is refunded to the sender.
    event Refunded(bytes32 indexed id);

    /// @notice Thrown when the caller is not the bridge contract.
    error Compliance_OnlyBridge();

    /// @notice Thrown when the caller is not the owner.
    error Compliance_OnlyOwner();

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

    /// @notice Checks a transaction against all configured compliance rules.
    /// @param _from       The sender of the transaction.
    /// @param _to         The recipient of the transaction.
    /// @param _value      The ETH value of the transaction.
    /// @param _gasLimit   The gas limit for execution.
    /// @param _isCreation Whether the transaction creates a contract.
    /// @param _data       The calldata of the transaction.
    /// @param _nonce      The nonce of the transaction (used for L2 withdrawals).
    /// @return allowed_   True if the transaction is approved to proceed immediately.
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
    /// @dev    Masks off bit 255 (the owner-override flag) and returns the Status
    ///         enum value from the least significant bits. A status is considered
    ///         final when it is Refunded (always final) or when the owner-override
    ///         flag (bit 255) is set (Pending or Rejected with an explicit owner
    ///         decision). Rule re-evaluation in settle() must respect finality:
    ///         a finalized status takes precedence even if rules would return a
    ///         stricter result.
    /// @param _id The hashed cross chain message identifier.
    /// @return isFinal_ True if the status is final and cannot be changed by rule re-evaluation.
    /// @return status_ The current Status of the message.
    function status(bytes32 _id) external view returns (bool isFinal_, Status status_);

    /// @notice Owner approves a pending transaction.
    /// @param _id The unique identifier of the transaction.
    function approve(bytes32 _id) external;

    /// @notice Owner rejects a pending transaction.
    /// @param _id The unique identifier of the transaction.
    function reject(bytes32 _id) external;

    /// @notice Settles a pending transaction. Anyone can call this.
    ///         The caller provides the full preimage fields; the contract recomputes the
    ///         hash and verifies it against the stored status. If the owner has set an
    ///         override, uses that. Otherwise re-evaluates rules. Approved transactions
    ///         are forwarded to the bridge. Rejected transactions have their ETH refunded
    ///         to the sender.
    /// @param _from       The original depositor/withdrawer address.
    /// @param _to         The recipient address on the remote chain.
    /// @param _value      The ETH value.
    /// @param _mint       The ETH to mint on L2 (msg.value for deposits, always 0 for withdrawals).
    /// @param _gasLimit   The gas limit for remote execution.
    /// @param _isCreation Whether this is a contract creation (always false for withdrawals).
    /// @param _data       The calldata.
    /// @param _nonce      The reserved nonce (always 0 for deposits).
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
    /// @param _rule The address of the rule contract to add.
    function addRule(address _rule) external;

    /// @notice Removes a compliance rule from the set.
    /// @param _rule The address of the rule contract to remove.
    function removeRule(address _rule) external;

    /// @notice Returns the bridge contract address.
    function bridge() external view returns (address payable);

    /// @notice Returns all rule addresses.
    /// @return The array of rule contract addresses.
    function rules() external view returns (address[] memory);

    /// @notice Returns whether the given address is a registered rule.
    /// @param _rule The address to check.
    /// @return True if the address is a registered rule.
    function hasRule(address _rule) external view returns (bool);

    /// @notice Initializes the compliance contract.
    /// @param _bridge The address of the bridge contract.
    /// @param _owner  The address of the owner.
    function initialize(address _bridge, address _owner) external;

    function version() external pure returns (string memory);
}
