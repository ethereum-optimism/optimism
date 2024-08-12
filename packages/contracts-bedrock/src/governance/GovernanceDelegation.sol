// SPDX-License-Identifier: MIT
pragma solidity 0.8.25;

import { ERC20Votes } from "@openzeppelin/contracts/token/ERC20/extensions/ERC20Votes.sol";
import { SafeCast } from "@openzeppelin/contracts/utils/math/SafeCast.sol";
import { Math } from "@openzeppelin/contracts/utils/math/Math.sol";
import { EnumerableMap } from "@openzeppelin/contracts/utils/structs/EnumerableMap.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";
import { IGovernanceDelegation } from "src/governance/IGovernanceDelegation.sol";

/// @notice Thrown when the caller is not the GovernanceToken contract.
error NotGovernanceToken();

/// @notice Thrown when the number of delegations exceeds the maximum allowed.
error LimitExceeded(uint256 length, uint256 maxLength);

/// @notice Thrown when the provided numerator is zero.
error InvalidNumeratorZero();

/// @notice Thrown when the sum of the numerators exceeds the denominator.
error NumeratorSumExceedsDenominator(uint256 numerator, uint96 denominator);

/// @notice The provided delegatee list is not sorted or contains duplicates.
error DuplicateOrUnsortedDelegatees(address delegatee);

/// @notice Thrown when a block number is not yet mined.
error BlockNotYetMined(uint256 blockNumber);

/// @custom:predeploy 0x4200000000000000000000000000000000000043
/// @title GovernanceDelegation
/// @notice A contract that allows delegation of votes to other accounts. It is used to implement advanced delegation
///         functionality in the Optimism Governance system. It provides a way to migrate accounts from the Governance
///         token to the GovernanceDelegation contract, and delegate votes to other accounts using advanced delegations.
contract GovernanceDelegation is IGovernanceDelegation {
    using EnumerableMap for EnumerableMap.AddressToUintMap;

    /// @notice The maximum number of delegations allowed.
    uint256 public constant MAX_DELEGATIONS = 20;

    /// @notice The denominator used for relative delegations.
    uint96 public constant DENOMINATOR = 10_000;

    /// @notice Semantic version.
    /// @custom:semver 1.0.0-beta.1
    string public constant version = "1.0.0-beta.1";

    /// @notice Flags to indicate if a account has been migrated to the GovernanceDelegation contract.
    mapping(address => bool) public migrated;

    /// @notice Delegations for an account.
    mapping(address => Delegation[]) internal _delegations;

    /// @notice Checkpoints of votes for an account.
    mapping(address => ERC20Votes.Checkpoint[]) internal _checkpoints;

    /// @notice Checkpoints of total supply.
    ERC20Votes.Checkpoint[] internal _totalSupplyCheckpoints;

    /// @notice Store temporary delegation adjusments.
    EnumerableMap.AddressToUintMap private _adjustments;

    /// @notice Emitted when an account's delegations are changed.
    event DelegationsChanged(address indexed account, Delegation[] oldDelegations, Delegation[] newDelegations);

    /// @notice Emitted when a user's voting power changes.
    event DelegateVotesChanged(address indexed delegate, uint256 previousBalance, uint256 newBalance);

    modifier migrate(address _account) {
        if (!migrated[_account]) _migrate(_account);
        _;
    }

    modifier onlyToken() {
        if (msg.sender != Predeploys.GOVERNANCE_TOKEN) {
            revert NotGovernanceToken();
        }
        _;
    }

    /// @notice Stores the total supply checkpoints.
    /// @param __checkpoints The total supply checkpoints to set.
    constructor(ERC20Votes.Checkpoint[] memory __checkpoints) {
        uint256 _checkpointsLength = __checkpoints.length;

        for (uint32 i; i < _checkpointsLength; i++) {
            _totalSupplyCheckpoints.push(__checkpoints[i]);
        }
    }

    /// @notice Returns the delegations for a given account.
    /// @param _account The account to get the delegations for.
    /// @return         The delegations.
    function delegations(address _account) external view returns (Delegation[] memory) {
        return _delegations[_account];
    }

    /// @notice Returns the checkpoint for a given account.
    /// @param _account The account to get the checkpoint for.
    /// @param _pos     The position to get the checkpoint for.
    /// @return         The checkpoint.
    function checkpoints(address _account, uint32 _pos) external view returns (ERC20Votes.Checkpoint memory) {
        return _checkpoints[_account][_pos];
    }

    /// @notice Returns the number of checkpoints for a account.
    /// @param _account The account to get the total supply checkpoints for.
    /// @return         The total supply checkpoints.
    function numCheckpoints(address _account) external view returns (uint32) {
        return SafeCast.toUint32(_checkpoints[_account].length);
    }

    /// @notice Returns the delegatee with the most voting power for a given account.
    /// @param _account The account to get the delegatee for.
    function delegates(address _account) public view returns (address) {
        return _delegations[_account][0].delegatee;
    }

    /// @notice Returns the number of votes for a given account.
    /// @param _account     The account to get the number of votes for.
    /// @return             The number of votes.
    function getVotes(address _account) external view returns (uint256) {
        uint256 pos = _checkpoints[_account].length;
        return pos == 0 ? 0 : _checkpoints[_account][pos - 1].votes;
    }

    /// @notice Returns the number of votes for `_account` at the end of `_blockNumber`.
    /// @param _account     The address of the account to get the number of votes for.
    /// @param _blockNumber The block number to get the number of votes for.
    /// @return             The number of votes.
    function getPastVotes(address _account, uint256 _blockNumber) external view returns (uint256) {
        if (_blockNumber >= block.number) revert BlockNotYetMined(_blockNumber);
        return _checkpointsLookup(_checkpoints[_account], _blockNumber);
    }

    /// @notice Returns the total supply at a block.
    /// @param _blockNumber The block number to get the total supply.
    /// @return         The total supply of the token for the given block.
    function getPastTotalSupply(uint256 _blockNumber) external view returns (uint256) {
        if (_blockNumber >= block.number) revert BlockNotYetMined(_blockNumber);
        return _checkpointsLookup(_totalSupplyCheckpoints, _blockNumber);
    }

    /// @notice Apply a delegation.
    /// @param _delegation         The delegeation to apply.
    function delegate(Delegation calldata _delegation) external {
        Delegation[] memory delegation = new Delegation[](1);
        delegation[0] = _delegation;
        _delegate(msg.sender, delegation);
    }

    /// @notice Apply a basic delegation from `_delegator` to `_delegatee`.
    /// @param _delegator          The address delegating.
    /// @param _delegatee          The address to delegate to.
    function delegateFromToken(address _delegator, address _delegatee) external onlyToken {
        Delegation[] memory delegation = new Delegation[](1);
        delegation[0] = Delegation(AllowanceType.Relative, _delegatee, DENOMINATOR);
        _delegate(_delegator, delegation);
    }

    /// @notice Apply multiple delegations.
    /// @param _newDelegations The delegations to apply.
    function delegateBatched(Delegation[] calldata _newDelegations) external {
        _delegate(msg.sender, _newDelegations);
    }

    /// @notice Callback called after token transfer in the GovernanceToken contract.
    /// @param _from   The account sending tokens.
    /// @param _to     The account receiving tokens.
    /// @param _amount The amount of tokens being transfered.
    function afterTokenTransfer(
        address _from,
        address _to,
        uint256 _amount
    )
        external
        onlyToken
        migrate(_from)
        migrate(_to)
    {
        _moveVotingPower(_from, _to, _amount);
    }

    /// @notice Migrate accounts' delegation state from the GovernanceToken contract to the
    /// GovernanceDelegation contract.
    /// @param _accounts The accounts to migrate.
    function migrateAccounts(address[] calldata _accounts) external {
        for (uint256 i; i < _accounts.length; i++) {
            address _account = _accounts[i];
            if (!migrated[_account]) _migrate(_account);
        }
    }

    /// @notice Migrate the delegation state of accounts from the token.
    /// @param _account The account to migrate.
    function _migrate(address _account) internal {
        // Get the number of checkpoints.
        uint32 _numCheckpoints = ERC20Votes(Predeploys.GOVERNANCE_TOKEN).numCheckpoints(_account);

        // Itereate over the checkpoints and store them.
        for (uint32 i; i < _numCheckpoints; i++) {
            ERC20Votes.Checkpoint memory checkpoint = ERC20Votes(Predeploys.GOVERNANCE_TOKEN).checkpoints(_account, i);
            _checkpoints[_account].push(checkpoint);
        }

        // Set migrated flag
        migrated[_account] = true;
    }

    /// @notice Delegate `_delegator`'s voting units to delegations specified in `_newDelegations`.
    /// @param _delegator         The delegator to delegate votes from.
    /// @param _newDelegations    The delegations to delegate votes to.
    function _delegate(address _delegator, Delegation[] memory _newDelegations) internal migrate(_delegator) {
        uint256 _newDelegationsLength = _newDelegations.length;
        if (_newDelegationsLength > MAX_DELEGATIONS) {
            revert LimitExceeded(_newDelegationsLength, MAX_DELEGATIONS);
        }

        Delegation[] memory _oldDelegations = _delegations[_delegator];
        uint256 _oldDelegationsLength = _oldDelegations.length;

        uint256 _delegatorVotes = ERC20Votes(Predeploys.GOVERNANCE_TOKEN).balanceOf(_delegator);

        // Net the old and new delegations and create checkpoints.
        _createCheckpoints(
            _calculateWeightDistribution(_oldDelegations, _delegatorVotes),
            _calculateWeightDistribution(_newDelegations, _delegatorVotes)
        );

        // Store the last delegatee to check for sorting and uniqueness.
        address _lastDelegatee;

        // Store new delegations.
        for (uint256 i; i < _newDelegationsLength; i++) {
            // Check sorting and uniqueness of delegatees.
            if (i == 0 && _newDelegations[i].delegatee == address(0)) {
                // zero delegation is allowed if in 0th position
            } else if (_newDelegations[i].delegatee <= _lastDelegatee) {
                revert DuplicateOrUnsortedDelegatees(_newDelegations[i].delegatee);
            }

            // Add new delegations by either updating or pushing.
            if (i < _oldDelegationsLength) {
                _delegations[_delegator][i] = _newDelegations[i];
            } else {
                _delegations[_delegator].push(_newDelegations[i]);
            }

            _lastDelegatee = _newDelegations[i].delegatee;
        }
        // Remove any old delegations.
        if (_oldDelegationsLength > _newDelegationsLength) {
            for (uint256 i = _newDelegationsLength; i < _oldDelegationsLength; i++) {
                _delegations[_delegator].pop();
            }
        }

        emit DelegationsChanged(_delegator, _oldDelegations, _newDelegations);
    }

    /// @notice Aggregates delegation adjustments and creates checkpoints.
    /// @param _old The old delegation set.
    /// @param _new The new delegation set.
    function _createCheckpoints(DelegationAdjustment[] memory _old, DelegationAdjustment[] memory _new) internal {
        for (uint256 i; i < _old.length; i++) {
            _adjustments.set(_old[i].delegatee, uint256(_old[i].amount));
        }

        for (uint256 i; i < _new.length; i++) {
            address delegatee = _new[i].delegatee;
            if (delegatee == address(0)) continue;

            Op op = Op.ADD;
            uint256 amount = _new[i].amount;

            // Any duplicate delegations will revert in `_delegate`.
            if (_adjustments.contains(delegatee)) {
                uint256 oldAmount = _adjustments.get(delegatee);
                (amount, op) = oldAmount > amount ? (oldAmount - amount, Op.SUBTRACT) : (amount - oldAmount, Op.ADD);
                _adjustments.remove(delegatee);
            }

            (uint256 oldValue, uint256 newValue) =
                _writeCheckpoint(_checkpoints[delegatee], op == Op.ADD ? _add : _subtract, amount);

            emit DelegateVotesChanged(delegatee, oldValue, newValue);
        }

        uint256 _adjustmentsLength = _adjustments.length();

        for (uint256 i; i < _adjustmentsLength; i++) {
            (address delegatee, uint256 amount) = _adjustments.at(i);
            (uint256 oldValue, uint256 newValue) = _writeCheckpoint(_checkpoints[delegatee], _subtract, amount);

            _adjustments.remove(delegatee);

            emit DelegateVotesChanged(delegatee, oldValue, newValue);
        }
    }

    /// @notice Calculate the weight distribution for a list of delegations.
    /// @param _delegationSet The delegations to calculate the weight distribution for.
    /// @param _amount        The sum of vote amounts to calculate the weight distribution for.
    function _calculateWeightDistribution(
        Delegation[] memory _delegationSet,
        uint256 _amount
    )
        internal
        returns (DelegationAdjustment[] memory)
    {
        uint256 _delegationsLength = _delegationSet.length;
        DelegationAdjustment[] memory _delegationAdjustments = new DelegationAdjustment[](_delegationsLength);

        // For relative delegations, keep track of total numerator to ensure it doesn't exceed DENOMINATOR
        uint256 _totalNumerator;

        // Iterate through partial delegations to calculate vote weight
        for (uint256 i; i < _delegationsLength; i++) {
            if (!migrated[_delegationSet[i].delegatee]) {
                _migrate(_delegationSet[i].delegatee);
            }

            if (_delegationSet[i].allowanceType == AllowanceType.Relative) {
                if (_delegationSet[i].amount == 0) {
                    revert InvalidNumeratorZero();
                }

                _delegationAdjustments[i] = DelegationAdjustment(
                    _delegationSet[i].delegatee, uint208((_amount * _delegationSet[i].amount) / DENOMINATOR)
                );
                _totalNumerator += _delegationSet[i].amount;

                if (_totalNumerator > DENOMINATOR) {
                    revert NumeratorSumExceedsDenominator(_totalNumerator, DENOMINATOR);
                }
            } else {
                _delegationAdjustments[i] =
                    DelegationAdjustment(_delegationSet[i].delegatee, uint208(_delegationSet[i].amount));
            }
        }
        return _delegationAdjustments;
    }

    /// @notice Moves voting power from `_src` to `_dst` by `_amount`.
    /// @param _from    The address of the source account.
    /// @param _to    The address of the destination account.
    /// @param _amount The amount of voting power to move.
    function _moveVotingPower(address _from, address _to, uint256 _amount) internal {
        // Skip when addresses are equal or amount is zero.
        if (_from == _to || _amount == 0) {
            return;
        }

        // Increase total supply checkpoint for mint
        if (_from == address(0)) {
            _writeCheckpoint(_totalSupplyCheckpoints, _add, _amount);
        }

        // Decrease total supply checkpoint for burn
        if (_to == address(0)) {
            _writeCheckpoint(_totalSupplyCheckpoints, _subtract, _amount);
        }

        // Create checkpoints for the `from` delegatees.
        uint256 _fromLength = _delegations[_from].length;
        if (_fromLength > 0) {
            uint256 _fromVotes = ERC20Votes(Predeploys.GOVERNANCE_TOKEN).balanceOf(_from);
            DelegationAdjustment[] memory from = _calculateWeightDistribution(_delegations[_from], _fromVotes + _amount);
            DelegationAdjustment[] memory fromNew = _calculateWeightDistribution(_delegations[_from], _fromVotes);
            for (uint256 i; i < _fromLength; i++) {
                (uint256 oldValue, uint256 newValue) = _writeCheckpoint(
                    _checkpoints[_delegations[_from][i].delegatee], _subtract, from[i].amount - fromNew[i].amount
                );

                emit DelegateVotesChanged(_delegations[_from][i].delegatee, oldValue, newValue);
            }
        }

        // Create checkpoints for the `to` delegatees.
        uint256 _toLength = _delegations[_to].length;
        if (_toLength > 0) {
            uint256 _toVotes = ERC20Votes(Predeploys.GOVERNANCE_TOKEN).balanceOf(_to);
            DelegationAdjustment[] memory to = _calculateWeightDistribution(_delegations[_to], _toVotes - _amount);
            DelegationAdjustment[] memory toNew = _calculateWeightDistribution(_delegations[_to], _toVotes);

            for (uint256 i; i < _toLength; i++) {
                (uint256 oldValue, uint256 newValue) =
                    _writeCheckpoint(_checkpoints[_delegations[_to][i].delegatee], _add, toNew[i].amount - to[i].amount);

                emit DelegateVotesChanged(_delegations[_to][i].delegatee, oldValue, newValue);
            }
        }
    }

    /// @notice Returns the checkpoints for a given token and account.
    /// @param _ckpts The checkpoints to get the checkpoints for.
    /// @param _blockNumber The block number to get the checkpoints for.
    function _checkpointsLookup(
        ERC20Votes.Checkpoint[] storage _ckpts,
        uint256 _blockNumber
    )
        private
        view
        returns (uint256)
    {
        // We run a binary search to look for the earliest checkpoint taken after `_blockNumber`.
        //
        // During the loop, the index of the wanted checkpoint remains in the range [low-1, high).
        // With each iteration, either `low` or `high` is moved towards the middle of the range to maintain the
        // invariant.
        // - If the middle checkpoint is after `_blockNumber`, we look in [low, mid)
        // - If the middle checkpoint is before or equal to `_blockNumber`, we look in [mid+1, high)
        // Once we reach a single value (when low == high), we've found the right checkpoint at the index high-1, if not
        // out of bounds (in which case we're looking too far in the past and the result is 0).
        // Note that if the latest checkpoint available is exactly for `_blockNumber`, we end up with an index that is
        // past the end of the array, so we technically don't find a checkpoint after `_blockNumber`, but it works out
        // the same.
        uint256 high = _ckpts.length;
        uint256 low = 0;
        while (low < high) {
            uint256 mid = Math.average(low, high);
            if (_ckpts[mid].fromBlock > _blockNumber) {
                high = mid;
            } else {
                low = mid + 1;
            }
        }

        return high == 0 ? 0 : _ckpts[high - 1].votes;
    }

    /// @notice Writes a checkpoint with `_delta` and `op` to `_ckpts`.
    /// @param _ckpts      The checkpoints to write to.
    /// @param _op         The operation to perform.
    /// @param _delta      The amount to add or subtract.
    /// @return _oldWeight The old weight.
    /// @return _newWeight The new weight.
    function _writeCheckpoint(
        ERC20Votes.Checkpoint[] storage _ckpts,
        function(uint256, uint256) view returns (uint256) _op,
        uint256 _delta
    )
        private
        returns (uint256 _oldWeight, uint256 _newWeight)
    {
        uint256 pos = _ckpts.length;
        _oldWeight = pos == 0 ? 0 : _ckpts[pos - 1].votes;
        _newWeight = _op(_oldWeight, _delta);

        if (pos > 0 && _ckpts[pos - 1].fromBlock == block.number) {
            _ckpts[pos - 1].votes = SafeCast.toUint224(_newWeight);
        } else {
            _ckpts.push(
                ERC20Votes.Checkpoint({
                    fromBlock: SafeCast.toUint32(block.number),
                    votes: SafeCast.toUint224(_newWeight)
                })
            );
        }
    }

    /// @notice Adds two numbers.
    /// @param _a The first number.
    /// @param _b The second number.
    /// @return  The sum of the two numbers.
    function _add(uint256 _a, uint256 _b) internal pure returns (uint256) {
        return _a + _b;
    }

    /// @notice Subtracts two numbers.
    /// @param _a The first number.
    /// @param _b The second number.
    /// @return  The difference of the two numbers.
    function _subtract(uint256 _a, uint256 _b) internal pure returns (uint256) {
        return _a - _b;
    }

    function _extract_32_4(bytes32 _self, uint8 _offset) internal pure returns (bytes4 _result) {
        assembly ("memory-safe") {
            _result := and(shl(mul(8, _offset), _self), shl(224, not(0)))
        }
    }

    function _pack_28_4(bytes28 _left, bytes4 _right) internal pure returns (bytes32 _result) {
        assembly ("memory-safe") {
            _left := and(_left, shl(32, not(0)))
            _right := and(_right, shl(224, not(0)))
            _result := or(_left, shr(224, _right))
        }
    }

    function _extract_32_28(bytes32 _self, uint8 _offset) internal pure returns (bytes28 _result) {
        assembly ("memory-safe") {
            _result := and(shl(mul(8, _offset), _self), shl(32, not(0)))
        }
    }
}
