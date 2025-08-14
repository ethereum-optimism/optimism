// SPDX-License-Identifier: MIT
pragma solidity 0.8.25;

// Safe
import { GnosisSafe as Safe } from "safe-contracts/GnosisSafe.sol";
import { Enum } from "safe-contracts/common/Enum.sol";
import { ModuleManager } from "safe-contracts/base/ModuleManager.sol";

// Contracts
import { ReentrancyGuard } from "@openzeppelin/contracts-v5/utils/ReentrancyGuard.sol";

// Interfaces
import { ISemver } from "interfaces/universal/ISemver.sol";

/// @title UnorderedExecutionModule
/// @notice Safe Module that allows for unordered execution of transactions on a Safe, by leveraging
///         the Safe's checkSignatures function. Replay prevention is handled on this module using
///         keccak256 hashes in place of the Safe's nonce. This ensures that any signatures submitted
///         to the safe will have a nonce which is so high that it is impossible for the Safe's nonce
///         to ever collide with a nonce used by this module.
///         Optionally the hash of a previous transaction can be provided, this enables transaction
///         ordering constraints when needed.
/// @dev This module is intended to be installed on multiple Safe contracts, in order to avoid
///      the need to deploy distinct modules for each Safe.
contract UnorderedExecutionModule is ISemver, ReentrancyGuard {
    /// @notice Semantic version.
    /// @custom:semver 1.0.0
    string public constant version = "1.0.0";

    /// @notice Arguments to ModuleManager.execTransactionFromModule()
    struct ExecTransactionFromModuleParams {
        address to;
        uint256 value;
        bytes data;
        Enum.Operation operation;
    }

    /// @notice Input parameters for hash-once calculation and prerequisite validation
    /// @dev This struct contains two fields that together form a unique transaction identifier:
    ///      - prevHashOnce: Optional hash of a previously executed transaction (zero if no prerequisite)
    ///      - mixHash: Additional entropy to ensure uniqueness of the hash-once value
    struct HashOnceInputs {
        bytes32 prevHashOnce;
        bytes32 mixHash;
    }

    /// @notice Used for replay protection
    mapping(bytes32 => bool) public executedTransactions;

    /// @notice Emitted when a transaction is successfully executed
    event TransactionExecuted(
        address indexed safe, bytes32 indexed txHash, address to, uint256 value, bytes data, Enum.Operation operation
    );

    /// @notice Error thrown when a transaction has already been executed
    error TransactionAlreadyExecuted();

    /// @notice Error thrown when module execution fails
    error ModuleExecutionFailed();

    /// @notice Allows for unordered execution of transactions on a Safe, by leveraging the Safe's
    /// checkSignatures function. Replay prevention is handled on this module using keccak256 hashes
    /// in place of the Safe's nonce.
    /// @param safe The Safe contract to execute the transaction on
    /// @param hashOnceInputs Input parameters containing prerequisite transaction hash and mix hash
    /// @param params The transaction parameters
    /// @param signatures Signature data that should be verified
    /// @return success Whether the transaction was successful
    function execTransactionOnSafe(
        Safe safe,
        HashOnceInputs calldata hashOnceInputs,
        ExecTransactionFromModuleParams memory params,
        bytes memory signatures
    )
        public
        payable
        nonReentrant
        returns (bool success)
    {
        // Calculate the hash-once value from the input parameters
        // This serves as both the transaction nonce and replay protection key
        bytes32 hashOnce = keccak256(abi.encode(hashOnceInputs));

        // If a prerequisite transaction is specified, verify it has been executed
        // This enables transaction ordering constraints when needed
        if (hashOnceInputs.prevHashOnce != bytes32(0)) {
            if (executedTransactions[hashOnceInputs.prevHashOnce] != true) {
                revert("UnorderedExecutionModule: prereq prevHashOnce not executed");
            }
        }

        // Use the Safe's getTransactionHash function to generate a deterministic hash,
        // this hash includes the Safe's domain separator.
        // This hash is used both for replay protection and for signature verification.
        (bytes32 safesInternalTxHash, bytes memory txHashData) =
            getSafesInternalTxHashAndTxHashData(safe, hashOnce, params);

        if (executedTransactions[safesInternalTxHash]) {
            revert TransactionAlreadyExecuted();
        }
        executedTransactions[safesInternalTxHash] = true;

        // Verify signatures using Safe's signature checking
        // Failure will bubble up
        safe.checkSignatures(safesInternalTxHash, txHashData, signatures);

        // Execute transaction through Safe's module system
        success = safe.execTransactionFromModule(params.to, params.value, params.data, params.operation);

        if (!success) {
            revert ModuleExecutionFailed();
        }

        emit TransactionExecuted(
            address(safe), safesInternalTxHash, params.to, params.value, params.data, params.operation
        );
    }

    /// @notice Utility function to check if this module is enabled on a given Safe
    /// @param safe The Safe to check
    /// @return enabled Whether this module is enabled on the Safe
    function isEnabledOnSafe(Safe safe) external view returns (bool enabled) {
        return ModuleManager(address(safe)).isModuleEnabled(address(this));
    }

    function getSafesInternalTxHashAndTxHashData(
        Safe safe,
        bytes32 nonce,
        ExecTransactionFromModuleParams memory params
    )
        internal
        view
        returns (bytes32, bytes memory)
    {
        return (
            safe.getTransactionHash({
                to: params.to,
                value: params.value,
                data: params.data,
                operation: params.operation,
                safeTxGas: 0,
                baseGas: 0,
                gasPrice: 0,
                gasToken: address(0),
                refundReceiver: address(0),
                _nonce: uint256(nonce)
            }),
            safe.encodeTransactionData({
                to: params.to,
                value: params.value,
                data: params.data,
                operation: params.operation,
                safeTxGas: 0,
                baseGas: 0,
                gasPrice: 0,
                gasToken: address(0),
                refundReceiver: address(0),
                _nonce: uint256(nonce)
            })
        );
    }
}
