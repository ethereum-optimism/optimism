// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

contract Alligator {
    // =============================================================
    //                             ERRORS
    // =============================================================

    error LengthMismatch();
    // error InvalidSignature(ECDSA.RecoverError recoverError);
    error NotDelegated(address from, address to);
    error TooManyRedelegations(address from, address to);
    error NotValidYet(address from, address to, uint256 willBeValidFrom);
    error NotValidAnymore(address from, address to, uint256 wasValidUntil);
    error TooEarly(address from, address to, uint256 blocksBeforeVoteCloses);
    error InvalidCustomRule(address from, address to, address customRule);

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

    /**
     * @param maxRedelegations The maximum number of times the delegated votes can be redelegated.
     * @param blocksBeforeVoteCloses The number of blocks before the vote closes that the delegation is valid.
     * @param notValidBefore The timestamp after which the delegation is valid.
     * @param notValidAfter The timestamp before which the delegation is valid.
     * @param baseRules The base subdelegation rules.
     * @param allowanceType The type of allowance. If Absolute, the amount of votes delegated is fixed.
     * If Relative, the amount of votes delegated is relative to the total amount of votes the delegator has.
     * @param allowance The amount of votes delegated. If `allowanceType` is Relative 100% of allowance corresponds
     * to 1e5, otherwise this is the exact amount of votes delegated.
     */
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

    // =============================================================
    //                         CONSTRUCTOR
    // =============================================================

    constructor() { }

    // =============================================================
    //                        SUBDELEGATIONS
    // =============================================================

    /**
     * Subdelegate `to` with `subdelegationRules`.
     *
     * @param token The address of the token.
     * @param to The address to subdelegate to.
     * @param subdelegationRules The rules to apply to the subdelegation.
     */
    function subdelegate(address token, address to, SubdelegationRules calldata subdelegationRules) external {
        subdelegations[token][msg.sender][to] = subdelegationRules;
        emit Subdelegation(token, msg.sender, to, subdelegationRules);
    }

    /**
     * Subdelegate `targets` with `subdelegationRules`.
     *
     * @param token The address of the token.
     * @param targets The addresses to subdelegate to.
     * @param subdelegationRules The rules to apply to the subdelegations.
     */
    function subdelegateBatched(
        address token,
        address[] calldata targets,
        SubdelegationRules calldata subdelegationRules
    )
        external
    {
        uint256 targetsLength = targets.length;
        for (uint256 i; i < targetsLength;) {
            subdelegations[token][msg.sender][targets[i]] = subdelegationRules;

            unchecked {
                ++i;
            }
        }

        emit Subdelegations(token, msg.sender, targets, subdelegationRules);
    }

    /**
     * Subdelegate `targets` with different `subdelegationRules` for each target.
     *
     * @param token The address of the token.
     * @param targets The addresses to subdelegate to.
     * @param subdelegationRules The rules to apply to the subdelegations.
     */
    function subdelegateBatched(
        address token,
        address[] calldata targets,
        SubdelegationRules[] calldata subdelegationRules
    )
        external
    {
        uint256 targetsLength = targets.length;
        if (targetsLength != subdelegationRules.length) revert LengthMismatch();

        for (uint256 i; i < targetsLength;) {
            subdelegations[token][msg.sender][targets[i]] = subdelegationRules[i];

            unchecked {
                ++i;
            }
        }

        emit Subdelegations(token, msg.sender, targets, subdelegationRules);
    }
}
