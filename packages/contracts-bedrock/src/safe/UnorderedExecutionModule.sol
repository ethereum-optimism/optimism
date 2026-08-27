// SPDX-License-Identifier: MIT
pragma solidity 0.8.25;

// Safe
import { Safe } from "safe-contracts/Safe.sol";
import { Enum } from "safe-contracts/common/Enum.sol";

// Libraries
import { SafeSigners } from "src/safe/SafeSigners.sol";
import { TransientContext } from "src/libraries/TransientContext.sol";

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
/// Liveness guard support:
///     For the duration of the task's call, the addresses recovered from the authorizing
///     signatures are stored and exposed via signers(), so that a module guard invoked by the
///     Safe (checkModuleTransaction(), available from Safe 1.5.0) — e.g. a liveness guard — can
///     read which owners approved the transaction and record their liveness. The array lives in
///     transient storage and is cleared before execute() returns. On Safe versions before
///     1.5.0, module transactions invoke no guard and the array simply goes unread.
/// Security notes:
///     - On Safe versions before 1.5.0, module transactions are NOT inspected by any guard, so
///       transactions executed through this module bypass e.g. a timelock guard enabled on the
///       Safe. From 1.5.0 a module guard set via setModuleGuard() inspects them.
///     - The relayer chooses the gas for execute(). A failed call reverts and leaves the
///       hash-once value unconsumed, but a target that swallows an inner failure — a nested
///       Safe's execTransaction(), or a multicall that allows failures — can report success on
///       a partial, possibly gas-starved execution and consume the hash-once value anyway. Task
///       authors should route calls through failure-propagating targets such as
///       MultiSendCallOnly.
///     - execute() is non-reentrant per Safe, so the exposed signers always belong to the one
///       transaction in flight for that Safe. A task must not execute another task for the same
///       Safe from within its call.
///     - Replay protection lives in this module's storage, so it is scoped per module instance:
///       a Safe must never have more than one instance of this module enabled at a time, or the
///       same signatures could execute the same transaction once through each instance.
/// Safe compatibility:
///     Works with Safes exposing the interfaces this module relies on, all present in 1.3.0,
///     1.4.1 and 1.5.0: domainSeparator(), getTransactionHash(),
///     checkSignatures(bytes32,bytes,bytes), getThreshold(), isModuleEnabled() and
///     execTransactionFromModuleReturnData(). The module builds the EIP-712 transaction
///     encoding itself (Safe 1.5.0 removed encodeTransactionData()) and execute() verifies it
///     against the Safe's getTransactionHash(), rejecting any Safe whose hashing scheme
///     diverges. The encoding is passed to checkSignatures() as the preimage that EIP-1271
///     owners validate on 1.3.0/1.4.1; Safe 1.5.0 ignores it and has EIP-1271 owners validate
///     the transaction hash directly. All other semantics of those functions (signature
///     checking, domain separation) are trusted as-is, so the module should only be enabled on
///     Safe versions known to implement them correctly; 1.3.0, 1.4.1 and 1.5.0 do. Tests
///     exercise the vendored 1.4.1 implementation.
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

    /// @notice Error for when a task's call re-enters execute() for the same Safe.
    error UnorderedExecutionModule_ReentrantExecution();

    /// @notice Error for when the Safe's transaction hashing diverges from the scheme this
    ///         module relies on.
    error UnorderedExecutionModule_UnsupportedSafe();

    /// @notice Error for when the transaction's call reverts, carrying the raw revert data.
    error UnorderedExecutionModule_ExecutionFailed(bytes data);

    /// @notice Emitted when a transaction is executed.
    /// @param safe The Safe the transaction was executed through.
    /// @param txHash The Safe transaction hash identifying the transaction.
    /// @param hashOnce The hash-once value consumed by the execution.
    event TransactionExecuted(Safe indexed safe, bytes32 indexed txHash, uint256 indexed hashOnce);

    /// @notice EIP-712 typehash of a Safe transaction, identical across Safe 1.3.0, 1.4.1 and
    ///         1.5.0 (0xbb8310d486368db6bd6f849402fdd73ad53d316b5a4b2644ad6efe0f941286d8).
    bytes32 internal constant SAFE_TX_TYPEHASH = keccak256(
        "SafeTx(address to,uint256 value,bytes data,uint8 operation,uint256 safeTxGas,uint256 baseGas,uint256 gasPrice,address gasToken,address refundReceiver,uint256 nonce)"
    );

    /// @notice Seed for the per-Safe transient storage slots holding the signers of the
    ///         transaction currently being executed. The array length lives at the derived base
    ///         slot and element i at base + 1 + i.
    bytes32 internal constant SIGNERS_SLOT_SEED = keccak256("UnorderedExecutionModule.signers");

    /// @notice Seed for the per-Safe transient reentrancy lock slot.
    bytes32 internal constant LOCK_SLOT_SEED = keccak256("UnorderedExecutionModule.lock");

    /// @notice Hash-once values that have been consumed by an execution, keyed by Safe.
    mapping(Safe => mapping(uint256 => bool)) internal _executed;

    /// @notice Semantic version.
    /// @custom:semver 1.0.0-beta.1
    string public constant version = "1.0.0-beta.1";

    /// @notice Getter function for whether a hash-once value has been consumed.
    ///
    /// @param _safe The Safe the hash-once value belongs to.
    /// @param _hashOnce The hash-once value.
    ///
    /// @return executed_ Whether the hash-once value has been consumed by an execution.
    function executed(Safe _safe, uint256 _hashOnce) public view returns (bool executed_) {
        executed_ = _executed[_safe][_hashOnce];
    }

    /// @notice Getter function for the signers of the transaction currently being executed.
    ///         Non-empty only while a task's call is in flight, so that a module guard (e.g. a
    ///         liveness guard) invoked during the call can read which owners approved it.
    ///
    /// @param _safe The Safe whose in-flight transaction signers to return.
    ///
    /// @return signers_ Addresses recovered from the authorizing signatures, or an empty array
    ///                  when no transaction is being executed for the Safe.
    function signers(Safe _safe) public view returns (address[] memory signers_) {
        bytes32 slot = _signersSlot(_safe);
        uint256 length = TransientContext.get(slot);
        signers_ = new address[](length);
        for (uint256 i; i < length; i++) {
            signers_[i] = address(uint160(TransientContext.get(bytes32(uint256(slot) + 1 + i))));
        }
    }

    /// @notice Derives a canonical hash-once value from a unique string, e.g. the hashOnceInput
    ///         pinned in a superchain-ops task's configuration.
    ///
    /// @param _input Unique string identifying the task.
    ///
    /// @return hashOnce_ Hash-once value to use as the transaction's nonce.
    function deriveHashOnce(string memory _input) public pure returns (uint256 hashOnce_) {
        hashOnce_ = uint256(keccak256(bytes(_input)));
    }

    /// @notice Computes the EIP-712 encoding of the transaction with the hash-once value in the
    ///         nonce slot and zeroed gas and refund fields — the preimage of transactionHash().
    ///         Built locally because Safe 1.5.0 removed encodeTransactionData(); execute()
    ///         verifies it against the Safe's own getTransactionHash().
    ///
    /// @param _safe The Safe the transaction is to be executed through.
    /// @param _params The parameters of the transaction.
    /// @param _hashOnce The transaction's hash-once value.
    ///
    /// @return txHashData_ EIP-712 encoded transaction data.
    function encodeTransactionData(
        Safe _safe,
        ExecTransactionParams calldata _params,
        uint256 _hashOnce
    )
        public
        view
        returns (bytes memory txHashData_)
    {
        bytes32 safeTxHash = keccak256(
            abi.encode(
                SAFE_TX_TYPEHASH,
                _params.to,
                _params.value,
                keccak256(_params.data),
                _params.operation,
                uint256(0),
                uint256(0),
                uint256(0),
                address(0),
                address(0),
                _hashOnce
            )
        );
        txHashData_ = abi.encodePacked(bytes1(0x19), bytes1(0x01), _safe.domainSeparator(), safeTxHash);
    }

    /// @notice Computes the Safe transaction hash that owners must sign to authorize a
    ///         transaction. Identical to the Safe's own getTransactionHash() with the hash-once
    ///         value in the nonce slot and zeroed gas and refund fields.
    ///
    /// @param _safe The Safe the transaction is to be executed through.
    /// @param _params The parameters of the transaction.
    /// @param _hashOnce The transaction's hash-once value.
    ///
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
    ///
    /// @param _safe The Safe to execute the transaction through.
    /// @param _params The parameters of the transaction.
    /// @param _hashOnce The transaction's hash-once value. Must be greater than
    ///                  type(uint128).max.
    /// @param _signatures Owner signatures over transactionHash(), in the format accepted by the
    ///                    Safe's checkSignatures() function.
    ///
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
        // Lock per Safe so that a task's call cannot re-enter execute() for the same Safe,
        // which would overwrite and then clear the signers exposed for the outer transaction
        // while it is still in flight.
        if (TransientContext.get(_lockSlot(_safe)) != 0) {
            revert UnorderedExecutionModule_ReentrantExecution();
        }
        TransientContext.set(_lockSlot(_safe), 1);

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
        bytes memory txHashData = encodeTransactionData(_safe, _params, _hashOnce);

        // The digest owners sign must be the hash of the locally built encoding, or the encoding
        // passed to checkSignatures() below would not be the preimage EIP-1271 owners validate
        // on Safe 1.3.0/1.4.1. This holds on 1.3.0, 1.4.1 and 1.5.0; a Safe whose hashing
        // scheme diverges is rejected.
        if (keccak256(txHashData) != txHash) {
            revert UnorderedExecutionModule_UnsupportedSafe();
        }

        // Verify signatures using the Safe's own logic, so that the Safe's current owner set and
        // threshold apply and all Safe-supported signing methods work. Reverts if invalid.
        _safe.checkSignatures(txHash, txHashData, _signatures);

        // Consume the hash-once value before the external call so that the call cannot execute
        // the same transaction again.
        _executed[_safe][_hashOnce] = true;

        // Expose the approving signers for the duration of the call, so that a module guard
        // (e.g. a liveness guard) invoked by the Safe can read them via signers().
        _setSigners(_safe, SafeSigners.getNSigners(txHash, _signatures, _safe.getThreshold()));

        // Execute the transaction through the Safe.
        bool success;
        (success, returnData_) =
            _safe.execTransactionFromModuleReturnData(_params.to, _params.value, _params.data, _params.operation);

        // Clear the exposed signers and release the lock now that the call has completed, so a
        // later execution in the same transaction can never observe them.
        TransientContext.set(_signersSlot(_safe), 0);
        TransientContext.set(_lockSlot(_safe), 0);

        // Unlike the Safe's execTransaction(), revert if the call failed. The revert unwinds the
        // consumed hash-once value so the transaction can be retried.
        if (!success) {
            revert UnorderedExecutionModule_ExecutionFailed(returnData_);
        }

        emit TransactionExecuted(_safe, txHash, _hashOnce);
    }

    /// @notice Derives the base transient storage slot for a Safe's exposed signers.
    ///
    /// @param _safe The Safe to derive the slot for.
    ///
    /// @return slot_ Base transient storage slot.
    function _signersSlot(Safe _safe) internal pure returns (bytes32 slot_) {
        slot_ = keccak256(abi.encode(SIGNERS_SLOT_SEED, _safe));
    }

    /// @notice Derives the transient storage slot for a Safe's reentrancy lock.
    ///
    /// @param _safe The Safe to derive the slot for.
    ///
    /// @return slot_ Transient storage slot of the lock.
    function _lockSlot(Safe _safe) internal pure returns (bytes32 slot_) {
        slot_ = keccak256(abi.encode(LOCK_SLOT_SEED, _safe));
    }

    /// @notice Writes the signers of the transaction being executed to transient storage.
    ///
    /// @param _safe The Safe the transaction is being executed through.
    /// @param _txSigners Addresses recovered from the authorizing signatures.
    function _setSigners(Safe _safe, address[] memory _txSigners) internal {
        bytes32 slot = _signersSlot(_safe);
        TransientContext.set(slot, _txSigners.length);
        for (uint256 i; i < _txSigners.length; i++) {
            TransientContext.set(bytes32(uint256(slot) + 1 + i), uint256(uint160(_txSigners[i])));
        }
    }
}
