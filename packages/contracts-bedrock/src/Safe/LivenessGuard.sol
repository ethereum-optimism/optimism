// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Safe } from "safe-contracts/Safe.sol";
import { BaseGuard, GuardManager } from "safe-contracts/base/GuardManager.sol";
import { ModuleManager } from "safe-contracts/base/ModuleManager.sol";
import { SafeSigners } from "src/Safe/SafeSigners.sol";
import { Enum } from "safe-contracts/common/Enum.sol";
import { ISemver } from "src/universal/ISemver.sol";
import { EnumerableSet } from "@openzeppelin/contracts/utils/structs/EnumerableSet.sol";

/// @title LivenessGuard
/// @notice This Guard contract is used to track the liveness of Safe owners.
/// @dev It keeps track of the last time each owner participated in signing a transaction.
///      If an owner does not participate in a transaction for a certain period of time, they are considered inactive.
///      This Guard is intended to be used in conjunction with the LivenessModule contract, but does
///      not depend on it.
contract LivenessGuard is ISemver, BaseGuard {
    using EnumerableSet for EnumerableSet.AddressSet;

    /// @notice Emitted when a new set of signers is recorded.
    /// @param signers An arrary of signer addresses.
    event SignersRecorded(bytes32 indexed txHash, address[] signers);

    /// @notice Semantic version.
    /// @custom:semver 1.0.0
    string public constant version = "1.0.0";

    /// @notice The safe account for which this contract will be the guard.
    Safe public immutable safe;

    /// @notice A mapping of the timestamp at which an owner last participated in signing a
    ///         an executed transaction, or called showLiveness.
    mapping(address => uint256) public lastLive;

    /// @notice An enumerable set of addresses used to store the list of owners before execution,
    ///         and then to update the lastLive mapping according to changes in the set observed
    ///         after execution.
    EnumerableSet.AddressSet private ownersBefore;

    /// @notice Constructor.
    /// @param _safe The safe account for which this contract will be the guard.
    constructor(Safe _safe) {
        safe = _safe;
    }

    /// @notice We use this post execution hook to compare the set of owners before and after.
    ///         If the set of owners has changed then we:
    ///         1. Add new owners to the lastLive mapping
    ///         2. Delete removed owners from the lastLive mapping
    function checkAfterExecution(bytes32, bool) external {
        // Get the current set of owners
        address[] memory ownersAfter = safe.getOwners();

        // Iterate over the current owners, and remove one at a time from the ownersBefore set.
        uint256 ownersAfterLength = ownersAfter.length;
        for (uint256 i = 0; i < ownersAfterLength; i++) {
            // If the value was present, remove() returns true.
            address ownerAfter = ownersAfter[i];
            if (ownersBefore.remove(ownerAfter) == false) {
                // This address was not already an owner, add it to the lastLive mapping
                lastLive[ownerAfter] = block.timestamp;
            }
        }
        // Now iterate over the remaining ownersBefore entries. Any remaining addresses are no longer an owner, so we
        // delete them from the lastLive mapping.
        for (uint256 j = 0; j < ownersBefore.length(); j++) {
            address ownerBefore = ownersBefore.at(j);
            delete lastLive[ownerBefore];
        }
    }

    /// @notice Records the most recent time which any owner has signed a transaction.
    /// @dev This method is called by the Safe contract, it is critical that it does not revert, otherwise
    ///      the Safe contract will be unable to execute transactions.
    function checkTransaction(
        address to,
        uint256 value,
        bytes memory data,
        Enum.Operation operation,
        uint256 safeTxGas,
        uint256 baseGas,
        uint256 gasPrice,
        address gasToken,
        address payable refundReceiver,
        bytes memory signatures,
        address
    )
        external
    {
        require(msg.sender == address(safe), "LivenessGuard: only Safe can call this function");

        // Cache the set of owners prior to execution.
        // This will be used in the checkAfterExecution method.
        address[] memory owners = safe.getOwners();
        for (uint256 i = 0; i < owners.length; i++) {
            ownersBefore.add(owners[i]);
        }

        // This call will reenter to the Safe which is calling it. This is OK because it is only reading the
        // nonce, and using the getTransactionHash() method.
        bytes32 txHash = Safe(payable(msg.sender)).getTransactionHash({
            to: to,
            value: value,
            data: data,
            operation: operation,
            safeTxGas: safeTxGas,
            baseGas: baseGas,
            gasPrice: gasPrice,
            gasToken: gasToken,
            refundReceiver: refundReceiver,
            _nonce: Safe(payable(msg.sender)).nonce() - 1
        });

        uint256 threshold = safe.getThreshold();
        address[] memory signers =
            SafeSigners.getNSigners({ dataHash: txHash, signatures: signatures, requiredSignatures: threshold });

        for (uint256 i = 0; i < signers.length; i++) {
            lastLive[signers[i]] = block.timestamp;
        }
        emit SignersRecorded(txHash, signers);
    }

    /// @notice Enables an owner to demonstrate liveness by calling this method directly.
    ///         This is useful for owners who have not recently signed a transaction via the Safe.
    function showLiveness() external {
        require(safe.isOwner(msg.sender), "LivenessGuard: only Safe owners may demontstrate liveness");
        lastLive[msg.sender] = block.timestamp;
        address[] memory signers = new address[](1);
        signers[0] = msg.sender;

        emit SignersRecorded(0x0, signers);
    }
}
