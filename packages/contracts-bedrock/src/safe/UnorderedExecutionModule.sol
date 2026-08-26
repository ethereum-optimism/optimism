// SPDX-License-Identifier: MIT
pragma solidity 0.8.25;

// Safe
import { Safe } from "safe-contracts/Safe.sol";
import { Enum } from "safe-contracts/common/Enum.sol";

// Libraries
import { SemverComp } from "src/libraries/SemverComp.sol";

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
///     getTransactionHash() with the hash-once value as the nonce — using any method the Safe
///     itself supports: private key signatures, Safe.approveHash(), or EIP-1271 contract
///     signatures (including nested Safes). Existing Safe signing tooling works unchanged.
///     Once a threshold of signatures has been collected, anyone may call execute() — the same
///     permissionless-relay model as the Safe's own execTransaction().
/// Hash-once values:
///     The hash-once value must be unique per task and strictly greater than type(uint128).max,
///     so that it can never collide with the Safe's real sequential nonce. Without this floor, a
///     signature produced for this module could also become executable through the ordinary
///     execTransaction() path once the Safe's nonce reached the same value. deriveHashOnce()
///     computes a canonical value from a unique string.
/// Replay protection and cancellation:
///     Each (Safe, transaction hash) pair can be executed at most once. There is no on-chain
///     cancellation or expiry: abandoning a signed task is handled off-chain by changing the
///     task's hash-once input and re-signing, and the abandoned signatures simply go unused.
///     Note that abandoned signatures remain technically executable indefinitely; signatures are
///     verified against the Safe's *current* owner set and threshold, so rotating owners or
///     raising the threshold invalidates them.
/// Signature validity:
///     Signatures are verified with the Safe's own checkSignatures() logic at execution time.
///     The Safe's EIP-712 domain separator binds the Safe address and chain id, so a signature
///     cannot be replayed against another Safe or chain.
/// Security notes:
///     - Module transactions are NOT inspected by a transaction guard on Safe versions 1.3.0 and
///       1.4.1, so transactions executed through this module bypass any guard (e.g. a timelock
///       guard) enabled on the Safe.
///     - The signed safeTxGas field is honored as a minimum gas commitment, so a relayer cannot
///       cause a partial, low-gas execution that would otherwise permanently consume the
///       transaction. Gas refund fields (gasPrice, gasToken, refundReceiver) are part of the
///       signed payload but refunds are not paid by this module, so gasPrice must be zero.
/// Safe Version Compatibility:
///     Compatible with Safe versions 1.3.0 and 1.4.1, which share the encodeTransactionData(),
///     checkSignatures(), isModuleEnabled() and execTransactionFromModuleReturnData() interfaces
///     this module relies on.
contract UnorderedExecutionModule is ISemver {
    /// @notice Parameters for the Safe transaction being executed, matching the fields of the
    ///         Safe's own execTransaction() with the nonce supplied separately as the hash-once
    ///         value.
    /// @custom:field to The address of the contract to call.
    /// @custom:field value The ETH value to send with the call.
    /// @custom:field data The calldata to send with the call.
    /// @custom:field operation The operation to perform (Call or DelegateCall).
    /// @custom:field safeTxGas Minimum gas that must be available to execute() at the point of
    ///                         the transaction's call. The call itself receives at least 63/64 of
    ///                         this (EIP-150), so signers should include a safety margin.
    /// @custom:field baseGas Unused by this module, part of the signed payload for tooling
    ///                       compatibility.
    /// @custom:field gasPrice Must be zero: this module does not pay gas refunds.
    /// @custom:field gasToken Unused by this module, part of the signed payload for tooling
    ///                        compatibility.
    /// @custom:field refundReceiver Unused by this module, part of the signed payload for tooling
    ///                              compatibility.
    struct ExecTransactionParams {
        address to;
        uint256 value;
        bytes data;
        Enum.Operation operation;
        uint256 safeTxGas;
        uint256 baseGas;
        uint256 gasPrice;
        address gasToken;
        address payable refundReceiver;
    }

    /// @notice Error for when the Safe's version is not supported.
    error UnorderedExecutionModule_InvalidSafeVersion();

    /// @notice Error for when this module is not enabled on the Safe.
    error UnorderedExecutionModule_ModuleNotEnabled();

    /// @notice Error for when the hash-once value is small enough to collide with a real Safe
    ///         nonce.
    error UnorderedExecutionModule_HashOnceTooSmall();

    /// @notice Error for when the transaction requests a gas refund, which this module does not
    ///         pay.
    error UnorderedExecutionModule_RefundNotSupported();

    /// @notice Error for when a transaction has already been executed.
    error UnorderedExecutionModule_AlreadyExecuted();

    /// @notice Error for when less gas is available than the transaction's signed safeTxGas.
    error UnorderedExecutionModule_InsufficientGas();

    /// @notice Error for when the transaction's call reverts, carrying the raw revert data.
    error UnorderedExecutionModule_ExecutionFailed(bytes);

    /// @notice Emitted when a transaction is executed.
    /// @param safe The Safe the transaction was executed through.
    /// @param txHash The Safe transaction hash identifying the transaction.
    event TransactionExecuted(Safe indexed safe, bytes32 indexed txHash);

    /// @notice Transactions that have been executed, keyed by Safe and Safe transaction hash.
    mapping(Safe => mapping(bytes32 => bool)) internal _executed;

    /// @notice Semantic version.
    /// @custom:semver 1.0.0-beta.1
    string public constant version = "1.0.0-beta.1";

    /// @notice Getter function for whether a transaction has been executed.
    /// @param _safe The Safe the transaction belongs to.
    /// @param _txHash The Safe transaction hash identifying the transaction.
    /// @return executed_ Whether the transaction has been executed.
    function executed(Safe _safe, bytes32 _txHash) public view returns (bool executed_) {
        executed_ = _executed[_safe][_txHash];
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
    ///         value in the nonce slot.
    /// @param _safe The Safe the transaction is to be executed through.
    /// @param _params The parameters of the transaction.
    /// @param _hashOnce The transaction's hash-once value.
    /// @return txHash_ Safe transaction hash of the transaction.
    function transactionHash(
        Safe _safe,
        ExecTransactionParams memory _params,
        uint256 _hashOnce
    )
        public
        view
        returns (bytes32 txHash_)
    {
        txHash_ = keccak256(_encodeTransactionData(_safe, _params, _hashOnce));
    }

    /// @notice Executes a transaction once a threshold of owner signatures has been collected.
    ///         Callable by anyone, matching the relay model of the Safe's own execTransaction().
    ///         Reverts if the transaction's call reverts, leaving the transaction unexecuted so
    ///         that it can be retried.
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
        // Only Safe versions 1.3.0 and 1.4.1 are supported. Later versions may change the
        // transaction encoding or signature checking logic this module depends on.
        string memory safeVersion = _safe.VERSION();
        if (!SemverComp.eq(safeVersion, "1.3.0") && !SemverComp.eq(safeVersion, "1.4.1")) {
            revert UnorderedExecutionModule_InvalidSafeVersion();
        }

        // The Safe itself would reject the module call, but checking here gives a clear error.
        if (!_safe.isModuleEnabled(address(this))) {
            revert UnorderedExecutionModule_ModuleNotEnabled();
        }

        // A hash-once value that a sequential nonce could actually reach would let the same
        // signatures execute both through this module and through execTransaction().
        if (_hashOnce <= type(uint128).max) {
            revert UnorderedExecutionModule_HashOnceTooSmall();
        }

        // This module never pays gas refunds, so refuse payloads that promise one.
        if (_params.gasPrice != 0) {
            revert UnorderedExecutionModule_RefundNotSupported();
        }

        // The transaction hash is the keccak256 of the encoded data on Safe 1.3.0 and 1.4.1,
        // which the version gate above pins.
        bytes memory txHashData = _encodeTransactionData(_safe, _params, _hashOnce);
        bytes32 txHash = keccak256(txHashData);

        // Each transaction hash can be executed at most once.
        if (_executed[_safe][txHash]) {
            revert UnorderedExecutionModule_AlreadyExecuted();
        }

        // Verify signatures using the Safe's own logic, so that the Safe's current owner set and
        // threshold apply and all Safe-supported signing methods work. Reverts if invalid.
        _safe.checkSignatures(txHash, txHashData, _signatures);

        // Honor the signed gas commitment so a relayer cannot cause a partial, low-gas execution
        // that would permanently consume the transaction.
        if (gasleft() < _params.safeTxGas) {
            revert UnorderedExecutionModule_InsufficientGas();
        }

        // Mark the transaction executed before the external call so that the call cannot
        // re-enter and execute the same transaction again.
        _executed[_safe][txHash] = true;

        // Execute the transaction through the Safe.
        bool success;
        (success, returnData_) =
            _safe.execTransactionFromModuleReturnData(_params.to, _params.value, _params.data, _params.operation);

        // Unlike the Safe's execTransaction(), revert if the call failed. The revert unwinds the
        // executed state so the transaction can be retried.
        if (!success) {
            revert UnorderedExecutionModule_ExecutionFailed(returnData_);
        }

        emit TransactionExecuted(_safe, txHash);
    }

    /// @notice Fetches the Safe's own EIP-712 encoding of the transaction with the hash-once
    ///         value in the nonce slot.
    /// @param _safe The Safe the transaction is to be executed through.
    /// @param _params The parameters of the transaction.
    /// @param _hashOnce The transaction's hash-once value.
    /// @return txHashData_ Encoded transaction data.
    function _encodeTransactionData(
        Safe _safe,
        ExecTransactionParams memory _params,
        uint256 _hashOnce
    )
        internal
        view
        returns (bytes memory txHashData_)
    {
        txHashData_ = _safe.encodeTransactionData(
            _params.to,
            _params.value,
            _params.data,
            _params.operation,
            _params.safeTxGas,
            _params.baseGas,
            _params.gasPrice,
            _params.gasToken,
            _params.refundReceiver,
            _hashOnce
        );
    }
}
