// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { ERC20Votes } from "@openzeppelin/contracts/token/ERC20/extensions/ERC20Votes.sol";
import { SafeCast } from "@openzeppelin/contracts/utils/math/SafeCast.sol";
import { Math } from "@openzeppelin/contracts/utils/math/Math.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";

/// @notice Allowance type of a delegation.
/// @param Absolute The amount of votes delegated is fixed.
/// @param Relative The amount of votes delegated is relative to the total amount of votes the delegator has.
enum AllowanceType {
    Absolute,
    Relative
}

/// @notice Delegation of voting power.
/// @param delegatee              The address to delegate to.
/// @param allowanceType          Type of allowance.
/// @param amount                 Amount of votes delegated. If `allowanceType` is Relative, `amount` acts
///                               as a numerator and `DENOMINATOR` as a denominator. For example, 100% of allowance
///                               corresponds to 1e4. Otherwise, this is the exact amount of votes delegated.
struct Delegation {
    AllowanceType allowanceType;
    address delegatee;
    uint256 amount;
}

/// @notice Adjustment of delegation.
/// @param delegatee              The address to delegate to.
/// @param amount                 Amount of votes delegated.
struct DelegationAdjustment {
    address delegatee;
    uint208 amount;
}

/// @notice Operations for delegation adjustments.
/// @param ADD      Add votes to the delegatee.
/// @param SUBTRACT Subtract votes from the delegatee.
enum Op {
    ADD,
    SUBTRACT
}

/// @custom:predeploy 0x4200000000000000000000000000000000000043
/// @title GovernanceDelegation
/// @notice A contract that allows delegation of votes to other accounts. It is used to implement advanced delegation
///         functionality in the Optimism Governance system. It provides a way to migrate accounts from the Governance
///         token to the GovernanceDelegation contract, and delegate votes to other accounts using advanced delegations.
contract GovernanceDelegation {
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

    /// @notice The maximum number of delegations allowed.
    uint256 public constant MAX_DELEGATIONS = 20;

    /// @notice The denominator used for relative delegations.
    uint96 public constant DENOMINATOR = 10_000;

    /// @notice Flags to indicate if a account has been migrated to the GovernanceDelegation contract.
    mapping(address => bool) public migrated;

    /// @notice Delegations for an account.
    mapping(address => Delegation[]) internal delegations;

    /// @notice Checkpoints of votes for an account.
    mapping(address => ERC20Votes.Checkpoint[]) internal _checkpoints;

    /// @notice Checkpoints of total supply.
    ERC20Votes.Checkpoint[] internal _totalSupplyCheckpoints;

    /// @notice Emitted when delegations are created.
    event DelegationsCreated(address indexed account, Delegation[] delegations);

    /// @notice Emitted when a user's voting power changes.
    event DelegateVotesChanged(address indexed delegate, uint256 previousBalance, uint256 newBalance);

    modifier migrate(address _account) {
        if (!migrated[_account]) _migrate(_account);
        _;
    }

    modifier onlyToken() {
        if (msg.sender != Predeploys.GOVERNANCE_TOKEN) revert NotGovernanceToken();
        _;
    }

    /// @notice Returns the checkpoints for a given account.
    /// @param _account The account to get the checkpoints for.
    /// @return         The checkpoints.
    function checkpoints(address _account) external view returns (ERC20Votes.Checkpoint[] memory) {
        return _checkpoints[_account];
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
        return delegations[_account][0].delegatee;
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
    function delegate(Delegation calldata _delegation) external migrate(msg.sender) {
        Delegation[] memory delegation = new Delegation[](1);
        delegation[0] = _delegation;
        _delegate(msg.sender, delegation);
        emit DelegationsCreated(msg.sender, delegation);
    }

    /// @notice Apply a basic delegation from `_delegator` to `_delegatee`.
    /// @param _delegator          The address delegating.
    /// @param _delegatee          The address to delegate to.
    function delegateFromToken(
        address _delegator,
        address _delegatee
    )
        external
        onlyToken
        migrate(_delegator)
        migrate(_delegatee)
    {
        Delegation[] memory delegation = new Delegation[](1);
        delegation[0] = Delegation({
            delegatee: _delegatee,
            allowanceType: AllowanceType.Relative,
            amount: 1e4 // 100%
         });

        _delegate(_delegator, delegation);

        emit DelegationsCreated(_delegator, delegation);
    }

    /// @notice Apply multiple delegations.
    /// @param _delegations The delegations to apply.
    function delegateBatched(Delegation[] calldata _delegations) external migrate(msg.sender) {
        // TODO: migration inside the _delegate??
        _delegate(msg.sender, _delegations);
        emit DelegationsCreated(msg.sender, _delegations);
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
        for (uint256 i; i < _accounts.length;) {
            address _account = _accounts[i];
            if (!migrated[_account]) _migrate(_account);

            unchecked {
                i++;
            }
        }
    }

    /// @notice Delegate `_delegator`'s voting units to delegations specified in `_newDelegations`.
    /// @param _delegator         The delegator to delegate votes from.
    /// @param _newDelegations    The delegations to delegate votes to.
    function _delegate(address _delegator, Delegation[] memory _newDelegations) internal virtual {
        uint256 _newDelegationsLength = _newDelegations.length;
        if (_newDelegationsLength > MAX_DELEGATIONS) {
            revert LimitExceeded(_newDelegationsLength, MAX_DELEGATIONS);
        }

        // Calculate adjustments for old delegatee set, if it exists.
        Delegation[] memory _oldDelegations = delegations[_delegator];
        uint256 _oldDelegateLength = _oldDelegations.length;

        DelegationAdjustment[] memory _old = new DelegationAdjustment[](_oldDelegateLength);
        uint256 _delegatorVotes = ERC20Votes(Predeploys.GOVERNANCE_TOKEN).balanceOf(_delegator);
        if (_oldDelegateLength > 0) {
            _old = calculateWeightDistribution(_oldDelegations, _delegatorVotes);
        }

        // Calculate adjustments for new delegatee set.
        DelegationAdjustment[] memory _new = calculateWeightDistribution(_newDelegations, _delegatorVotes);

        // Now we want a collated list of all delegatee changes, combining the old subtractions with the new additions.
        // Ideally we'd like to process this only once.
        _aggregateDelegationAdjustmentsAndCreateCheckpoints(_old, _new);

        // The rest of this method body replaces in storage the old delegatees with the new ones.
        // keep track of last delegatee to ensure ordering / uniqueness:
        address _lastDelegatee;

        for (uint256 i; i < _newDelegationsLength; i++) {
            // check sorting and uniqueness
            if (i == 0 && _newDelegations[i].delegatee == address(0)) {
                // zero delegation is allowed if in 0th position
            } else if (_newDelegations[i].delegatee <= _lastDelegatee) {
                revert DuplicateOrUnsortedDelegatees(_newDelegations[i].delegatee);
            }

            // replace existing delegatees in storage
            if (i < _oldDelegateLength) {
                delegations[_delegator][i] = _newDelegations[i];
            }
            // or add new delegatees
            else {
                delegations[_delegator].push(_newDelegations[i]);
            }
            _lastDelegatee = _newDelegations[i].delegatee;
        }
        // remove any remaining old delegatees
        if (_oldDelegateLength > _newDelegationsLength) {
            for (uint256 i = _newDelegationsLength; i < _oldDelegateLength; i++) {
                delegations[_delegator].pop();
            }
        }
        // TODO: event below.
        // emit DelegateChanged(_delegator, _oldDelegations, _newDelegations);
    }

    /// @notice Calculate the delegation adjusments and checkpoints given an old and new delegation set.
    /// @param _old The old delegation set.
    /// @param _new The new delegation set.
    function _aggregateDelegationAdjustmentsAndCreateCheckpoints(
        DelegationAdjustment[] memory _old,
        DelegationAdjustment[] memory _new
    )
        internal
    {
        // start with ith member of _old and jth member of _new.
        // If they are the same delegatee, combine them, check if result is 0, and iterate i and j.
        // If _old[i] > _new[j], add _new[j] to the final array and iterate j. If _new[j] > _old[i], add _old[i] and
        // iterate
        // i.
        uint256 i;
        uint256 j;
        uint256 _oldLength = _old.length;
        uint256 _newLength = _new.length;
        while (i < _oldLength || j < _newLength) {
            DelegationAdjustment memory _delegationAdjustment;
            Op _op;

            // same address is present in both arrays
            if (i < _oldLength && j < _newLength && _old[i].delegatee == _new[j].delegatee) {
                // combine, checkpoint, and iterate
                _delegationAdjustment.delegatee = _old[i].delegatee;
                if (_old[i].amount != _new[j].amount) {
                    if (_old[i].amount > _new[j].amount) {
                        _op = Op.SUBTRACT;
                        _delegationAdjustment.amount = _old[i].amount - _new[j].amount;
                    } else {
                        _op = Op.ADD;
                        _delegationAdjustment.amount = _new[j].amount - _old[i].amount;
                    }
                }
                i++;
                j++;
            } else if (
                j == _newLength // if we've exhausted the new array, we can just checkpoint the old values
                    || (i != _oldLength && _old[i].delegatee < _new[j].delegatee) // or, if the ith old delegatee is next in
                    // line
            ) {
                // skip if 0...
                _delegationAdjustment.delegatee = _old[i].delegatee;
                if (_old[i].amount != 0) {
                    _op = Op.SUBTRACT;
                    _delegationAdjustment.amount = _old[i].amount;
                }
                i++;
            } else {
                // skip if 0...
                _delegationAdjustment.delegatee = _new[j].delegatee;
                if (_new[j].amount != 0) {
                    _op = Op.ADD;
                    _delegationAdjustment.amount = _new[j].amount;
                }
                j++;
            }

            if (_delegationAdjustment.amount != 0 && _delegationAdjustment.delegatee != address(0)) {
                (uint256 oldValue, uint256 newValue) = _writeCheckpoint(
                    _checkpoints[_delegationAdjustment.delegatee],
                    _op == Op.ADD ? _add : _subtract,
                    _delegationAdjustment.amount
                );

                emit DelegateVotesChanged(_delegationAdjustment.delegatee, oldValue, newValue);
            }
        }
    }

    /// @notice Calculate the weight distribution for a list of delegations.
    /// @param _delegations The delegations to calculate the weight distribution for.
    /// @param _amount      The sum of vote amounts to calculate the weight distribution for.
    function calculateWeightDistribution(
        Delegation[] memory _delegations,
        uint256 _amount
    )
        public
        pure
        returns (DelegationAdjustment[] memory)
    {
        uint256 _delegationsLength = _delegations.length;
        DelegationAdjustment[] memory _delegationAdjustments = new DelegationAdjustment[](_delegationsLength);

        // For relative delegations, keep track of total numerator to ensure it doesn't exceed DENOMINATOR
        uint256 _totalNumerator;

        // Iterate through partial delegations to calculate vote weight
        for (uint256 i; i < _delegationsLength; i++) {
            if (_delegations[i].allowanceType == AllowanceType.Relative) {
                if (_delegations[i].amount == 0) {
                    revert InvalidNumeratorZero();
                }

                _delegationAdjustments[i] = DelegationAdjustment(
                    _delegations[i].delegatee, uint208(_amount * _delegations[i].amount / DENOMINATOR)
                );
                _totalNumerator += _delegations[i].amount;

                if (_totalNumerator > DENOMINATOR) {
                    revert NumeratorSumExceedsDenominator(_totalNumerator, DENOMINATOR);
                }
            } else {
                _delegationAdjustments[i] =
                    DelegationAdjustment(_delegations[i].delegatee, uint208(_delegations[i].amount));
            }
        }
        return _delegationAdjustments;
    }

    /// @notice Migrate an account to the GovernanceDelegation contract.
    /// @param _account The account to migrate.
    function _migrate(address _account) internal {
        // Get the number of checkpoints.
        uint32 _numCheckpoints = ERC20Votes(Predeploys.GOVERNANCE_TOKEN).numCheckpoints(_account);

        // Itereate over the checkpoints and store them.
        for (uint32 i; i < _numCheckpoints;) {
            ERC20Votes.Checkpoint memory checkpoint = ERC20Votes(Predeploys.GOVERNANCE_TOKEN).checkpoints(_account, i);
            _checkpoints[_account].push(checkpoint);
            unchecked {
                ++i;
            }
        }

        // Set migrated flag
        migrated[_account] = true;
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

    /// @notice Moves voting power from `_src` to `_dst` by `_amount`.
    /// @param _from    The address of the source account.
    /// @param _to    The address of the destination account.
    /// @param _amount The amount of voting power to move.
    function _moveVotingPower(address _from, address _to, uint256 _amount) internal {
        // skip from==to no-op, as the math would require special handling
        if (_from == _to) {
            return;
        }

        // update total supply checkpoints if mint/burn
        if (_from == address(0)) {
            _writeCheckpoint(_totalSupplyCheckpoints, _add, _amount);
        }
        if (_to == address(0)) {
            _writeCheckpoint(_totalSupplyCheckpoints, _subtract, _amount);
        }

        // finally, calculate delegatee vote changes and create checkpoints accordingly
        uint256 _fromLength = delegations[_from].length;
        DelegationAdjustment[] memory delegationAdjustmentsFrom = new DelegationAdjustment[](_fromLength);
        // We'll need to adjust the delegatee votes for both "_from" and "_to" delegatee sets.
        if (_fromLength > 0) {
            uint256 _fromVotes = ERC20Votes(Predeploys.GOVERNANCE_TOKEN).balanceOf(_from);
            DelegationAdjustment[] memory from = calculateWeightDistribution(delegations[_from], _fromVotes + _amount);
            DelegationAdjustment[] memory fromNew = calculateWeightDistribution(delegations[_from], _fromVotes);
            for (uint256 i; i < _fromLength; i++) {
                delegationAdjustmentsFrom[i] = DelegationAdjustment({
                    delegatee: delegations[_from][i].delegatee,
                    amount: from[i].amount - fromNew[i].amount
                });
            }
        }

        uint256 _toLength = delegations[_to].length;
        DelegationAdjustment[] memory delegationAdjustmentsTo = new DelegationAdjustment[](_toLength);
        if (_toLength > 0) {
            uint256 _toVotes = ERC20Votes(Predeploys.GOVERNANCE_TOKEN).balanceOf(_to);
            DelegationAdjustment[] memory to = calculateWeightDistribution(delegations[_to], _toVotes - _amount);
            DelegationAdjustment[] memory toNew = calculateWeightDistribution(delegations[_to], _toVotes);

            for (uint256 i; i < _toLength; i++) {
                delegationAdjustmentsTo[i] = (
                    DelegationAdjustment({
                        delegatee: delegations[_to][i].delegatee,
                        amount: toNew[i].amount - to[i].amount
                    })
                );
            }
        }
        _aggregateDelegationAdjustmentsAndCreateCheckpoints(delegationAdjustmentsFrom, delegationAdjustmentsTo);
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
}
