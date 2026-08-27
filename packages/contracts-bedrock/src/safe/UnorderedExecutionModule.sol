// SPDX-License-Identifier: MIT
pragma solidity 0.8.25;

// Safe
import { Safe } from "safe-contracts/Safe.sol";
import { Enum } from "safe-contracts/common/Enum.sol";

// Interfaces
import { ISemver } from "interfaces/universal/ISemver.sol";

/// @title UnorderedExecutionModule
/// @notice Safe Module that allows a Safe's owners to authorize a transaction by signing the
///         Safe's own transaction hash computed with a "hash-once" value in the nonce slot
///         instead of the Safe's sequential nonce. Because the hash-once value is pinned per
///         task (derived from a unique string in the task's configuration) and never depends on
///         execution order, signatures remain valid regardless of the order in which tasks are
///         executed, and reordering a queue of pending tasks never requires re-signing.
/// Usage:
///     The Safe must first enable this module via ModuleManager.enableModule(). Owners then sign
///     the digest returned by transactionHash() — which is exactly the Safe's own
///     getTransactionHash() with the hash-once value as the nonce and the gas and refund fields
///     (safeTxGas, baseGas, gasPrice, gasToken, refundReceiver) all zero, matching how
///     superchain-ops tasks are signed — using any method the Safe itself supports: private key
///     signatures, Safe.approveHash(), or EIP-1271 contract signatures (including nested Safes).
///     Existing Safe signing tooling works unchanged. Once a threshold of signatures has been
///     collected, anyone may call execute() — the same permissionless-relay model as the Safe's
///     own execTransaction().
/// Hash-once values:
///     The hash-once value must be unique per task and strictly greater than type(uint128).max,
///     so that it can never collide with the Safe's real sequential nonce. Without this floor, a
///     signature produced for this module could also become executable through the ordinary
///     execTransaction() path once the Safe's nonce reached the same value — a collision we
///     realize is extremely unlikely, but the floor makes it impossible. deriveHashOnce()
///     computes a canonical value from a unique string.
/// Replay protection and cancellation:
///     Each (Safe, hash-once value) pair can be executed at most once. There is no on-chain
///     cancellation or expiry mechanism: to cancel a signed task, sign and execute a no-op
///     transaction (e.g. a zero-value call to address(0)) using the same hash-once value. The
///     no-op consumes the hash-once value and permanently prevents the unwanted task from being
///     executed. Until a task's hash-once value has been consumed, its signatures remain valid
///     indefinitely; they are checked against the Safe's *current* owner set and threshold, so
///     rotating owners or raising the threshold also invalidates them.
/// Security notes:
///     - Module transactions are NOT inspected by a transaction guard on Safe versions 1.3.0 and
///       1.4.1, so transactions executed through this module bypass any guard (e.g. a timelock
///       guard) enabled on the Safe.
///     - The relayer chooses the gas for execute(). A failed call reverts and leaves the
///       hash-once value unconsumed, but a target that swallows an inner failure — a nested
///       Safe's execTransaction(), or a multicall that allows failures — can report success on
///       a partial, possibly gas-starved execution and consume the hash-once value anyway. Task
///       authors should route calls through failure-propagating targets such as
///       MultiSendCallOnly.
/// Safe compatibility:
///     Works with Safes exposing the 1.3.0/1.4.1 interfaces this module relies on:
///     encodeTransactionData(), getTransactionHash(), checkSignatures(bytes32,bytes,bytes),
///     isModuleEnabled() and execTransactionFromModuleReturnData(). Rather than pinning
///     versions, execute() verifies the single property that could otherwise diverge silently —
///     that getTransactionHash() is the keccak256 of encodeTransactionData(). All other
///     semantics of those functions (signature checking, domain separation) are trusted as-is,
///     so the module should only be enabled on Safe versions known to implement them correctly;
///     1.3.0 and 1.4.1 do. Tests exercise the vendored 1.4.1 implementation.
contract UnorderedExecutionModule is ISemver {
    /// @notice Parameters for the Safe transaction being executed. These are the fields
    ///         execTransactionFromModuleReturnData() consumes; the remaining fields of the
    ///         signed Safe transaction (safeTxGas, baseGas, gasPrice, gasToken, refundReceiver)
    ///         are hard-coded to zero, and the nonce is supplied separately as the hash-once
    ///         value.
    /// @custom:field to The address of the contract to call.
    /// @custom:field value The ETH value to send with the call.
    /// @custom:field data The calldata to send with the call.
    /// @custom:field operation The operation to perform (Call or DelegateCall).
    struct ExecTransactionParams {
        address to;
        uint256 value;
        bytes data;
        Enum.Operation operation;
    }

    /// @notice Error for when this module is not enabled on the Safe.
    error UnorderedExecutionModule_ModuleNotEnabled();

    /// @notice Error for when the hash-once value is small enough to collide with a real Safe
    ///         nonce.
    error UnorderedExecutionModule_HashOnceTooSmall();

    /// @notice Error for when the hash-once value has already been consumed by an execution.
    error UnorderedExecutionModule_HashOnceAlreadyUsed();

    /// @notice Error for when the Safe's transaction hashing diverges from the scheme this
    ///         module relies on.
    error UnorderedExecutionModule_UnsupportedSafe();

    /// @notice Error for when the transaction's call reverts, carrying the raw revert data.
    error UnorderedExecutionModule_ExecutionFailed(bytes);

    /// @notice Emitted when a transaction is executed.
    /// @param safe The Safe the transaction was executed through.
    /// @param txHash The Safe transaction hash identifying the transaction.
    /// @param hashOnce The hash-once value consumed by the execution.
    event TransactionExecuted(Safe indexed safe, bytes32 indexed txHash, uint256 indexed hashOnce);

    /// @notice Hash-once values that have been consumed by an execution, keyed by Safe.
    mapping(Safe => mapping(uint256 => bool)) internal _executed;

    /// @notice Semantic version.
    /// @custom:semver 1.0.0-beta.1
    string public constant version = "1.0.0-beta.1";

    /// @notice Getter function for whether a hash-once value has been consumed.
    /// @param _safe The Safe the hash-once value belongs to.
    /// @param _hashOnce The hash-once value.
    /// @return executed_ Whether the hash-once value has been consumed by an execution.
    function executed(Safe _safe, uint256 _hashOnce) public view returns (bool executed_) {
        executed_ = _executed[_safe][_hashOnce];
    }

    /// @notice Derives a canonical hash-once value from a unique string, e.g. the hashOnceInput
    ///         pinned in a superchain-ops task's configuration.
    /// @param _input Unique string identifying the task.
    /// @return hashOnce_ Hash-once value to use as the transaction's nonce.
    function deriveHashOnce(string memory _input) public pure returns (uint256 hashOnce_) {
        hashOnce_ = uint256(keccak256(bytes(_input)));
    }

    /// @notice Computes the Safe transaction hash that owners must sign to authorize a
    ///         transaction. Identical to the Safe's own getTransactionHash() with the hash-once
    ///         value in the nonce slot and zeroed gas and refund fields.
    /// @param _safe The Safe the transaction is to be executed through.
    /// @param _params The parameters of the transaction.
    /// @param _hashOnce The transaction's hash-once value.
    /// @return txHash_ Safe transaction hash of the transaction.
    function transactionHash(
        Safe _safe,
        ExecTransactionParams calldata _params,
        uint256 _hashOnce
    )
        public
        view
        returns (bytes32 txHash_)
    {
        txHash_ = _safe.getTransactionHash(
            _params.to, _params.value, _params.data, _params.operation, 0, 0, 0, address(0), address(0), _hashOnce
        );
    }

    /// @notice Executes a transaction once a threshold of owner signatures has been collected.
    ///         Callable by anyone, matching the relay model of the Safe's own execTransaction().
    ///         Reverts if the transaction's call reverts, leaving the hash-once value unconsumed
    ///         so that the transaction can be retried.
    /// @param _safe The Safe to execute the transaction through.
    /// @param _params The parameters of the transaction.
    /// @param _hashOnce The transaction's hash-once value. Must be greater than
    ///                  type(uint128).max.
    /// @param _signatures Owner signatures over transactionHash(), in the format accepted by the
    ///                    Safe's checkSignatures() function.
    /// @return returnData_ Return data of the transaction's call.
    function execute(
        Safe _safe,
        ExecTransactionParams calldata _params,
        uint256 _hashOnce,
        bytes calldata _signatures
    )
        external
        returns (bytes memory returnData_)
    {
        // The Safe itself would reject the module call, but checking here gives a clear error.
        if (!_safe.isModuleEnabled(address(this))) {
            revert UnorderedExecutionModule_ModuleNotEnabled();
        }

        // A hash-once value that a sequential nonce could actually reach would let the same
        // signatures execute both through this module and through execTransaction().
        if (_hashOnce <= type(uint128).max) {
            revert UnorderedExecutionModule_HashOnceTooSmall();
        }

        // Each hash-once value can be consumed at most once. This also lets a signed task be
        // cancelled by executing a no-op transaction with the same hash-once value.
        if (_executed[_safe][_hashOnce]) {
            revert UnorderedExecutionModule_HashOnceAlreadyUsed();
        }

        bytes32 txHash = transactionHash(_safe, _params, _hashOnce);
        bytes memory txHashData = _encodeTransactionData(_safe, _params, _hashOnce);

        // The digest owners sign must be the hash of the preimage passed to checkSignatures(),
        // or EIP-1271 owners would validate different data than what was signed. This holds on
        // Safe 1.3.0 and 1.4.1; a Safe whose hashing scheme diverges is rejected.
        if (keccak256(txHashData) != txHash) {
            revert UnorderedExecutionModule_UnsupportedSafe();
        }

        // Verify signatures using the Safe's own logic, so that the Safe's current owner set and
        // threshold apply and all Safe-supported signing methods work. Reverts if invalid.
        _safe.checkSignatures(txHash, txHashData, _signatures);

        // Consume the hash-once value before the external call so that the call cannot re-enter
        // and execute the same transaction again.
        _executed[_safe][_hashOnce] = true;

        // Execute the transaction through the Safe.
        bool success;
        (success, returnData_) =
            _safe.execTransactionFromModuleReturnData(_params.to, _params.value, _params.data, _params.operation);

        // Unlike the Safe's execTransaction(), revert if the call failed. The revert unwinds the
        // consumed hash-once value so the transaction can be retried.
        if (!success) {
            revert UnorderedExecutionModule_ExecutionFailed(returnData_);
        }

        emit TransactionExecuted(_safe, txHash, _hashOnce);
    }

    /// @notice Fetches the Safe's own encoding of the transaction with the hash-once value in
    ///         the nonce slot. Kept as a helper rather than inlined into execute() because the
    ///         ten-argument external call exceeds the stack limit of non-via-ir compilation
    ///         there.
    /// @param _safe The Safe the transaction is to be executed through.
    /// @param _params The parameters of the transaction.
    /// @param _hashOnce The transaction's hash-once value.
    /// @return txHashData_ Encoded transaction data.
    function _encodeTransactionData(
        Safe _safe,
        ExecTransactionParams calldata _params,
        uint256 _hashOnce
    )
        internal
        view
        returns (bytes memory txHashData_)
    {
        txHashData_ = _safe.encodeTransactionData(
            _params.to, _params.value, _params.data, _params.operation, 0, 0, 0, address(0), address(0), _hashOnce
        );
    }
}
