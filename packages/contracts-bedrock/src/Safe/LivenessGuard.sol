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
///      Note: Both `checkTransaction` and `checkAfterExecution` are called once each by the Safe contract
///      before and after the execution of a transaction. It is critical that neither function revert,
///      otherwise the Safe contract will be unable to execute a transaction.
contract LivenessGuard is ISemver, BaseGuard {
    using EnumerableSet for EnumerableSet.AddressSet;

    /// @notice Emitted when an owner is recorded.
    /// @param owner The owner's address.
    event OwnerRecorded(address owner);

    /// @notice Semantic version.
    /// @custom:semver 1.0.0
    string public constant version = "1.0.0";

    /// @notice The safe account for which this contract will be the guard.
    Safe internal immutable SAFE;

    /// @notice A mapping of the timestamp at which an owner last participated in signing a
    ///         an executed transaction, or called showLiveness.
    mapping(address => uint256) public lastLive;

    /// @notice An enumerable set of addresses used to store the list of owners before execution,
    ///         and then to update the lastLive mapping according to changes in the set observed
    ///         after execution.
    EnumerableSet.AddressSet internal ownersBefore;

    /// @notice Constructor.
    /// @param _safe The safe account for which this contract will be the guard.
    constructor(Safe _safe) {
        SAFE = _safe;
        address[] memory owners = _safe.getOwners();
        for (uint256 i = 0; i < owners.length; i++) {
            address owner = owners[i];
            lastLive[owner] = block.timestamp;
            emit OwnerRecorded(owner);
        }
    }

    /// @notice Getter function for the Safe contract instance
    /// @return safe_ The Safe contract instance
    function safe() public view returns (Safe safe_) {
        safe_ = SAFE;
    }

    /// @notice Internal function to ensure that only the Safe can call certain functions.
    function _requireOnlySafe() internal view {
        require(msg.sender == address(SAFE), "LivenessGuard: only Safe can call this function");
    }

    /// @notice Records the most recent time which any owner has signed a transaction.
    /// @dev Called by the Safe contract before execution of a transaction.
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
        address msgSender
    )
        external
    {
        msgSender; // silence unused variable warning
        _requireOnlySafe();

        // Cache the set of owners prior to execution.
        // This will be used in the checkAfterExecution method.
        address[] memory owners = SAFE.getOwners();
        for (uint256 i = 0; i < owners.length; i++) {
            ownersBefore.add(owners[i]);
        }

        // This call will reenter to the Safe which is calling it. This is OK because it is only reading the
        // nonce, and using the getTransactionHash() method.
        bytes32 txHash = SAFE.getTransactionHash({
            to: to,
            value: value,
            data: data,
            operation: operation,
            safeTxGas: safeTxGas,
            baseGas: baseGas,
            gasPrice: gasPrice,
            gasToken: gasToken,
            refundReceiver: refundReceiver,
            _nonce: SAFE.nonce() - 1
        });

        uint256 threshold = SAFE.getThreshold();
        address[] memory signers =
            SafeSigners.getNSigners({ dataHash: txHash, signatures: signatures, requiredSignatures: threshold });

        for (uint256 i = 0; i < signers.length; i++) {
            lastLive[signers[i]] = block.timestamp;
            emit OwnerRecorded(signers[i]);
        }
    }

    /// @notice Update the lastLive mapping according to the set of owners before and after execution.
    /// @dev Called by the Safe contract after the execution of a transaction.
    ///      We use this post execution hook to compare the set of owners before and after.
    ///      If the set of owners has changed then we:
    ///      1. Add new owners to the lastLive mapping
    ///      2. Delete removed owners from the lastLive mapping
    function checkAfterExecution(bytes32, bool) external {
        _requireOnlySafe();
        // Get the current set of owners
        address[] memory ownersAfter = SAFE.getOwners();

        // Iterate over the current owners, and remove one at a time from the ownersBefore set.
        for (uint256 i = 0; i < ownersAfter.length; i++) {
            // If the value was present, remove() returns true.
            address ownerAfter = ownersAfter[i];
            if (ownersBefore.remove(ownerAfter) == false) {
                // This address was not already an owner, add it to the lastLive mapping
                lastLive[ownerAfter] = block.timestamp;
            }
        }

        // Now iterate over the remaining ownersBefore entries. Any remaining addresses are no longer an owner, so we
        // delete them from the lastLive mapping.
        // We cache the ownersBefore set before iterating over it, because the remove() method mutates the set.
        address[] memory ownersBeforeCache = ownersBefore.values();
        for (uint256 i = 0; i < ownersBeforeCache.length; i++) {
            address ownerBefore = ownersBeforeCache[i];
            delete lastLive[ownerBefore];
            ownersBefore.remove(ownerBefore);
        }
    }

    /// @notice Enables an owner to demonstrate liveness by calling this method directly.
    ///         This is useful for owners who have not recently signed a transaction via the Safe.
    function showLiveness() external {
        require(SAFE.isOwner(msg.sender), "LivenessGuard: only Safe owners may demonstrate liveness");
        lastLive[msg.sender] = block.timestamp;

        emit OwnerRecorded(msg.sender);
    }
}
