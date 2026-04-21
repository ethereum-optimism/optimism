// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing utilities
import { CommonTest } from "test/setup/CommonTest.sol";
import { TestERC20 } from "test/mocks/TestERC20.sol";

// Interfaces
import { IERC20 } from "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import { IPolicyEngineStaking } from "interfaces/periphery/staking/IPolicyEngineStaking.sol";

// Libraries
import { Predeploys } from "src/libraries/Predeploys.sol";

/// @title PolicyEngineStaking_TestInit
/// @notice Reusable test initialization for `PolicyEngineStaking` tests.
abstract contract PolicyEngineStaking_TestInit is CommonTest {
    address internal carol = address(0xC4101);

    IPolicyEngineStaking internal staking;
    address internal owner;

    event Staked(address indexed account, uint128 amount);
    event Unstaked(address indexed account, uint128 amount);
    event BeneficiarySet(address indexed staker, address indexed beneficiary);
    event BeneficiaryRemoved(address indexed staker, address indexed previousBeneficiary);
    event EffectiveStakeChanged(address indexed account, uint256 newEffectiveStake);
    event BeneficiaryAllowlistUpdated(address indexed beneficiary, address indexed staker, bool allowed);
    event Paused();
    event Unpaused();
    event OwnershipTransferStarted(address indexed previousOwner, address indexed newOwner);

    function setUp() public virtual override {
        super.setUp();
        owner = makeAddr("owner");
        staking = IPolicyEngineStaking(
            vm.deployCode("PolicyEngineStaking.sol:PolicyEngineStaking", abi.encode(owner, Predeploys.GOVERNANCE_TOKEN))
        );

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

/// @title PolicyEngineStaking_TransferOwnership_Test
/// @notice Tests the two-step `transferOwnership` / `acceptOwnership` pattern.
contract PolicyEngineStaking_TransferOwnership_Test is PolicyEngineStaking_TestInit {
    /// @notice Tests that transferOwnership nominates a pending owner without changing owner.
    function testFuzz_transferOwnership_succeeds(address _newOwner) external {
        vm.assume(_newOwner != address(0));

        vm.expectEmit(address(staking));
        emit OwnershipTransferStarted(owner, _newOwner);

        vm.prank(owner);
        staking.transferOwnership(_newOwner);

        // Owner should NOT change yet
        assertEq(staking.owner(), owner);
        assertEq(staking.pendingOwner(), _newOwner);
    }

    /// @notice Tests that pendingOwner can accept ownership.
    function test_acceptOwnership_succeeds() external {
        address newOwner = makeAddr("newOwner");

        vm.prank(owner);
        staking.transferOwnership(newOwner);

        vm.expectEmit(address(staking));
        emit OwnershipTransferred(owner, newOwner);

        vm.prank(newOwner);
        staking.acceptOwnership();

        assertEq(staking.owner(), newOwner);
        assertEq(staking.pendingOwner(), address(0));
    }

    /// @notice Tests that new owner can exercise ownership after accepting.
    function test_transferOwnership_newOwnerCanPause_succeeds() external {
        address newOwner = makeAddr("newOwner");

        vm.prank(owner);
        staking.transferOwnership(newOwner);

        vm.prank(newOwner);
        staking.acceptOwnership();

        vm.prank(newOwner);
        staking.pause();
        assertTrue(staking.paused());
    }

    /// @notice Tests that old owner loses ownership after transfer is accepted.
    function test_transferOwnership_oldOwnerReverts_reverts() external {
        address newOwner = makeAddr("newOwner");

        vm.prank(owner);
        staking.transferOwnership(newOwner);

        vm.prank(newOwner);
        staking.acceptOwnership();

        vm.prank(owner);
        vm.expectRevert(abi.encodeWithSelector(IPolicyEngineStaking.OwnableUnauthorizedAccount.selector, owner));
        staking.pause();
    }

    /// @notice Tests that non-owner cannot call transferOwnership.
    function testFuzz_transferOwnership_notOwner_reverts(address _caller) external {
        vm.assume(_caller != owner && _caller != address(0));

        vm.expectRevert(abi.encodeWithSelector(IPolicyEngineStaking.OwnableUnauthorizedAccount.selector, _caller));
        vm.prank(_caller);
        staking.transferOwnership(alice);
    }

    /// @notice Tests that transferring to zero address cancels pending transfer.
    function test_transferOwnership_zeroAddressCancels_succeeds() external {
        address newOwner = makeAddr("newOwner");

        vm.prank(owner);
        staking.transferOwnership(newOwner);
        assertEq(staking.pendingOwner(), newOwner);

        vm.prank(owner);
        staking.transferOwnership(address(0));
        assertEq(staking.pendingOwner(), address(0));
    }

    /// @notice Tests that non-pending-owner cannot accept ownership.
    function test_acceptOwnership_notPendingOwner_reverts() external {
        address newOwner = makeAddr("newOwner");

        vm.prank(owner);
        staking.transferOwnership(newOwner);

        vm.prank(alice);
        vm.expectRevert(abi.encodeWithSelector(IPolicyEngineStaking.OwnableUnauthorizedAccount.selector, alice));
        staking.acceptOwnership();
    }

    /// @notice Tests that pendingOwner view returns the correct value.
    function test_pendingOwner_succeeds() external {
        assertEq(staking.pendingOwner(), address(0));

        address newOwner = makeAddr("newOwner");
        vm.prank(owner);
        staking.transferOwnership(newOwner);

        assertEq(staking.pendingOwner(), newOwner);
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
        vm.expectRevert(abi.encodeWithSelector(IPolicyEngineStaking.OwnableUnauthorizedAccount.selector, _caller));
        vm.prank(_caller);
        staking.pause();
    }

    /// @notice Tests that non-owner cannot unpause.
    function testFuzz_unpause_notOwner_reverts(address _caller) external {
        vm.prank(owner);
        staking.pause();

        vm.assume(_caller != owner && _caller != address(0));
        vm.expectRevert(abi.encodeWithSelector(IPolicyEngineStaking.OwnableUnauthorizedAccount.selector, _caller));
        vm.prank(_caller);
        staking.unpause();
    }

    /// @notice Tests that stake reverts when paused.
    function test_stake_whenPaused_reverts() external {
        vm.prank(owner);
        staking.pause();

        vm.expectRevert(IPolicyEngineStaking.PolicyEngineStaking_Paused.selector);
        vm.prank(alice);
        staking.stake(uint128(100 ether), alice);
    }

    /// @notice Tests that changeBeneficiary works when paused.
    function test_changeBeneficiary_whenPaused_reverts() external {
        vm.prank(alice);
        staking.stake(uint128(100 ether), alice);
        vm.prank(bob);
        staking.setAllowedStaker(alice, true);

        vm.prank(owner);
        staking.pause();

        vm.expectRevert(IPolicyEngineStaking.PolicyEngineStaking_Paused.selector);
        vm.prank(alice);
        staking.changeBeneficiary(bob);
    }

    /// @notice Tests that unstake works when paused.
    function test_unstake_whenPaused_succeeds() external {
        vm.prank(alice);
        staking.stake(uint128(100 ether), alice);
        vm.prank(owner);
        staking.pause();

        uint256 balanceBefore = IERC20(Predeploys.GOVERNANCE_TOKEN).balanceOf(alice);
        vm.prank(alice);
        staking.unstake(uint128(100 ether));
        assertEq(IERC20(Predeploys.GOVERNANCE_TOKEN).balanceOf(alice), balanceBefore + 100 ether);
    }
}

/// @title PolicyEngineStaking_Stake_Test
/// @notice Tests the `stake` function.
contract PolicyEngineStaking_Stake_Test is PolicyEngineStaking_TestInit {
    /// @notice Tests that stake with self-attribution succeeds.
    function testFuzz_stake_selfAttribution_succeeds(uint128 _amount) external {
        _amount = uint128(bound(_amount, 1, 1_000 ether));

        vm.expectEmit(address(staking));
        emit BeneficiarySet(alice, alice);
        vm.expectEmit(address(staking));
        emit EffectiveStakeChanged(alice, _amount);
        vm.expectEmit(address(staking));
        emit Staked(alice, _amount);

        vm.prank(alice);
        staking.stake(_amount, alice);

        (uint128 staked, address beneficiary) = staking.stakingData(alice);
        (uint128 effectiveStake, uint128 lastUpdate) = staking.peData(alice);

        assertEq(staked, _amount);
        assertEq(beneficiary, alice);
        assertEq(effectiveStake, _amount);
        assertEq(lastUpdate, block.timestamp);
    }

    /// @notice Tests that multiple stake calls to same beneficiary succeed.
    function test_stake_severalToSameBeneficiary_succeeds() external {
        vm.prank(alice);
        staking.stake(uint128(100 ether), alice);
        vm.prank(alice);
        staking.stake(uint128(200 ether), alice);
        vm.prank(alice);
        staking.stake(uint128(300 ether), alice);

        (uint128 aliceStaked, address aliceBeneficiary) = staking.stakingData(alice);
        assertEq(aliceStaked, 600 ether);
        assertEq(aliceBeneficiary, alice);
        (uint128 aliceEffectiveStake, uint128 aliceLastUpdate) = staking.peData(alice);
        assertEq(aliceEffectiveStake, 600 ether);
        assertEq(aliceLastUpdate, block.timestamp);
    }

    /// @notice Tests that stake to another beneficiary with allowlist succeeds.
    function test_stake_toBeneficiaryWithAllowlist_succeeds() external {
        vm.prank(bob);
        staking.setAllowedStaker(alice, true);

        vm.expectEmit(address(staking));
        emit BeneficiarySet(alice, bob);
        vm.expectEmit(address(staking));
        emit EffectiveStakeChanged(bob, 100 ether);
        vm.expectEmit(address(staking));
        emit Staked(alice, uint128(100 ether));

        vm.prank(alice);
        staking.stake(uint128(100 ether), bob);

        (uint128 staked, address beneficiary) = staking.stakingData(alice);
        (uint128 effectiveStake, uint128 lastUpdate) = staking.peData(alice);
        assertEq(staked, 100 ether);
        assertEq(beneficiary, bob);
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
        staking.stake(uint128(100 ether), bob);

        vm.prank(alice);
        staking.stake(uint128(50 ether), bob);

        (uint128 staked,) = staking.stakingData(alice);
        assertEq(staked, 150 ether);
        (uint128 bobEffective,) = staking.peData(bob);
        assertEq(bobEffective, 150 ether);
    }

    /// @notice Tests that stake changes beneficiary atomically.
    function test_stake_changeBeneficiary_succeeds() external {
        // Alice stakes to self
        vm.prank(alice);
        staking.stake(uint128(100 ether), alice);

        (uint128 aliceEffBefore,) = staking.peData(alice);
        assertEq(aliceEffBefore, 100 ether);

        // Bob allows alice
        vm.prank(bob);
        staking.setAllowedStaker(alice, true);

        // Alice changes beneficiary to bob with additional stake
        vm.expectEmit(address(staking));
        emit EffectiveStakeChanged(alice, 0); // decrease alice's PE
        vm.expectEmit(address(staking));
        emit BeneficiaryRemoved(alice, alice);
        vm.expectEmit(address(staking));
        emit BeneficiarySet(alice, bob);
        vm.expectEmit(address(staking));
        emit EffectiveStakeChanged(bob, 150 ether); // single increase: old stake + new amount
        vm.expectEmit(address(staking));
        emit Staked(alice, uint128(50 ether));

        vm.prank(alice);
        staking.stake(uint128(50 ether), bob);

        (uint128 staked, address beneficiary) = staking.stakingData(alice);
        assertEq(staked, 150 ether);
        assertEq(beneficiary, bob);
        (uint128 aliceEffAfter,) = staking.peData(alice);
        assertEq(aliceEffAfter, 0);
        (uint128 bobEff,) = staking.peData(bob);
        assertEq(bobEff, 150 ether);
    }

    /// @notice Tests that stake with zero amount reverts.
    function test_stake_zeroAmount_reverts() external {
        vm.prank(alice);
        vm.expectRevert(IPolicyEngineStaking.PolicyEngineStaking_ZeroAmount.selector);
        staking.stake(0, alice);
    }

    /// @notice Tests that stake with zero beneficiary reverts.
    function test_stake_zeroBeneficiary_reverts() external {
        vm.prank(alice);
        vm.expectRevert(IPolicyEngineStaking.PolicyEngineStaking_ZeroBeneficiary.selector);
        staking.stake(uint128(100 ether), address(0));
    }

    /// @notice Tests that stake to beneficiary without allowlist reverts.
    function test_stake_toBeneficiaryWithoutAllowlist_reverts() external {
        vm.prank(alice);
        vm.expectRevert(IPolicyEngineStaking.PolicyEngineStaking_NotAllowedToSetBeneficiary.selector);
        staking.stake(uint128(100 ether), bob);
    }

    /// @notice Tests change beneficiary reverts without allowlist.
    function test_stake_changeBeneficiaryWithoutAllowlist_reverts() external {
        vm.prank(alice);
        staking.stake(uint128(100 ether), alice);

        vm.prank(alice);
        vm.expectRevert(IPolicyEngineStaking.PolicyEngineStaking_NotAllowedToSetBeneficiary.selector);
        staking.stake(uint128(50 ether), bob);
    }
}

/// @title PolicyEngineStaking_Unstake_Test
/// @notice Tests the `unstake` function.
contract PolicyEngineStaking_Unstake_Test is PolicyEngineStaking_TestInit {
    /// @notice Tests that full unstake succeeds, auto-clears beneficiary, and preserves balance.
    function testFuzz_unstake_full_succeeds(uint128 _amount) external {
        _amount = uint128(bound(_amount, 1, 1_000 ether));

        vm.prank(alice);
        staking.stake(_amount, alice);

        uint256 aliceBalanceBefore = IERC20(Predeploys.GOVERNANCE_TOKEN).balanceOf(alice);

        vm.expectEmit(address(staking));
        emit EffectiveStakeChanged(alice, 0);
        vm.expectEmit(address(staking));
        emit BeneficiaryRemoved(alice, alice);
        vm.expectEmit(address(staking));
        emit Unstaked(alice, _amount);

        vm.prank(alice);
        staking.unstake(_amount);

        (uint128 aliceStaked, address beneficiary) = staking.stakingData(alice);
        assertEq(aliceStaked, 0);
        assertEq(beneficiary, address(0));
        assertEq(IERC20(Predeploys.GOVERNANCE_TOKEN).balanceOf(alice), aliceBalanceBefore + _amount);
    }

    /// @notice Tests that unstake with zero amount reverts.
    function test_unstake_zeroAmount_reverts() external {
        vm.prank(alice);
        staking.stake(uint128(100 ether), alice);

        vm.prank(alice);
        vm.expectRevert(IPolicyEngineStaking.PolicyEngineStaking_ZeroAmount.selector);
        staking.unstake(0);
    }

    /// @notice Tests that unstake with no stake reverts.
    function test_unstake_noStake_reverts() external {
        vm.prank(alice);
        vm.expectRevert(IPolicyEngineStaking.PolicyEngineStaking_InsufficientStake.selector);
        staking.unstake(uint128(100 ether));
    }

    /// @notice Tests that unstake more than staked reverts.
    function test_unstake_insufficientStake_reverts() external {
        vm.prank(alice);
        staking.stake(uint128(100 ether), alice);

        vm.prank(alice);
        vm.expectRevert(IPolicyEngineStaking.PolicyEngineStaking_InsufficientStake.selector);
        staking.unstake(uint128(101 ether));
    }

    /// @notice Tests partial unstake preserves correct remaining balance.
    function testFuzz_unstake_partialAmount_succeeds(uint128 _stakeAmount, uint128 _unstakeAmount) external {
        _stakeAmount = uint128(bound(_stakeAmount, 2, 1_000 ether));
        _unstakeAmount = uint128(bound(_unstakeAmount, 1, _stakeAmount - 1));

        vm.prank(alice);
        staking.stake(_stakeAmount, alice);

        uint128 remaining = _stakeAmount - _unstakeAmount;

        vm.expectEmit(address(staking));
        emit EffectiveStakeChanged(alice, remaining);
        vm.expectEmit(address(staking));
        emit Unstaked(alice, _unstakeAmount);

        vm.prank(alice);
        staking.unstake(_unstakeAmount);

        (uint128 staked, address beneficiary) = staking.stakingData(alice);
        assertEq(staked, remaining);
        assertEq(beneficiary, alice);
        (uint128 effective,) = staking.peData(alice);
        assertEq(effective, remaining);
    }

    /// @notice Tests partial unstake with beneficiary preserves remaining stake attribution.
    function testFuzz_unstake_partialWithBeneficiary_succeeds(uint128 _stakeAmount, uint128 _unstakeAmount) external {
        _stakeAmount = uint128(bound(_stakeAmount, 2, 1_000 ether));
        _unstakeAmount = uint128(bound(_unstakeAmount, 1, _stakeAmount - 1));

        vm.prank(bob);
        staking.setAllowedStaker(alice, true);

        vm.prank(alice);
        staking.stake(_stakeAmount, bob);

        uint128 remaining = _stakeAmount - _unstakeAmount;

        vm.expectEmit(address(staking));
        emit EffectiveStakeChanged(bob, remaining);
        vm.expectEmit(address(staking));
        emit Unstaked(alice, _unstakeAmount);

        vm.prank(alice);
        staking.unstake(_unstakeAmount);

        (uint128 staked, address beneficiary) = staking.stakingData(alice);
        assertEq(staked, remaining);
        assertEq(beneficiary, bob);
        (uint128 bobEffective,) = staking.peData(bob);
        assertEq(bobEffective, remaining);
    }
}

/// @title PolicyEngineStaking_ChangeBeneficiary_Test
/// @notice Tests the `changeBeneficiary` function.
contract PolicyEngineStaking_ChangeBeneficiary_Test is PolicyEngineStaking_TestInit {
    /// @notice Tests that changing beneficiary succeeds.
    function testFuzz_changeBeneficiary_succeeds(uint128 _amount) external {
        _amount = uint128(bound(_amount, 1, 1_000 ether));
        vm.prank(alice);
        staking.stake(_amount, alice);

        vm.prank(bob);
        staking.setAllowedStaker(alice, true);

        vm.expectEmit(address(staking));
        emit EffectiveStakeChanged(alice, 0);
        vm.expectEmit(address(staking));
        emit BeneficiaryRemoved(alice, alice);
        vm.expectEmit(address(staking));
        emit EffectiveStakeChanged(bob, _amount);
        vm.expectEmit(address(staking));
        emit BeneficiarySet(alice, bob);

        vm.prank(alice);
        staking.changeBeneficiary(bob);

        (uint128 staked, address beneficiary) = staking.stakingData(alice);
        assertEq(staked, _amount);
        assertEq(beneficiary, bob);
        (uint128 aliceEffective,) = staking.peData(alice);
        assertEq(aliceEffective, 0);
        (uint128 bobEffective,) = staking.peData(bob);
        assertEq(bobEffective, _amount);
    }

    /// @notice Tests that changing from one beneficiary to another succeeds.
    function test_changeBeneficiary_fromOneToAnother_succeeds() external {
        vm.prank(bob);
        staking.setAllowedStaker(alice, true);
        vm.prank(alice);
        staking.stake(uint128(100 ether), bob);

        vm.prank(carol);
        staking.setAllowedStaker(alice, true);

        vm.prank(alice);
        staking.changeBeneficiary(carol);

        (, address beneficiary) = staking.stakingData(alice);
        assertEq(beneficiary, carol);
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
        staking.stake(uint128(100 ether), bob);

        vm.prank(alice);
        staking.changeBeneficiary(alice);

        (, address beneficiary) = staking.stakingData(alice);
        assertEq(beneficiary, alice);
        (uint128 aliceEffective,) = staking.peData(alice);
        assertEq(aliceEffective, 100 ether);
        (uint128 bobEffective,) = staking.peData(bob);
        assertEq(bobEffective, 0);
    }

    /// @notice Tests that changing to same beneficiary reverts.
    function test_changeBeneficiary_sameBeneficiary_reverts() external {
        vm.prank(alice);
        staking.stake(uint128(100 ether), alice);

        vm.prank(alice);
        vm.expectRevert(IPolicyEngineStaking.PolicyEngineStaking_SameBeneficiary.selector);
        staking.changeBeneficiary(alice);
    }

    /// @notice Tests that changeBeneficiary with zero beneficiary reverts.
    function test_changeBeneficiary_zeroBeneficiary_reverts() external {
        vm.prank(alice);
        staking.stake(uint128(100 ether), alice);

        vm.prank(alice);
        vm.expectRevert(IPolicyEngineStaking.PolicyEngineStaking_ZeroBeneficiary.selector);
        staking.changeBeneficiary(address(0));
    }

    /// @notice Tests that changeBeneficiary without allowlist reverts.
    function test_changeBeneficiary_notAllowed_reverts() external {
        vm.prank(alice);
        staking.stake(uint128(100 ether), alice);

        vm.prank(alice);
        vm.expectRevert(IPolicyEngineStaking.PolicyEngineStaking_NotAllowedToSetBeneficiary.selector);
        staking.changeBeneficiary(bob);
    }

    /// @notice Tests that changeBeneficiary with no stake reverts.
    function test_changeBeneficiary_noStake_reverts() external {
        vm.prank(bob);
        staking.setAllowedStaker(alice, true);

        vm.prank(alice);
        vm.expectRevert(IPolicyEngineStaking.PolicyEngineStaking_NoStake.selector);
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
        uint128 amount = uint128(100 ether);
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

    /// @notice Tests that the constructor reverts when owner is zero address.
    function test_constructor_zeroOwner_reverts() external {
        vm.expectRevert(abi.encodeWithSelector(IPolicyEngineStaking.OwnableInvalidOwner.selector, address(0)));
        vm.deployCode(
            "PolicyEngineStaking.sol:PolicyEngineStaking", abi.encode(address(0), Predeploys.GOVERNANCE_TOKEN)
        );
    }

    /// @notice Tests that the constructor reverts when token is zero address.
    function test_constructor_zeroToken_reverts() external {
        vm.expectRevert(IPolicyEngineStaking.PolicyEngineStaking_ZeroAddress.selector);
        vm.deployCode("PolicyEngineStaking.sol:PolicyEngineStaking", abi.encode(owner, address(0)));
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

    /// @notice Tests that setAllowedStaker reverts when staker is msg.sender.
    function test_setAllowedStaker_selfAllowlist_reverts() external {
        vm.prank(bob);
        vm.expectRevert(IPolicyEngineStaking.PolicyEngineStaking_SelfAllowlist.selector);
        staking.setAllowedStaker(bob, true);
    }
}

/// @title PolicyEngineStaking_Integration_Test
/// @notice Integration tests for the full stake/changeBeneficiary/unstake flow.
contract PolicyEngineStaking_Integration_Test is PolicyEngineStaking_TestInit {
    /// @notice Tests full flow: stake -> stake more -> changeBeneficiary -> partial unstake -> full unstake.
    function test_fullFlow_succeeds() external {
        // Step 1: Alice stakes 100 to self
        vm.prank(alice);
        staking.stake(uint128(100 ether), alice);
        (uint128 staked,) = staking.stakingData(alice);
        assertEq(staked, 100 ether);

        // Step 2: Alice stakes 50 more
        vm.prank(alice);
        staking.stake(uint128(50 ether), alice);
        (staked,) = staking.stakingData(alice);
        assertEq(staked, 150 ether);

        // Step 3: Alice changes beneficiary to bob
        vm.prank(bob);
        staking.setAllowedStaker(alice, true);
        vm.prank(alice);
        staking.changeBeneficiary(bob);

        (, address beneficiary) = staking.stakingData(alice);
        assertEq(beneficiary, bob);
        (uint128 bobEff,) = staking.peData(bob);
        assertEq(bobEff, 150 ether);
        (uint128 aliceEff,) = staking.peData(alice);
        assertEq(aliceEff, 0);

        // Step 4: Partial unstake
        vm.prank(alice);
        staking.unstake(uint128(50 ether));
        (staked, beneficiary) = staking.stakingData(alice);
        assertEq(staked, 100 ether);
        assertEq(beneficiary, bob);
        (bobEff,) = staking.peData(bob);
        assertEq(bobEff, 100 ether);

        // Step 5: Full unstake (auto-unlinks)
        uint256 aliceBalanceBefore = IERC20(Predeploys.GOVERNANCE_TOKEN).balanceOf(alice);
        vm.prank(alice);
        staking.unstake(uint128(100 ether));
        assertEq(IERC20(Predeploys.GOVERNANCE_TOKEN).balanceOf(alice), aliceBalanceBefore + 100 ether);
        (staked, beneficiary) = staking.stakingData(alice);
        assertEq(staked, 0);
        assertEq(beneficiary, address(0));
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
        staking.stake(uint128(100 ether), bob);
        vm.prank(carol);
        staking.stake(uint128(50 ether), bob);

        (uint128 bobEffective,) = staking.peData(bob);
        assertEq(bobEffective, 150 ether);
    }

    /// @notice Tests that a beneficiary with own stake plus received stake has correct effective stake.
    function test_beneficiaryWithOwnStakeAndReceived_succeeds() external {
        vm.prank(bob);
        staking.stake(uint128(50 ether), bob);
        vm.prank(bob);
        staking.setAllowedStaker(alice, true);
        vm.prank(alice);
        staking.stake(uint128(100 ether), bob);

        (uint128 bobStaked,) = staking.stakingData(bob);
        assertEq(bobStaked, 50 ether);
        (uint128 bobEffective,) = staking.peData(bob);
        assertEq(bobEffective, 150 ether);
    }

    /// @notice Tests that revoking allowlist auto-resets beneficiary to self.
    function test_revokeAllowlist_resetsBeneficiaryToSelf_succeeds() external {
        vm.prank(alice);
        staking.setAllowedStaker(bob, true);
        vm.prank(bob);
        staking.stake(uint128(100 ether), alice);

        (uint128 bobStaked, address bobBeneficiary) = staking.stakingData(bob);
        (uint128 aliceEffective,) = staking.peData(alice);
        assertEq(bobStaked, 100 ether);
        assertEq(bobBeneficiary, alice);
        assertEq(aliceEffective, 100 ether);

        // Alice revokes bob from allowlist
        vm.expectEmit(address(staking));
        emit BeneficiaryAllowlistUpdated(alice, bob, false);

        // Bob is unlinked from Alice
        vm.expectEmit(address(staking));
        emit EffectiveStakeChanged(alice, 0);
        vm.expectEmit(address(staking));
        emit BeneficiaryRemoved(bob, alice);

        // Bob is linked to self
        vm.expectEmit(address(staking));
        emit EffectiveStakeChanged(bob, 100 ether);
        vm.expectEmit(address(staking));
        emit BeneficiarySet(bob, bob);

        vm.prank(alice);
        staking.setAllowedStaker(bob, false);

        // Bob is now linked to self, alice's effective stake is zeroed
        (bobStaked, bobBeneficiary) = staking.stakingData(bob);
        (aliceEffective,) = staking.peData(alice);
        (uint128 bobEffective,) = staking.peData(bob);
        assertEq(bobStaked, 100 ether);
        assertEq(bobBeneficiary, bob);
        assertEq(aliceEffective, 0);
        assertEq(bobEffective, 100 ether);

        // Bob fully unstakes
        vm.prank(bob);
        staking.unstake(uint128(100 ether));

        (bobStaked, bobBeneficiary) = staking.stakingData(bob);
        (bobEffective,) = staking.peData(bob);
        assertEq(bobStaked, 0);
        assertEq(bobBeneficiary, address(0));
        assertEq(bobEffective, 0);
    }

    /// @notice Tests that stake to a beneficiary reverts after the beneficiary revokes allowlist.
    function test_stake_afterAllowlistRevoked_reverts() external {
        vm.prank(alice);
        staking.setAllowedStaker(bob, true);
        vm.prank(bob);
        staking.stake(uint128(100 ether), alice);

        // Alice revokes bob
        vm.prank(alice);
        staking.setAllowedStaker(bob, false);

        // Bob tries to stake to alice again
        vm.prank(bob);
        vm.expectRevert(IPolicyEngineStaking.PolicyEngineStaking_NotAllowedToSetBeneficiary.selector);
        staking.stake(uint128(50 ether), alice);
    }

    /// @notice Tests that lastUpdate is updated after new staking and linking when time advances.
    function test_lastUpdate_updatesAfterStakingAndLinking_succeeds() external {
        // Initial stake
        vm.prank(alice);
        staking.stake(uint128(100 ether), alice);
        (, uint128 lastUpdate0) = staking.peData(alice);
        uint256 ts0 = block.timestamp;
        assertEq(lastUpdate0, ts0);

        // Warp time and stake again; lastUpdate should advance
        vm.warp(block.timestamp + 1);
        vm.prank(alice);
        staking.stake(uint128(50 ether), alice);
        (, uint128 lastUpdate1) = staking.peData(alice);
        assertEq(lastUpdate1, ts0 + 1);

        // Warp time and change beneficiary to bob; bob's lastUpdate should be the new timestamp
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
        staking.stake(uint128(100 ether), alice);
        vm.prank(alice);
        staking.unstake(uint128(100 ether));

        (uint128 staked, address beneficiary) = staking.stakingData(alice);
        assertEq(staked, 0);
        assertEq(beneficiary, address(0));

        // Re-enter with a different beneficiary
        vm.prank(bob);
        staking.setAllowedStaker(alice, true);
        vm.prank(alice);
        staking.stake(uint128(50 ether), bob);

        (staked, beneficiary) = staking.stakingData(alice);
        assertEq(staked, 50 ether);
        assertEq(beneficiary, bob);
        (uint128 bobEffective,) = staking.peData(bob);
        assertEq(bobEffective, 50 ether);
    }

    /// @notice Tests stake to beneficiary and full unstake preserves staker balance.
    function testFuzz_stakeToBeneficiaryAndUnstake_succeeds(uint128 _amount) external {
        _amount = uint128(bound(_amount, 1, 1_000 ether));

        vm.prank(bob);
        staking.setAllowedStaker(alice, true);

        uint256 balanceBefore = IERC20(Predeploys.GOVERNANCE_TOKEN).balanceOf(alice);
        vm.prank(alice);
        staking.stake(_amount, bob);
        vm.prank(alice);
        staking.unstake(_amount);
        uint256 balanceAfter = IERC20(Predeploys.GOVERNANCE_TOKEN).balanceOf(alice);

        assertEq(balanceAfter, balanceBefore);
        (uint128 aliceStaked, address beneficiary) = staking.stakingData(alice);
        assertEq(aliceStaked, 0);
        assertEq(beneficiary, address(0));
        (uint128 bobEffective,) = staking.peData(bob);
        assertEq(bobEffective, 0);
    }

    /// @notice Tests stake -> change beneficiary -> unstake full cycle.
    function testFuzz_beneficiaryCycle_succeeds(uint128 _amount) external {
        _amount = uint128(bound(_amount, 1, 1_000 ether));

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
        uint128 _amount1,
        uint128 _amount2,
        uint128 _amount3
    )
        external
    {
        _amount1 = uint128(bound(_amount1, 1, 300 ether));
        _amount2 = uint128(bound(_amount2, 1, 300 ether));
        _amount3 = uint128(bound(_amount3, 1, 300 ether));

        uint128 total = _amount1 + _amount2 + _amount3;

        uint256 balanceBefore = IERC20(Predeploys.GOVERNANCE_TOKEN).balanceOf(alice);
        vm.prank(alice);
        staking.stake(_amount1, alice);
        vm.prank(alice);
        staking.stake(_amount2, alice);
        vm.prank(alice);
        staking.stake(_amount3, alice);

        (uint128 staked,) = staking.stakingData(alice);
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
        uint128 _amount
    )
        external
    {
        address[] memory accounts = _accounts();
        _stakerIdx = uint8(bound(_stakerIdx, 0, 2));
        _beneficiaryIdx = uint8(bound(_beneficiaryIdx, 0, 2));
        if (_stakerIdx == _beneficiaryIdx) return; // self-attribution, skip
        address staker = accounts[_stakerIdx];
        address beneficiary = accounts[_beneficiaryIdx];
        _amount = uint128(bound(_amount, 1, 300 ether));

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

/// @title PolicyEngineStaking_Unstake_LastUpdate_Test
/// @notice Tests lastUpdate behavior on partial vs full unstake.
contract PolicyEngineStaking_Unstake_LastUpdate_Test is PolicyEngineStaking_TestInit {
    /// @notice Tests that partial unstake preserves lastUpdate (does not reset it).
    function test_unstake_partialPreservesLastUpdate_succeeds() external {
        uint128 stakeAmount = uint128(100 ether);
        uint128 unstakeAmount = uint128(40 ether);

        vm.prank(alice);
        staking.stake(stakeAmount, alice);
        (, uint128 lastUpdateAfterStake) = staking.peData(alice);

        // Warp time forward
        vm.warp(block.timestamp + 1000);

        // Partial unstake — lastUpdate should NOT change
        vm.prank(alice);
        staking.unstake(unstakeAmount);

        (uint128 effectiveStake, uint128 lastUpdateAfterUnstake) = staking.peData(alice);
        assertEq(effectiveStake, stakeAmount - unstakeAmount);
        assertEq(lastUpdateAfterUnstake, lastUpdateAfterStake, "lastUpdate should be preserved on partial unstake");
    }

    /// @notice Tests that full unstake resets lastUpdate to block.timestamp.
    function test_unstake_fullResetsLastUpdate_succeeds() external {
        uint128 stakeAmount = uint128(100 ether);

        vm.prank(alice);
        staking.stake(stakeAmount, alice);

        // Warp time forward
        uint256 newTimestamp = block.timestamp + 1000;
        vm.warp(newTimestamp);

        // Full unstake — lastUpdate should reset to block.timestamp
        vm.prank(alice);
        staking.unstake(stakeAmount);

        (, uint128 lastUpdateAfterUnstake) = staking.peData(alice);
        assertEq(lastUpdateAfterUnstake, newTimestamp, "lastUpdate should reset on full unstake");
    }
}
