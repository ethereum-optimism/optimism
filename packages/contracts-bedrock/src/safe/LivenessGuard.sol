// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Safe
import { Safe } from "safe-contracts/Safe.sol";
import { BaseGuard, Guard } from "safe-contracts/base/GuardManager.sol";
import { Enum } from "safe-contracts/common/Enum.sol";
import { IERC165 } from "safe-contracts/interfaces/IERC165.sol";

// Libraries
import { EnumerableSet } from "@openzeppelin/contracts/utils/structs/EnumerableSet.sol";
import { SafeSigners } from "src/safe/SafeSigners.sol";

// Interfaces
import { ISemver } from "interfaces/universal/ISemver.sol";
import { IModuleGuard } from "interfaces/safe/IModuleGuard.sol";
import { IUnorderedExecutionModule } from "interfaces/safe/IUnorderedExecutionModule.sol";

/// @title LivenessGuard
/// @notice This Guard contract is used to track the liveness of Safe owners.
/// @dev It keeps track of the last time each owner participated in signing a transaction.
///      If an owner does not participate in a transaction for a certain period of time, they are considered inactive.
///      This Guard is intended to be used in conjunction with the LivenessModule contract, but does
///      not depend on it.
///      Liveness is also recorded for executions performed through the trusted
///      UnorderedExecutionModule (nonceless execution) via the Safe v1.5.0 module guard hooks. The
///      module hooks use their own owner snapshot (moduleOwnersBefore) so owner-set changes are
///      reconciled after either execution path, and so the two paths cannot corrupt each other's
///      bookkeeping when a module execution happens inside a Safe transaction (or vice versa).
///      Note: it is critical that none of the Safe hooks revert (beyond the onlySafe check),
///      otherwise the Safe contract would be unable to execute a transaction — or to evict a stale
///      owner through a module.
contract LivenessGuard is ISemver, BaseGuard, IModuleGuard {
    using EnumerableSet for EnumerableSet.AddressSet;

    /// @notice Emitted when an owner is recorded.
    /// @param owner The owner's address.
    event OwnerRecorded(address owner);

    /// @notice Semantic version.
    /// @custom:semver 1.2.0
    string public constant version = "1.2.0";

    /// @notice The safe account for which this contract will be the guard.
    Safe internal immutable SAFE;

    /// @notice The UnorderedExecutionModule trusted to report the signers of a nonceless (module)
    ///         execution. May be the zero address, which disables module-path liveness recording.
    IUnorderedExecutionModule internal immutable UNORDERED_EXECUTION_MODULE;

    /// @notice A mapping of the timestamp at which an owner last participated in signing a
    ///         an executed transaction, or called showLiveness.
    mapping(address => uint256) public lastLive;

    /// @notice An enumerable set of addresses used to store the list of owners before execution,
    ///         and then to update the lastLive mapping according to changes in the set observed
    ///         after execution.
    EnumerableSet.AddressSet internal ownersBefore;

    /// @notice The owner snapshot used by the module-guard execution path.
    EnumerableSet.AddressSet internal moduleOwnersBefore;

    /// @notice Constructor.
    /// @param _safe The safe account for which this contract will be the guard.
    /// @param _unorderedExecutionModule The UnorderedExecutionModule trusted for module-path
    ///        liveness recording (zero address to disable).
    constructor(Safe _safe, IUnorderedExecutionModule _unorderedExecutionModule) {
        SAFE = _safe;
        UNORDERED_EXECUTION_MODULE = _unorderedExecutionModule;
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

    /// @notice Getter function for the trusted UnorderedExecutionModule.
    /// @return module_ The UnorderedExecutionModule (zero address if module recording is disabled).
    function unorderedExecutionModule() public view returns (IUnorderedExecutionModule module_) {
        module_ = UNORDERED_EXECUTION_MODULE;
    }

    /// @inheritdoc IERC165
    /// @dev Safe v1.5.0 requires the guard to support the Guard interface id for setGuard (GS300)
    ///      and the IModuleGuard interface id for setModuleGuard (GS301).
    function supportsInterface(bytes4 _interfaceId) external pure override(BaseGuard, IERC165) returns (bool) {
        return _interfaceId == type(Guard).interfaceId || _interfaceId == type(IModuleGuard).interfaceId
            || _interfaceId == type(IERC165).interfaceId;
    }

    /// @notice Internal function to ensure that only the Safe can call certain functions.
    function _requireOnlySafe() internal view {
        require(msg.sender == address(SAFE), "LivenessGuard: only Safe can call this function");
    }

    /// @notice Records the most recent time which any owner has signed a transaction.
    /// @dev Called by the Safe contract before execution of a transaction.
    function checkTransaction(
        address _to,
        uint256 _value,
        bytes memory _data,
        Enum.Operation _operation,
        uint256 _safeTxGas,
        uint256 _baseGas,
        uint256 _gasPrice,
        address _gasToken,
        address payable _refundReceiver,
        bytes memory _signatures,
        address _msgSender
    )
        external
    {
        _msgSender; // silence unused variable warning
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
            to: _to,
            value: _value,
            data: _data,
            operation: _operation,
            safeTxGas: _safeTxGas,
            baseGas: _baseGas,
            gasPrice: _gasPrice,
            gasToken: _gasToken,
            refundReceiver: _refundReceiver,
            _nonce: SAFE.nonce() - 1
        });

        uint256 threshold = SAFE.getThreshold();
        address[] memory signers =
            SafeSigners.getNSigners({ _dataHash: txHash, _signatures: _signatures, _requiredSignatures: threshold });

        for (uint256 i = 0; i < signers.length; i++) {
            lastLive[signers[i]] = block.timestamp;
            emit OwnerRecorded(signers[i]);
        }
    }

    /// @inheritdoc IModuleGuard
    /// @dev Called by the Safe (>= v1.5.0) before a module transaction is executed. Records
    ///      liveness only for executions performed by the trusted UnorderedExecutionModule: that
    ///      module exposes the signer set it already validated via Safe.checkSignatures earlier in
    ///      this same call, so no unvalidated signatures are ever parsed here and every reported
    ///      signer is necessarily a current owner. Executions by any other module (e.g. the
    ///      LivenessModule evicting a stale owner) perform no recording at all, so liveness
    ///      tracking can never block third-party module execution. For the trusted module the
    ///      signer read is a simple view call into a contract deployed alongside this guard.
    function checkModuleTransaction(
        address,
        uint256,
        bytes memory,
        Enum.Operation,
        address _module
    )
        external
        returns (bytes32 moduleTxHash_)
    {
        _requireOnlySafe();

        // Cache the set of owners prior to execution.
        // This will be used in the checkAfterModuleExecution method.
        address[] memory owners = SAFE.getOwners();
        for (uint256 i = 0; i < owners.length; i++) {
            moduleOwnersBefore.add(owners[i]);
        }

        IUnorderedExecutionModule module = UNORDERED_EXECUTION_MODULE;
        if (address(module) != address(0) && _module == address(module)) {
            address[] memory signers = module.signers(SAFE);
            for (uint256 i = 0; i < signers.length; i++) {
                lastLive[signers[i]] = block.timestamp;
                emit OwnerRecorded(signers[i]);
            }
        }
        return bytes32(0);
    }

    /// @inheritdoc IModuleGuard
    function checkAfterModuleExecution(bytes32, bool) external {
        _requireOnlySafe();
        _reconcileOwners(moduleOwnersBefore);
    }

    /// @notice Update the lastLive mapping according to the set of owners before and after execution.
    /// @dev Called by the Safe contract after the execution of a transaction.
    ///      We use this post execution hook to compare the set of owners before and after.
    ///      If the set of owners has changed then we:
    ///      1. Add new owners to the lastLive mapping
    ///      2. Delete removed owners from the lastLive mapping
    function checkAfterExecution(bytes32, bool) external {
        _requireOnlySafe();
        _reconcileOwners(ownersBefore);
    }

    /// @notice Reconciles owner liveness after a Safe transaction may have changed the owner set.
    function _reconcileOwners(EnumerableSet.AddressSet storage _ownersBefore) internal {
        // An empty snapshot means this post-hook has no matching pre-hook: either a nested
        // execution already reconciled (and drained) the snapshot, or the hook was invoked without
        // one. Reconciling against an empty set would mark every current owner as live, so do
        // nothing instead. Owner changes made by a nested execution are reconciled by that
        // execution's own hook pair.
        if (_ownersBefore.length() == 0) return;

        // Get the current set of owners
        address[] memory ownersAfter = SAFE.getOwners();

        // Iterate over the current owners, and remove one at a time from the ownersBefore set.
        for (uint256 i = 0; i < ownersAfter.length; i++) {
            // If the value was present, remove() returns true.
            address ownerAfter = ownersAfter[i];
            if (_ownersBefore.remove(ownerAfter) == false) {
                // This address was not already an owner, add it to the lastLive mapping
                lastLive[ownerAfter] = block.timestamp;
            }
        }

        // Now iterate over the remaining ownersBefore entries. Any remaining addresses are no longer an owner, so we
        // delete them from the lastLive mapping.
        // We cache the ownersBefore set before iterating over it, because the remove() method mutates the set.
        address[] memory ownersBeforeCache = _ownersBefore.values();
        for (uint256 i = 0; i < ownersBeforeCache.length; i++) {
            address ownerBefore = ownersBeforeCache[i];
            delete lastLive[ownerBefore];
            _ownersBefore.remove(ownerBefore);
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
