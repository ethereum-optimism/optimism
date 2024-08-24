// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import "forge-std/Test.sol";
import { StdCheats } from "forge-std/StdCheats.sol";

import { ERC20Votes } from "@openzeppelin/contracts/token/ERC20/extensions/ERC20Votes.sol";

import { CommonTest } from "test/setup/CommonTest.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";
import { IGovernanceDelegation } from "src/governance/IGovernanceDelegation.sol";
import { IGovernanceTokenInterop } from "src/governance/IGovernanceTokenInterop.sol";

contract GovernanceDelegation_Init is CommonTest {
    address owner;
    address rando;

    event DelegationsChanged(
        address indexed account,
        IGovernanceDelegation.Delegation[] oldDelegations,
        IGovernanceDelegation.Delegation[] newDelegations
    );
    event DelegateVotesChanged(address indexed delegate, uint256 previousBalance, uint256 newBalance);

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
                governanceDelegation.getVotes(_delegations[i].delegatee),
                _expectedVoteWeight,
                "incorrect vote weight for delegate"
            );
            _totalWeight += _votes[i].amount;
        }
        assertLe(_totalWeight, _amount, "incorrect total weight");
    }

    /// @dev Copied from GovernanceDelegation
    function calculateWeightDistribution(
        IGovernanceDelegation.Delegation[] memory _delegationSet,
        uint256 _balance
    )
        internal
        view
        returns (IGovernanceDelegation.DelegationAdjustment[] memory)
    {
        uint256 _delegationsLength = _delegationSet.length;
        IGovernanceDelegation.DelegationAdjustment[] memory _delegationAdjustments =
            new IGovernanceDelegation.DelegationAdjustment[](_delegationsLength);

        // For relative delegations, keep track of total numerator to ensure it doesn't exceed DENOMINATOR
        uint256 _total;
        IGovernanceDelegation.AllowanceType _type;

        // Iterate through partial delegations to calculate delegation adjustments.
        for (uint256 i; i < _delegationsLength; i++) {
            address delegatee = _delegationSet[i].delegatee;
            uint256 amount = _delegationSet[i].amount;

            if (i > 0 && _delegationSet[i].allowanceType != _type) revert IGovernanceDelegation.InconsistentType();

            if (_delegationSet[i].allowanceType == IGovernanceDelegation.AllowanceType.Relative) {
                if (amount == 0) revert IGovernanceDelegation.InvalidAmountZero();
                _delegationAdjustments[i] = IGovernanceDelegation.DelegationAdjustment(
                    delegatee, (_balance * amount) / governanceDelegation.DENOMINATOR()
                );
                _total += amount;
                if (_total > governanceDelegation.DENOMINATOR()) {
                    revert IGovernanceDelegation.NumeratorSumExceedsDenominator(
                        _total, governanceDelegation.DENOMINATOR()
                    );
                }
            } else {
                amount = _balance < amount ? _balance : amount;
                _delegationAdjustments[i] = IGovernanceDelegation.DelegationAdjustment(delegatee, amount);
                _balance -= amount;
                if (_balance == 0) break;
            }

            _type = _delegationSet[i].allowanceType;
        }
        return _delegationAdjustments;
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
                        vm.expectEmit(address(governanceDelegation));
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
                    vm.expectEmit(address(governanceDelegation));
                    emit DelegateVotesChanged(_fromDelegations[i].delegatee, _initialVotes[i].amount, 0);
                }
                i++;
                // If new delegatee comes before the old delegatee OR old delegatees have been exhausted
            } else {
                if (_votes[j].amount != 0) {
                    vm.expectEmit(address(governanceDelegation));
                    emit DelegateVotesChanged(_toDelegations[j].delegatee, 0, _votes[j].amount);
                }
                j++;
            }
        }
    }

    function _createValidPartialDelegation(
        uint256 _n,
        uint256 _seed,
        uint256 _max,
        bool relative
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
        uint256 _total;
        for (uint256 i = 0; i < _n; i++) {
            uint256 _value;
            if (relative) {
                _value = uint256(
                    bound(
                        uint256(keccak256(abi.encode(_seed + i))) % governanceDelegation.DENOMINATOR(),
                        1,
                        governanceDelegation.DENOMINATOR() - _total - (_n - i)
                    )
                );
                delegations[i] = IGovernanceDelegation.Delegation(
                    IGovernanceDelegation.AllowanceType.Relative, address(uint160(uint160(vm.addr(_seed)) + i)), _value
                );
            } else {
                if (_max == 0) {
                    delegations[i] = IGovernanceDelegation.Delegation(
                        IGovernanceDelegation.AllowanceType.Absolute, address(uint160(uint160(vm.addr(_seed)) + i)), 0
                    );
                } else {
                    _value = uint256(bound(uint256(keccak256(abi.encode(_seed + i))) % type(uint224).max, 0, _max));
                    delegations[i] = IGovernanceDelegation.Delegation(
                        IGovernanceDelegation.AllowanceType.Absolute,
                        address(uint160(uint160(vm.addr(_seed)) + i)),
                        _value
                    );
                    _max -= _value;
                }
            }

            _total += _value;
        }
        return delegations;
    }

    function _migrated(address _account) internal view returns (bool) {
        return governanceToken.delegates(_account) == address(0);
    }
}

contract GovernanceDelegation_Delegate_Test is GovernanceDelegation_Init {
    function testFuzz_delegate_singleAddress_succeeds(
        address _delegatee,
        uint256 _value,
        uint256 _amount,
        bool relative
    )
        public
    {
        vm.assume(_delegatee != address(0));

        if (relative) {
            _value = bound(_value, 1, governanceDelegation.DENOMINATOR());
        } else {
            _value = bound(_value, 1, type(uint224).max);
        }
        _amount = bound(_amount, 1, type(uint224).max);

        vm.prank(owner);
        governanceToken.mint(rando, _amount);

        IGovernanceDelegation.Delegation[] memory delegations = new IGovernanceDelegation.Delegation[](1);
        if (relative) {
            delegations[0] =
                IGovernanceDelegation.Delegation(IGovernanceDelegation.AllowanceType.Relative, _delegatee, _value);
        } else {
            delegations[0] =
                IGovernanceDelegation.Delegation(IGovernanceDelegation.AllowanceType.Absolute, _delegatee, _value);
        }

        _expectEmitDelegateVotesChangedEvents(_amount, new IGovernanceDelegation.Delegation[](0), delegations);
        vm.expectEmit(address(governanceDelegation));
        emit DelegationsChanged(rando, new IGovernanceDelegation.Delegation[](0), delegations);
        vm.prank(rando);
        governanceDelegation.delegate(delegations[0]);

        assertEq(governanceDelegation.delegations(rando), delegations);
        IGovernanceDelegation.DelegationAdjustment[] memory adjustments =
            calculateWeightDistribution(delegations, _amount);
        assertEq(governanceDelegation.getVotes(_delegatee), adjustments[0].amount);
    }

    function testFuzz_delegate_zeroAddress_succeeds(
        address _actor,
        uint256 _value,
        uint256 _amount,
        bool relative
    )
        public
    {
        vm.assume(_actor != address(0));
        address _delegatee = address(0);

        if (relative) {
            _value = bound(_value, 1, governanceDelegation.DENOMINATOR());
        } else {
            _value = bound(_value, 1, type(uint224).max);
        }
        _amount = bound(_amount, 0, type(uint224).max);

        vm.prank(owner);
        governanceToken.mint(_actor, _amount);

        IGovernanceDelegation.Delegation[] memory delegations = new IGovernanceDelegation.Delegation[](1);
        if (relative) {
            delegations[0] =
                IGovernanceDelegation.Delegation(IGovernanceDelegation.AllowanceType.Relative, _delegatee, _value);
        } else {
            delegations[0] =
                IGovernanceDelegation.Delegation(IGovernanceDelegation.AllowanceType.Absolute, _delegatee, _value);
        }

        vm.expectEmit(address(governanceDelegation));
        emit DelegationsChanged(_actor, new IGovernanceDelegation.Delegation[](0), delegations);
        vm.prank(_actor);
        governanceDelegation.delegate(delegations[0]);

        assertEq(governanceDelegation.delegations(_actor), delegations);
        assertEq(governanceDelegation.getVotes(_delegatee), 0);
    }

    function testFuzz_delegateBatched_multipleAddresses_succeeds(
        address _actor,
        uint256 _amount,
        uint256 _n,
        uint256 _seed,
        bool relative
    )
        public
    {
        vm.assume(_actor != address(0));
        _amount = bound(_amount, 0, type(uint224).max);
        _n = bound(_n, 1, governanceDelegation.MAX_DELEGATIONS());
        IGovernanceDelegation.Delegation[] memory delegations =
            _createValidPartialDelegation(_n, _seed, _amount, relative);

        vm.prank(owner);
        governanceToken.mint(_actor, _amount);

        _expectEmitDelegateVotesChangedEvents(_amount, new IGovernanceDelegation.Delegation[](0), delegations);
        vm.expectEmit(address(governanceDelegation));
        emit DelegationsChanged(_actor, new IGovernanceDelegation.Delegation[](0), delegations);
        vm.prank(_actor);
        governanceDelegation.delegateBatched(delegations);

        assertEq(governanceDelegation.delegations(_actor), delegations);
        assertCorrectVotes(delegations, _amount);
    }

    function testFuzz_delegateBatched_multipleAddressesDouble_succeeds(
        address _actor,
        uint256 _amount,
        uint256 _n,
        uint256 _seed,
        bool relative
    )
        public
    {
        vm.assume(_actor != address(0));
        _amount = bound(_amount, 0, type(uint224).max);
        _n = bound(_n, 1, governanceDelegation.MAX_DELEGATIONS());
        IGovernanceDelegation.Delegation[] memory delegations =
            _createValidPartialDelegation(_n, _seed, _amount, relative);

        vm.prank(owner);
        governanceToken.mint(_actor, _amount);

        _expectEmitDelegateVotesChangedEvents(_amount, new IGovernanceDelegation.Delegation[](0), delegations);
        vm.expectEmit(address(governanceDelegation));
        emit DelegationsChanged(_actor, new IGovernanceDelegation.Delegation[](0), delegations);
        vm.prank(_actor);
        governanceDelegation.delegateBatched(delegations);

        assertEq(governanceDelegation.delegations(_actor), delegations);
        IGovernanceDelegation.Delegation[] memory newDelegations =
            _createValidPartialDelegation(0, uint256(keccak256(abi.encode(_seed))), _amount, relative);

        vm.expectEmit(address(governanceDelegation));
        emit DelegationsChanged(_actor, delegations, newDelegations);
        vm.prank(_actor);
        governanceDelegation.delegateBatched(newDelegations);

        assertEq(governanceDelegation.delegations(_actor), newDelegations);
        assertCorrectVotes(newDelegations, _amount);
        // initial delegates should have 0 vote power (assuming set union is empty)
        for (uint256 i = 0; i < delegations.length; i++) {
            assertEq(governanceDelegation.getVotes(delegations[i].delegatee), 0, "initial delegate has vote power");
        }
    }

    function testFuzz_delegateFromToken_succeeds(address _delegatee, uint256 _amount) public {
        vm.assume(_delegatee != address(0));

        _amount = bound(_amount, 0, type(uint224).max);

        vm.prank(owner);
        governanceToken.mint(rando, _amount);

        IGovernanceDelegation.Delegation[] memory delegations = new IGovernanceDelegation.Delegation[](1);
        delegations[0] = IGovernanceDelegation.Delegation(
            IGovernanceDelegation.AllowanceType.Relative, _delegatee, governanceDelegation.DENOMINATOR()
        );

        if (_amount != 0) {
            vm.expectEmit(address(governanceDelegation));
            emit DelegateVotesChanged(_delegatee, 0, _amount);
        }

        vm.expectEmit(address(governanceDelegation));
        emit DelegationsChanged(rando, new IGovernanceDelegation.Delegation[](0), delegations);

        vm.prank(address(governanceToken));
        governanceDelegation.delegateFromToken(rando, _delegatee);

        assertEq(governanceDelegation.delegations(rando), delegations);
        IGovernanceDelegation.DelegationAdjustment[] memory adjustments =
            calculateWeightDistribution(delegations, _amount);
        assertEq(governanceDelegation.getVotes(_delegatee), adjustments[0].amount);
    }

    function testFuzz_afterTokenTransfer_succeeds(address _from, address _to, uint256 _amount) public {
        vm.assume(_from != address(0));
        vm.assume(_to != address(0));
        vm.assume(_from != _to);

        _amount = bound(_amount, 0, type(uint224).max);

        vm.prank(_from);
        governanceToken.delegate(_from);

        vm.prank(_to);
        governanceToken.delegate(_to);

        vm.prank(owner);
        emit DelegateVotesChanged(_from, 0, _amount);
        governanceToken.mint(_from, _amount);

        assertEq(governanceDelegation.getVotes(_from), _amount);
        assertEq(governanceDelegation.getVotes(_to), 0);

        // simulate transfer from `_from` to `_to`
        vm.prank(_from);
        emit DelegateVotesChanged(_from, _amount, 0);
        emit DelegateVotesChanged(_to, 0, _amount);
        governanceToken.transfer(_to, _amount);

        assertEq(governanceDelegation.getVotes(_from), 0);
        assertEq(governanceDelegation.getVotes(_to), _amount);
    }
}

contract GovernanceDelegation_Delegate_TestFail is GovernanceDelegation_Init {
    function testFuzz_delegateBatched_duplicatesRelative_reverts(
        address _actor,
        address _delegatee,
        uint256 _amount,
        uint96 _numerator
    )
        public
    {
        vm.assume(_actor != address(0));
        vm.assume(_delegatee != address(0));
        _amount = bound(_amount, 0, type(uint224).max);
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
        _amount = bound(_amount, 0, type(uint224).max);

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
        _amount = bound(_amount, 0, type(uint224).max);
        IGovernanceDelegation.Delegation[] memory delegations = _createValidPartialDelegation(0, _seed, _amount, true);
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
                IGovernanceDelegation.NumeratorSumExceedsDenominator.selector,
                sumOfNumerators,
                governanceDelegation.DENOMINATOR()
            )
        );
        vm.prank(_actor);
        governanceDelegation.delegateBatched(delegations);
    }

    function testFuzz_delegateBatched_limitExceedRelative_reverts(
        address _actor,
        uint256 _amount,
        uint256 _numOfDelegatees,
        uint256 _seed,
        bool relative
    )
        public
    {
        vm.assume(_actor != address(0));
        _amount = bound(_amount, 0, type(uint224).max);
        _numOfDelegatees = bound(
            _numOfDelegatees, governanceDelegation.MAX_DELEGATIONS() + 1, governanceDelegation.MAX_DELEGATIONS() + 500
        );
        IGovernanceDelegation.Delegation[] memory delegations =
            _createValidPartialDelegation(_numOfDelegatees, _seed, _amount, relative);

        vm.prank(owner);
        governanceToken.mint(_actor, _amount);

        vm.expectRevert(
            abi.encodeWithSelector(
                IGovernanceDelegation.LimitExceeded.selector, _numOfDelegatees, governanceDelegation.MAX_DELEGATIONS()
            )
        );
        vm.prank(_actor);
        governanceDelegation.delegateBatched(delegations);
    }

    function testFuzz_delegateBatched_unsortedRelative_reverts(
        address _actor,
        uint256 _amount,
        uint256 _numOfDelegatees,
        address _replacedDelegatee,
        uint256 _seed,
        bool relative
    )
        public
    {
        vm.assume(_actor != address(0));
        _amount = bound(_amount, 0, type(uint224).max);
        _numOfDelegatees = bound(_numOfDelegatees, 2, governanceDelegation.MAX_DELEGATIONS());
        IGovernanceDelegation.Delegation[] memory delegations =
            _createValidPartialDelegation(_numOfDelegatees, _seed, _amount, relative);
        address lastDelegatee = delegations[delegations.length - 1].delegatee;
        vm.assume(_replacedDelegatee <= lastDelegatee);
        delegations[delegations.length - 1].delegatee = _replacedDelegatee;

        vm.prank(owner);
        governanceToken.mint(_actor, _amount);

        vm.expectRevert(
            abi.encodeWithSelector(IGovernanceDelegation.DuplicateOrUnsortedDelegatees.selector, _replacedDelegatee)
        );
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
        _amount = bound(_amount, 0, type(uint224).max);
        IGovernanceDelegation.Delegation[] memory delegations = _createValidPartialDelegation(0, _seed, _amount, true);
        _delegationIndex = bound(_delegationIndex, 0, delegations.length - 1);

        delegations[_delegationIndex].amount = 0;

        vm.prank(owner);
        governanceToken.mint(_actor, _amount);

        vm.expectRevert(IGovernanceDelegation.InvalidAmountZero.selector);
        vm.prank(_actor);
        governanceDelegation.delegateBatched(delegations);

        vm.expectRevert(IGovernanceDelegation.InvalidAmountZero.selector);
        vm.prank(_actor);
        governanceDelegation.delegate(delegations[_delegationIndex]);
    }

    function testFuzz_delegateFromToken_onlyTokenReletive_reverts(address _delegatee, uint256 _amount) public {
        vm.assume(_delegatee != address(0));

        _amount = bound(_amount, 0, type(uint224).max);

        vm.prank(owner);
        governanceToken.mint(rando, _amount);

        vm.expectRevert(IGovernanceDelegation.NotGovernanceToken.selector);
        governanceDelegation.delegateFromToken(rando, _delegatee);

        assertEq(governanceDelegation.delegations(rando), new IGovernanceDelegation.Delegation[](0));
        assertEq(governanceDelegation.getVotes(_delegatee), 0);
    }

    function testFuzz_aferTokenTransfer_onlyTokenRelative_reverts(address _from, address _to, uint256 _amount) public {
        vm.expectRevert(IGovernanceDelegation.NotGovernanceToken.selector);
        governanceDelegation.afterTokenTransfer(_from, _to, _amount);
    }
}

contract GovernanceDelegation_Migration_Test is GovernanceDelegation_Init {
    function testFuzz_migrate_succeeds(uint224 _votes) public {
        vm.assume(_votes != 0);

        StdCheats.deployCodeTo("GovernanceToken.sol:GovernanceToken", "", Predeploys.GOVERNANCE_TOKEN);

        vm.prank(rando);
        governanceToken.delegate(rando);
        governanceToken.mint(rando, _votes);

        vm.roll(block.number + 1);

        StdCheats.deployCodeTo("GovernanceDelegation.sol:GovernanceDelegation", "", Predeploys.GOVERNANCE_DELEGATION);
        StdCheats.deployCodeTo("GovernanceTokenInterop.sol:GovernanceTokenInterop", "", Predeploys.GOVERNANCE_TOKEN);

        IGovernanceTokenInterop(address(governanceToken)).migrate(rando);

        vm.roll(block.number + 1);

        assertEq(governanceDelegation.delegates(rando), rando);
        assertEq(governanceDelegation.checkpoints(rando, 0).fromBlock, block.number - 1);
        assertEq(governanceDelegation.checkpoints(rando, 0).votes, _votes);
        assertEq(governanceDelegation.numCheckpoints(rando), 1);
        assertEq(governanceDelegation.getVotes(rando), _votes);
        assertEq(governanceDelegation.getPastVotes(rando, block.number - 1), _votes);
        assertEq(governanceDelegation.getPastTotalSupply(block.number - 1), _votes);
        assertTrue(_migrated(rando));
    }

    function testFuzz_migrateAccounts_noDuplicate_succeeds(uint224 _votes) public {
        vm.assume(_votes != 0);

        StdCheats.deployCodeTo("GovernanceToken.sol:GovernanceToken", "", Predeploys.GOVERNANCE_TOKEN);

        vm.prank(rando);
        governanceToken.delegate(rando);
        governanceToken.mint(rando, _votes);

        vm.roll(block.number + 1);

        StdCheats.deployCodeTo("GovernanceDelegation.sol:GovernanceDelegation", "", Predeploys.GOVERNANCE_DELEGATION);
        StdCheats.deployCodeTo("GovernanceTokenInterop.sol:GovernanceTokenInterop", "", Predeploys.GOVERNANCE_TOKEN);

        IGovernanceTokenInterop(address(governanceToken)).migrate(rando);

        vm.roll(block.number + 1);

        IGovernanceTokenInterop(address(governanceToken)).migrate(rando);

        assertEq(governanceDelegation.delegates(rando), rando);
        assertEq(governanceDelegation.checkpoints(rando, 0).fromBlock, block.number - 1);
        assertEq(governanceDelegation.checkpoints(rando, 0).votes, _votes);
        assertEq(governanceDelegation.numCheckpoints(rando), 1);
        assertEq(governanceDelegation.getVotes(rando), _votes);
        assertEq(governanceDelegation.getPastVotes(rando, block.number - 1), _votes);
        assertEq(governanceDelegation.getPastTotalSupply(block.number - 1), _votes);
        assertTrue(_migrated(rando));
    }
}

contract GovernanceDelegation_Getters_Test is GovernanceDelegation_Init {
    /// @dev Tests that the constructor sets the correct initial state.
    function test_constructor_succeeds() external view {
        assertEq(governanceToken.owner(), owner);
        assertEq(governanceToken.name(), "Optimism");
        assertEq(governanceToken.symbol(), "OP");
        assertEq(governanceToken.decimals(), 18);
        assertEq(governanceToken.totalSupply(), 0);
    }

    function testFuzz_getters_migrateAccounts_succeeds(uint224 _votes) public {
        vm.assume(_votes != 0);

        StdCheats.deployCodeTo("GovernanceToken.sol:GovernanceToken", "", Predeploys.GOVERNANCE_TOKEN);

        vm.prank(rando);
        governanceToken.delegate(rando);
        governanceToken.mint(rando, _votes);

        vm.roll(block.number + 1);

        StdCheats.deployCodeTo("GovernanceDelegation.sol:GovernanceDelegation", "", Predeploys.GOVERNANCE_DELEGATION);
        StdCheats.deployCodeTo("GovernanceTokenInterop.sol:GovernanceTokenInterop", "", Predeploys.GOVERNANCE_TOKEN);

        address[] memory accounts = new address[](1);
        accounts[0] = rando;
        governanceDelegation.migrateAccounts(accounts);

        vm.roll(block.number + 1);

        assertEq(governanceDelegation.delegates(rando), rando);
        assertEq(governanceDelegation.checkpoints(rando, 0).fromBlock, block.number - 1);
        assertEq(governanceDelegation.checkpoints(rando, 0).votes, _votes);
        assertEq(governanceDelegation.numCheckpoints(rando), 1);
        assertEq(governanceDelegation.getVotes(rando), _votes);
        assertEq(governanceDelegation.getPastVotes(rando, block.number - 1), _votes);
        assertEq(governanceDelegation.getPastTotalSupply(block.number - 1), _votes);
        assertTrue(_migrated(rando));
    }
}

contract GovernanceDelegation_Getters_TestFail is GovernanceDelegation_Init {
    function testFuzz_getPastVotes_blockNotYetMined_reverts(uint256 _amount) public {
        _amount = bound(_amount, 1, type(uint224).max);

        vm.prank(rando);
        governanceToken.delegate(rando);

        vm.prank(owner);
        governanceToken.mint(rando, _amount);

        vm.expectRevert(abi.encodeWithSelector(IGovernanceDelegation.BlockNotYetMined.selector, block.timestamp));
        governanceDelegation.getPastVotes(rando, block.timestamp);

        vm.roll(block.timestamp + 1);

        assertEq(governanceDelegation.getPastVotes(rando, block.timestamp), _amount);
    }

    function testFuzz_getPastTotalSupply_blockNotYetMined_reverts(uint256 _amount) public {
        _amount = bound(_amount, 1, type(uint224).max);

        vm.prank(rando);
        governanceToken.delegate(rando);

        vm.prank(owner);
        governanceToken.mint(rando, _amount);

        vm.expectRevert(abi.encodeWithSelector(IGovernanceDelegation.BlockNotYetMined.selector, block.timestamp));
        governanceDelegation.getPastTotalSupply(block.timestamp);

        vm.roll(block.timestamp + 1);

        assertEq(governanceDelegation.getPastTotalSupply(block.timestamp), _amount);
    }
}
