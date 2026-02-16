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
    /// @notice Tests that the owner can mint tokens.
    function testFuzz_mint_fromOwner_succeeds(address _account, uint256 _amount) external {
        vm.assume(_account != address(0));
        _amount = bound(_amount, 0, type(uint224).max);

        vm.prank(owner);
        governanceToken.mint(_account, _amount);

        assertEq(governanceToken.balanceOf(_account), _amount);
        assertEq(governanceToken.totalSupply(), _amount);
    }

    /// @notice Tests that `mint` reverts for non-owners.
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
    /// @notice Tests that a user can burn their tokens.
    function testFuzz_burn_succeeds(uint256 _mintAmount, uint256 _burnAmount) external {
        _mintAmount = bound(_mintAmount, 0, type(uint224).max);
        _burnAmount = bound(_burnAmount, 0, _mintAmount);

        vm.prank(owner);
        governanceToken.mint(rando, _mintAmount);

        vm.prank(rando);
        governanceToken.burn(_burnAmount);

        assertEq(governanceToken.balanceOf(rando), _mintAmount - _burnAmount);
        assertEq(governanceToken.totalSupply(), _mintAmount - _burnAmount);
    }

    /// @notice Tests that `burnFrom` works with approval.
    function testFuzz_burnFrom_succeeds(uint256 _mintAmount, uint256 _burnAmount) external {
        _mintAmount = bound(_mintAmount, 0, type(uint224).max);
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

    /// @notice Tests that `transfer` correctly moves tokens.
    function testFuzz_transfer_succeeds(uint256 _mintAmount, uint256 _transferAmount) external {
        _mintAmount = bound(_mintAmount, 0, type(uint224).max);
        _transferAmount = bound(_transferAmount, 0, _mintAmount);

        vm.prank(owner);
        governanceToken.mint(rando, _mintAmount);

        vm.prank(rando);
        governanceToken.transfer(owner, _transferAmount);

        assertEq(governanceToken.balanceOf(owner), _transferAmount);
        assertEq(governanceToken.balanceOf(rando), _mintAmount - _transferAmount);
        assertEq(governanceToken.totalSupply(), _mintAmount);
    }

    /// @notice Tests that `approve` correctly sets allowances.
    function testFuzz_approve_succeeds(uint256 _amount) external {
        vm.prank(rando);
        governanceToken.approve(owner, _amount);

        assertEq(governanceToken.allowance(rando, owner), _amount);
    }

    /// @notice Tests that `transferFrom` correctly moves tokens.
    function testFuzz_transferFrom_succeeds(uint256 _mintAmount, uint256 _transferAmount) external {
        _mintAmount = bound(_mintAmount, 0, type(uint224).max);
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

    /// @notice Tests that `increaseAllowance` increases allowances.
    function testFuzz_increaseAllowance_succeeds(uint256 _initialAllowance, uint256 _increase) external {
        _initialAllowance = bound(_initialAllowance, 0, type(uint128).max);
        _increase = bound(_increase, 0, type(uint128).max);

        vm.prank(rando);
        governanceToken.approve(owner, _initialAllowance);

        vm.prank(rando);
        governanceToken.increaseAllowance(owner, _increase);

        assertEq(governanceToken.allowance(rando, owner), _initialAllowance + _increase);
    }

    /// @notice Tests that `decreaseAllowance` decreases allowances.
    function testFuzz_decreaseAllowance_succeeds(uint256 _initialAllowance, uint256 _decrease) external {
        _initialAllowance = bound(_initialAllowance, 0, type(uint192).max);
        _decrease = bound(_decrease, 0, _initialAllowance);

        vm.prank(rando);
        governanceToken.approve(owner, _initialAllowance);

        vm.prank(rando);
        governanceToken.decreaseAllowance(owner, _decrease);

        assertEq(governanceToken.allowance(rando, owner), _initialAllowance - _decrease);
    }
}
