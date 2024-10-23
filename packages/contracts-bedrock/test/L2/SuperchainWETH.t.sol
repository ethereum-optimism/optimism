// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { CommonTest } from "test/setup/CommonTest.sol";

// Libraries
import { Predeploys } from "src/libraries/Predeploys.sol";
import { NotCustomGasToken } from "src/libraries/errors/CommonErrors.sol";

// Interfaces
import { IETHLiquidity } from "src/L2/interfaces/IETHLiquidity.sol";
import { ISuperchainWETH } from "src/L2/interfaces/ISuperchainWETH.sol";

/// @title SuperchainWETH_Test
/// @notice Contract for testing the SuperchainWETH contract.
contract SuperchainWETH_Test is CommonTest {
    /// @notice Emitted when a transfer is made.
    event Transfer(address indexed src, address indexed dst, uint256 wad);

    /// @notice Emitted when a deposit is made.
    event Deposit(address indexed dst, uint256 wad);

    /// @notice Emitted when a withdrawal is made.
    event Withdrawal(address indexed src, uint256 wad);

    /// @notice Emitted when a crosschain transfer mints tokens.
    event CrosschainMint(address indexed to, uint256 amount);

    /// @notice Emitted when a crosschain transfer burns tokens.
    event CrosschainBurn(address indexed from, uint256 amount);

    address internal constant ZERO_ADDRESS = address(0);

    /// @notice Test setup.
    function setUp() public virtual override {
        super.enableInterop();
        super.setUp();
    }

    /// @notice Helper function to setup a mock and expect a call to it.
    function _mockAndExpect(address _receiver, bytes memory _calldata, bytes memory _returned) internal {
        vm.mockCall(_receiver, _calldata, _returned);
        vm.expectCall(_receiver, _calldata);
    }

    /// @notice Tests that the deposit function can be called on a non-custom gas token chain.
    /// @param _amount The amount of WETH to send.
    function testFuzz_deposit_fromNonCustomGasTokenChain_succeeds(uint256 _amount) public {
        // Assume
        _amount = bound(_amount, 0, type(uint248).max - 1);

        // Arrange
        vm.deal(alice, _amount);
        _mockAndExpect(address(l1Block), abi.encodeCall(l1Block.isCustomGasToken, ()), abi.encode(false));

        // Act
        vm.expectEmit(address(superchainWeth));
        emit Deposit(alice, _amount);
        vm.prank(alice);
        superchainWeth.deposit{ value: _amount }();

        // Assert
        assertEq(alice.balance, 0);
        assertEq(superchainWeth.balanceOf(alice), _amount);
    }

    /// @notice Tests that the deposit function reverts when called on a custom gas token chain.
    /// @param _amount The amount of WETH to send.
    function testFuzz_deposit_fromCustomGasTokenChain_fails(uint256 _amount) public {
        // Assume
        _amount = bound(_amount, 0, type(uint248).max - 1);

        // Arrange
        vm.deal(address(alice), _amount);
        _mockAndExpect(address(l1Block), abi.encodeCall(l1Block.isCustomGasToken, ()), abi.encode(true));

        // Act
        vm.prank(alice);
        vm.expectRevert(NotCustomGasToken.selector);
        superchainWeth.deposit{ value: _amount }();

        // Assert
        assertEq(alice.balance, _amount);
        assertEq(superchainWeth.balanceOf(alice), 0);
    }

    /// @notice Tests that the withdraw function can be called on a non-custom gas token chain.
    /// @param _amount The amount of WETH to send.
    function testFuzz_withdraw_fromNonCustomGasTokenChain_succeeds(uint256 _amount) public {
        // Assume
        _amount = bound(_amount, 0, type(uint248).max - 1);

        // Arrange
        vm.deal(alice, _amount);
        vm.prank(alice);
        superchainWeth.deposit{ value: _amount }();
        _mockAndExpect(address(l1Block), abi.encodeCall(l1Block.isCustomGasToken, ()), abi.encode(false));

        // Act
        vm.expectEmit(address(superchainWeth));
        emit Withdrawal(alice, _amount);
        vm.prank(alice);
        superchainWeth.withdraw(_amount);

        // Assert
        assertEq(alice.balance, _amount);
        assertEq(superchainWeth.balanceOf(alice), 0);
    }

    /// @notice Tests that the withdraw function reverts when called on a custom gas token chain.
    /// @param _amount The amount of WETH to send.
    function testFuzz_withdraw_fromCustomGasTokenChain_fails(uint256 _amount) public {
        // Assume
        _amount = bound(_amount, 0, type(uint248).max - 1);

        // Arrange
        vm.deal(alice, _amount);
        vm.prank(alice);
        superchainWeth.deposit{ value: _amount }();
        _mockAndExpect(address(l1Block), abi.encodeCall(l1Block.isCustomGasToken, ()), abi.encode(true));

        // Act
        vm.prank(alice);
        vm.expectRevert(NotCustomGasToken.selector);
        superchainWeth.withdraw(_amount);

        // Assert
        assertEq(alice.balance, 0);
        assertEq(superchainWeth.balanceOf(alice), _amount);
    }

    /// @notice Tests the `crosschainMint` function reverts when the caller is not the `SuperchainTokenBridge`.
    function testFuzz_crosschainMint_callerNotBridge_reverts(address _caller, address _to, uint256 _amount) public {
        // Ensure the caller is not the bridge
        vm.assume(_caller != Predeploys.SUPERCHAIN_TOKEN_BRIDGE);

        // Expect the revert with `Unauthorized` selector
        vm.expectRevert(ISuperchainWETH.Unauthorized.selector);

        // Call the `mint` function with the non-bridge caller
        vm.prank(_caller);
        superchainWeth.crosschainMint(_to, _amount);
    }

    /// @notice Tests the `crosschainMint` with non custom gas token succeeds and emits the `CrosschainMint` event.
    function testFuzz_crosschainMint_fromBridgeNonCustomGasTokenChain_succeeds(address _to, uint256 _amount) public {
        // Ensure `_to` is not the zero address
        vm.assume(_to != ZERO_ADDRESS);
        _amount = bound(_amount, 0, type(uint248).max - 1);

        // Get the total supply and balance of `_to` before the mint to compare later on the assertions
        uint256 _totalSupplyBefore = superchainWeth.totalSupply();
        uint256 _toBalanceBefore = superchainWeth.balanceOf(_to);

        // Look for the emit of the `Transfer` event
        vm.expectEmit(address(superchainWeth));
        emit Transfer(ZERO_ADDRESS, _to, _amount);

        // Look for the emit of the `CrosschainMint` event
        vm.expectEmit(address(superchainWeth));
        emit CrosschainMint(_to, _amount);

        // Mock the `isCustomGasToken` function to return false
        _mockAndExpect(address(l1Block), abi.encodeCall(l1Block.isCustomGasToken, ()), abi.encode(false));

        // Expect the call to the `mint` function in the `ETHLiquidity` contract
        vm.expectCall(Predeploys.ETH_LIQUIDITY, abi.encodeCall(IETHLiquidity.mint, (_amount)), 1);

        // Call the `mint` function with the bridge caller
        vm.prank(Predeploys.SUPERCHAIN_TOKEN_BRIDGE);
        superchainWeth.crosschainMint(_to, _amount);

        // Check the total supply and balance of `_to` after the mint were updated correctly
        assertEq(superchainWeth.totalSupply(), _totalSupplyBefore + _amount);
        assertEq(superchainWeth.balanceOf(_to), _toBalanceBefore + _amount);
        assertEq(address(superchainWeth).balance, _amount);
    }

    /// @notice Tests the `crosschainMint` with custom gas token succeeds and emits the `CrosschainMint` event.
    function testFuzz_crosschainMint_fromBridgeCustomGasTokenChain_succeeds(address _to, uint256 _amount) public {
        // Ensure `_to` is not the zero address
        vm.assume(_to != ZERO_ADDRESS);
        _amount = bound(_amount, 0, type(uint248).max - 1);

        // Get the balance of `_to` before the mint to compare later on the assertions
        uint256 _toBalanceBefore = superchainWeth.balanceOf(_to);

        // Look for the emit of the `Transfer` event
        vm.expectEmit(address(superchainWeth));
        emit Transfer(ZERO_ADDRESS, _to, _amount);

        // Look for the emit of the `CrosschainMint` event
        vm.expectEmit(address(superchainWeth));
        emit CrosschainMint(_to, _amount);

        // Mock the `isCustomGasToken` function to return false
        _mockAndExpect(address(l1Block), abi.encodeCall(l1Block.isCustomGasToken, ()), abi.encode(true));

        // Expect to not call the `mint` function in the `ETHLiquidity` contract
        vm.expectCall(Predeploys.ETH_LIQUIDITY, abi.encodeCall(IETHLiquidity.mint, (_amount)), 0);

        // Call the `mint` function with the bridge caller
        vm.prank(Predeploys.SUPERCHAIN_TOKEN_BRIDGE);
        superchainWeth.crosschainMint(_to, _amount);

        // Check the total supply and balance of `_to` after the mint were updated correctly
        assertEq(superchainWeth.balanceOf(_to), _toBalanceBefore + _amount);
        assertEq(superchainWeth.totalSupply(), 0);
        assertEq(address(superchainWeth).balance, 0);
    }

    /// @notice Tests the `crosschainBurn` function reverts when the caller is not the `SuperchainTokenBridge`.
    function testFuzz_crosschainBurn_callerNotBridge_reverts(address _caller, address _from, uint256 _amount) public {
        // Ensure the caller is not the bridge
        vm.assume(_caller != Predeploys.SUPERCHAIN_TOKEN_BRIDGE);

        // Expect the revert with `Unauthorized` selector
        vm.expectRevert(ISuperchainWETH.Unauthorized.selector);

        // Call the `burn` function with the non-bridge caller
        vm.prank(_caller);
        superchainWeth.crosschainBurn(_from, _amount);
    }

    /// @notice Tests the `crosschainBurn` with non custom gas token burns the amount and emits the `CrosschainBurn`
    /// event.
    function testFuzz_crosschainBurn_fromBridgeNonCustomGasTokenChain_succeeds(address _from, uint256 _amount) public {
        // Ensure `_from` is not the zero address
        vm.assume(_from != ZERO_ADDRESS);
        _amount = bound(_amount, 0, type(uint248).max - 1);

        // Deposit some tokens to `_from` so then they can be burned
        vm.deal(_from, _amount);
        vm.prank(_from);
        superchainWeth.deposit{ value: _amount }();

        // Get the total supply and balance of `_from` before the burn to compare later on the assertions
        uint256 _totalSupplyBefore = superchainWeth.totalSupply();
        uint256 _fromBalanceBefore = superchainWeth.balanceOf(_from);

        // Look for the emit of the `Transfer` event
        vm.expectEmit(address(superchainWeth));
        emit Transfer(_from, ZERO_ADDRESS, _amount);

        // Look for the emit of the `CrosschainBurn` event
        vm.expectEmit(address(superchainWeth));
        emit CrosschainBurn(_from, _amount);

        // Mock the `isCustomGasToken` function to return false
        _mockAndExpect(address(l1Block), abi.encodeCall(l1Block.isCustomGasToken, ()), abi.encode(false));

        // Expect the call to the `burn` function in the `ETHLiquidity` contract
        vm.expectCall(Predeploys.ETH_LIQUIDITY, abi.encodeCall(IETHLiquidity.burn, ()), 1);

        // Call the `burn` function with the bridge caller
        vm.prank(Predeploys.SUPERCHAIN_TOKEN_BRIDGE);
        superchainWeth.crosschainBurn(_from, _amount);

        // Check the total supply and balance of `_from` after the burn were updated correctly
        assertEq(superchainWeth.totalSupply(), _totalSupplyBefore - _amount);
        assertEq(superchainWeth.balanceOf(_from), _fromBalanceBefore - _amount);
        assertEq(address(superchainWeth).balance, 0);
    }

    /// @notice Tests the `crosschainBurn` with custom gas token burns the amount and emits the `CrosschainBurn`
    /// event.
    function testFuzz_crosschainBurn_fromBridgeCustomGasTokenChain_succeeds(address _from, uint256 _amount) public {
        // Ensure `_from` is not the zero address
        vm.assume(_from != ZERO_ADDRESS);
        _amount = bound(_amount, 0, type(uint248).max - 1);

        // Mock the `isCustomGasToken` function to return false
        _mockAndExpect(address(l1Block), abi.encodeCall(l1Block.isCustomGasToken, ()), abi.encode(true));

        // Mint some tokens to `_from` so then they can be burned
        vm.prank(Predeploys.SUPERCHAIN_TOKEN_BRIDGE);
        superchainWeth.crosschainMint(_from, _amount);

        // Get the total supply and balance of `_from` before the burn to compare later on the assertions
        uint256 _totalSupplyBefore = superchainWeth.totalSupply();
        uint256 _fromBalanceBefore = superchainWeth.balanceOf(_from);

        // Look for the emit of the `Transfer` event
        vm.expectEmit(address(superchainWeth));
        emit Transfer(_from, ZERO_ADDRESS, _amount);

        // Look for the emit of the `CrosschainBurn` event
        vm.expectEmit(address(superchainWeth));
        emit CrosschainBurn(_from, _amount);

        // Expect to not call the `burn` function in the `ETHLiquidity` contract
        vm.expectCall(Predeploys.ETH_LIQUIDITY, abi.encodeCall(IETHLiquidity.burn, ()), 0);

        // Call the `burn` function with the bridge caller
        vm.prank(Predeploys.SUPERCHAIN_TOKEN_BRIDGE);
        superchainWeth.crosschainBurn(_from, _amount);

        // Check the total supply and balance of `_from` after the burn were updated correctly
        assertEq(superchainWeth.balanceOf(_from), _fromBalanceBefore - _amount);
        assertEq(superchainWeth.totalSupply(), _totalSupplyBefore);
        assertEq(address(superchainWeth).balance, 0);
    }

    /// @notice Tests that the `crosschainBurn` function reverts when called with insufficient balance.
    function testFuzz_crosschainBurn_insufficientBalance_fails(address _from, uint256 _amount) public {
        // Assume
        vm.assume(_from != ZERO_ADDRESS);
        _amount = bound(_amount, 0, type(uint248).max - 1);

        // Arrange
        vm.deal(_from, _amount);
        vm.prank(_from);
        superchainWeth.deposit{ value: _amount }();

        // Act
        vm.expectRevert();
        superchainWeth.crosschainBurn(_from, _amount + 1);
    }

    /// @notice Test that the internal mint function reverts to protect against accidentally changing the visibility.
    function testFuzz_calling_internal_mint_function_reverts(address _caller, address _to, uint256 _amount) public {
        // Arrange
        bytes memory _calldata = abi.encodeWithSignature("_mint(address,uint256)", _to, _amount);
        vm.expectRevert(bytes(""));

        // Act
        vm.prank(_caller);
        (bool success,) = address(superchainWeth).call(_calldata);

        // Assert
        assertFalse(success);
    }

    /// @notice Test that the mint function reverts to protect against accidentally changing the visibility.
    function testFuzz_calling_mint_function_reverts(address _caller, address _to, uint256 _amount) public {
        // Arrange
        bytes memory _calldata = abi.encodeWithSignature("mint(address,uint256)", _to, _amount);
        vm.expectRevert(bytes(""));

        // Act
        vm.prank(_caller);
        (bool success,) = address(superchainWeth).call(_calldata);

        // Assert
        assertFalse(success);
    }

    /// @notice Test that the internal burn function reverts to protect against accidentally changing the visibility.
    function testFuzz_calling_internal_burn_function_reverts(address _caller, address _from, uint256 _amount) public {
        // Arrange
        bytes memory _calldata = abi.encodeWithSignature("_burn(address,uint256)", _from, _amount);
        vm.expectRevert(bytes(""));

        // Act
        vm.prank(_caller);
        (bool success,) = address(superchainWeth).call(_calldata);

        // Assert
        assertFalse(success);
    }

    /// @notice Test that the burn function reverts to protect against accidentally changing the visibility.
    function testFuzz_calling_burn_function_reverts(address _caller, address _from, uint256 _amount) public {
        // Arrange
        bytes memory _calldata = abi.encodeWithSignature("burn(address,uint256)", _from, _amount);
        vm.expectRevert(bytes(""));

        // Act
        vm.prank(_caller);
        (bool success,) = address(superchainWeth).call(_calldata);

        // Assert
        assertFalse(success);
    }
}
