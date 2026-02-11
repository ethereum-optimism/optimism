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

/// @title PolicyEngineStaking_TestHarness
/// @notice Extends PolicyEngineStaking to expose internal _stakingData for testing.
contract PolicyEngineStaking_ForTest is PolicyEngineStaking {
    constructor(address _owner) PolicyEngineStaking(_owner) { }

    /// @notice Exposes _stakingData for tests. Not in production contract.
    function stakingData(address _account)
        external
        view
        returns (uint256 stakedAmount, uint256 receivedStake, address linkedTo)
    {
        StakedData memory d = _stakingData[_account];
        return (d.stakedAmount, d.receivedStake, d.linkedTo);
    }
}

/// @title PolicyEngineStaking_TestInit
/// @notice Reusable test initialization for `PolicyEngineStaking` tests.
abstract contract PolicyEngineStaking_TestInit is CommonTest {
    address internal carol = address(0xC4101);

    PolicyEngineStaking_ForTest internal staking;
    address internal owner;

    event Staked(address indexed account, address indexed beneficiary, uint256 amount);
    event Unstaked(address indexed account, uint256 amount);
    event Linked(address indexed staker, address indexed beneficiary);
    event Unlinked(address indexed staker, address indexed previousBeneficiary);
    event BeneficiaryAllowlistUpdated(address indexed beneficiary, address indexed staker, bool allowed);
    event Paused();
    event Unpaused();

    function setUp() public virtual override {
        super.setUp();
        owner = makeAddr("owner");
        staking = new PolicyEngineStaking_ForTest(owner);

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
    }

    /// @notice Approves the staking contract and stakes tokens.
    function _stake(address _account, uint256 _amount, address _beneficiary) internal {
        vm.prank(_account);
        IERC20(Predeploys.GOVERNANCE_TOKEN).approve(address(staking), _amount);
        vm.prank(_account);
        staking.stake(_amount, _beneficiary);
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
    function test_pause_notOwner_reverts() external {
        vm.expectRevert(PolicyEngineStaking.PolicyEngineStaking_OnlyOwner.selector);
        vm.prank(alice);
        staking.pause();
    }

    /// @notice Tests that non-owner cannot unpause.
    function test_unpause_notOwner_reverts() external {
        vm.prank(owner);
        staking.pause();

        vm.expectRevert(PolicyEngineStaking.PolicyEngineStaking_OnlyOwner.selector);
        vm.prank(alice);
        staking.unpause();
    }

    /// @notice Tests that stake reverts when paused.
    function test_stake_whenPaused_reverts() external {
        vm.prank(owner);
        staking.pause();

        vm.prank(alice);
        IERC20(Predeploys.GOVERNANCE_TOKEN).approve(address(staking), 100 ether);

        vm.expectRevert(PolicyEngineStaking.PolicyEngineStaking_Paused.selector);
        vm.prank(alice);
        staking.stake(100 ether, address(0));
    }

    /// @notice Tests that link reverts when paused.
    function test_link_whenPaused_reverts() external {
        _stake(alice, 100 ether, address(0));
        vm.prank(bob);
        staking.setAllowedStaker(alice, true);

        vm.prank(owner);
        staking.pause();

        vm.expectRevert(PolicyEngineStaking.PolicyEngineStaking_Paused.selector);
        vm.prank(alice);
        staking.link(bob);
    }

    /// @notice Tests that unstake works when paused.
    function test_unstake_whenPaused_succeeds() external {
        _stake(alice, 100 ether, address(0));
        vm.prank(owner);
        staking.pause();

        uint256 balanceBefore = IERC20(Predeploys.GOVERNANCE_TOKEN).balanceOf(alice);
        vm.prank(alice);
        staking.unstake();
        assertEq(IERC20(Predeploys.GOVERNANCE_TOKEN).balanceOf(alice), balanceBefore + 100 ether);
    }

    /// @notice Tests that unlink works when paused.
    function test_unlink_whenPaused_succeeds() external {
        _stake(alice, 100 ether, address(0));
        vm.prank(bob);
        staking.setAllowedStaker(alice, true);
        vm.prank(alice);
        staking.link(bob);

        vm.prank(owner);
        staking.pause();

        vm.prank(alice);
        staking.unlink();

        (uint256 staked,, address linkedTo) = staking.stakingData(alice);
        assertEq(staked, 100 ether);
        assertEq(linkedTo, address(0));
    }
}

/// @title PolicyEngineStaking_Stake_Test
/// @notice Tests the `stake` function of the `PolicyEngineStaking` contract.
contract PolicyEngineStaking_Stake_Test is PolicyEngineStaking_TestInit {
    /// @notice Tests that staking with self-attribution succeeds.
    function test_stake_selfAttribution_succeeds() external {
        uint256 amount = 100 ether;

        vm.prank(alice);
        IERC20(Predeploys.GOVERNANCE_TOKEN).approve(address(staking), amount);

        vm.expectEmit(address(staking));
        emit Staked(alice, address(0), amount);

        vm.prank(alice);
        staking.stake(amount, address(0));

        (uint256 staked, uint256 received, address linkedTo) = staking.stakingData(alice);
        (uint128 effectiveStake, uint128 lastUpdate) = staking.peData(alice);

        assertEq(staked, amount);
        assertEq(received, 0);
        assertEq(linkedTo, address(0));
        assertEq(effectiveStake, amount);
        assertEq(lastUpdate, block.timestamp);
    }

    function test_stake_severalSelfAttributions_succeeds() external {
        _stake(alice, 100 ether, address(0));
        _stake(alice, 200 ether, address(0));
        _stake(alice, 300 ether, address(0));

        (uint256 aliceStaked, uint256 aliceReceived,) = staking.stakingData(alice);
        assertEq(aliceStaked, 600 ether);
        assertEq(aliceReceived, 0);
        (uint128 aliceEffectiveStake, uint128 aliceLastUpdate) = staking.peData(alice);
        assertEq(aliceEffectiveStake, 600 ether);
        assertEq(aliceLastUpdate, block.timestamp);
    }

    /// @notice Tests that staking to another beneficiary with allowlist succeeds.
    function test_stake_toBeneficiaryWithAllowlist_succeeds() external {
        vm.prank(bob);
        staking.setAllowedStaker(alice, true);

        vm.prank(alice);
        IERC20(Predeploys.GOVERNANCE_TOKEN).approve(address(staking), 100 ether);

        vm.expectEmit(address(staking));
        emit Staked(alice, bob, 100 ether);

        vm.prank(alice);
        staking.stake(100 ether, bob);

        (uint256 staked, uint256 received, address linkedTo) = staking.stakingData(alice);
        (uint128 effectiveStake, uint128 lastUpdate) = staking.peData(alice);
        assertEq(staked, 100 ether);
        assertEq(received, 0);
        assertEq(linkedTo, bob);
        assertEq(effectiveStake, 0);
        assertEq(lastUpdate, 0);

        (uint256 bobStaked, uint256 bobReceived,) = staking.stakingData(bob);
        assertEq(bobStaked, 0);
        assertEq(bobReceived, 100 ether);
        (uint128 bobEffectiveStake, uint128 bobLastUpdate) = staking.peData(bob);
        assertEq(bobEffectiveStake, 100 ether);
        assertEq(bobLastUpdate, block.timestamp);
    }

    /// @notice Tests that staking more to the same beneficiary when linked succeeds.
    function test_stake_moreToSameBeneficiary_succeeds() external {
        vm.prank(bob);
        staking.setAllowedStaker(alice, true);
        _stake(alice, 100 ether, bob);

        vm.prank(alice);
        IERC20(Predeploys.GOVERNANCE_TOKEN).approve(address(staking), 50 ether);
        vm.prank(alice);
        staking.stake(50 ether, bob);

        (uint256 staked,,) = staking.stakingData(alice);
        assertEq(staked, 150 ether);
        (, uint256 bobReceived,) = staking.stakingData(bob);
        assertEq(bobReceived, 150 ether);
        (uint128 bobEffective,) = staking.peData(bob);
        assertEq(bobEffective, 150 ether);
    }

    /// @notice Verifies stake succeeds with various amounts.
    function testFuzz_stake_amount_succeeds(uint256 _amount) external {
        _amount = bound(_amount, 1, 1_000 ether);

        vm.prank(alice);
        IERC20(Predeploys.GOVERNANCE_TOKEN).approve(address(staking), _amount);

        vm.prank(alice);
        staking.stake(_amount, address(0));

        (uint128 effectiveStake, uint128 lastUpdate) = staking.peData(alice);
        assertEq(effectiveStake, _amount);
        assertEq(lastUpdate, block.timestamp);
    }

    /// @notice Tests that staking with zero amount reverts.
    function test_stake_zeroAmount_reverts() external {
        vm.prank(alice);
        vm.expectRevert(PolicyEngineStaking.PolicyEngineStaking_ZeroAmount.selector);
        staking.stake(0, address(0));
    }

    /// @notice Tests that staking to self (msg.sender as beneficiary) reverts.
    function test_stake_toSelf_reverts() external {
        vm.prank(alice);
        IERC20(Predeploys.GOVERNANCE_TOKEN).approve(address(staking), 100 ether);

        vm.prank(alice);
        vm.expectRevert(PolicyEngineStaking.PolicyEngineStaking_CannotLinkToSelf.selector);
        staking.stake(100 ether, alice);
    }

    /// @notice Tests that staking to another beneficiary without allowlist reverts.
    function test_stake_toBeneficiaryWithoutAllowlist_reverts() external {
        vm.prank(alice);
        IERC20(Predeploys.GOVERNANCE_TOKEN).approve(address(staking), 100 ether);

        vm.prank(alice);
        vm.expectRevert(PolicyEngineStaking.PolicyEngineStaking_NotAllowedToLink.selector);
        staking.stake(100 ether, bob);
    }

    /// @notice Tests that staking with self-attribution when already linked reverts.
    function test_stake_selfAttributionWhenLinked_reverts() external {
        vm.prank(bob);
        staking.setAllowedStaker(alice, true);
        _stake(alice, 100 ether, bob);

        vm.prank(alice);
        IERC20(Predeploys.GOVERNANCE_TOKEN).approve(address(staking), 100 ether);

        vm.prank(alice);
        vm.expectRevert(PolicyEngineStaking.PolicyEngineStaking_AlreadyLinked.selector);
        staking.stake(100 ether, address(0));
    }

    /// @notice Tests that staking to beneficiary with existing self-stake reverts (must link or unstake first).
    function test_stake_mustLinkOrUnstakeFirst_reverts() external {
        _stake(alice, 100 ether, address(0));

        vm.prank(bob);
        staking.setAllowedStaker(alice, true);

        vm.prank(alice);
        IERC20(Predeploys.GOVERNANCE_TOKEN).approve(address(staking), 50 ether);

        vm.prank(alice);
        vm.expectRevert(PolicyEngineStaking.PolicyEngineStaking_MustLinkOrUnstakeFirst.selector);
        staking.stake(50 ether, bob);
    }

    /// @notice Tests that staking to different beneficiary when linked reverts.
    function test_stake_differentBeneficiaryWhenLinked_reverts() external {
        vm.prank(bob);
        staking.setAllowedStaker(alice, true);
        _stake(alice, 100 ether, bob);

        vm.prank(carol);
        staking.setAllowedStaker(alice, true);
        vm.prank(alice);
        IERC20(Predeploys.GOVERNANCE_TOKEN).approve(address(staking), 50 ether);

        vm.prank(alice);
        vm.expectRevert(PolicyEngineStaking.PolicyEngineStaking_AlreadyLinked.selector);
        staking.stake(50 ether, carol);
    }

    /// @notice Tests that staking to beneficiary not in allowlist reverts.
    function test_stake_notAllowedToLink_reverts() external {
        vm.prank(alice);
        IERC20(Predeploys.GOVERNANCE_TOKEN).approve(address(staking), 100 ether);

        vm.prank(alice);
        vm.expectRevert(PolicyEngineStaking.PolicyEngineStaking_NotAllowedToLink.selector);
        staking.stake(100 ether, carol);
    }

    /// @notice Tests that staking more fails when beneficiary removes staker from allowlist.
    function test_stake_beneficiaryRemovedFromAllowlist_reverts() external {
        vm.prank(bob);
        staking.setAllowedStaker(alice, true);
        _stake(alice, 100 ether, bob);

        vm.prank(bob);
        staking.setAllowedStaker(alice, false);

        vm.prank(alice);
        IERC20(Predeploys.GOVERNANCE_TOKEN).approve(address(staking), 50 ether);
        vm.prank(alice);
        vm.expectRevert(PolicyEngineStaking.PolicyEngineStaking_NotAllowedToLink.selector);
        staking.stake(50 ether, bob);
    }

    /// @notice Tests that staking with amount > type(uint128).max reverts (effectiveStake limit).
    function test_stake_amountExceedsUint128_reverts() external {
        uint256 excessAmount = type(uint128).max;
        excessAmount++;
        // Revert happens in _increasePeData before transfer; no need to mint (avoids token overflow).

        vm.prank(alice);
        vm.expectRevert(PolicyEngineStaking.PolicyEngineStaking_AmountExceedsEffectiveStakeLimit.selector);
        staking.stake(excessAmount, address(0));
    }

    /// @notice Tests that staking with amount == type(uint128).max succeeds (boundary).
    function test_stake_amountEqualsUint128Max_succeeds() external {
        uint256 maxAmount = type(uint128).max;
        TestERC20(Predeploys.GOVERNANCE_TOKEN).mint(alice, maxAmount);

        vm.prank(alice);
        IERC20(Predeploys.GOVERNANCE_TOKEN).approve(address(staking), maxAmount);

        vm.prank(alice);
        staking.stake(maxAmount, address(0));

        (uint128 effectiveStake,) = staking.peData(alice);
        assertEq(effectiveStake, maxAmount);
    }
}

/// @title PolicyEngineStaking_Unstake_Test
/// @notice Tests the `unstake` function of the `PolicyEngineStaking` contract.
contract PolicyEngineStaking_Unstake_Test is PolicyEngineStaking_TestInit {
    /// @notice Tests that unstaking succeeds.
    function test_unstake_succeeds() external {
        _stake(alice, 100 ether, address(0));

        uint256 aliceBalanceBefore = IERC20(Predeploys.GOVERNANCE_TOKEN).balanceOf(alice);

        vm.expectEmit(address(staking));
        emit Unstaked(alice, 100 ether);

        vm.prank(alice);
        staking.unstake();

        (uint256 aliceStaked,,) = staking.stakingData(alice);
        assertEq(aliceStaked, 0);
        assertEq(IERC20(Predeploys.GOVERNANCE_TOKEN).balanceOf(alice), aliceBalanceBefore + 100 ether);
    }

    /// @notice Tests that unstaking when linked returns tokens to staker.
    function test_unstake_whenLinked_succeeds() external {
        vm.prank(bob);
        staking.setAllowedStaker(alice, true);
        _stake(alice, 100 ether, bob);

        uint256 aliceBalanceBefore = IERC20(Predeploys.GOVERNANCE_TOKEN).balanceOf(alice);

        vm.prank(alice);
        staking.unstake();

        (uint256 aliceStaked,,) = staking.stakingData(alice);
        assertEq(aliceStaked, 0);
        assertEq(IERC20(Predeploys.GOVERNANCE_TOKEN).balanceOf(alice), aliceBalanceBefore + 100 ether);
        (, uint256 bobReceived,) = staking.stakingData(bob);
        assertEq(bobReceived, 0);
    }

    /// @notice Tests that unstaking with no stake reverts.
    function test_unstake_noStake_reverts() external {
        vm.prank(alice);
        vm.expectRevert(PolicyEngineStaking.PolicyEngineStaking_NoStake.selector);
        staking.unstake();
    }

    /// @notice Tests that unstaking after staking type(uint128).max succeeds (_decreasePeData at boundary).
    function test_unstake_afterStakingUint128Max_succeeds() external {
        uint256 maxAmount = type(uint128).max;
        TestERC20(Predeploys.GOVERNANCE_TOKEN).mint(alice, maxAmount);

        vm.prank(alice);
        IERC20(Predeploys.GOVERNANCE_TOKEN).approve(address(staking), maxAmount);
        vm.prank(alice);
        staking.stake(maxAmount, address(0));

        (uint128 effectiveBefore,) = staking.peData(alice);
        assertEq(effectiveBefore, maxAmount);

        uint256 balanceBefore = IERC20(Predeploys.GOVERNANCE_TOKEN).balanceOf(alice);
        vm.prank(alice);
        staking.unstake();

        (uint256 staked,,) = staking.stakingData(alice);
        (uint128 effectiveAfter,) = staking.peData(alice);
        assertEq(staked, 0);
        assertEq(effectiveAfter, 0);
        assertEq(IERC20(Predeploys.GOVERNANCE_TOKEN).balanceOf(alice), balanceBefore + maxAmount);
    }
}

/// @title PolicyEngineStaking_Link_Test
/// @notice Tests the `link` function of the `PolicyEngineStaking` contract.
contract PolicyEngineStaking_Link_Test is PolicyEngineStaking_TestInit {
    /// @notice Tests that linking to beneficiary succeeds.
    function test_link_succeeds() external {
        uint256 amount = 100 ether;
        _stake(alice, amount, address(0));

        vm.prank(bob);
        staking.setAllowedStaker(alice, true);

        vm.expectEmit(address(staking));
        emit Linked(alice, bob);

        vm.prank(alice);
        staking.link(bob);

        (, uint256 bobReceived,) = staking.stakingData(bob);
        assertEq(bobReceived, amount);
    }

    /// @notice Tests that linking to self reverts (would break receivedStake accounting).
    function test_link_linkToSelf_reverts() external {
        _stake(alice, 100 ether, address(0));

        vm.prank(alice);
        vm.expectRevert(PolicyEngineStaking.PolicyEngineStaking_CannotLinkToSelf.selector);
        staking.link(alice);
    }

    /// @notice Tests that linking with zero beneficiary reverts.
    function test_link_zeroBeneficiary_reverts() external {
        _stake(alice, 100 ether, address(0));

        vm.prank(alice);
        vm.expectRevert(PolicyEngineStaking.PolicyEngineStaking_ZeroBeneficiary.selector);
        staking.link(address(0));
    }

    /// @notice Tests that linking without allowlist reverts.
    function test_link_notAllowed_reverts() external {
        _stake(alice, 100 ether, address(0));

        vm.prank(alice);
        vm.expectRevert(PolicyEngineStaking.PolicyEngineStaking_NotAllowedToLink.selector);
        staking.link(bob);
    }

    /// @notice Tests that linking with no stake reverts.
    function test_link_noStake_reverts() external {
        vm.prank(alice);
        vm.expectRevert(PolicyEngineStaking.PolicyEngineStaking_NoStake.selector);
        staking.link(bob);
    }

    /// @notice Tests that linking when already linked reverts.
    function test_link_alreadyLinked_reverts() external {
        vm.prank(bob);
        staking.setAllowedStaker(alice, true);
        _stake(alice, 100 ether, bob);

        vm.prank(alice);
        vm.expectRevert(PolicyEngineStaking.PolicyEngineStaking_AlreadyLinked.selector);
        staking.link(carol);
    }
}

/// @title PolicyEngineStaking_Unlink_Test
/// @notice Tests the `unlink` function of the `PolicyEngineStaking` contract.
contract PolicyEngineStaking_Unlink_Test is PolicyEngineStaking_TestInit {
    /// @notice Tests that unlinking succeeds.
    function test_unlink_succeeds() external {
        vm.prank(bob);
        staking.setAllowedStaker(alice, true);
        _stake(alice, 100 ether, bob);

        vm.expectEmit(address(staking));
        emit Unlinked(alice, bob);

        vm.prank(alice);
        staking.unlink();

        (, uint256 bobReceived,) = staking.stakingData(bob);
        assertEq(bobReceived, 0);
        (uint256 aliceStaked,,) = staking.stakingData(alice);
        assertEq(aliceStaked, 100 ether);
    }

    /// @notice Tests that unlinking when not linked reverts.
    function test_unlink_notLinked_reverts() external {
        vm.prank(alice);
        vm.expectRevert(PolicyEngineStaking.PolicyEngineStaking_NotLinked.selector);
        staking.unlink();
    }
}

/// @title PolicyEngineStaking_Allowlist_Test
/// @notice Tests the allowlist functions of the `PolicyEngineStaking` contract.
contract PolicyEngineStaking_Allowlist_Test is PolicyEngineStaking_TestInit {
    /// @notice Tests that setAllowedStaker updates allowlist correctly.
    function test_setAllowedStaker_succeeds() external {
        // allowlist(beneficiary, staker) returns tuple (bool allowed) for AllowlistEntry struct
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

/// @title PolicyEngineStaking_Uncategorized_Test
/// @notice Tests the view functions and public mappings of the `PolicyEngineStaking` contract.
contract PolicyEngineStaking_Uncategorized_Test is PolicyEngineStaking_TestInit {
    /// @notice Tests that owner is set correctly.
    function test_owner_succeeds() external view {
        assertEq(staking.owner(), owner);
    }

    /// @notice Tests that PE_DATA_SLOT is 0.
    function test_peDataSlot_isZero_succeeds() external view {
        assertEq(staking.PE_DATA_SLOT(), bytes32(uint256(0)));
    }

    /// @notice Tests that stakingData returns correct staked amount.
    function test_stakingData_stakedAmount_succeeds() external {
        (uint256 staked,,) = staking.stakingData(alice);
        assertEq(staked, 0);

        _stake(alice, 100 ether, address(0));
        (staked,,) = staking.stakingData(alice);
        assertEq(staked, 100 ether);

        _stake(alice, 50 ether, address(0));
        (staked,,) = staking.stakingData(alice);
        assertEq(staked, 150 ether);
    }

    /// @notice Tests that stakingData returns correct values.
    function test_stakingData_succeeds() external {
        vm.prank(bob);
        staking.setAllowedStaker(alice, true);
        _stake(alice, 100 ether, bob);

        (uint256 staked, uint256 received, address linkedTo) = staking.stakingData(alice);
        assertEq(staked, 100 ether);
        assertEq(received, 0);
        assertEq(linkedTo, bob);

        (, uint256 bobReceived,) = staking.stakingData(bob);
        assertEq(bobReceived, 100 ether);
    }
}

/// @title PolicyEngineStaking_Integration_Test
/// @notice Integration tests for the full stake/link/unlink/unstake flow.
contract PolicyEngineStaking_Integration_Test is PolicyEngineStaking_TestInit {
    /// @notice Tests full flow: stake -> link -> stake more -> unlink -> unstake.
    function test_fullFlow_succeeds() external {
        _stake(alice, 100 ether, address(0));
        (uint256 staked,,) = staking.stakingData(alice);
        assertEq(staked, 100 ether);

        vm.prank(bob);
        staking.setAllowedStaker(alice, true);
        vm.prank(alice);
        staking.link(bob);

        vm.prank(alice);
        IERC20(Predeploys.GOVERNANCE_TOKEN).approve(address(staking), 50 ether);
        vm.prank(alice);
        staking.stake(50 ether, bob);

        (staked,,) = staking.stakingData(alice);
        assertEq(staked, 150 ether);
        (, uint256 bobReceived,) = staking.stakingData(bob);
        assertEq(bobReceived, 150 ether);

        vm.prank(alice);
        staking.unlink();

        (staked,,) = staking.stakingData(alice);
        assertEq(staked, 150 ether);
        (, bobReceived,) = staking.stakingData(bob);
        assertEq(bobReceived, 0);

        uint256 aliceBalanceBefore = IERC20(Predeploys.GOVERNANCE_TOKEN).balanceOf(alice);
        vm.prank(alice);
        staking.unstake();
        assertEq(IERC20(Predeploys.GOVERNANCE_TOKEN).balanceOf(alice), aliceBalanceBefore + 150 ether);
        (staked,,) = staking.stakingData(alice);
        assertEq(staked, 0);
    }

    /// @notice Tests that multiple stakers can stake to the same beneficiary.
    function test_multipleStakersToSameBeneficiary_succeeds() external {
        vm.prank(bob);
        staking.setAllowedStaker(alice, true);
        vm.prank(bob);
        staking.setAllowedStaker(carol, true);

        _stake(alice, 100 ether, bob);
        _stake(carol, 50 ether, bob);

        (, uint256 bobReceived,) = staking.stakingData(bob);
        assertEq(bobReceived, 150 ether);
        (uint128 bobEffective,) = staking.peData(bob);
        assertEq(bobEffective, 150 ether);
    }

    /// @notice Tests that a beneficiary with own stake plus received stake has correct effective stake.
    function test_beneficiaryWithOwnStakeAndReceived_succeeds() external {
        _stake(bob, 50 ether, address(0));
        vm.prank(bob);
        staking.setAllowedStaker(alice, true);
        _stake(alice, 100 ether, bob);

        (uint256 bobStaked, uint256 bobReceived,) = staking.stakingData(bob);
        assertEq(bobStaked, 50 ether);
        assertEq(bobReceived, 100 ether);
        (uint128 bobEffective,) = staking.peData(bob);
        assertEq(bobEffective, 150 ether);
    }

    /// @notice Tests switching beneficiary: unlink from one, link to another.
    function test_switchBeneficiary_unlinkThenLinkToOther_succeeds() external {
        vm.prank(bob);
        staking.setAllowedStaker(alice, true);
        _stake(alice, 100 ether, bob);

        (, uint256 bobReceived,) = staking.stakingData(bob);
        assertEq(bobReceived, 100 ether);

        vm.prank(alice);
        staking.unlink();

        vm.prank(carol);
        staking.setAllowedStaker(alice, true);
        vm.prank(alice);
        staking.link(carol);

        (, bobReceived,) = staking.stakingData(bob);
        assertEq(bobReceived, 0);
        (, uint256 carolReceived,) = staking.stakingData(carol);
        assertEq(carolReceived, 100 ether);
        (uint128 carolEffective,) = staking.peData(carol);
        assertEq(carolEffective, 100 ether);
    }

    /// @notice Tests stake -> pause -> stake reverts -> unpause -> stake works.
    function test_stakePauseUnpause_stakeFlow_succeeds() external {
        _stake(alice, 100 ether, address(0));
        (uint256 staked,,) = staking.stakingData(alice);
        assertEq(staked, 100 ether);

        vm.prank(owner);
        staking.pause();
        assertTrue(staking.paused());

        vm.prank(alice);
        IERC20(Predeploys.GOVERNANCE_TOKEN).approve(address(staking), 50 ether);
        vm.expectRevert(PolicyEngineStaking.PolicyEngineStaking_Paused.selector);
        vm.prank(alice);
        staking.stake(50 ether, address(0));

        vm.prank(owner);
        staking.unpause();
        assertFalse(staking.paused());

        vm.prank(alice);
        staking.stake(50 ether, address(0));
        (staked,,) = staking.stakingData(alice);
        assertEq(staked, 150 ether);
    }

    /// @notice Tests that revoking allowlist does not auto-unlink: staker keeps link until explicit unlink.
    function test_revokeAllowlist_stakeRemainsUntilUnlink_succeeds() external {
        vm.prank(alice);
        staking.setAllowedStaker(bob, true);
        _stake(bob, 100 ether, alice);

        (uint256 bobStaked,, address bobLinkedTo) = staking.stakingData(bob);
        (, uint256 aliceReceived,) = staking.stakingData(alice);
        (uint128 aliceEffective,) = staking.peData(alice);
        assertEq(bobStaked, 100 ether);
        assertEq(bobLinkedTo, alice);
        assertEq(aliceReceived, 100 ether);
        assertEq(aliceEffective, 100 ether);

        vm.prank(alice);
        staking.setAllowedStaker(bob, false);

        // Bob stays linked - stake and effective power remain with Alice until Bob unlinks
        (bobStaked,, bobLinkedTo) = staking.stakingData(bob);
        (, aliceReceived,) = staking.stakingData(alice);
        (aliceEffective,) = staking.peData(alice);
        assertEq(bobStaked, 100 ether);
        assertEq(aliceReceived, 100 ether);
        assertEq(aliceEffective, 100 ether);

        vm.prank(bob);
        staking.unlink();

        // Now Bob is self-attributed, Alice lost the received stake
        (bobStaked,, bobLinkedTo) = staking.stakingData(bob);
        (, aliceReceived,) = staking.stakingData(alice);
        (uint128 bobEffective,) = staking.peData(bob);
        (aliceEffective,) = staking.peData(alice);
        assertEq(bobStaked, 100 ether);
        assertEq(bobLinkedTo, address(0));
        assertEq(aliceReceived, 0);
        assertEq(bobEffective, 100 ether);
        assertEq(aliceEffective, 0);
    }
}
