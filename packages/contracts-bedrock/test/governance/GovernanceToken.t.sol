// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing utilities
import { CommonTest } from "test/setup/CommonTest.sol";

/// @title GovernanceToken_TestInit
/// @notice Reusable test initialization for `GovernanceToken` tests.
abstract contract GovernanceToken_TestInit is CommonTest {
    address owner;
    address rando;

    /// @notice Sets up the test suite.
    function setUp() public virtual override {
        super.setUp();
        owner = governanceToken.owner();
        rando = makeAddr("rando");
    }
}

/// @title GovernanceToken_Constructor_Test
/// @notice Tests the constructor of the `GovernanceToken` contract.
contract GovernanceToken_Constructor_Test is GovernanceToken_TestInit {
    /// @notice Tests that the constructor sets the correct initial state.
    function test_constructor_succeeds() external view {
        assertEq(governanceToken.owner(), owner);
        assertEq(governanceToken.name(), "Optimism");
        assertEq(governanceToken.symbol(), "OP");
        assertEq(governanceToken.decimals(), 18);
        assertEq(governanceToken.totalSupply(), 0);
    }
}

/// @title GovernanceToken_Mint_Test
/// @notice Tests the `mint` function of the `GovernanceToken` contract.
contract GovernanceToken_Mint_Test is GovernanceToken_TestInit {
    /// @notice Tests that the owner can mint tokens to any
    ///         valid recipient with any valid amount.
    function testFuzz_mint_fromOwner_succeeds(address _to, uint256 _amount) external {
        vm.assume(_to != address(0));
        _amount = bound(_amount, 0, type(uint224).max);

        vm.prank(owner);
        governanceToken.mint(_to, _amount);

        assertEq(governanceToken.balanceOf(_to), _amount);
        assertEq(governanceToken.totalSupply(), _amount);
    }

    /// @notice Tests that `mint` reverts when called by
    ///         any non-owner address.
    function testFuzz_mint_fromNotOwner_reverts(address _caller) external {
        vm.assume(_caller != owner);

        vm.prank(_caller);
        vm.expectRevert("Ownable: caller is not the owner");
        governanceToken.mint(owner, 100);

        assertEq(governanceToken.balanceOf(owner), 0);
        assertEq(governanceToken.totalSupply(), 0);
    }
}

/// @title GovernanceToken_Uncategorized_Test
/// @notice General tests that are not testing any function directly of the `GovernanceToken`
///         contract or are testing multiple functions at once.
contract GovernanceToken_Uncategorized_Test is GovernanceToken_TestInit {
    /// @notice Tests that a user can burn a portion of
    ///         their tokens.
    function testFuzz_burn_succeeds(uint256 _mintAmount, uint256 _burnAmount) external {
        _mintAmount = bound(_mintAmount, 1, type(uint224).max);
        _burnAmount = bound(_burnAmount, 0, _mintAmount);

        vm.prank(owner);
        governanceToken.mint(rando, _mintAmount);

        vm.prank(rando);
        governanceToken.burn(_burnAmount);

        assertEq(governanceToken.balanceOf(rando), _mintAmount - _burnAmount);
        assertEq(governanceToken.totalSupply(), _mintAmount - _burnAmount);
    }

    /// @notice Tests that an approved spender can burn
    ///         tokens from another account.
    function testFuzz_burnFrom_succeeds(uint256 _mintAmount, uint256 _burnAmount) external {
        _mintAmount = bound(_mintAmount, 1, type(uint224).max);
        _burnAmount = bound(_burnAmount, 0, _mintAmount);

        vm.prank(owner);
        governanceToken.mint(rando, _mintAmount);

        vm.prank(rando);
        governanceToken.approve(owner, _burnAmount);

        vm.prank(owner);
        governanceToken.burnFrom(rando, _burnAmount);

        assertEq(governanceToken.balanceOf(rando), _mintAmount - _burnAmount);
        assertEq(governanceToken.totalSupply(), _mintAmount - _burnAmount);
    }

    /// @notice Tests that `transfer` correctly moves
    ///         tokens between accounts.
    function testFuzz_transfer_succeeds(uint256 _mintAmount, uint256 _transferAmount) external {
        _mintAmount = bound(_mintAmount, 1, type(uint224).max);
        _transferAmount = bound(_transferAmount, 0, _mintAmount);

        vm.prank(owner);
        governanceToken.mint(rando, _mintAmount);

        vm.prank(rando);
        governanceToken.transfer(owner, _transferAmount);

        assertEq(governanceToken.balanceOf(owner), _transferAmount);
        assertEq(governanceToken.balanceOf(rando), _mintAmount - _transferAmount);
        assertEq(governanceToken.totalSupply(), _mintAmount);
    }

    /// @notice Tests that `approve` correctly sets
    ///         allowances for any amount.
    function testFuzz_approve_succeeds(uint256 _amount) external {
        vm.prank(rando);
        governanceToken.approve(owner, _amount);

        assertEq(governanceToken.allowance(rando, owner), _amount);
    }

    /// @notice Tests that `transferFrom` correctly moves
    ///         tokens using an allowance.
    function testFuzz_transferFrom_succeeds(uint256 _mintAmount, uint256 _transferAmount) external {
        _mintAmount = bound(_mintAmount, 1, type(uint224).max);
        _transferAmount = bound(_transferAmount, 0, _mintAmount);

        vm.prank(owner);
        governanceToken.mint(rando, _mintAmount);

        vm.prank(rando);
        governanceToken.approve(owner, _transferAmount);

        vm.prank(owner);
        governanceToken.transferFrom(rando, owner, _transferAmount);

        assertEq(governanceToken.balanceOf(owner), _transferAmount);
        assertEq(governanceToken.balanceOf(rando), _mintAmount - _transferAmount);
        assertEq(governanceToken.totalSupply(), _mintAmount);
    }

    /// @notice Tests that `increaseAllowance` correctly
    ///         increases an existing allowance.
    function testFuzz_increaseAllowance_succeeds(uint256 _initialApproval, uint256 _increase) external {
        _initialApproval = bound(_initialApproval, 0, type(uint256).max / 2);
        _increase = bound(_increase, 0, type(uint256).max - _initialApproval);

        vm.prank(rando);
        governanceToken.approve(owner, _initialApproval);

        vm.prank(rando);
        governanceToken.increaseAllowance(owner, _increase);

        assertEq(governanceToken.allowance(rando, owner), _initialApproval + _increase);
    }

    /// @notice Tests that `decreaseAllowance` correctly
    ///         decreases an existing allowance.
    function testFuzz_decreaseAllowance_succeeds(uint256 _initialApproval, uint256 _decrease) external {
        _initialApproval = bound(_initialApproval, 0, type(uint256).max);
        _decrease = bound(_decrease, 0, _initialApproval);

        vm.prank(rando);
        governanceToken.approve(owner, _initialApproval);

        vm.prank(rando);
        governanceToken.decreaseAllowance(owner, _decrease);

        assertEq(governanceToken.allowance(rando, owner), _initialApproval - _decrease);
    }
}
