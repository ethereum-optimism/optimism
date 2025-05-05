// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing utilities
import { CommonTest } from "test/setup/CommonTest.sol";

// Error imports
import { Unauthorized } from "src/libraries/errors/CommonErrors.sol";
import { InvalidAmount } from "src/libraries/errors/CommonErrors.sol";

/// @title ETHLiquidity_Test
/// @notice Contract for testing the ETHLiquidity contract.
contract ETHLiquidity_Test is CommonTest {
    /// @notice Emitted when an address burns ETH liquidity.
    event LiquidityBurned(address indexed caller, uint256 value);

    /// @notice Emitted when an address mints ETH liquidity.
    event LiquidityMinted(address indexed caller, uint256 value);

    /// @notice Emitted when an address funds the contract.
    event LiquidityFunded(address indexed funder, uint256 amount);

    /// @notice The starting balance of the ETHLiquidity contract.
    uint256 public constant STARTING_LIQUIDITY_BALANCE = type(uint248).max;

    /// @notice Test setup.
    function setUp() public virtual override {
        super.enableInterop();
        super.setUp();
    }

    /// @notice Tests that contract is set up with the correct starting balance.
    function test_setup_succeeds() public view {
        // Assert
        assertEq(address(ethLiquidity).balance, STARTING_LIQUIDITY_BALANCE);
    }

    /// @notice Tests that the burn function can always be called by an authorized caller.
    /// @param _amount Amount of ETH (in wei) to call the burn function with.
    function testFuzz_burn_fromAuthorizedCaller_succeeds(uint256 _amount) public {
        // Assume
        _amount = bound(_amount, 0, type(uint248).max - 1);

        // Arrange
        vm.deal(address(superchainETHBridge), _amount);

        // Act
        vm.expectEmit(address(ethLiquidity));
        emit LiquidityBurned(address(superchainETHBridge), _amount);
        vm.prank(address(superchainETHBridge));
        ethLiquidity.burn{ value: _amount }();

        // Assert
        assertEq(address(superchainETHBridge).balance, 0);
        assertEq(address(ethLiquidity).balance, STARTING_LIQUIDITY_BALANCE + _amount);
    }

    /// @notice Tests that the burn function always reverts when called by an unauthorized caller.
    /// @param _amount Amount of ETH (in wei) to call the burn function with.
    /// @param _caller Address of the caller to call the burn function with.
    function testFuzz_burn_fromUnauthorizedCaller_fails(uint256 _amount, address _caller) public {
        // Assume
        vm.assume(_caller != address(superchainETHBridge));
        vm.assume(_caller != address(ethLiquidity));
        _amount = bound(_amount, 0, type(uint248).max - 1);

        // Arrange
        vm.deal(_caller, _amount);

        // Act
        vm.prank(_caller);
        vm.expectRevert(Unauthorized.selector);
        ethLiquidity.burn{ value: _amount }();

        // Assert
        assertEq(_caller.balance, _amount);
        assertEq(address(ethLiquidity).balance, STARTING_LIQUIDITY_BALANCE);
    }

    /// @notice Tests that the mint function fails when the amount requested is greater than the
    ///         available balance. In practice this should never happen because the starting
    ///         balance is expected to be uint248 wei, the total ETH supply is far less than that
    ///         amount, and the only contract that pulls from here is the SuperchainETHBridge contract
    ///         which will always burn ETH somewhere before minting it somewhere else. It needs to
    ///         be a system-wide invariant that this condition is never triggered in the first
    ///         place but it is the behavior we expect if it does happen.
    function test_mint_moreThanAvailableBalance_fails() public {
        // Arrange
        uint256 amount = STARTING_LIQUIDITY_BALANCE + 1;

        // Act
        vm.expectRevert(); // nosemgrep: sol-safety-expectrevert-no-args
        ethLiquidity.mint(amount);

        // Assert
        assertEq(address(superchainETHBridge).balance, 0);
        assertEq(address(ethLiquidity).balance, STARTING_LIQUIDITY_BALANCE);
    }

    /// @notice Tests that the mint function can always be called by an authorized caller.
    /// @param _amount Amount of ETH (in wei) to call the mint function with.
    function testFuzz_mint_fromAuthorizedCaller_succeeds(uint256 _amount) public {
        // Assume
        _amount = bound(_amount, 0, type(uint248).max - 1);

        // Get balances before
        uint256 superchainETHBridgeBalanceBefore = address(superchainETHBridge).balance;

        // Act
        vm.expectEmit(address(ethLiquidity));
        emit LiquidityMinted(address(superchainETHBridge), _amount);
        vm.prank(address(superchainETHBridge));
        ethLiquidity.mint(_amount);

        // Assert
        assertEq(address(superchainETHBridge).balance, superchainETHBridgeBalanceBefore + _amount);
        assertEq(address(ethLiquidity).balance, STARTING_LIQUIDITY_BALANCE - _amount);
    }

    /// @notice Tests that the mint function always reverts when called by an unauthorized caller.
    /// @param _amount Amount of ETH (in wei) to call the mint function with.
    /// @param _caller Address of the caller to call the mint function with.
    function testFuzz_mint_fromUnauthorizedCaller_fails(uint256 _amount, address _caller) public {
        // Assume
        vm.assume(_caller != address(superchainETHBridge));
        vm.assume(address(_caller).balance == 0);
        _amount = bound(_amount, 0, type(uint248).max - 1);

        // Arrange
        // Nothing to arrange.

        // Act
        vm.prank(_caller);
        vm.expectRevert(Unauthorized.selector);
        ethLiquidity.mint(_amount);

        // Assert
        assertEq(_caller.balance, 0);
        assertEq(address(ethLiquidity).balance, STARTING_LIQUIDITY_BALANCE);
        assertEq(address(superchainETHBridge).balance, 0);
    }

    /// @notice Tests that the fund function succeeds when called with a non-zero value.
    /// @param _amount Amount of ETH (in wei) to call the fund function with.
    /// @param _caller Address of the caller to call the fund function with.
    function testFuzz_fund_succeeds(uint256 _amount, address _caller) public {
        // Assume
        vm.assume(_amount > 0); // Fund amount must be greater than 0
        // Bound amount reasonably, e.g., up to 1 million ETH
        _amount = bound(_amount, 1, STARTING_LIQUIDITY_BALANCE);
        vm.assume(_caller != address(0));
        vm.assume(_caller != address(ethLiquidity)); // Prevent contract from calling itself

        // Arrange
        uint256 initialContractBalance = address(ethLiquidity).balance;
        vm.deal(_caller, _amount);

        // Act
        vm.expectEmit(address(ethLiquidity));
        emit LiquidityFunded(_caller, _amount);
        vm.prank(_caller);
        ethLiquidity.fund{ value: _amount }();

        // Assert
        // Caller should have 0 balance after funding the exact amount they were dealt
        assertEq(_caller.balance, 0);
        assertEq(address(ethLiquidity).balance, initialContractBalance + _amount);
    }

    /// @notice Tests that the fund function reverts when called with zero value.
    function test_fund_zeroAmount_reverts() public {
        // Arrange
        // Nothing to arrange.

        // Act
        vm.expectRevert(InvalidAmount.selector);
        ethLiquidity.fund{ value: 0 }();

        // Assert
        assertEq(address(ethLiquidity).balance, STARTING_LIQUIDITY_BALANCE);
    }
}
