// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { IGovernor } from "@openzeppelin/contracts/governance/IGovernor.sol";
import { ERC20Votes } from "@openzeppelin/contracts/token/ERC20/extensions/ERC20Votes.sol";

contract Alligator is ERC20Votes {
    // =============================================================
    //                             ERRORS
    // =============================================================

    error LengthMismatch();
    error NotDelegated(address from, address to);
    error TooManyRedelegations(address from, address to);
    error NotValidYet(address from, address to, uint256 willBeValidFrom);
    error NotValidAnymore(address from, address to, uint256 wasValidUntil);
    error TooEarly(address from, address to, uint256 blocksBeforeVoteCloses);

    // =============================================================
    //                             EVENTS
    // =============================================================

    event Subdelegation(
        address indexed token, address indexed from, address indexed to, SubdelegationRules subdelegationRules
    );
    event Subdelegations(
        address indexed token, address indexed from, address[] to, SubdelegationRules subdelegationRules
    );
    event Subdelegations(
        address indexed token, address indexed from, address[] to, SubdelegationRules[] subdelegationRules
    );

    // =============================================================
    //                       IMMUTABLE STORAGE
    // =============================================================

    enum AllowanceType {
        Absolute,
        Relative
    }

    /// @param maxRedelegations The maximum number of times the delegated votes can be redelegated.
    /// @param blocksBeforeVoteCloses The number of blocks before the vote closes that the delegation is valid.
    /// @param notValidBefore The timestamp after which the delegation is valid.
    /// @param notValidAfter The timestamp before which the delegation is valid.
    /// @param baseRules The base subdelegation rules.
    /// @param allowanceType The type of allowance. If Absolute, the amount of votes delegated is fixed.
    /// If Relative, the amount of votes delegated is relative to the total amount of votes the delegator has.
    /// @param allowance The amount of votes delegated. If `allowanceType` is Relative 100% of allowance corresponds
    /// to 1e5, otherwise this is the exact amount of votes delegated.
    struct SubdelegationRules {
        uint8 maxRedelegations;
        uint16 blocksBeforeVoteCloses;
        uint32 notValidBefore;
        uint32 notValidAfter;
        AllowanceType allowanceType;
        uint256 allowance;
    }

    // =============================================================
    //                        MUTABLE STORAGE
    // =============================================================

    // token => from => to => subdelegationRules
    mapping(address => mapping(address => mapping(address => SubdelegationRules))) public subdelegations;

    // token => user => bool
    mapping(address => mapping(address => bool)) public migrated;

    // =============================================================
    //                         CONSTRUCTOR
    // =============================================================

    constructor() { }

    // =============================================================
    //                          TOKEN HOOK
    // =============================================================

    /// @notice Callback called after a token transfer.
    /// @param from   The account sending tokens.
    /// @param to     The account receiving tokens.
    /// @param amount The amount of tokens being transfered.
    function afterTokenTransfer(address from, address to, uint256 amount) external { }

    // =============================================================
    //                        SUBDELEGATIONS
    // =============================================================

    /// @notice Subdelegate `to` with `subdelegationRules`.
    /// @param from The address subdelegating.
    /// @param to The address to subdelegate to.
    /// @param subdelegationRules The rules to apply to the subdelegation.
    function subdelegate(address from, address to, SubdelegationRules calldata subdelegationRules) external {
        subdelegations[msg.sender][from][to] = subdelegationRules;
        emit Subdelegation(msg.sender, from, to, subdelegationRules);
    }

    /// @notice Subdelegate `targets` with `subdelegationRules`.
    /// @param from The address subdelegating.
    /// @param targets The addresses to subdelegate to.
    /// @param subdelegationRules The rules to apply to the subdelegations.
    function subdelegateBatched(
        address from,
        address[] calldata targets,
        SubdelegationRules calldata subdelegationRules
    )
        external
    {
        uint256 targetsLength = targets.length;
        for (uint256 i; i < targetsLength;) {
            subdelegations[msg.sender][from][targets[i]] = subdelegationRules;

            unchecked {
                ++i;
            }
        }

        emit Subdelegations(msg.sender, from, targets, subdelegationRules);
    }

    /// @notice Subdelegate `targets` with different `subdelegationRules` for each target.
    /// @param from The address subdelegating.
    /// @param targets The addresses to subdelegate to.
    /// @param subdelegationRules The rules to apply to the subdelegations.
    function subdelegateBatched(
        address from,
        address[] calldata targets,
        SubdelegationRules[] calldata subdelegationRules
    )
        external
    {
        uint256 targetsLength = targets.length;
        if (targetsLength != subdelegationRules.length) revert LengthMismatch();

        for (uint256 i; i < targetsLength;) {
            subdelegations[msg.sender][from][targets[i]] = subdelegationRules[i];

            unchecked {
                ++i;
            }
        }

        emit Subdelegations(msg.sender, from, targets, subdelegationRules);
    }

    // =============================================================
    //                         VIEW FUNCTIONS
    // =============================================================

    /// @notice Validate subdelegation rules and partial delegation allowances.
    /// @param proxy The address of the proxy.
    /// @param sender The sender address to validate.
    /// @param authority The authority chain to validate against.
    /// @param proposalId The id of the proposal for which validation is being performed.
    /// @param support The support value for the vote. 0=against, 1=for, 2=abstain, 0xFF=proposal
    /// @param voterAllowance The allowance of the voter.
    /// @return votesToCast The number of votes to cast by `sender`.
    function validate(
        address token,
        address proxy,
        address sender,
        address[] calldata authority,
        uint256 proposalId,
        uint256 support,
        uint256 voterAllowance
    )
        internal
        view
        returns (uint256 votesToCast)
    {
        address from = authority[0];

        /// @dev Cannot underflow as `weightCast` is always less than or equal to total votes.
        unchecked {
            // TODO: update governor address below.
            // uint256 weightCast = IGovernor(address(0)).weightCast(proposalId, proxy);
            // votesToCast = weightCast == 0 ? voterAllowance : voterAllowance - weightCast;
        }

        // If `sender` is the proxy owner, only the proxy rules are validated.
        if (from == sender) {
            return (votesToCast);
        }

        address to;
        SubdelegationRules memory subdelegationRules;
        uint256 votesCastFactor;
        for (uint256 i = 1; i < authority.length;) {
            to = authority[i];

            subdelegationRules = subdelegations[token][from][to];

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
                // votesCastFactor = subdelegationRules.allowance * voterAllowance / 1e5
                //     - votesCastByAuthorityChain[proxy][proposalId][keccak256(abi.encode(authority[0:i]))][to];

                // Adjust `votesToCast` to the minimum between `votesCastFactor` and `votesToCast`
                if (votesCastFactor < votesToCast) {
                    votesToCast = votesCastFactor;
                }
            } else {
                // `votesCastFactor`: total votes cast by the delegate
                // Retrieve votes cast by `to` via `from` regardless of the used authority chain

                // TODO: below
                // votesCastFactor = votesCast[proxy][proposalId][from][to];

                // Adjust allowance by subtracting eventual votes already cast by the delegate
                // Reverts for underflow when `votesCastFactor > voterAllowance`, when delegate has exceeded the
                // allowance.
                if (votesCastFactor != 0) {
                    subdelegationRules.allowance = subdelegationRules.allowance - votesCastFactor;
                }
            }

            // Calculate `voterAllowance` based on allowance given by `from`
            voterAllowance =
                _getVoterAllowance(subdelegationRules.allowanceType, subdelegationRules.allowance, voterAllowance);

            unchecked {
                _validateRules(
                    subdelegationRules,
                    authority.length,
                    proposalId,
                    from,
                    to,
                    ++i // pass `i + 1` and increment at the same time
                );
            }

            from = to;
        }

        if (from != sender) revert NotDelegated(from, sender);

        votesToCast = voterAllowance > votesToCast ? votesToCast : voterAllowance;
    }

    // =============================================================
    //                     RESTRICTED, INTERNAL
    // =============================================================

    function _validateRules(
        SubdelegationRules memory rules,
        uint256 authorityLength,
        uint256 proposalId,
        address from,
        address to,
        uint256 redelegationIndex
    )
        internal
        view
    {
        /// @dev `maxRedelegation` cannot overflow as it increases by 1 each iteration
        /// @dev block.number + rules.blocksBeforeVoteCloses cannot overflow uint256
        unchecked {
            if (uint256(rules.maxRedelegations) + redelegationIndex < authorityLength) {
                revert TooManyRedelegations(from, to);
            }
            if (block.timestamp < rules.notValidBefore) {
                revert NotValidYet(from, to, rules.notValidBefore);
            }
            if (rules.notValidAfter != 0) {
                if (block.timestamp > rules.notValidAfter) revert NotValidAnymore(from, to, rules.notValidAfter);
            }
            if (rules.blocksBeforeVoteCloses != 0) {
                // TODO: update governor below?
                if (
                    IGovernor(address(0)).proposalDeadline(proposalId)
                        > uint256(block.number) + uint256(rules.blocksBeforeVoteCloses)
                ) {
                    revert TooEarly(from, to, rules.blocksBeforeVoteCloses);
                }
            }
        }
    }

    /// @notice Return the allowance of a voter, used in `validate`.
    function _getVoterAllowance(
        AllowanceType allowanceType,
        uint256 subdelegationAllowance,
        uint256 delegatorAllowance
    )
        private
        pure
        returns (uint256)
    {
        if (allowanceType == AllowanceType.Relative) {
            return
                subdelegationAllowance >= 1e5 ? delegatorAllowance : delegatorAllowance * subdelegationAllowance / 1e5;
        }

        return delegatorAllowance > subdelegationAllowance ? subdelegationAllowance : delegatorAllowance;
    }
}
