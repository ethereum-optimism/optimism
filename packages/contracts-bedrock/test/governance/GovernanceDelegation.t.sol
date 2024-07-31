// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import "forge-std/Test.sol";

import { CommonTest } from "test/setup/CommonTest.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";
import "src/governance/IGovernanceDelegation.sol";

contract GovernanceDelegation_Test is CommonTest {
    address owner;
    address rando;

    // Can't get events from GovernanceDelegation as it's using 0.8.25
    event DelegationsCreated(address indexed account, IGovernanceDelegation.Delegation[] delegations);
    event DelegateVotesChanged(address indexed delegate, uint256 previousBalance, uint256 newBalance);

    /// @dev Sets up the test suite.
    function setUp() public virtual override {
        super.enableInterop();
        super.setUp();
        owner = governanceToken.owner();
        rando = makeAddr("rando");
    }

    /// @dev Tests that the constructor sets the correct initial state.
    function test_constructor_succeeds() external view {
        assertEq(governanceToken.owner(), owner);
        assertEq(governanceToken.name(), "Optimism");
        assertEq(governanceToken.symbol(), "OP");
        assertEq(governanceToken.decimals(), 18);
        assertEq(governanceToken.totalSupply(), 0);
    }

    function assertEq(
        IGovernanceDelegation.Delegation[] memory a,
        IGovernanceDelegation.Delegation[] memory b
    )
        internal
        pure
    {
        assertEq(a.length, b.length, "length mismatch");
        for (uint256 i = 0; i < a.length; i++) {
            assertEq(a[i].delegatee, b[i].delegatee, "delegatee mismatch");
            assertEq(a[i].amount, b[i].amount, "amount mismatch");
            assertEq(uint8(a[i].allowanceType), uint8(b[i].allowanceType), "type mismatch");
        }
    }

    function assertCorrectVotes(
        IGovernanceDelegation.Delegation[] memory _delegations,
        uint256 _amount
    )
        internal
        view
    {
        IGovernanceDelegation.DelegationAdjustment[] memory _votes = calculateWeightDistribution(_delegations, _amount);
        uint256 _totalWeight = 0;
        for (uint256 i = 0; i < _delegations.length; i++) {
            uint256 _expectedVoteWeight = _delegations[i].delegatee == address(0) ? 0 : _votes[i].amount;
            assertEq(
                governanceToken.getVotes(_delegations[i].delegatee),
                _expectedVoteWeight,
                "incorrect vote weight for delegate"
            );
            _totalWeight += _votes[i].amount;
        }
        assertLe(_totalWeight, _amount, "incorrect total weight");
    }

    function assertCorrectPastVotes(
        IGovernanceDelegation.Delegation[] memory _delegations,
        uint256 _amount,
        uint256 _timepoint
    )
        internal
        view
    {
        IGovernanceDelegation.DelegationAdjustment[] memory _votes = calculateWeightDistribution(_delegations, _amount);
        uint256 _totalWeight = 0;
        for (uint256 i = 0; i < _delegations.length; i++) {
            uint256 _expectedVoteWeight = _votes[i].amount;
            assertEq(
                governanceToken.getPastVotes(_delegations[i].delegatee, _timepoint),
                _expectedVoteWeight,
                "incorrect past vote weight for delegate"
            );
            _totalWeight += _votes[i].amount;
        }
        assertLe(_totalWeight, _amount, "incorrect total weight");
    }

    /// @dev Copied from GovernanceDelegation
    function calculateWeightDistribution(
        IGovernanceDelegation.Delegation[] memory _delegations,
        uint256 _amount
    )
        internal
        view
        returns (IGovernanceDelegation.DelegationAdjustment[] memory)
    {
        uint256 _delegationsLength = _delegations.length;
        IGovernanceDelegation.DelegationAdjustment[] memory _delegationAdjustments =
            new IGovernanceDelegation.DelegationAdjustment[](_delegationsLength);

        // For relative delegations, keep track of total numerator to ensure it doesn't exceed DENOMINATOR
        uint256 _totalNumerator;

        // Iterate through partial delegations to calculate vote weight
        for (uint256 i; i < _delegationsLength; i++) {
            if (_delegations[i].allowanceType == IGovernanceDelegation.AllowanceType.Relative) {
                require(_delegations[i].amount != 0);

                _delegationAdjustments[i] = IGovernanceDelegation.DelegationAdjustment(
                    _delegations[i].delegatee,
                    uint208(_amount * _delegations[i].amount / governanceDelegation.DENOMINATOR())
                );
                _totalNumerator += _delegations[i].amount;

                require(_totalNumerator <= governanceDelegation.DENOMINATOR());
            } else {
                _delegationAdjustments[i] = IGovernanceDelegation.DelegationAdjustment(
                    _delegations[i].delegatee, uint208(_delegations[i].amount)
                );
            }
        }
        return _delegationAdjustments;
    }

    function _createSingleFullDelegation(address _delegatee)
        internal
        view
        returns (IGovernanceDelegation.Delegation[] memory)
    {
        IGovernanceDelegation.Delegation[] memory delegations = new IGovernanceDelegation.Delegation[](1);
        delegations[0] = IGovernanceDelegation.Delegation(
            IGovernanceDelegation.AllowanceType.Relative, _delegatee, governanceDelegation.DENOMINATOR()
        );
        return delegations;
    }

    function _expectEmitDelegateVotesChangedEvents(
        uint256 _amount,
        uint256 _toExistingBalance,
        IGovernanceDelegation.Delegation[] memory _fromDelegations,
        IGovernanceDelegation.Delegation[] memory _toDelegations
    )
        internal
    {
        IGovernanceDelegation.DelegationAdjustment[] memory _fromVotes =
            calculateWeightDistribution(_fromDelegations, _amount);
        IGovernanceDelegation.DelegationAdjustment[] memory _toInitialVotes =
            calculateWeightDistribution(_toDelegations, _toExistingBalance);
        IGovernanceDelegation.DelegationAdjustment[] memory _toVotes =
            calculateWeightDistribution(_toDelegations, _amount + _toExistingBalance);

        uint256 i;
        uint256 j;
        while (i < _fromDelegations.length || j < _toDelegations.length) {
            // If both delegations have the same delegatee
            if (
                i < _fromDelegations.length && j < _toDelegations.length
                    && _fromDelegations[i].delegatee == _toDelegations[j].delegatee
            ) {
                // if the numerator is different
                if (_fromDelegations[i].amount != _toDelegations[j].amount) {
                    if (_toVotes[j].amount != 0 || _fromVotes[j].amount != 0) {
                        vm.expectEmit();
                        emit DelegateVotesChanged(
                            _fromDelegations[i].delegatee, _fromVotes[j].amount, _toVotes[j].amount
                        );
                    }
                }
                i++;
                j++;
                // Old delegatee comes before the new delegatee OR new delegatees have been exhausted
            } else if (
                j == _toDelegations.length
                    || (i != _fromDelegations.length && _fromDelegations[i].delegatee < _toDelegations[j].delegatee)
            ) {
                if (_fromVotes[i].amount != 0) {
                    vm.expectEmit();
                    emit DelegateVotesChanged(_fromDelegations[i].delegatee, _fromVotes[i].amount, 0);
                }
                i++;
                // If new delegatee comes before the old delegatee OR old delegatees have been exhausted
            } else {
                // If the new delegatee vote weight is not the same as its previous vote weight
                if (_toVotes[j].amount != 0 && _toVotes[j].amount != _toInitialVotes[j].amount) {
                    vm.expectEmit();
                    emit DelegateVotesChanged(
                        _toDelegations[j].delegatee, _toInitialVotes[j].amount, _toVotes[j].amount
                    );
                }
                j++;
            }
        }
    }

    function testFuzz_delegates_single_non_zero_address(
        address _delegatee,
        uint96 _numerator,
        uint256 _amount
    )
        public
    {
        vm.assume(_delegatee != address(0));

        _numerator = uint96(bound(_numerator, 1, governanceDelegation.DENOMINATOR()));
        _amount = bound(_amount, 0, type(uint208).max);

        vm.prank(owner);
        governanceToken.mint(rando, _amount);

        IGovernanceDelegation.Delegation[] memory delegations = new IGovernanceDelegation.Delegation[](1);
        delegations[0] =
            IGovernanceDelegation.Delegation(IGovernanceDelegation.AllowanceType.Relative, _delegatee, _numerator);

        vm.prank(rando);
        governanceDelegation.delegate(delegations[0]);

        // assertEq(governanceDelegation.delegates(rando), delegations);
        // IGovernanceDelegation.DelegationAdjustment[] memory adjustments =
        //     calculateWeightDistribution(delegations, _amount);
        // assertEq(governanceDelegation.getVotes(_delegatee), adjustments[0].amount);
    }
}
