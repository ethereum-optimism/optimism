// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import "forge-std/Test.sol";

import { CommonTest } from "test/setup/CommonTest.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";
import { IGovernanceDelegation } from "src/governance/IGovernanceDelegation.sol";

contract GovernanceDelegation_Init is CommonTest {
    address owner;
    address rando;

    // Can't get events and errors from GovernanceDelegation as it's using 0.8.25
    event DelegationsCreated(address indexed account, IGovernanceDelegation.Delegation[] delegations);
    event DelegateVotesChanged(address indexed delegate, uint256 previousBalance, uint256 newBalance);

    error LimitExceeded(uint256 length, uint256 maxLength);
    error InvalidNumeratorZero();
    error NumeratorSumExceedsDenominator(uint256 numerator, uint96 denominator);
    error DuplicateOrUnsortedDelegatees(address delegatee);

    /// @dev Sets up the test suite.
    function setUp() public virtual override {
        super.enableInterop();
        super.setUp();
        owner = governanceToken.owner();
        rando = makeAddr("rando");
    }

    // HELPERS

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

    // TODO: combine into one function

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

    function _expectEmitDelegateVotesChangedEvents(
        uint256 _amount,
        IGovernanceDelegation.Delegation[] memory _fromDelegations,
        IGovernanceDelegation.Delegation[] memory _toDelegations
    )
        internal
    {
        IGovernanceDelegation.DelegationAdjustment[] memory _initialVotes =
            calculateWeightDistribution(_fromDelegations, _amount);
        IGovernanceDelegation.DelegationAdjustment[] memory _votes =
            calculateWeightDistribution(_toDelegations, _amount);

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
                    if (_votes[j].amount != 0 || _initialVotes[i].amount != 0) {
                        vm.expectEmit();
                        emit DelegateVotesChanged(
                            _fromDelegations[i].delegatee, _initialVotes[i].amount, _votes[j].amount
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
                if (_initialVotes[i].amount != 0) {
                    vm.expectEmit();
                    emit DelegateVotesChanged(_fromDelegations[i].delegatee, _initialVotes[i].amount, 0);
                }
                i++;
                // If new delegatee comes before the old delegatee OR old delegatees have been exhausted
            } else {
                if (_votes[j].amount != 0) {
                    vm.expectEmit();
                    emit DelegateVotesChanged(_toDelegations[j].delegatee, 0, _votes[j].amount);
                }
                j++;
            }
        }
    }

    function _createValidPartialDelegation(
        uint256 _n,
        uint256 _seed
    )
        internal
        view
        returns (IGovernanceDelegation.Delegation[] memory)
    {
        _seed = bound(
            _seed,
            1,
            /* private key can't be bigger than secp256k1 curve order */
            115_792_089_237_316_195_423_570_985_008_687_907_852_837_564_279_074_904_382_605_163_141_518_161_494_337 - 1
        );
        _n = _n != 0 ? _n : (_seed % governanceDelegation.MAX_DELEGATIONS()) + 1;
        IGovernanceDelegation.Delegation[] memory delegations = new IGovernanceDelegation.Delegation[](_n);
        uint96 _totalNumerator;
        for (uint256 i = 0; i < _n; i++) {
            uint96 _numerator = uint96(
                bound(
                    uint256(keccak256(abi.encode(_seed + i))) % governanceDelegation.DENOMINATOR(), // initial value of
                        // the numerator
                    1,
                    governanceDelegation.DENOMINATOR() - _totalNumerator - (_n - i) // ensure that there is enough
                        // numerator left for
                        // the
                        // remaining delegations
                )
            );
            delegations[i] = IGovernanceDelegation.Delegation(
                IGovernanceDelegation.AllowanceType.Relative, address(uint160(uint160(vm.addr(_seed)) + i)), _numerator
            );
            _totalNumerator += _numerator;
        }
        return delegations;
    }
}

contract GovernanceDelegation_Delegate_Test is GovernanceDelegation_Init {
    /// @dev Tests that the constructor sets the correct initial state.
    function test_constructor_succeeds() external view {
        assertEq(governanceToken.owner(), owner);
        assertEq(governanceToken.name(), "Optimism");
        assertEq(governanceToken.symbol(), "OP");
        assertEq(governanceToken.decimals(), 18);
        assertEq(governanceToken.totalSupply(), 0);
    }

    function testFuzz_delegate_singleAddress_succeeds(address _delegatee, uint96 _numerator, uint256 _amount) public {
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

        assertEq(governanceDelegation.delegations(rando), delegations);
        IGovernanceDelegation.DelegationAdjustment[] memory adjustments =
            calculateWeightDistribution(delegations, _amount);
        assertEq(governanceDelegation.getVotes(_delegatee), adjustments[0].amount);
    }

    function testFuzz_delegate_zeroAddress_succeeds(address _actor, uint96 _numerator, uint256 _amount) public {
        vm.assume(_actor != address(0));
        address _delegatee = address(0);
        _numerator = uint96(bound(_numerator, 1, governanceDelegation.DENOMINATOR()));
        _amount = bound(_amount, 0, type(uint208).max);

        vm.prank(owner);
        governanceToken.mint(_actor, _amount);

        IGovernanceDelegation.Delegation[] memory delegations = new IGovernanceDelegation.Delegation[](1);
        delegations[0] =
            IGovernanceDelegation.Delegation(IGovernanceDelegation.AllowanceType.Relative, _delegatee, _numerator);
        vm.prank(_actor);
        governanceDelegation.delegate(delegations[0]);

        assertEq(governanceDelegation.delegations(_actor), delegations);
        assertEq(governanceDelegation.getVotes(_delegatee), 0);
    }

    function testFuzz_delegateBatched_multipleAddresses_succeeds(
        address _actor,
        uint256 _amount,
        uint256 _n,
        uint256 _seed
    )
        public
    {
        vm.assume(_actor != address(0));
        _amount = bound(_amount, 0, type(uint208).max);
        _n = bound(_n, 1, governanceDelegation.MAX_DELEGATIONS());
        IGovernanceDelegation.Delegation[] memory delegations = _createValidPartialDelegation(_n, _seed);

        vm.prank(owner);
        governanceToken.mint(_actor, _amount);

        vm.prank(_actor);
        governanceDelegation.delegateBatched(delegations);

        assertEq(governanceDelegation.delegations(_actor), delegations);
        assertCorrectVotes(delegations, _amount);
    }

    function testFuzz_delegateBatched_multipleAddressesDouble_succeeds(
        address _actor,
        uint256 _amount,
        uint256 _n,
        uint256 _seed
    )
        public
    {
        vm.assume(_actor != address(0));
        _amount = bound(_amount, 0, type(uint208).max);
        _n = bound(_n, 1, governanceDelegation.MAX_DELEGATIONS());
        IGovernanceDelegation.Delegation[] memory delegations = _createValidPartialDelegation(_n, _seed);

        vm.prank(owner);
        governanceToken.mint(_actor, _amount);

        vm.prank(_actor);
        governanceDelegation.delegateBatched(delegations);

        assertEq(governanceDelegation.delegations(_actor), delegations);
        IGovernanceDelegation.Delegation[] memory newDelegations =
            _createValidPartialDelegation(0, uint256(keccak256(abi.encode(_seed))));

        vm.prank(_actor);
        governanceDelegation.delegateBatched(newDelegations);

        assertEq(governanceDelegation.delegations(_actor), newDelegations);
        assertCorrectVotes(newDelegations, _amount);
        // initial delegates should have 0 vote power (assuming set union is empty)
        for (uint256 i = 0; i < delegations.length; i++) {
            assertEq(governanceDelegation.getVotes(delegations[i].delegatee), 0, "initial delegate has vote power");
        }
    }

    function testFuzz_EmitsDelegateChangedEvents(address _actor, uint256 _amount, uint256 _n, uint256 _seed) public {
        vm.assume(_actor != address(0));
        _amount = bound(_amount, 0, type(uint208).max);
        _n = bound(_n, 1, governanceDelegation.MAX_DELEGATIONS());
        IGovernanceDelegation.Delegation[] memory delegations = _createValidPartialDelegation(_n, _seed);

        vm.prank(owner);
        governanceToken.mint(_actor, _amount);

        vm.expectEmit();
        emit DelegationsCreated(_actor, delegations);
        vm.prank(_actor);
        governanceDelegation.delegateBatched(delegations);
    }

    function testFuzz_EmitsDelegateVotesChanged(address _actor, uint256 _amount, uint256 _n, uint256 _seed) public {
        vm.assume(_actor != address(0));
        _amount = bound(_amount, 0, type(uint208).max);
        _n = bound(_n, 1, governanceDelegation.MAX_DELEGATIONS());
        IGovernanceDelegation.Delegation[] memory delegations = _createValidPartialDelegation(_n, _seed);

        vm.prank(owner);
        governanceToken.mint(_actor, _amount);

        _expectEmitDelegateVotesChangedEvents(_amount, new IGovernanceDelegation.Delegation[](0), delegations);
        vm.prank(_actor);
        governanceDelegation.delegateBatched(delegations);
    }

    function testFuzz_EmitsDelegateChangedEventsWhenDelegateesAreRemoved(
        address _actor,
        uint256 _amount,
        uint256 _oldN,
        uint256 _numOfDelegateesToRemove,
        uint256 _seed
    )
        public
    {
        vm.assume(_actor != address(0));
        _amount = bound(_amount, 0, type(uint208).max);
        _oldN = bound(_oldN, 1, governanceDelegation.MAX_DELEGATIONS());
        _numOfDelegateesToRemove = bound(_numOfDelegateesToRemove, 0, _oldN - 1);
        IGovernanceDelegation.Delegation[] memory oldDelegations = _createValidPartialDelegation(_oldN, _seed);

        vm.prank(owner);
        governanceToken.mint(_actor, _amount);

        vm.prank(_actor);
        governanceDelegation.delegateBatched(oldDelegations);

        IGovernanceDelegation.Delegation[] memory newDelegations =
            new IGovernanceDelegation.Delegation[](_oldN - _numOfDelegateesToRemove);
        for (uint256 i; i < newDelegations.length; i++) {
            newDelegations[i] = oldDelegations[i];
        }

        vm.expectEmit();
        emit DelegationsCreated(_actor, newDelegations);
        vm.prank(_actor);
        governanceDelegation.delegateBatched(newDelegations);
    }

    function testFuzz_EmitsDelegateChangedEventsWhenAllNumeratorsForCurrentDelegateesAreChanged(
        address _actor,
        uint256 _amount,
        uint256 _oldN,
        uint256 _newN,
        uint256 _seed
    )
        public
    {
        vm.assume(_actor != address(0));
        _amount = bound(_amount, 0, type(uint208).max);
        _oldN = bound(_oldN, 1, governanceDelegation.MAX_DELEGATIONS());
        _newN = bound(_newN, 1, governanceDelegation.MAX_DELEGATIONS());
        IGovernanceDelegation.Delegation[] memory oldDelegations = _createValidPartialDelegation(_oldN, _seed);

        vm.prank(owner);
        governanceToken.mint(_actor, _amount);

        vm.prank(_actor);
        governanceDelegation.delegateBatched(oldDelegations);
        IGovernanceDelegation.Delegation[] memory newDelegations = oldDelegations;

        // Arthimatic overflow/underflow error without this bounding.
        _seed = bound(
            _seed,
            1,
            /* private key can't be bigger than secp256k1 curve order */
            115_792_089_237_316_195_423_570_985_008_687_907_852_837_564_279_074_904_382_605_163_141_518_161_494_337 - 1
        );
        uint96 _totalNumerator;
        for (uint256 i = 0; i < _oldN; i++) {
            uint96 _numerator = uint96(
                bound(
                    uint256(keccak256(abi.encode(_seed + i))) % governanceDelegation.DENOMINATOR(), // initial value of
                        // the
                        // numerator
                    1,
                    governanceDelegation.DENOMINATOR() - _totalNumerator - (_oldN - i) // ensure that there is enough
                        // numerator
                        // left for the
                        // remaining delegations
                )
            );
            newDelegations[i].amount = _numerator;
            _totalNumerator += _numerator;
        }

        vm.expectEmit();
        emit DelegationsCreated(_actor, newDelegations);
        vm.prank(_actor);
        governanceDelegation.delegateBatched(newDelegations);
        vm.stopPrank();
    }

    function testFuzz_EmitsDelegateChangedEventsWhenAllDelegatesAreReplaced(
        address _actor,
        uint256 _amount,
        uint256 _oldN,
        uint256 _newN,
        uint256 _seed
    )
        public
    {
        vm.assume(_actor != address(0));
        _amount = bound(_amount, 0, type(uint208).max);
        _oldN = bound(_oldN, 1, governanceDelegation.MAX_DELEGATIONS());
        _newN = bound(_newN, 1, governanceDelegation.MAX_DELEGATIONS());
        IGovernanceDelegation.Delegation[] memory oldDelegations = _createValidPartialDelegation(_oldN, _seed);

        vm.prank(owner);
        governanceToken.mint(_actor, _amount);

        vm.prank(_actor);
        governanceDelegation.delegateBatched(oldDelegations);

        IGovernanceDelegation.Delegation[] memory newDelegations =
            _createValidPartialDelegation(_newN, uint256(keccak256(abi.encode(_seed))));

        vm.expectEmit();
        emit DelegationsCreated(_actor, newDelegations);
        vm.prank(_actor);
        governanceDelegation.delegateBatched(newDelegations);
    }

    function testFuzz_EmitsDelegateVotesChangedEventsWhenAllNumeratorsForCurrentDelegateesAreChanged(
        address _actor,
        uint256 _amount,
        uint256 _oldN,
        uint256 _newN,
        uint256 _seed
    )
        public
    {
        vm.assume(_actor != address(0));
        _amount = bound(_amount, 0, type(uint208).max);
        _oldN = bound(_oldN, 1, governanceDelegation.MAX_DELEGATIONS());
        _newN = bound(_newN, 1, governanceDelegation.MAX_DELEGATIONS());
        IGovernanceDelegation.Delegation[] memory oldDelegations = _createValidPartialDelegation(_oldN, _seed);

        vm.prank(owner);
        governanceToken.mint(_actor, _amount);

        vm.prank(_actor);
        governanceDelegation.delegateBatched(oldDelegations);

        IGovernanceDelegation.Delegation[] memory newDelegations = oldDelegations;

        _seed = bound(
            _seed,
            1,
            /* private key can't be bigger than secp256k1 curve order */
            115_792_089_237_316_195_423_570_985_008_687_907_852_837_564_279_074_904_382_605_163_141_518_161_494_337 - 1
        );
        uint96 _totalNumerator;
        for (uint256 i = 0; i < _oldN; i++) {
            uint96 _numerator = uint96(
                bound(
                    uint256(keccak256(abi.encode(_seed + i))) % governanceDelegation.DENOMINATOR(), // initial value of
                        // the
                        // numerator
                    1,
                    governanceDelegation.DENOMINATOR() - _totalNumerator - (_oldN - i) // ensure that there is enough
                        // numerator
                        // left for the
                        // remaining delegations
                )
            );
            newDelegations[i].amount = _numerator;
            _totalNumerator += _numerator;
        }

        _expectEmitDelegateVotesChangedEvents(_amount, oldDelegations, newDelegations);
        vm.prank(_actor);
        governanceDelegation.delegateBatched(newDelegations);
    }

    function testFuzz_EmitsDelegateVotesChangedEventsWhenDelegateesAreRemoved(
        address _actor,
        uint256 _amount,
        uint256 _oldN,
        uint256 _numOfDelegateesToRemove,
        uint256 _seed
    )
        public
    {
        vm.assume(_actor != address(0));
        _amount = bound(_amount, 0, type(uint208).max);
        _oldN = bound(_oldN, 1, governanceDelegation.MAX_DELEGATIONS());
        _numOfDelegateesToRemove = bound(_numOfDelegateesToRemove, 0, _oldN - 1);
        IGovernanceDelegation.Delegation[] memory oldDelegations = _createValidPartialDelegation(_oldN, _seed);

        vm.prank(owner);
        governanceToken.mint(_actor, _amount);

        vm.prank(_actor);
        governanceDelegation.delegateBatched(oldDelegations);

        IGovernanceDelegation.Delegation[] memory newDelegations =
            new IGovernanceDelegation.Delegation[](_oldN - _numOfDelegateesToRemove);
        for (uint256 i; i < newDelegations.length; i++) {
            newDelegations[i] = oldDelegations[i];
        }

        _expectEmitDelegateVotesChangedEvents(_amount, oldDelegations, newDelegations);
        vm.prank(_actor);
        governanceDelegation.delegateBatched(newDelegations);
    }

    function testFuzz_EmitsDelegateVotesChangedEventsWhenAllDelegatesAreReplaced(
        address _actor,
        uint256 _amount,
        uint256 _oldN,
        uint256 _newN,
        uint256 _seed
    )
        public
    {
        vm.assume(_actor != address(0));
        _amount = bound(_amount, 0, type(uint208).max);
        _oldN = bound(_oldN, 1, governanceDelegation.MAX_DELEGATIONS());
        _newN = bound(_newN, 1, governanceDelegation.MAX_DELEGATIONS());

        IGovernanceDelegation.Delegation[] memory oldDelegations = _createValidPartialDelegation(_oldN, _seed);

        vm.prank(owner);
        governanceToken.mint(_actor, _amount);

        vm.prank(_actor);
        governanceDelegation.delegateBatched(oldDelegations);

        IGovernanceDelegation.Delegation[] memory newDelegations =
            _createValidPartialDelegation(_newN, uint256(keccak256(abi.encode(_seed))));

        _expectEmitDelegateVotesChangedEvents(_amount, oldDelegations, newDelegations);
        vm.prank(_actor);
        governanceDelegation.delegateBatched(newDelegations);
    }
}

contract GovernanceDelegation_Delegate_TestFail is GovernanceDelegation_Init {
    function testFuzz_delegateBatched_duplicates_reverts(
        address _actor,
        address _delegatee,
        uint256 _amount,
        uint96 _numerator
    )
        public
    {
        vm.assume(_actor != address(0));
        vm.assume(_delegatee != address(0));
        _amount = bound(_amount, 0, type(uint208).max);
        _numerator = uint96(bound(_numerator, 1, governanceDelegation.DENOMINATOR() - 1));

        vm.prank(owner);
        governanceToken.mint(_actor, _amount);

        IGovernanceDelegation.Delegation[] memory delegations = new IGovernanceDelegation.Delegation[](2);
        delegations[0] =
            IGovernanceDelegation.Delegation(IGovernanceDelegation.AllowanceType.Relative, _delegatee, _numerator);
        delegations[1] = IGovernanceDelegation.Delegation(
            IGovernanceDelegation.AllowanceType.Relative, _delegatee, governanceDelegation.DENOMINATOR() - _numerator
        );

        vm.expectRevert();
        vm.prank(_actor);
        governanceDelegation.delegateBatched(delegations);
    }

    function testFuzz_delegate_numeratorExceed_reverts(
        address _actor,
        address _delegatee,
        uint256 _amount,
        uint96 _numerator
    )
        public
    {
        vm.assume(_actor != address(0));
        vm.assume(_delegatee != address(0));
        _numerator = uint96(bound(_numerator, governanceDelegation.DENOMINATOR() + 1, type(uint96).max));
        _amount = bound(_amount, 0, type(uint208).max);

        vm.prank(owner);
        governanceToken.mint(_actor, _amount);

        vm.expectRevert();
        governanceDelegation.delegate(
            IGovernanceDelegation.Delegation(IGovernanceDelegation.AllowanceType.Relative, _delegatee, _numerator)
        );
        vm.stopPrank();
    }

    function testFuzz_delegateBatched_multipleNumeratorExceed_reverts(
        address _actor,
        uint256 _amount,
        uint256 _delegationIndex,
        uint256 _seed
    )
        public
    {
        vm.assume(_actor != address(0));
        _amount = bound(_amount, 0, type(uint208).max);
        IGovernanceDelegation.Delegation[] memory delegations = _createValidPartialDelegation(0, _seed);
        _delegationIndex = bound(_delegationIndex, 0, delegations.length - 1);

        delegations[_delegationIndex].amount = governanceDelegation.DENOMINATOR() + 1;
        uint256 sumOfNumerators;
        for (uint256 i; i < delegations.length; i++) {
            sumOfNumerators += delegations[i].amount;
            if (sumOfNumerators > governanceDelegation.DENOMINATOR()) {
                break;
            }
        }

        vm.prank(owner);
        governanceToken.mint(_actor, _amount);

        vm.expectRevert(
            abi.encodeWithSelector(
                NumeratorSumExceedsDenominator.selector, sumOfNumerators, governanceDelegation.DENOMINATOR()
            )
        );
        vm.prank(_actor);
        governanceDelegation.delegateBatched(delegations);
    }

    function testFuzz_delegateBatched_limitExceed_reverts(
        address _actor,
        uint256 _amount,
        uint256 _numOfDelegatees,
        uint256 _seed
    )
        public
    {
        vm.assume(_actor != address(0));
        _amount = bound(_amount, 0, type(uint208).max);
        _numOfDelegatees = bound(
            _numOfDelegatees, governanceDelegation.MAX_DELEGATIONS() + 1, governanceDelegation.MAX_DELEGATIONS() + 500
        );
        IGovernanceDelegation.Delegation[] memory delegations = _createValidPartialDelegation(_numOfDelegatees, _seed);

        vm.prank(owner);
        governanceToken.mint(_actor, _amount);

        vm.expectRevert(
            abi.encodeWithSelector(LimitExceeded.selector, _numOfDelegatees, governanceDelegation.MAX_DELEGATIONS())
        );
        vm.prank(_actor);
        governanceDelegation.delegateBatched(delegations);
    }

    function testFuzz_delegateBatched_unsorted_reverts(
        address _actor,
        uint256 _amount,
        uint256 _numOfDelegatees,
        address _replacedDelegatee,
        uint256 _seed
    )
        public
    {
        vm.assume(_actor != address(0));
        _amount = bound(_amount, 0, type(uint208).max);
        _numOfDelegatees = bound(_numOfDelegatees, 2, governanceDelegation.MAX_DELEGATIONS());
        IGovernanceDelegation.Delegation[] memory delegations = _createValidPartialDelegation(_numOfDelegatees, _seed);
        address lastDelegatee = delegations[delegations.length - 1].delegatee;
        vm.assume(_replacedDelegatee <= lastDelegatee);
        delegations[delegations.length - 1].delegatee = _replacedDelegatee;

        vm.prank(owner);
        governanceToken.mint(_actor, _amount);

        vm.expectRevert(abi.encodeWithSelector(DuplicateOrUnsortedDelegatees.selector, _replacedDelegatee));
        vm.prank(_actor);
        governanceDelegation.delegateBatched(delegations);
    }

    function testFuzz_delegate_zeroNumerator_reverts(
        address _actor,
        uint256 _amount,
        uint256 _delegationIndex,
        uint256 _seed
    )
        public
    {
        vm.assume(_actor != address(0));
        _amount = bound(_amount, 0, type(uint208).max);
        IGovernanceDelegation.Delegation[] memory delegations = _createValidPartialDelegation(0, _seed);
        _delegationIndex = bound(_delegationIndex, 0, delegations.length - 1);

        delegations[_delegationIndex].amount = 0;

        vm.prank(owner);
        governanceToken.mint(_actor, _amount);

        vm.expectRevert(InvalidNumeratorZero.selector);
        vm.prank(_actor);
        governanceDelegation.delegateBatched(delegations);

        vm.expectRevert(InvalidNumeratorZero.selector);
        vm.prank(_actor);
        governanceDelegation.delegate(delegations[_delegationIndex]);
    }
}
