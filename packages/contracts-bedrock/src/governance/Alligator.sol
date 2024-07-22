// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { IGovernor } from "@openzeppelin/contracts/governance/IGovernor.sol";
import { ERC20Votes } from "@openzeppelin/contracts/token/ERC20/extensions/ERC20Votes.sol";
import { SafeCast } from "@openzeppelin/contracts/utils/math/SafeCast.sol";
import { Math } from "@openzeppelin/contracts/utils/math/Math.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";

/// @notice Allowance type.
/// @param Absolute The amount of votes delegated is fixed.
/// @param Relative The amount of votes delegated is relative to the total amount of votes the delegator has.
enum AllowanceType {
    Absolute,
    Relative
}

/// @notice Subdelegation rules.
/// @param maxRedelegations       Maximum number of times the delegated votes can be redelegated.
/// @param blocksBeforeVoteCloses Number of blocks before the vote closes that the delegation is valid.
/// @param notValidBefore         Timestamp after which the delegation is valid.
/// @param notValidAfter          Timestamp before which the delegation is valid.
/// @param baseRules              Base subdelegation rules.
/// @param allowanceType          Type of allowance.
/// @param allowance              Amount of votes delegated. If `allowanceType` is Relative, 100% of allowance
///                               corresponds to 1e5. Otherwise, this is the exact amount of votes delegated.
struct SubdelegationRules {
    uint8 maxRedelegations;
    uint16 blocksBeforeVoteCloses;
    uint32 notValidBefore;
    uint32 notValidAfter;
    AllowanceType allowanceType;
    uint256 allowance;
}

/// @custom:predeploy 0x4200000000000000000000000000000000000043
/// @title Alligator
/// @notice A contract that allows delegation of votes to other accounts. It is used to implement subdelegation
///         functionality in the Optimism Governance system. It provides a way to migrate accounts from the Governance
///         token to the Alligator contract, and then delegate votes to other accounts under subdelegation rules.
contract Alligator {
    /// @notice Thrown when the account has not been migrated to the Alligator contract.
    error NotMigrated(address account);

    /// @notice Thrown when the caller is not the GovernanceToken contract.
    error NotGovernanceToken();

    /// @notice Thrown when there's a mismatch between the length of the `targets` and `subdelegationRules` arrays.
    error LengthMismatch();

    /// @notice Thrown when the delegation is not delegated.
    error NotDelegated(address from, address to);

    /// @notice Thrown when the delegation is delegated too many times.
    error TooManyRedelegations(address from, address to);

    /// @notice Thrown when the delegation is not valid yet.
    error NotValidYet(address from, address to, uint256 willBeValidFrom);

    /// @notice Thrown when the delegation is not valid anymore.
    error NotValidAnymore(address from, address to, uint256 wasValidUntil);

    /// @notice Thrown when the delegation is valid too early.
    error TooEarly(address from, address to, uint256 blocksBeforeVoteCloses);

    /// @notice Thrown when a block number is not yet mined.
    error BlockNotYetMined(uint256 blockNumber);

    /// @notice Flags to indicate if a account has been migrated to the Alligator contract.
    mapping(address => bool) public migrated;

    /// @notice Subdelegation rules for an account and delegatee.
    mapping(address => mapping(address => SubdelegationRules)) internal _subdelegations;

    /// @notice Checkpoints of votes for an account.
    mapping(address => ERC20Votes.Checkpoint[]) internal _checkpoints;

    /// @notice Checkpoints of total supply.
    ERC20Votes.Checkpoint[] internal _totalSupplyCheckpoints;

    /// @notice Emitted when a subdelegation is created.
    event Subdelegation(address indexed account, address indexed delegatee, SubdelegationRules subdelegationRules);

    /// @notice Emitted when multiple subdelegations are created under a single SubdelegationRules.
    event Subdelegations(address indexed account, address[] delegatee, SubdelegationRules subdelegationRules);

    /// @notice Emitted when multiple subdelegations are created under multiple SubdelegationRules.
    event Subdelegations(address indexed account, address[] delegatee, SubdelegationRules[] subdelegationRules);

    /// @notice Emitted when a delegator's voting power changes.
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

    /// @notice Returns the subdelegation rules for a given account and delegatee.
    /// @param _account   The account subdelegating.
    /// @param _delegatee The delegatee to get the subdelegation rules for.
    /// @return           The subdelegation rules.
    function subdelegations(address _account, address _delegatee) external view returns (SubdelegationRules memory) {
        return _subdelegations[_account][_delegatee];
    }

    /// @notice Subdelegate `to` with `subdelegationRules`.
    /// @param _delegatee          The address to subdelegate to.
    /// @param _subdelegationRules The rules to apply to the subdelegation.
    function subdelegate(
        address _delegatee,
        SubdelegationRules calldata _subdelegationRules
    )
        external
        migrate(msg.sender)
        migrate(_delegatee)
    {
        _subdelegations[msg.sender][_delegatee] = _subdelegationRules;
        emit Subdelegation(msg.sender, _delegatee, _subdelegationRules);

        // TODO: fix line below -> how to get some account's delegates? should we subtract the amount from the account?
        // what if we have several delegates?
        //_moveVotingPower({ _token: msg.sender, _src: delegates(_from), _dst: delegates(_to), _amount: _amount });
    }

    /// @notice Subdelegate `to` with `subdelegationRules`. This function can only be called from the GovernanceToken
    /// contract.
    /// @param _account            The address subdelegating.
    /// @param _delegatee          The address to subdelegate to.
    /// @param _subdelegationRules The rules to apply to the subdelegation.
    function subdelegateFromToken(
        address _account,
        address _delegatee,
        SubdelegationRules calldata _subdelegationRules
    )
        external
        onlyToken
        migrate(_account)
        migrate(_delegatee)
    {
        _subdelegations[_account][_delegatee] = _subdelegationRules;
        emit Subdelegation(_account, _delegatee, _subdelegationRules);

        // TODO: fix line below -> how to get some account's delegates? should we subtract the amount from the account?
        // what if we have several delegates?
        //_moveVotingPower({ _token: msg.sender, _src: delegates(_from), _dst: delegates(_to), _amount: _amount });
    }

    /// @notice Subdelegate `targets` with same `subdelegationRules` rule.
    /// @param _delegatees         The addresses to subdelegate to.
    /// @param _subdelegationRules The rule to apply to the subdelegations.
    function subdelegateBatched(
        address[] calldata _delegatees,
        SubdelegationRules calldata _subdelegationRules
    )
        external
        migrate(msg.sender)
    {
        for (uint256 i; i < _delegatees.length;) {
            address delegatee = _delegatees[i];

            // Migreate delegatee if it hasn't been migrated.
            if (!migrated[delegatee]) _migrate(delegatee);

            _subdelegations[msg.sender][delegatee] = _subdelegationRules;

            unchecked {
                ++i;
            }
        }

        emit Subdelegations(msg.sender, _delegatees, _subdelegationRules);
    }

    /// @notice Subdelegate `targets` with different `subdelegationRules` for each target.
    /// @param _delegatees         The addresses to subdelegate to.
    /// @param _subdelegationRules The rules to apply to the subdelegations.
    function subdelegateBatched(
        address[] calldata _delegatees,
        SubdelegationRules[] calldata _subdelegationRules
    )
        external
        migrate(msg.sender)
    {
        uint256 targetsLength = _delegatees.length;
        if (targetsLength != _subdelegationRules.length) revert LengthMismatch();

        for (uint256 i; i < targetsLength;) {
            address delegatee = _delegatees[i];

            // Migreate delegatee if it hasn't been migrated.
            if (!migrated[delegatee]) _migrate(delegatee);

            _subdelegations[msg.sender][delegatee] = _subdelegationRules[i];

            unchecked {
                ++i;
            }
        }

        emit Subdelegations(msg.sender, _delegatees, _subdelegationRules);
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
        // TODO: fix line below -> how to get some account's delegates? should we subtract the amount from the account?
        // what if we have several delegates?
        //_moveVotingPower({ _token: msg.sender, _src: delegates(_from), _dst: delegates(_to), _amount: _amount });
    }

    /// @notice Migrate an account's delegation state from the GovernanceToken contract to the Alligator contract.
    /// @param _account The account to migrate.
    function migrateAccount(address _account) external {
        if (!migrated[_account]) _migrate(_account);
    }

    /// @notice Validate subdelegation rules and partial delegation allowances.
    /// @param _token          The token to validate.
    /// @param _proxy          The address of the proxy.
    /// @param _sender         The sender address to validate.
    /// @param _authority      The authority chain to validate against.
    /// @param _proposalId     The id of the proposal for which validation is being performed.
    /// @param _support        The support value for the vote. 0=against, 1=for, 2=abstain, 0xFF=proposal
    /// @param _voterAllowance The allowance of the voter.
    /// @return _votesToCast   The number of votes to cast by `sender`.
    function _validate(
        address _token,
        address _proxy,
        address _sender,
        address[] calldata _authority,
        uint256 _proposalId,
        uint256 _support,
        uint256 _voterAllowance
    )
        internal
        view
        returns (uint256 _votesToCast)
    {
        address from = _authority[0];

        /// @dev Cannot underflow as `weightCast` is always less than or equal to total votes.
        unchecked {
            // TODO: update governor address below.
            // uint256 weightCast = IGovernor(address(0)).weightCast(_proposalId, _proxy);
            // _votesToCast = weightCast == 0 ? _voterAllowance : _voterAllowance - weightCast;
        }

        // If `_sender` is the proxy owner, only the proxy rules are validated.
        if (from == _sender) {
            return (_votesToCast);
        }

        address to;
        SubdelegationRules memory subdelegationRules;
        uint256 votesCastFactor;
        for (uint256 i = 1; i < _authority.length;) {
            to = _authority[i];

            subdelegationRules = _subdelegations[from][to];

            if (subdelegationRules.allowance == 0) {
                revert NotDelegated(from, to);
            }

            // Prevent double spending of votes already cast by previous delegators by adjusting
            // `subdelegationRules.allowance`.
            if (subdelegationRules.allowanceType == AllowanceType.Relative) {
                // `votesCastFactor`: remaining votes to cast by the delegate
                // Get `votesCastFactor` by subtracting `votesCastByAuthorityChain` to given allowance amount
                // Reverts for underflow when `votesCastByAuthorityChain > votesCastFactor`, when delegate has exceeded
                // the allowance.

                // TODO: below
                // votesCastFactor = subdelegationRules.allowance * _voterAllowance / 1e5
                //     - votesCastByAuthorityChain[_proxy][_proposalId][keccak256(abi.encode(_authority[0:i]))][to];

                // Adjust `_votesToCast` to the minimum between `votesCastFactor` and `_votesToCast`
                if (votesCastFactor < _votesToCast) {
                    _votesToCast = votesCastFactor;
                }
            } else {
                // `votesCastFactor`: total votes cast by the delegate
                // Retrieve votes cast by `to` via `from` regardless of the used authority chain

                // TODO: below
                // votesCastFactor = votesCast[_proxy][_proposalId][from][to];

                // Adjust allowance by subtracting eventual votes already cast by the delegate
                // Reverts for underflow when `votesCastFactor > _voterAllowance`, when delegate has exceeded the
                // allowance.
                if (votesCastFactor != 0) {
                    subdelegationRules.allowance = subdelegationRules.allowance - votesCastFactor;
                }
            }

            // Calculate `_voterAllowance` based on allowance given by `from`
            _voterAllowance =
                _getVoterAllowance(subdelegationRules.allowanceType, subdelegationRules.allowance, _voterAllowance);

            unchecked {
                _validateRules(
                    subdelegationRules,
                    _authority.length,
                    _proposalId,
                    from,
                    to,
                    ++i // pass `i + 1` and increment at the same time
                );
            }

            from = to;
        }

        if (from != _sender) revert NotDelegated(from, _sender);

        _votesToCast = _voterAllowance > _votesToCast ? _votesToCast : _voterAllowance;
    }

    /// @notice Validate subdelegation rules and partial delegation allowances.
    /// @param _subdelegationRules The rules to validate.
    /// @param _authorityLength    The length of the authority chain.
    /// @param _proposalId         The id of the proposal for which validation is being performed.
    /// @param _account            The address subdelegating.
    /// @param _delegatee          The address to subdelegate to.
    /// @param _redelegationIndex  The index of the redelegation in the authority chain.
    function _validateRules(
        SubdelegationRules memory _subdelegationRules,
        uint256 _authorityLength,
        uint256 _proposalId,
        address _account,
        address _delegatee,
        uint256 _redelegationIndex
    )
        internal
        view
    {
        /// @dev `maxRedelegation` cannot overflow as it increases by 1 each iteration
        /// @dev block.number + _subdelegationRules.blocksBeforeVoteCloses cannot overflow uint256
        unchecked {
            if (uint256(_subdelegationRules.maxRedelegations) + _redelegationIndex < _authorityLength) {
                revert TooManyRedelegations(_account, _delegatee);
            }
            if (block.timestamp < _subdelegationRules.notValidBefore) {
                revert NotValidYet(_account, _delegatee, _subdelegationRules.notValidBefore);
            }
            if (_subdelegationRules.notValidAfter != 0) {
                if (block.timestamp > _subdelegationRules.notValidAfter) {
                    revert NotValidAnymore(_account, _delegatee, _subdelegationRules.notValidAfter);
                }
            }
            if (_subdelegationRules.blocksBeforeVoteCloses != 0) {
                // TODO: update governor below?
                if (
                    IGovernor(address(0)).proposalDeadline(_proposalId)
                        > uint256(block.number) + uint256(_subdelegationRules.blocksBeforeVoteCloses)
                ) {
                    revert TooEarly(_account, _delegatee, _subdelegationRules.blocksBeforeVoteCloses);
                }
            }
        }
    }

    /// @notice Migrate an account to the Alligator contract.
    /// @param _account The account to migrate.
    function _migrate(address _account) internal {
        // Get the number of checkpoints.
        uint32 numCheckpoints = ERC20Votes(Predeploys.GOVERNANCE_TOKEN).numCheckpoints(_account);

        // Itereate over the checkpoints and store them.
        for (uint32 i; i < numCheckpoints;) {
            ERC20Votes.Checkpoint memory checkpoint = ERC20Votes(Predeploys.GOVERNANCE_TOKEN).checkpoints(_account, i);
            _checkpoints[_account].push(checkpoint);
            unchecked {
                ++i;
            }
        }

        // Set migrated flag
        migrated[_account] = true;
    }

    /// @notice Return the allowance of a voter, used in `validate`.
    /// @param _allowanceType          The type of allowance.
    /// @param _subdelegationAllowance The allowance of the subdelegation.
    /// @param _delegatorAllowance     The allowance of the delegator.
    /// @return                        The allowance of the voter.
    function _getVoterAllowance(
        AllowanceType _allowanceType,
        uint256 _subdelegationAllowance,
        uint256 _delegatorAllowance
    )
        internal
        pure
        returns (uint256)
    {
        if (_allowanceType == AllowanceType.Relative) {
            return _subdelegationAllowance >= 1e5
                ? _delegatorAllowance
                : _delegatorAllowance * _subdelegationAllowance / 1e5;
        }

        return _delegatorAllowance > _subdelegationAllowance ? _subdelegationAllowance : _delegatorAllowance;
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
    /// @param _src    The address of the source account.
    /// @param _dst    The address of the destination account.
    /// @param _amount The amount of voting power to move.
    function _moveVotingPower(address _src, address _dst, uint256 _amount) internal {
        if (_src != _dst && _amount > 0) {
            if (_src != address(0)) {
                (uint256 oldWeight, uint256 newWeight) = _writeCheckpoint(_checkpoints[_src], _subtract, _amount);
                emit DelegateVotesChanged(_src, oldWeight, newWeight);
                // Check if burn to update total supply checkpoint.
                if (_dst == address(0)) _writeCheckpoint(_totalSupplyCheckpoints, _subtract, _amount);
            }

            if (_dst != address(0)) {
                (uint256 oldWeight, uint256 newWeight) = _writeCheckpoint(_checkpoints[_dst], _add, _amount);
                emit DelegateVotesChanged(_dst, oldWeight, newWeight);
                // Check if mint to update total supply checkpoint.
                if (_src == address(0)) _writeCheckpoint(_totalSupplyCheckpoints, _add, _amount);
            }
        }
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
    /// @param a The first number.
    /// @param b The second number.
    /// @return  The sum of the two numbers.
    function _add(uint256 a, uint256 b) internal pure returns (uint256) {
        return a + b;
    }

    /// @notice Subtracts two numbers.
    /// @param a The first number.
    /// @param b The second number.
    /// @return  The difference of the two numbers.
    function _subtract(uint256 a, uint256 b) internal pure returns (uint256) {
        return a - b;
    }
}
