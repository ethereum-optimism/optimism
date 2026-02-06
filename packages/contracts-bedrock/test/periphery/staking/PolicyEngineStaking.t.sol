// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { Test } from "test/setup/Test.sol";

// Contracts
import { ERC20 } from "@openzeppelin/contracts/token/ERC20/ERC20.sol";
import { PolicyEngineStaking } from "src/periphery/staking/PolicyEngineStaking.sol";
import { Proxy } from "src/universal/Proxy.sol";
import { ProxyAdmin } from "src/universal/ProxyAdmin.sol";
import { ProxyAdminOwnedBase } from "src/L1/ProxyAdminOwnedBase.sol";

// Libraries
import { Predeploys } from "src/libraries/Predeploys.sol";

/// @title MockOPToken
/// @notice Simple ERC20 mock for testing PolicyEngineStaking at the OP token predeploy address.
contract MockOPToken is ERC20 {
    constructor() ERC20("Optimism", "OP") { }

    function mint(address _to, uint256 _amount) external {
        _mint(_to, _amount);
    }
}

/// @title PolicyEngineStaking_TestInit
/// @notice Reusable test initialization for `PolicyEngineStaking` tests.
abstract contract PolicyEngineStaking_TestInit is Test {
    address internal alice = address(0xA11CE);
    address internal bob = address(0xB0B);
    address internal carol = address(0xC4101);

    PolicyEngineStaking internal implementation;
    PolicyEngineStaking internal staking;
    ProxyAdmin internal proxyAdmin;
    address internal proxyAdminOwner;

    event Staked(address indexed account, address indexed beneficiary, uint256 amount);
    event Unstaked(address indexed account, uint256 amount);
    event Linked(address indexed staker, address indexed beneficiary);
    event Unlinked(address indexed staker, address indexed previousBeneficiary);
    event BeneficiaryAllowlistUpdated(address indexed beneficiary, address indexed staker, bool allowed);

    function setUp() public virtual {
        proxyAdminOwner = makeAddr("proxyAdminOwner");
        proxyAdmin = new ProxyAdmin(proxyAdminOwner);

        implementation = new PolicyEngineStaking();

        Proxy proxy = new Proxy(address(proxyAdmin));

        bytes memory initData = abi.encodeCall(PolicyEngineStaking.initialize, ());
        vm.prank(proxyAdminOwner);
        proxyAdmin.upgradeAndCall(payable(address(proxy)), address(implementation), initData);

        staking = PolicyEngineStaking(payable(address(proxy)));

        _setupMockOPToken();

        vm.label(alice, "alice");
        vm.label(bob, "bob");
        vm.label(carol, "carol");
        vm.label(address(staking), "PolicyEngineStaking");
    }

    /// @notice Deploys a mock OP token at the predeploy address and funds test accounts.
    function _setupMockOPToken() internal {
        MockOPToken opToken = new MockOPToken();
        vm.etch(Predeploys.GOVERNANCE_TOKEN, address(opToken).code);

        deal(Predeploys.GOVERNANCE_TOKEN, alice, 1_000 ether);
        deal(Predeploys.GOVERNANCE_TOKEN, bob, 1_000 ether);
        deal(Predeploys.GOVERNANCE_TOKEN, carol, 1_000 ether);
    }

    /// @notice Approves the staking contract and stakes tokens.
    function _stake(address _account, uint256 _amount, address _beneficiary) internal {
        vm.prank(_account);
        ERC20(Predeploys.GOVERNANCE_TOKEN).approve(address(staking), _amount);
        vm.prank(_account);
        staking.stake(_amount, _beneficiary);
    }
}

/// @title PolicyEngineStaking_Initialize_Test
/// @notice Tests the `initialize` function of the `PolicyEngineStaking` contract.
contract PolicyEngineStaking_Initialize_Test is PolicyEngineStaking_TestInit {
    /// @notice Tests that version is set correctly.
    function test_initialize_version_succeeds() external view {
        assertEq(staking.version(), "1.0.0");
    }

    /// @notice Tests that initialize reverts when called by unauthorized address.
    function testFuzz_initialize_notProxyAdminOrProxyAdminOwner_reverts(address _sender) public {
        vm.assume(_sender != address(proxyAdmin) && _sender != proxyAdminOwner);

        PolicyEngineStaking newImpl = new PolicyEngineStaking();
        Proxy newProxy = new Proxy(address(proxyAdmin));

        vm.prank(proxyAdminOwner);
        proxyAdmin.upgrade(payable(address(newProxy)), address(newImpl));

        vm.expectRevert(ProxyAdminOwnedBase.ProxyAdminOwnedBase_NotProxyAdminOrProxyAdminOwner.selector);
        vm.prank(_sender);
        PolicyEngineStaking(payable(address(newProxy))).initialize();
    }
}

/// @title PolicyEngineStaking_Stake_Test
/// @notice Tests the `stake` function of the `PolicyEngineStaking` contract.
contract PolicyEngineStaking_Stake_Test is PolicyEngineStaking_TestInit {
    /// @notice Tests that staking with self-attribution succeeds.
    function test_stake_selfAttribution_succeeds() external {
        uint256 amount = 100 ether;

        vm.prank(alice);
        ERC20(Predeploys.GOVERNANCE_TOKEN).approve(address(staking), amount);

        vm.expectEmit(address(staking));
        emit Staked(alice, address(0), amount);

        vm.prank(alice);
        staking.stake(amount, address(0));

        (uint256 staked, uint256 received, address linkedTo) = staking.getStakedData(alice);
        (uint128 effectiveStake, uint64 lastUpdate) = staking.getPEData(alice);

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

        (uint256 aliceStaked, uint256 aliceReceived,) = staking.getStakedData(alice);
        assertEq(aliceStaked, 600 ether);
        assertEq(aliceReceived, 0);
        (uint128 aliceEffectiveStake, uint64 aliceLastUpdate) = staking.getPEData(alice);
        assertEq(aliceEffectiveStake, 600 ether);
        assertEq(aliceLastUpdate, block.timestamp);
    }

    /// @notice Tests that staking to another beneficiary with allowlist succeeds.
    function test_stake_toBeneficiaryWithAllowlist_succeeds() external {
        vm.prank(bob);
        staking.setAllowedStaker(alice, true);

        vm.prank(alice);
        ERC20(Predeploys.GOVERNANCE_TOKEN).approve(address(staking), 100 ether);

        vm.expectEmit(address(staking));
        emit Staked(alice, bob, 100 ether);

        vm.prank(alice);
        staking.stake(100 ether, bob);

        (uint256 staked, uint256 received, address linkedTo) = staking.getStakedData(alice);
        (uint128 effectiveStake, uint64 lastUpdate) = staking.getPEData(alice);
        assertEq(staked, 100 ether);
        assertEq(received, 0);
        assertEq(linkedTo, bob);
        assertEq(effectiveStake, 0);
        assertEq(lastUpdate, 0);

        (uint256 bobStaked, uint256 bobReceived,) = staking.getStakedData(bob);
        assertEq(bobStaked, 0);
        assertEq(bobReceived, 100 ether);
        (uint128 bobEffectiveStake, uint64 bobLastUpdate) = staking.getPEData(bob);
        assertEq(bobEffectiveStake, 100 ether);
        assertEq(bobLastUpdate, block.timestamp);
    }

    /// @notice Tests that staking more to the same beneficiary when linked succeeds.
    function test_stake_moreToSameBeneficiary_succeeds() external {
        vm.prank(bob);
        staking.setAllowedStaker(alice, true);
        _stake(alice, 100 ether, bob);

        vm.prank(alice);
        ERC20(Predeploys.GOVERNANCE_TOKEN).approve(address(staking), 50 ether);
        vm.prank(alice);
        staking.stake(50 ether, bob);

        (uint256 staked,,) = staking.getStakedData(alice);
        assertEq(staked, 150 ether);
        (, uint256 bobReceived,) = staking.getStakedData(bob);
        assertEq(bobReceived, 150 ether);
        (uint128 bobEffective,) = staking.getPEData(bob);
        assertEq(bobEffective, 150 ether);
    }

    /// @notice Verifies stake succeeds with various amounts.
    function testFuzz_stake_amount_succeeds(uint256 _amount) external {
        _amount = bound(_amount, 1, 1_000 ether);

        vm.prank(alice);
        ERC20(Predeploys.GOVERNANCE_TOKEN).approve(address(staking), _amount);

        vm.prank(alice);
        staking.stake(_amount, address(0));

        (uint128 effectiveStake, uint64 lastUpdate) = staking.getPEData(alice);
        assertEq(effectiveStake, _amount);
        assertEq(lastUpdate, block.timestamp);
    }

    /// @notice Tests that staking with zero amount reverts.
    function test_stake_zeroAmount_reverts() external {
        vm.prank(alice);
        vm.expectRevert(PolicyEngineStaking.PolicyEngineStaking_ZeroAmount.selector);
        staking.stake(0, address(0));
    }

    /// @notice Tests that staking to another beneficiary without allowlist reverts.
    function test_stake_toBeneficiaryWithoutAllowlist_reverts() external {
        vm.prank(alice);
        ERC20(Predeploys.GOVERNANCE_TOKEN).approve(address(staking), 100 ether);

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
        ERC20(Predeploys.GOVERNANCE_TOKEN).approve(address(staking), 100 ether);

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
        ERC20(Predeploys.GOVERNANCE_TOKEN).approve(address(staking), 50 ether);

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
        ERC20(Predeploys.GOVERNANCE_TOKEN).approve(address(staking), 50 ether);

        vm.prank(alice);
        vm.expectRevert(PolicyEngineStaking.PolicyEngineStaking_AlreadyLinked.selector);
        staking.stake(50 ether, carol);
    }

    /// @notice Tests that staking to beneficiary not in allowlist reverts.
    function test_stake_notAllowedToLink_reverts() external {
        vm.prank(alice);
        ERC20(Predeploys.GOVERNANCE_TOKEN).approve(address(staking), 100 ether);

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
        ERC20(Predeploys.GOVERNANCE_TOKEN).approve(address(staking), 50 ether);
        vm.prank(alice);
        vm.expectRevert(PolicyEngineStaking.PolicyEngineStaking_NotAllowedToLink.selector);
        staking.stake(50 ether, bob);
    }
}

/// @title PolicyEngineStaking_Unstake_Test
/// @notice Tests the `unstake` function of the `PolicyEngineStaking` contract.
contract PolicyEngineStaking_Unstake_Test is PolicyEngineStaking_TestInit {
    /// @notice Tests that unstaking succeeds.
    function test_unstake_succeeds() external {
        _stake(alice, 100 ether, address(0));

        uint256 aliceBalanceBefore = ERC20(Predeploys.GOVERNANCE_TOKEN).balanceOf(alice);

        vm.expectEmit(address(staking));
        emit Unstaked(alice, 100 ether);

        vm.prank(alice);
        staking.unstake();

        (uint256 aliceStaked,,) = staking.getStakedData(alice);
        assertEq(aliceStaked, 0);
        assertEq(ERC20(Predeploys.GOVERNANCE_TOKEN).balanceOf(alice), aliceBalanceBefore + 100 ether);
    }

    /// @notice Tests that unstaking when linked returns tokens to staker.
    function test_unstake_whenLinked_succeeds() external {
        vm.prank(bob);
        staking.setAllowedStaker(alice, true);
        _stake(alice, 100 ether, bob);

        uint256 aliceBalanceBefore = ERC20(Predeploys.GOVERNANCE_TOKEN).balanceOf(alice);

        vm.prank(alice);
        staking.unstake();

        (uint256 aliceStaked,,) = staking.getStakedData(alice);
        assertEq(aliceStaked, 0);
        assertEq(ERC20(Predeploys.GOVERNANCE_TOKEN).balanceOf(alice), aliceBalanceBefore + 100 ether);
        (, uint256 bobReceived,) = staking.getStakedData(bob);
        assertEq(bobReceived, 0);
    }

    /// @notice Tests that unstaking with no stake reverts.
    function test_unstake_noStake_reverts() external {
        vm.prank(alice);
        vm.expectRevert(PolicyEngineStaking.PolicyEngineStaking_NoStake.selector);
        staking.unstake();
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

        (, uint256 bobReceived,) = staking.getStakedData(bob);
        assertEq(bobReceived, amount);
    }

    /// @notice Tests that self-attribution (link to self) succeeds without allowlist.
    function test_link_selfAttribution_succeeds() external {
        uint256 amount = 100 ether;
        _stake(alice, amount, address(0));

        vm.prank(alice);
        staking.link(alice);

        (uint256 aliceStaked,,) = staking.getStakedData(alice);
        assertEq(aliceStaked, amount);
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

        (, uint256 bobReceived,) = staking.getStakedData(bob);
        assertEq(bobReceived, 0);
        (uint256 aliceStaked,,) = staking.getStakedData(alice);
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
        assertFalse(staking.isAllowedToLink(bob, alice));

        vm.expectEmit(address(staking));
        emit BeneficiaryAllowlistUpdated(bob, alice, true);

        vm.prank(bob);
        staking.setAllowedStaker(alice, true);

        assertTrue(staking.isAllowedToLink(bob, alice));

        vm.prank(bob);
        staking.setAllowedStaker(alice, false);

        assertFalse(staking.isAllowedToLink(bob, alice));
    }

    /// @notice Tests that setAllowedStakers batch updates allowlist.
    function test_setAllowedStakers_succeeds() external {
        address[] memory stakers = new address[](2);
        stakers[0] = alice;
        stakers[1] = carol;

        vm.prank(bob);
        staking.setAllowedStakers(stakers, true);

        assertTrue(staking.isAllowedToLink(bob, alice));
        assertTrue(staking.isAllowedToLink(bob, carol));

        vm.prank(bob);
        staking.setAllowedStakers(stakers, false);

        assertFalse(staking.isAllowedToLink(bob, alice));
        assertFalse(staking.isAllowedToLink(bob, carol));
    }
}

/// @title PolicyEngineStaking_View_Test
/// @notice Tests the view functions of the `PolicyEngineStaking` contract.
contract PolicyEngineStaking_View_Test is PolicyEngineStaking_TestInit {
    /// @notice Tests that getStakedData returns correct staked amount.
    function test_getStakedData_stakedAmount_succeeds() external {
        (uint256 staked,,) = staking.getStakedData(alice);
        assertEq(staked, 0);

        _stake(alice, 100 ether, address(0));
        (staked,,) = staking.getStakedData(alice);
        assertEq(staked, 100 ether);

        _stake(alice, 50 ether, address(0));
        (staked,,) = staking.getStakedData(alice);
        assertEq(staked, 150 ether);
    }

    /// @notice Tests that getStakedData returns correct values.
    function test_getStakedData_succeeds() external {
        vm.prank(bob);
        staking.setAllowedStaker(alice, true);
        _stake(alice, 100 ether, bob);

        (uint256 staked, uint256 received, address linkedTo) = staking.getStakedData(alice);
        assertEq(staked, 100 ether);
        assertEq(received, 0);
        assertEq(linkedTo, bob);

        (, uint256 bobReceived,) = staking.getStakedData(bob);
        assertEq(bobReceived, 100 ether);
    }
}

/// @title PolicyEngineStaking_Integration_Test
/// @notice Integration tests for the full stake/link/unlink/unstake flow.
contract PolicyEngineStaking_Integration_Test is PolicyEngineStaking_TestInit {
    /// @notice Tests full flow: stake -> link -> stake more -> unlink -> unstake.
    function test_fullFlow_succeeds() external {
        _stake(alice, 100 ether, address(0));
        (uint256 staked,,) = staking.getStakedData(alice);
        assertEq(staked, 100 ether);

        vm.prank(bob);
        staking.setAllowedStaker(alice, true);
        vm.prank(alice);
        staking.link(bob);

        vm.prank(alice);
        ERC20(Predeploys.GOVERNANCE_TOKEN).approve(address(staking), 50 ether);
        vm.prank(alice);
        staking.stake(50 ether, bob);

        (staked,,) = staking.getStakedData(alice);
        assertEq(staked, 150 ether);
        (, uint256 bobReceived,) = staking.getStakedData(bob);
        assertEq(bobReceived, 150 ether);

        vm.prank(alice);
        staking.unlink();

        (staked,,) = staking.getStakedData(alice);
        assertEq(staked, 150 ether);
        (, bobReceived,) = staking.getStakedData(bob);
        assertEq(bobReceived, 0);

        uint256 aliceBalanceBefore = ERC20(Predeploys.GOVERNANCE_TOKEN).balanceOf(alice);
        vm.prank(alice);
        staking.unstake();
        assertEq(ERC20(Predeploys.GOVERNANCE_TOKEN).balanceOf(alice), aliceBalanceBefore + 150 ether);
        (staked,,) = staking.getStakedData(alice);
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

        (, uint256 bobReceived,) = staking.getStakedData(bob);
        assertEq(bobReceived, 150 ether);
        (uint128 bobEffective,) = staking.getPEData(bob);
        assertEq(bobEffective, 150 ether);
    }

    /// @notice Tests that a beneficiary with own stake plus received stake has correct effective stake.
    function test_beneficiaryWithOwnStakeAndReceived_succeeds() external {
        _stake(bob, 50 ether, address(0));
        vm.prank(bob);
        staking.setAllowedStaker(alice, true);
        _stake(alice, 100 ether, bob);

        (uint256 bobStaked, uint256 bobReceived,) = staking.getStakedData(bob);
        assertEq(bobStaked, 50 ether);
        assertEq(bobReceived, 100 ether);
        (uint128 bobEffective,) = staking.getPEData(bob);
        assertEq(bobEffective, 150 ether);
    }

    /// @notice Tests switching beneficiary: unlink from one, link to another.
    function test_switchBeneficiary_unlinkThenLinkToOther_succeeds() external {
        vm.prank(bob);
        staking.setAllowedStaker(alice, true);
        _stake(alice, 100 ether, bob);

        (, uint256 bobReceived,) = staking.getStakedData(bob);
        assertEq(bobReceived, 100 ether);

        vm.prank(alice);
        staking.unlink();

        vm.prank(carol);
        staking.setAllowedStaker(alice, true);
        vm.prank(alice);
        staking.link(carol);

        (, bobReceived,) = staking.getStakedData(bob);
        assertEq(bobReceived, 0);
        (, uint256 carolReceived,) = staking.getStakedData(carol);
        assertEq(carolReceived, 100 ether);
        (uint128 carolEffective,) = staking.getPEData(carol);
        assertEq(carolEffective, 100 ether);
    }
}
