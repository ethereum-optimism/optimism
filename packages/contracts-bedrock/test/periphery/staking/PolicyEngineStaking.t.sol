// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing utilities
import { CommonTest } from "test/setup/CommonTest.sol";
import { TestERC20 } from "test/mocks/TestERC20.sol";

// Contracts
import { PolicyEngineStaking } from "src/periphery/staking/PolicyEngineStaking.sol";

// Interfaces
import { IERC20 } from "@openzeppelin/contracts/token/ERC20/IERC20.sol";

// Libraries
import { Predeploys } from "src/libraries/Predeploys.sol";

/// @title PolicyEngineStaking_TestInit
/// @notice Reusable test initialization for `PolicyEngineStaking` tests.
abstract contract PolicyEngineStaking_TestInit is CommonTest {
    address internal carol = address(0xC4101);

    PolicyEngineStaking internal staking;
    address internal owner;

    event Staked(address indexed account, uint256 amount);
    event Unstaked(address indexed account, uint256 amount);
    event Linked(address indexed staker, address indexed beneficiary);
    event Unlinked(address indexed staker, address indexed previousBeneficiary);
    event EffectiveStakeChanged(address indexed account, uint256 newEffectiveStake);
    event BeneficiaryAllowlistUpdated(address indexed beneficiary, address indexed staker, bool allowed);
    event Paused();
    event Unpaused();

    function setUp() public virtual override {
        super.setUp();
        owner = makeAddr("owner");
        staking = new PolicyEngineStaking(owner, Predeploys.GOVERNANCE_TOKEN);

        _setupMockOPToken();

        vm.label(carol, "carol");
        vm.label(address(staking), "PolicyEngineStaking");
    }

    /// @notice Deploys TestERC20 at the predeploy address and funds test accounts.
    function _setupMockOPToken() internal {
        TestERC20 token = new TestERC20();
        vm.etch(Predeploys.GOVERNANCE_TOKEN, address(token).code);

        TestERC20(Predeploys.GOVERNANCE_TOKEN).mint(alice, 1_000 ether);
        TestERC20(Predeploys.GOVERNANCE_TOKEN).mint(bob, 1_000 ether);
        TestERC20(Predeploys.GOVERNANCE_TOKEN).mint(carol, 1_000 ether);

        vm.prank(alice);
        IERC20(Predeploys.GOVERNANCE_TOKEN).approve(address(staking), type(uint256).max);
        vm.prank(bob);
        IERC20(Predeploys.GOVERNANCE_TOKEN).approve(address(staking), type(uint256).max);
        vm.prank(carol);
        IERC20(Predeploys.GOVERNANCE_TOKEN).approve(address(staking), type(uint256).max);
    }
}

/// @title PolicyEngineStaking_Pause_Test
/// @notice Tests the pause/unpause functionality.
contract PolicyEngineStaking_Pause_Test is PolicyEngineStaking_TestInit {
    /// @notice Tests that owner can pause and unpause.
    function test_pauseUnpause_owner_succeeds() external {
        assertFalse(staking.paused());

        vm.expectEmit(address(staking));
        emit Paused();
        vm.prank(owner);
        staking.pause();

        assertTrue(staking.paused());

        vm.expectEmit(address(staking));
        emit Unpaused();
        vm.prank(owner);
        staking.unpause();

        assertFalse(staking.paused());
    }

    /// @notice Tests that non-owner cannot pause.
    function testFuzz_pause_notOwner_reverts(address _caller) external {
        vm.assume(_caller != owner && _caller != address(0));
        vm.expectRevert(PolicyEngineStaking.PolicyEngineStaking_OnlyOwner.selector);
        vm.prank(_caller);
        staking.pause();
    }

    /// @notice Tests that non-owner cannot unpause.
    function testFuzz_unpause_notOwner_reverts(address _caller) external {
        vm.prank(owner);
        staking.pause();

        vm.assume(_caller != owner && _caller != address(0));
        vm.expectRevert(PolicyEngineStaking.PolicyEngineStaking_OnlyOwner.selector);
        vm.prank(_caller);
        staking.unpause();
    }

    /// @notice Tests that stake reverts when paused.
    function test_stake_whenPaused_reverts() external {
        vm.prank(owner);
        staking.pause();

        vm.expectRevert(PolicyEngineStaking.PolicyEngineStaking_Paused.selector);
        vm.prank(alice);
        staking.stake(100 ether, alice);
    }

    /// @notice Tests that changeBeneficiary works when paused.
    function test_changeBeneficiary_whenPaused_succeeds() external {
        vm.prank(alice);
        staking.stake(100 ether, alice);
        vm.prank(bob);
        staking.setAllowedStaker(alice, true);

        vm.prank(owner);
        staking.pause();

        vm.prank(alice);
        staking.changeBeneficiary(bob);

        (, address linkedTo) = staking.stakingData(alice);
        assertEq(linkedTo, bob);
    }

    /// @notice Tests that unstake works when paused.
    function test_unstake_whenPaused_succeeds() external {
        vm.prank(alice);
        staking.stake(100 ether, alice);
        vm.prank(owner);
        staking.pause();

        uint256 balanceBefore = IERC20(Predeploys.GOVERNANCE_TOKEN).balanceOf(alice);
        vm.prank(alice);
        staking.unstake(100 ether);
        assertEq(IERC20(Predeploys.GOVERNANCE_TOKEN).balanceOf(alice), balanceBefore + 100 ether);
    }
}

/// @title PolicyEngineStaking_Stake_Test
/// @notice Tests the `stake` function.
contract PolicyEngineStaking_Stake_Test is PolicyEngineStaking_TestInit {
    /// @notice Tests that stake with self-attribution succeeds.
    function testFuzz_stake_selfAttribution_succeeds(uint256 _amount) external {
        _amount = bound(_amount, 1, 1_000 ether);

        vm.expectEmit(address(staking));
        emit Linked(alice, alice);
        vm.expectEmit(address(staking));
        emit EffectiveStakeChanged(alice, _amount);
        vm.expectEmit(address(staking));
        emit Staked(alice, _amount);

        vm.prank(alice);
        staking.stake(_amount, alice);

        (uint256 staked, address linkedTo) = staking.stakingData(alice);
        (uint128 effectiveStake, uint128 lastUpdate) = staking.peData(alice);

        assertEq(staked, _amount);
        assertEq(linkedTo, alice);
        assertEq(effectiveStake, _amount);
        assertEq(lastUpdate, block.timestamp);
    }

    /// @notice Tests that multiple stake calls to same beneficiary succeed.
    function test_stake_severalToSameBeneficiary_succeeds() external {
        vm.prank(alice);
        staking.stake(100 ether, alice);
        vm.prank(alice);
        staking.stake(200 ether, alice);
        vm.prank(alice);
        staking.stake(300 ether, alice);

        (uint256 aliceStaked, address aliceLinkedTo) = staking.stakingData(alice);
        assertEq(aliceStaked, 600 ether);
        assertEq(aliceLinkedTo, alice);
        (uint128 aliceEffectiveStake, uint128 aliceLastUpdate) = staking.peData(alice);
        assertEq(aliceEffectiveStake, 600 ether);
        assertEq(aliceLastUpdate, block.timestamp);
    }

    /// @notice Tests that stake to another beneficiary with allowlist succeeds.
    function test_stake_toBeneficiaryWithAllowlist_succeeds() external {
        vm.prank(bob);
        staking.setAllowedStaker(alice, true);

        vm.expectEmit(address(staking));
        emit Linked(alice, bob);
        vm.expectEmit(address(staking));
        emit EffectiveStakeChanged(bob, 100 ether);
        vm.expectEmit(address(staking));
        emit Staked(alice, 100 ether);

        vm.prank(alice);
        staking.stake(100 ether, bob);

        (uint256 staked, address linkedTo) = staking.stakingData(alice);
        (uint128 effectiveStake, uint128 lastUpdate) = staking.peData(alice);
        assertEq(staked, 100 ether);
        assertEq(linkedTo, bob);
        assertEq(effectiveStake, 0);
        assertEq(lastUpdate, 0);

        (uint128 bobEffectiveStake, uint128 bobLastUpdate) = staking.peData(bob);
        assertEq(bobEffectiveStake, 100 ether);
        assertEq(bobLastUpdate, block.timestamp);
    }

    /// @notice Tests that stake more to same beneficiary when already linked succeeds.
    function test_stake_moreToSameBeneficiary_succeeds() external {
        vm.prank(bob);
        staking.setAllowedStaker(alice, true);
        vm.prank(alice);
        staking.stake(100 ether, bob);

        vm.prank(alice);
        staking.stake(50 ether, bob);

        (uint256 staked,) = staking.stakingData(alice);
        assertEq(staked, 150 ether);
        (uint128 bobEffective,) = staking.peData(bob);
        assertEq(bobEffective, 150 ether);
    }

    /// @notice Tests that stake re-links to new beneficiary atomically.
    function test_stake_relink_succeeds() external {
        // Alice stakes to self
        vm.prank(alice);
        staking.stake(100 ether, alice);

        (uint128 aliceEffBefore,) = staking.peData(alice);
        assertEq(aliceEffBefore, 100 ether);

        // Bob allows alice
        vm.prank(bob);
        staking.setAllowedStaker(alice, true);

        // Alice re-links to bob with additional stake
        vm.expectEmit(address(staking));
        emit EffectiveStakeChanged(alice, 0); // decrease alice's PE
        vm.expectEmit(address(staking));
        emit Unlinked(alice, alice);
        vm.expectEmit(address(staking));
        emit EffectiveStakeChanged(bob, 100 ether); // increase bob's PE by existing stake
        vm.expectEmit(address(staking));
        emit Linked(alice, bob);
        vm.expectEmit(address(staking));
        emit EffectiveStakeChanged(bob, 150 ether); // increase bob's PE by new amount
        vm.expectEmit(address(staking));
        emit Staked(alice, 50 ether);

        vm.prank(alice);
        staking.stake(50 ether, bob);

        (uint256 staked, address linkedTo) = staking.stakingData(alice);
        assertEq(staked, 150 ether);
        assertEq(linkedTo, bob);
        (uint128 aliceEffAfter,) = staking.peData(alice);
        assertEq(aliceEffAfter, 0);
        (uint128 bobEff,) = staking.peData(bob);
        assertEq(bobEff, 150 ether);
    }

    /// @notice Tests that stake with zero amount reverts.
    function test_stake_zeroAmount_reverts() external {
        vm.prank(alice);
        vm.expectRevert(PolicyEngineStaking.PolicyEngineStaking_ZeroAmount.selector);
        staking.stake(0, alice);
    }

    /// @notice Tests that stake with zero beneficiary reverts.
    function test_stake_zeroBeneficiary_reverts() external {
        vm.prank(alice);
        vm.expectRevert(PolicyEngineStaking.PolicyEngineStaking_ZeroBeneficiary.selector);
        staking.stake(100 ether, address(0));
    }

    /// @notice Tests that stake to beneficiary without allowlist reverts.
    function test_stake_toBeneficiaryWithoutAllowlist_reverts() external {
        vm.prank(alice);
        vm.expectRevert(PolicyEngineStaking.PolicyEngineStaking_NotAllowedToLink.selector);
        staking.stake(100 ether, bob);
    }

    /// @notice Tests re-link from beneficiary to self reverts without allowlist removal.
    function test_stake_relinkToBeneficiaryWithoutAllowlist_reverts() external {
        vm.prank(alice);
        staking.stake(100 ether, alice);

        vm.prank(alice);
        vm.expectRevert(PolicyEngineStaking.PolicyEngineStaking_NotAllowedToLink.selector);
        staking.stake(50 ether, bob);
    }
}

/// @title PolicyEngineStaking_Unstake_Test
/// @notice Tests the `unstake` function.
contract PolicyEngineStaking_Unstake_Test is PolicyEngineStaking_TestInit {
    /// @notice Tests that full unstake succeeds, auto-unlinks, and preserves balance.
    function testFuzz_unstake_full_succeeds(uint256 _amount) external {
        _amount = bound(_amount, 1, 1_000 ether);

        vm.prank(alice);
        staking.stake(_amount, alice);

        uint256 aliceBalanceBefore = IERC20(Predeploys.GOVERNANCE_TOKEN).balanceOf(alice);

        vm.expectEmit(address(staking));
        emit EffectiveStakeChanged(alice, 0);
        vm.expectEmit(address(staking));
        emit Unlinked(alice, alice);
        vm.expectEmit(address(staking));
        emit Unstaked(alice, _amount);

        vm.prank(alice);
        staking.unstake(_amount);

        (uint256 aliceStaked, address linkedTo) = staking.stakingData(alice);
        assertEq(aliceStaked, 0);
        assertEq(linkedTo, address(0));
        assertEq(IERC20(Predeploys.GOVERNANCE_TOKEN).balanceOf(alice), aliceBalanceBefore + _amount);
    }

    /// @notice Tests that unstake with zero amount reverts.
    function test_unstake_zeroAmount_reverts() external {
        vm.prank(alice);
        staking.stake(100 ether, alice);

        vm.prank(alice);
        vm.expectRevert(PolicyEngineStaking.PolicyEngineStaking_ZeroAmount.selector);
        staking.unstake(0);
    }

    /// @notice Tests that unstake with no stake reverts.
    function test_unstake_noStake_reverts() external {
        vm.prank(alice);
        vm.expectRevert(PolicyEngineStaking.PolicyEngineStaking_InsufficientStake.selector);
        staking.unstake(100 ether);
    }

    /// @notice Tests that unstake more than staked reverts.
    function test_unstake_insufficientStake_reverts() external {
        vm.prank(alice);
        staking.stake(100 ether, alice);

        vm.prank(alice);
        vm.expectRevert(PolicyEngineStaking.PolicyEngineStaking_InsufficientStake.selector);
        staking.unstake(101 ether);
    }

    /// @notice Tests partial unstake preserves correct remaining balance.
    function testFuzz_unstake_partialAmount_succeeds(uint256 _stakeAmount, uint256 _unstakeAmount) external {
        _stakeAmount = bound(_stakeAmount, 2, 1_000 ether);
        _unstakeAmount = bound(_unstakeAmount, 1, _stakeAmount - 1);

        vm.prank(alice);
        staking.stake(_stakeAmount, alice);

        uint256 remaining = _stakeAmount - _unstakeAmount;

        vm.expectEmit(address(staking));
        emit EffectiveStakeChanged(alice, remaining);
        vm.expectEmit(address(staking));
        emit Unstaked(alice, _unstakeAmount);

        vm.prank(alice);
        staking.unstake(_unstakeAmount);

        (uint256 staked, address linkedTo) = staking.stakingData(alice);
        assertEq(staked, remaining);
        assertEq(linkedTo, alice);
        (uint128 effective,) = staking.peData(alice);
        assertEq(effective, remaining);
    }

    /// @notice Tests partial unstake with beneficiary preserves remaining stake attribution.
    function testFuzz_unstake_partialWithBeneficiary_succeeds(uint256 _stakeAmount, uint256 _unstakeAmount) external {
        _stakeAmount = bound(_stakeAmount, 2, 1_000 ether);
        _unstakeAmount = bound(_unstakeAmount, 1, _stakeAmount - 1);

        vm.prank(bob);
        staking.setAllowedStaker(alice, true);

        vm.prank(alice);
        staking.stake(_stakeAmount, bob);

        uint256 remaining = _stakeAmount - _unstakeAmount;

        vm.expectEmit(address(staking));
        emit EffectiveStakeChanged(bob, remaining);
        vm.expectEmit(address(staking));
        emit Unstaked(alice, _unstakeAmount);

        vm.prank(alice);
        staking.unstake(_unstakeAmount);

        (uint256 staked, address linkedTo) = staking.stakingData(alice);
        assertEq(staked, remaining);
        assertEq(linkedTo, bob);
        (uint128 bobEffective,) = staking.peData(bob);
        assertEq(bobEffective, remaining);
    }
}

/// @title PolicyEngineStaking_ChangeBeneficiary_Test
/// @notice Tests the `changeBeneficiary` function.
contract PolicyEngineStaking_ChangeBeneficiary_Test is PolicyEngineStaking_TestInit {
    /// @notice Tests that changing beneficiary succeeds.
    function testFuzz_changeBeneficiary_succeeds(uint256 _amount) external {
        _amount = bound(_amount, 1, 1_000 ether);
        vm.prank(alice);
        staking.stake(_amount, alice);

        vm.prank(bob);
        staking.setAllowedStaker(alice, true);

        vm.expectEmit(address(staking));
        emit EffectiveStakeChanged(alice, 0);
        vm.expectEmit(address(staking));
        emit Unlinked(alice, alice);
        vm.expectEmit(address(staking));
        emit EffectiveStakeChanged(bob, _amount);
        vm.expectEmit(address(staking));
        emit Linked(alice, bob);

        vm.prank(alice);
        staking.changeBeneficiary(bob);

        (uint256 staked, address linkedTo) = staking.stakingData(alice);
        assertEq(staked, _amount);
        assertEq(linkedTo, bob);
        (uint128 aliceEffective,) = staking.peData(alice);
        assertEq(aliceEffective, 0);
        (uint128 bobEffective,) = staking.peData(bob);
        assertEq(bobEffective, _amount);
    }

    /// @notice Tests that re-linking from one beneficiary to another succeeds.
    function test_changeBeneficiary_relink_succeeds() external {
        vm.prank(bob);
        staking.setAllowedStaker(alice, true);
        vm.prank(alice);
        staking.stake(100 ether, bob);

        vm.prank(carol);
        staking.setAllowedStaker(alice, true);

        vm.prank(alice);
        staking.changeBeneficiary(carol);

        (, address linkedTo) = staking.stakingData(alice);
        assertEq(linkedTo, carol);
        (uint128 bobEffective,) = staking.peData(bob);
        assertEq(bobEffective, 0);
        (uint128 carolEffective,) = staking.peData(carol);
        assertEq(carolEffective, 100 ether);
    }

    /// @notice Tests that changing beneficiary to self succeeds (no allowlist needed).
    function test_changeBeneficiary_toSelf_succeeds() external {
        vm.prank(bob);
        staking.setAllowedStaker(alice, true);
        vm.prank(alice);
        staking.stake(100 ether, bob);

        vm.prank(alice);
        staking.changeBeneficiary(alice);

        (, address linkedTo) = staking.stakingData(alice);
        assertEq(linkedTo, alice);
        (uint128 aliceEffective,) = staking.peData(alice);
        assertEq(aliceEffective, 100 ether);
        (uint128 bobEffective,) = staking.peData(bob);
        assertEq(bobEffective, 0);
    }

    /// @notice Tests that changing to same beneficiary is a no-op.
    function test_changeBeneficiary_sameBeneficiaryNoOp_succeeds() external {
        vm.prank(alice);
        staking.stake(100 ether, alice);

        (uint128 effectiveBefore, uint128 lastUpdateBefore) = staking.peData(alice);

        vm.warp(block.timestamp + 1);

        vm.prank(alice);
        staking.changeBeneficiary(alice);

        // PE data should be unchanged (no-op)
        (uint128 effectiveAfter, uint128 lastUpdateAfter) = staking.peData(alice);
        assertEq(effectiveAfter, effectiveBefore);
        assertEq(lastUpdateAfter, lastUpdateBefore);
    }

    /// @notice Tests that changeBeneficiary with zero beneficiary reverts.
    function test_changeBeneficiary_zeroBeneficiary_reverts() external {
        vm.prank(alice);
        staking.stake(100 ether, alice);

        vm.prank(alice);
        vm.expectRevert(PolicyEngineStaking.PolicyEngineStaking_ZeroBeneficiary.selector);
        staking.changeBeneficiary(address(0));
    }

    /// @notice Tests that changeBeneficiary without allowlist reverts.
    function test_changeBeneficiary_notAllowed_reverts() external {
        vm.prank(alice);
        staking.stake(100 ether, alice);

        vm.prank(alice);
        vm.expectRevert(PolicyEngineStaking.PolicyEngineStaking_NotAllowedToLink.selector);
        staking.changeBeneficiary(bob);
    }

    /// @notice Tests that changeBeneficiary with no stake reverts.
    function test_changeBeneficiary_noStake_reverts() external {
        vm.prank(alice);
        vm.expectRevert(PolicyEngineStaking.PolicyEngineStaking_NoStake.selector);
        staking.changeBeneficiary(bob);
    }
}

/// @title PolicyEngineStaking_Constructor_Test
/// @notice Tests constructor, view functions, and storage layout.
contract PolicyEngineStaking_Constructor_Test is PolicyEngineStaking_TestInit {
    /// @notice Tests that owner is set correctly.
    function test_owner_succeeds() external view {
        assertEq(staking.owner(), owner);
    }

    /// @notice Tests that PE_DATA_SLOT is 0.
    function test_peDataSlot_isZero_succeeds() external view {
        assertEq(staking.PE_DATA_SLOT(), bytes32(uint256(0)));
    }

    /// @notice Tests that peData storage layout matches PE_DATA_SLOT convention
    ///         across stake, changeBeneficiary, and unstake operations.
    function test_peData_storageLayout_succeeds() external {
        uint256 amount = 100 ether;
        bytes32 aliceSlot = keccak256(abi.encode(alice, staking.PE_DATA_SLOT()));
        bytes32 bobSlot = keccak256(abi.encode(bob, staking.PE_DATA_SLOT()));

        // After stake: staker's beneficiary slot is populated
        vm.prank(alice);
        staking.stake(amount, alice);
        bytes32 raw = vm.load(address(staking), aliceSlot);
        assertEq(uint128(uint256(raw)), amount);
        assertEq(uint128(uint256(raw) >> 128), block.timestamp);

        // After changeBeneficiary: stake moves to beneficiary's slot, staker's slot zeroed
        vm.prank(bob);
        staking.setAllowedStaker(alice, true);
        vm.warp(block.timestamp + 1);
        vm.prank(alice);
        staking.changeBeneficiary(bob);

        raw = vm.load(address(staking), aliceSlot);
        assertEq(uint128(uint256(raw)), 0);

        bytes32 bobRaw = vm.load(address(staking), bobSlot);
        assertEq(uint128(uint256(bobRaw)), amount);
        assertEq(uint128(uint256(bobRaw) >> 128), block.timestamp);

        // After full unstake: beneficiary's slot zeroed
        vm.prank(alice);
        staking.unstake(amount);
        bobRaw = vm.load(address(staking), bobSlot);
        assertEq(uint128(uint256(bobRaw)), 0);
    }
}

/// @title PolicyEngineStaking_SetAllowedStaker_Test
/// @notice Tests the `setAllowedStaker` and `setAllowedStakers` functions.
contract PolicyEngineStaking_SetAllowedStaker_Test is PolicyEngineStaking_TestInit {
    /// @notice Tests that setAllowedStaker updates allowlist correctly.
    function test_setAllowedStaker_succeeds() external {
        (bool allowed) = staking.allowlist(bob, alice);
        assertFalse(allowed);

        vm.expectEmit(address(staking));
        emit BeneficiaryAllowlistUpdated(bob, alice, true);

        vm.prank(bob);
        staking.setAllowedStaker(alice, true);

        (allowed) = staking.allowlist(bob, alice);
        assertTrue(allowed);

        vm.prank(bob);
        staking.setAllowedStaker(alice, false);

        (allowed) = staking.allowlist(bob, alice);
        assertFalse(allowed);
    }

    /// @notice Tests that setAllowedStakers batch updates allowlist.
    function test_setAllowedStakers_succeeds() external {
        address[] memory stakers = new address[](2);
        stakers[0] = alice;
        stakers[1] = carol;

        vm.prank(bob);
        staking.setAllowedStakers(stakers, true);

        (bool aliceAllowed) = staking.allowlist(bob, alice);
        (bool carolAllowed) = staking.allowlist(bob, carol);
        assertTrue(aliceAllowed);
        assertTrue(carolAllowed);

        vm.prank(bob);
        staking.setAllowedStakers(stakers, false);

        (aliceAllowed) = staking.allowlist(bob, alice);
        (carolAllowed) = staking.allowlist(bob, carol);
        assertFalse(aliceAllowed);
        assertFalse(carolAllowed);
    }
}

/// @title PolicyEngineStaking_Integration_Test
/// @notice Integration tests for the full stake/changeBeneficiary/unstake flow.
contract PolicyEngineStaking_Integration_Test is PolicyEngineStaking_TestInit {
    /// @notice Tests full flow: stake -> stake more -> changeBeneficiary -> partial unstake -> full unstake.
    function test_fullFlow_succeeds() external {
        // Step 1: Alice stakes 100 to self
        vm.prank(alice);
        staking.stake(100 ether, alice);
        (uint256 staked,) = staking.stakingData(alice);
        assertEq(staked, 100 ether);

        // Step 2: Alice stakes 50 more
        vm.prank(alice);
        staking.stake(50 ether, alice);
        (staked,) = staking.stakingData(alice);
        assertEq(staked, 150 ether);

        // Step 3: Alice changes beneficiary to bob
        vm.prank(bob);
        staking.setAllowedStaker(alice, true);
        vm.prank(alice);
        staking.changeBeneficiary(bob);

        (, address linkedTo) = staking.stakingData(alice);
        assertEq(linkedTo, bob);
        (uint128 bobEff,) = staking.peData(bob);
        assertEq(bobEff, 150 ether);
        (uint128 aliceEff,) = staking.peData(alice);
        assertEq(aliceEff, 0);

        // Step 4: Partial unstake
        vm.prank(alice);
        staking.unstake(50 ether);
        (staked, linkedTo) = staking.stakingData(alice);
        assertEq(staked, 100 ether);
        assertEq(linkedTo, bob);
        (bobEff,) = staking.peData(bob);
        assertEq(bobEff, 100 ether);

        // Step 5: Full unstake (auto-unlinks)
        uint256 aliceBalanceBefore = IERC20(Predeploys.GOVERNANCE_TOKEN).balanceOf(alice);
        vm.prank(alice);
        staking.unstake(100 ether);
        assertEq(IERC20(Predeploys.GOVERNANCE_TOKEN).balanceOf(alice), aliceBalanceBefore + 100 ether);
        (staked, linkedTo) = staking.stakingData(alice);
        assertEq(staked, 0);
        assertEq(linkedTo, address(0));
        (bobEff,) = staking.peData(bob);
        assertEq(bobEff, 0);
    }

    /// @notice Tests that multiple stakers can stake to the same beneficiary.
    function test_multipleStakersToSameBeneficiary_succeeds() external {
        vm.prank(bob);
        staking.setAllowedStaker(alice, true);
        vm.prank(bob);
        staking.setAllowedStaker(carol, true);

        vm.prank(alice);
        staking.stake(100 ether, bob);
        vm.prank(carol);
        staking.stake(50 ether, bob);

        (uint128 bobEffective,) = staking.peData(bob);
        assertEq(bobEffective, 150 ether);
    }

    /// @notice Tests that a beneficiary with own stake plus received stake has correct effective stake.
    function test_beneficiaryWithOwnStakeAndReceived_succeeds() external {
        vm.prank(bob);
        staking.stake(50 ether, bob);
        vm.prank(bob);
        staking.setAllowedStaker(alice, true);
        vm.prank(alice);
        staking.stake(100 ether, bob);

        (uint256 bobStaked,) = staking.stakingData(bob);
        assertEq(bobStaked, 50 ether);
        (uint128 bobEffective,) = staking.peData(bob);
        assertEq(bobEffective, 150 ether);
    }

    /// @notice Tests that revoking allowlist does not auto-unlink: staker keeps link until explicit action.
    function test_revokeAllowlist_stakeRemainsUntilUnstake_succeeds() external {
        vm.prank(alice);
        staking.setAllowedStaker(bob, true);
        vm.prank(bob);
        staking.stake(100 ether, alice);

        (uint256 bobStaked, address bobLinkedTo) = staking.stakingData(bob);
        (uint128 aliceEffective,) = staking.peData(alice);
        assertEq(bobStaked, 100 ether);
        assertEq(bobLinkedTo, alice);
        assertEq(aliceEffective, 100 ether);

        vm.prank(alice);
        staking.setAllowedStaker(bob, false);

        // Bob stays linked - stake and effective power remain with Alice
        (bobStaked, bobLinkedTo) = staking.stakingData(bob);
        (aliceEffective,) = staking.peData(alice);
        assertEq(bobStaked, 100 ether);
        assertEq(bobLinkedTo, alice);
        assertEq(aliceEffective, 100 ether);

        // Bob fully unstakes (auto-unlinks)
        vm.prank(bob);
        staking.unstake(100 ether);

        (bobStaked, bobLinkedTo) = staking.stakingData(bob);
        (aliceEffective,) = staking.peData(alice);
        assertEq(bobStaked, 0);
        assertEq(bobLinkedTo, address(0));
        assertEq(aliceEffective, 0);
    }

    /// @notice Tests that lastUpdate is updated after new staking and linking when time advances.
    function test_lastUpdate_updatesAfterStakingAndLinking_succeeds() external {
        // Initial stake
        vm.prank(alice);
        staking.stake(100 ether, alice);
        (, uint128 lastUpdate0) = staking.peData(alice);
        uint256 ts0 = block.timestamp;
        assertEq(lastUpdate0, ts0);

        // Warp time and stake again; lastUpdate should advance
        vm.warp(block.timestamp + 1);
        vm.prank(alice);
        staking.stake(50 ether, alice);
        (, uint128 lastUpdate1) = staking.peData(alice);
        assertEq(lastUpdate1, ts0 + 1);

        // Warp time and link to bob; bob's lastUpdate should be the new timestamp
        vm.warp(block.timestamp + 1);
        vm.prank(bob);
        staking.setAllowedStaker(alice, true);
        vm.prank(alice);
        staking.changeBeneficiary(bob);
        (, uint128 bobLastUpdate) = staking.peData(bob);
        assertEq(bobLastUpdate, ts0 + 2);
    }

    /// @notice Tests that stake after full unstake works (re-entry into system).
    function test_stake_afterFullUnstake_succeeds() external {
        vm.prank(alice);
        staking.stake(100 ether, alice);
        vm.prank(alice);
        staking.unstake(100 ether);

        (uint256 staked, address linkedTo) = staking.stakingData(alice);
        assertEq(staked, 0);
        assertEq(linkedTo, address(0));

        // Re-enter with a different beneficiary
        vm.prank(bob);
        staking.setAllowedStaker(alice, true);
        vm.prank(alice);
        staking.stake(50 ether, bob);

        (staked, linkedTo) = staking.stakingData(alice);
        assertEq(staked, 50 ether);
        assertEq(linkedTo, bob);
        (uint128 bobEffective,) = staking.peData(bob);
        assertEq(bobEffective, 50 ether);
    }

    /// @notice Tests stake to beneficiary and full unstake preserves staker balance.
    function testFuzz_stakeToBeneficiaryAndUnstake_succeeds(uint256 _amount) external {
        _amount = bound(_amount, 1, 1_000 ether);

        vm.prank(bob);
        staking.setAllowedStaker(alice, true);

        uint256 balanceBefore = IERC20(Predeploys.GOVERNANCE_TOKEN).balanceOf(alice);
        vm.prank(alice);
        staking.stake(_amount, bob);
        vm.prank(alice);
        staking.unstake(_amount);
        uint256 balanceAfter = IERC20(Predeploys.GOVERNANCE_TOKEN).balanceOf(alice);

        assertEq(balanceAfter, balanceBefore);
        (uint256 aliceStaked, address linkedTo) = staking.stakingData(alice);
        assertEq(aliceStaked, 0);
        assertEq(linkedTo, address(0));
        (uint128 bobEffective,) = staking.peData(bob);
        assertEq(bobEffective, 0);
    }

    /// @notice Tests stake -> changeBeneficiary -> unstake full cycle.
    function testFuzz_linkCycle_succeeds(uint256 _amount) external {
        _amount = bound(_amount, 1, 1_000 ether);

        vm.prank(bob);
        staking.setAllowedStaker(alice, true);

        uint256 balanceBefore = IERC20(Predeploys.GOVERNANCE_TOKEN).balanceOf(alice);
        vm.prank(alice);
        staking.stake(_amount, alice);
        vm.prank(alice);
        staking.changeBeneficiary(bob);

        (uint128 bobEff,) = staking.peData(bob);
        assertEq(bobEff, _amount);

        vm.prank(alice);
        staking.unstake(_amount);
        uint256 balanceAfter = IERC20(Predeploys.GOVERNANCE_TOKEN).balanceOf(alice);

        assertEq(balanceAfter, balanceBefore);
    }

    /// @notice Tests multiple stake calls and single full unstake.
    function testFuzz_multipleStakesAndUnstake_succeeds(
        uint256 _amount1,
        uint256 _amount2,
        uint256 _amount3
    )
        external
    {
        _amount1 = bound(_amount1, 1, 300 ether);
        _amount2 = bound(_amount2, 1, 300 ether);
        _amount3 = bound(_amount3, 1, 300 ether);

        uint256 total = _amount1 + _amount2 + _amount3;

        uint256 balanceBefore = IERC20(Predeploys.GOVERNANCE_TOKEN).balanceOf(alice);
        vm.prank(alice);
        staking.stake(_amount1, alice);
        vm.prank(alice);
        staking.stake(_amount2, alice);
        vm.prank(alice);
        staking.stake(_amount3, alice);

        (uint256 staked,) = staking.stakingData(alice);
        (uint128 effective,) = staking.peData(alice);
        assertEq(staked, total);
        assertEq(effective, total);

        vm.prank(alice);
        staking.unstake(total);
        assertEq(IERC20(Predeploys.GOVERNANCE_TOKEN).balanceOf(alice), balanceBefore);
    }

    /// @notice Tests stake with different staker-beneficiary pairs.
    function testFuzz_stakeToBeneficiaryDifferentAccounts_succeeds(
        uint8 _stakerIdx,
        uint8 _beneficiaryIdx,
        uint256 _amount
    )
        external
    {
        address[] memory accounts = _accounts();
        _stakerIdx = uint8(bound(_stakerIdx, 0, 2));
        _beneficiaryIdx = uint8(bound(_beneficiaryIdx, 0, 2));
        if (_stakerIdx == _beneficiaryIdx) return; // self-attribution, skip
        address staker = accounts[_stakerIdx];
        address beneficiary = accounts[_beneficiaryIdx];
        _amount = bound(_amount, 1, 300 ether);

        vm.prank(beneficiary);
        staking.setAllowedStaker(staker, true);

        uint256 balanceBefore = IERC20(Predeploys.GOVERNANCE_TOKEN).balanceOf(staker);
        vm.prank(staker);
        staking.stake(_amount, beneficiary);
        vm.prank(staker);
        staking.unstake(_amount);
        uint256 balanceAfter = IERC20(Predeploys.GOVERNANCE_TOKEN).balanceOf(staker);

        assertEq(balanceAfter, balanceBefore);
        (uint128 benEffective,) = staking.peData(beneficiary);
        assertEq(benEffective, 0);
    }

    function _accounts() internal view returns (address[] memory) {
        address[] memory a = new address[](3);
        a[0] = alice;
        a[1] = bob;
        a[2] = carol;
        return a;
    }
}
