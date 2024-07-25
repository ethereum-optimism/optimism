// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing utilities
import { CommonTest } from "test/setup/CommonTest.sol";
import "src/libraries/Predeploys.sol";
import "src/governance/GovernanceDelegation.sol";
import "@openzeppelin/contracts/token/ERC20/extensions/ERC20Votes.sol";

contract GovernanceDelegation_Test is CommonTest {
    address owner;
    address rando;

    event DelegationCreated(address indexed account, Delegation delegation);
    event DelegateVotesChanged(address indexed delegate, uint256 previousBalance, uint256 newBalance);

    /// @dev Sets up the test suite.
    function setUp() public virtual override {
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

    function assertEq(Delegation[] memory a, Delegation[] memory b) public {
        assertEq(a.length, b.length, "length mismatch");
        for (uint256 i = 0; i < a.length; i++) {
            assertEq(a[i].delegatee, b[i].delegatee, "delegatee mismatch");
            assertEq(a[i].amount, b[i].amount, "amount mismatch");
            assertEq(uint8(a[i].allowanceType), uint8(b[i].allowanceType), "type mismatch");
        }
    }

    function assertCorrectVotes(Delegation[] memory _delegations, uint256 _amount) internal {
        DelegationAdjustment[] memory _votes = governanceDelegation.calculateWeightDistribution(_delegations, _amount);
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

    function assertCorrectPastVotes(Delegation[] memory _delegations, uint256 _amount, uint256 _timepoint) internal {
        DelegationAdjustment[] memory _votes = governanceDelegation.calculateWeightDistribution(_delegations, _amount);
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

    function _createSingleFullDelegation(address _delegatee) internal view returns (Delegation[] memory) {
        Delegation[] memory delegations = new Delegation[](1);
        delegations[0] = Delegation(AllowanceType.Relative, _delegatee, governanceDelegation.DENOMINATOR());
        return delegations;
    }

    function _expectEmitDelegateVotesChangedEvents(
        uint256 _amount,
        uint256 _toExistingBalance,
        Delegation[] memory _fromDelegations,
        Delegation[] memory _toDelegations
    )
        internal
    {
        DelegationAdjustment[] memory _fromVotes =
            governanceDelegation.calculateWeightDistribution(_fromDelegations, _amount);
        DelegationAdjustment[] memory _toInitialVotes =
            governanceDelegation.calculateWeightDistribution(_toDelegations, _toExistingBalance);
        DelegationAdjustment[] memory _toVotes =
            governanceDelegation.calculateWeightDistribution(_toDelegations, _amount + _toExistingBalance);

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

        Delegation[] memory delegations = new Delegation[](1);
        delegations[0] = Delegation(AllowanceType.Relative, _delegatee, _numerator);

        vm.prank(rando);
        governanceDelegation.delegate(delegations[0]);

        assertEq(governanceDelegation.delegates(rando), delegations);
        DelegationAdjustment[] memory adjustments =
            governanceDelegation.calculateWeightDistribution(delegations, _amount);
        assertEq(governanceDelegation.getVotes(_delegatee), adjustments[0].amount);
    }
}
