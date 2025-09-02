// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing utilities
import { CommonTest } from "test/setup/CommonTest.sol";

// Error imports
import { Unauthorized, InvalidAmount } from "src/libraries/errors/CommonErrors.sol";

/// @title NativeAssetLiquidity_TestInit
/// @notice Reusable test initialization for `NativeAssetLiquidity` tests.
contract NativeAssetLiquidity_TestInit is CommonTest {
    /// @notice Emitted when an address withdraws native asset liquidity.
    event LiquidityWithdrawn(address indexed caller, uint256 value);

    /// @notice Emitted when an address deposits native asset liquidity.
    event LiquidityDeposited(address indexed caller, uint256 value);

    /// @notice Emitted when an address funds the contract.
    event LiquidityFunded(address indexed funder, uint256 value);

    /// @notice Test setup.
    function setUp() public virtual override {
        enableCustomGasToken();
        super.setUp();
    }

    /// @notice Tests that contract is set up correctly.
    function test_setup_succeeds() public view {
        // Assert
        assertEq(nativeAssetLiquidity.version(), "1.0.0");
    }
}

/// @title NativeAssetLiquidity_Deposit_Test
/// @notice Tests the `deposit` function of the `NativeAssetLiquidity` contract.
contract NativeAssetLiquidity_Deposit_Test is NativeAssetLiquidity_TestInit {
    /// @notice Tests that the deposit function can be called by the authorized caller.
    /// @param _amount Amount of native asset (in wei) to call the deposit function with.
    function testFuzz_deposit_fromAuthorizedCaller_succeeds(uint256 _amount) public {
        _amount = bound(_amount, 0, type(uint248).max);

        // Deal the LiquidityController with the amount to deposit
        vm.deal(address(liquidityController), _amount);
        uint256 nativeAssetBalanceBefore = address(nativeAssetLiquidity).balance;

        // Expect emit LiquidityDeposited event
        vm.expectEmit(address(nativeAssetLiquidity));
        emit LiquidityDeposited(address(liquidityController), _amount);

        // Call the deposit function with LiquidityController as the caller
        vm.prank(address(liquidityController));
        nativeAssetLiquidity.deposit{ value: _amount }();

        // Assert LiquidityController and NativeAssetLiquidity balances are updated correctly
        assertEq(address(liquidityController).balance, 0);
        assertEq(address(nativeAssetLiquidity).balance, nativeAssetBalanceBefore + _amount);
    }

    /// @notice Tests that the deposit function always reverts when called by an unauthorized caller.
    /// @param _amount Amount of native asset (in wei) to call the deposit function with.
    /// @param _caller Address of the caller to call the deposit function with.
    function testFuzz_deposit_fromUnauthorizedCaller_fails(uint256 _amount, address _caller) public {
        vm.assume(_caller != address(liquidityController));
        _amount = bound(_amount, 0, type(uint248).max);

        // Deal the unauthorized caller with the amount to deposit
        vm.deal(_caller, _amount);
        uint256 nativeAssetBalanceBefore = address(nativeAssetLiquidity).balance;

        // Call the deposit function with unauthorized caller
        vm.prank(_caller);
        // Expect revert with Unauthorized
        vm.expectRevert(Unauthorized.selector);
        nativeAssetLiquidity.deposit{ value: _amount }();

        // Assert caller and NativeAssetLiquidity balances remain unchanged
        assertEq(_caller.balance, _amount);
        assertEq(address(nativeAssetLiquidity).balance, nativeAssetBalanceBefore);
    }
}

/// @title NativeAssetLiquidity_Withdraw_Test
/// @notice Tests the `withdraw` function of the `NativeAssetLiquidity` contract.
contract NativeAssetLiquidity_Withdraw_Test is NativeAssetLiquidity_TestInit {
    /// @notice Tests that the withdraw function can be called by the authorized caller.
    /// @param _amount Amount of native asset (in wei) to call the withdraw function with.
    function testFuzz_withdraw_fromAuthorizedCaller_succeeds(uint256 _amount) public {
        _amount = bound(_amount, 1, type(uint248).max);

        // Deal NativeAssetLiquidity with the amount to withdraw
        vm.deal(address(nativeAssetLiquidity), _amount);
        uint256 nativeAssetBalanceBefore = address(nativeAssetLiquidity).balance;
        uint256 controllerBalanceBefore = address(liquidityController).balance;

        // Expect emit LiquidityWithdrawn event
        vm.expectEmit(address(nativeAssetLiquidity));
        emit LiquidityWithdrawn(address(liquidityController), _amount);
        vm.prank(address(liquidityController));
        nativeAssetLiquidity.withdraw(_amount);

        // Assert LiquidityController and NativeAssetLiquidity balances are updated correctly
        assertEq(address(liquidityController).balance, controllerBalanceBefore + _amount);
        assertEq(address(nativeAssetLiquidity).balance, nativeAssetBalanceBefore - _amount);
    }

    /// @notice Tests that the withdraw function always reverts when called by an unauthorized caller.
    /// @param _amount Amount of native asset (in wei) to call the withdraw function with.
    /// @param _caller Address of the caller to call the withdraw function with.
    function testFuzz_withdraw_fromUnauthorizedCaller_fails(uint256 _amount, address _caller) public {
        vm.assume(_caller != address(liquidityController));
        _amount = bound(_amount, 1, type(uint248).max);

        // Deal NativeAssetLiquidity with the amount to withdraw
        vm.deal(address(nativeAssetLiquidity), _amount);
        uint256 nativeAssetBalanceBefore = address(nativeAssetLiquidity).balance;
        uint256 callerBalanceBefore = _caller.balance;

        // Call the withdraw function with unauthorized caller
        vm.prank(_caller);
        // Expect revert with Unauthorized
        vm.expectRevert(Unauthorized.selector);
        nativeAssetLiquidity.withdraw(_amount);

        // Assert caller and NativeAssetLiquidity balances remain unchanged
        assertEq(_caller.balance, callerBalanceBefore);
        assertEq(address(nativeAssetLiquidity).balance, nativeAssetBalanceBefore);
    }

    /// @notice Tests that the withdraw function reverts when contract has insufficient balance.
    function test_withdraw_insufficientBalance_fails() public {
        // Try to withdraw more than available balance
        uint256 contractBalance = address(nativeAssetLiquidity).balance;
        uint256 amount = bound(contractBalance, contractBalance + 1, type(uint256).max);

        // Call the withdraw function with insufficient balance
        vm.prank(address(liquidityController));
        // Expect revert with OutOfFunds
        vm.expectRevert();
        nativeAssetLiquidity.withdraw(amount);

        // Assert contract and controller balances remain unchanged
        assertEq(address(nativeAssetLiquidity).balance, contractBalance);
        assertEq(address(liquidityController).balance, 0);
    }
}

/// @title NativeAssetLiquidity_Fund_Test
/// @notice Tests the `fund` function of the `NativeAssetLiquidity` contract.
contract NativeAssetLiquidity_Fund_Test is NativeAssetLiquidity_TestInit {
    /// @notice Tests that the fund function succeeds when called with a non-zero value.
    /// @param _amount Amount of native asset (in wei) to call the fund function with.
    /// @param _caller Address of the caller to call the fund function with.
    function testFuzz_fund_succeeds(uint256 _amount, address _caller) public {
        _amount = bound(_amount, 1, 1000 ether);
        vm.assume(_caller != address(0));
        vm.assume(_caller != address(nativeAssetLiquidity)); // Prevent contract from calling itself

        // Deal caller with the amount to fund
        vm.deal(_caller, _amount);
        uint256 initialContractBalance = address(nativeAssetLiquidity).balance;

        // Expect emit LiquidityFunded event
        vm.expectEmit(address(nativeAssetLiquidity));
        emit LiquidityFunded(_caller, _amount);
        vm.prank(_caller);
        nativeAssetLiquidity.fund{ value: _amount }();

        // Assert caller and contract balances are updated correctly
        assertEq(_caller.balance, 0);
        assertEq(address(nativeAssetLiquidity).balance, initialContractBalance + _amount);
    }

    /// @notice Tests that the fund function reverts when called with zero value.
    function test_fund_zeroAmount_reverts() public {
        uint256 initialContractBalance = address(nativeAssetLiquidity).balance;
        // Expect revert with InvalidAmount
        vm.expectRevert(InvalidAmount.selector);
        nativeAssetLiquidity.fund{ value: 0 }();

        // Assert contract balance does not change
        assertEq(address(nativeAssetLiquidity).balance, initialContractBalance);
    }
}
