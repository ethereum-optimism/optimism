// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing utilities
import { CommonTest } from "test/setup/CommonTest.sol";
import { stdStorage, StdStorage } from "forge-std/Test.sol";

// Libraries
import { Features } from "src/libraries/Features.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";

// Contracts
import { LiquidityController } from "src/L2/LiquidityController.sol";
import { NativeAssetLiquidity } from "src/L2/NativeAssetLiquidity.sol";

/// @title LiquidityController_TestInit
/// @notice Reusable test initialization for `LiquidityController` tests.
contract LiquidityController_TestInit is CommonTest {
    using stdStorage for StdStorage;

    /// @notice Emitted when an address withdraws native asset liquidity.
    event LiquidityWithdrawn(address indexed caller, uint256 value);

    /// @notice Emitted when an address deposits native asset liquidity.
    event LiquidityDeposited(address indexed caller, uint256 value);

    /// @notice Emitted when an address is deauthorized to mint/burn liquidity
    event MinterDeauthorized(address indexed minter);

    /// @notice Emitted when an address is authorized to mint/burn liquidity
    event MinterAuthorized(address indexed minter);

    /// @notice Emitted when liquidity is minted
    event LiquidityMinted(address indexed minter, address indexed to, uint256 amount);

    /// @notice Emitted when liquidity is burned
    event LiquidityBurned(address indexed minter, uint256 amount);

    /// @notice Test setup.
    function setUp() public virtual override {
        super.setUp();
        skipIfSysFeatureDisabled(Features.CUSTOM_GAS_TOKEN);
    }

    /// @notice Helper function to authorize a minter.
    function _authorizeMinter(address _minter) internal {
        assumeNotForgeAddress(_minter);
        assumeNotZeroAddress(_minter);
        // Authorize the minter
        stdstore.target(address(liquidityController)).sig(liquidityController.minters.selector).with_key(_minter)
            .checked_write(true);
    }
}

/// @title LiquidityController_Version_Test
/// @notice Tests the `version` function of the `LiquidityController` contract.
contract LiquidityController_Version_Test is LiquidityController_TestInit {
    /// @notice Tests that the version function returns a valid string.
    function test_version_succeeds() public view {
        assert(bytes(liquidityController.version()).length > 0);
    }
}

/// @title LiquidityController_GasPayingTokenName_Test
/// @notice Tests the `gasPayingTokenName` function of the `LiquidityController` contract.
contract LiquidityController_GasPayingTokenName_Test is LiquidityController_TestInit {
    /// @notice Tests that the `version` function returns the correct string. We avoid testing the
    ///         specific value of the string as it changes frequently.
    function test_gasPayingTokenName_succeeds() public view {
        assertTrue(bytes(liquidityController.gasPayingTokenName()).length > 0);
    }
}

/// @title LiquidityController_GasPayingTokenSymbol_Test
/// @notice Tests the `gasPayingTokenSymbol` function of the `LiquidityController` contract.
contract LiquidityController_GasPayingTokenSymbol_Test is LiquidityController_TestInit {
    /// @notice Tests that the gasPayingTokenSymbol function returns a valid string.
    function test_gasPayingTokenSymbol_succeeds() public view {
        assertTrue(bytes(liquidityController.gasPayingTokenSymbol()).length > 0);
    }
}

/// @title LiquidityController_AuthorizeMinter_Test
/// @notice Tests the `authorizeMinter` function of the `LiquidityController` contract.
contract LiquidityController_AuthorizeMinter_Test is LiquidityController_TestInit {
    /// @notice Tests that the authorizeMinter function can be called by the owner.
    function testFuzz_authorizeMinter_fromOwner_succeeds(address _minter) public {
        // Expect emit MinterAuthorized event
        vm.expectEmit(address(liquidityController));
        emit MinterAuthorized(_minter);
        // Call the authorizeMinter function with owner as the caller
        vm.prank(liquidityController.owner());
        liquidityController.authorizeMinter(_minter);

        // Assert minter is authorized
        assertTrue(liquidityController.minters(_minter));
    }

    /// @notice Tests that the authorizeMinter function reverts when called by non-owner.
    function testFuzz_authorizeMinter_fromNonOwner_fails(address _caller, address _minter) public {
        vm.assume(_caller != liquidityController.owner());

        // Call the authorizeMinter function with non-owner as the caller
        vm.prank(_caller);
        vm.expectRevert("Ownable: caller is not the owner");
        liquidityController.authorizeMinter(_minter);

        // Assert minter is not authorized
        assertFalse(liquidityController.minters(_minter));
    }
}

/// @title LiquidityController_DeauthorizeMinter_Test
/// @notice Tests the `deauthorizeMinter` function of the `LiquidityController` contract.
contract LiquidityController_DeauthorizeMinter_Test is LiquidityController_TestInit {
    /// @notice Tests that the deauthorizeMinter function can be called by the owner.
    function testFuzz_deauthorizeMinter_fromOwner_succeeds(address _minter) public {
        // Set minter to authorized
        _authorizeMinter(_minter);

        // Expect emit MinterDeauthorized event
        vm.expectEmit(address(liquidityController));
        emit MinterDeauthorized(_minter);
        // Call the deauthorizeMinter function with owner as the caller
        vm.prank(liquidityController.owner());
        liquidityController.deauthorizeMinter(_minter);

        // Assert minter is deauthorized
        assertFalse(liquidityController.minters(_minter));
    }

    /// @notice Tests that the deauthorizeMinter function reverts when called by non-owner.
    function testFuzz_deauthorizeMinter_fromNonOwner_fails(address _caller, address _minter) public {
        vm.assume(_caller != liquidityController.owner());

        // Set minter to authorized
        _authorizeMinter(_minter);

        // Call the deauthorizeMinter function with non-owner as the caller
        vm.prank(_caller);
        vm.expectRevert("Ownable: caller is not the owner");
        liquidityController.deauthorizeMinter(_minter);

        // Assert minter is still authorized
        assertTrue(liquidityController.minters(_minter));
    }
}

/// @title LiquidityController_Mint_Test
/// @notice Tests the `mint` function of the `LiquidityController` contract.
contract LiquidityController_Mint_Test is LiquidityController_TestInit {
    /// @notice Tests that the mint function can be called by an authorized minter.
    function testFuzz_mint_fromAuthorizedMinter_succeeds(address _to, uint256 _amount, address _minter) public {
        _authorizeMinter(_minter);
        vm.assume(_to != address(nativeAssetLiquidity));
        _amount = bound(_amount, 1, address(nativeAssetLiquidity).balance);

        // Record initial balances
        uint256 nativeAssetBalanceBefore = address(nativeAssetLiquidity).balance;
        uint256 toBalanceBefore = _to.balance;

        // Expect emit LiquidityWithdrawn event and call the mint function
        vm.expectEmit(address(nativeAssetLiquidity));
        emit LiquidityWithdrawn(address(liquidityController), _amount);
        // Expect emit LiquidityMinted event
        vm.expectEmit(address(liquidityController));
        emit LiquidityMinted(_minter, _to, _amount);
        vm.prank(_minter);
        liquidityController.mint(_to, _amount);

        // Assert recipient and NativeAssetLiquidity balances are updated correctly
        assertEq(_to.balance, toBalanceBefore + _amount);
        assertEq(address(nativeAssetLiquidity).balance, nativeAssetBalanceBefore - _amount);
    }

    /// @notice Tests that the mint function reverts when called by unauthorized address.
    function testFuzz_mint_fromUnauthorizedCaller_fails(address _caller, address _to, uint256 _amount) public {
        _amount = bound(_amount, 1, address(nativeAssetLiquidity).balance);

        uint256 nativeAssetBalanceBefore = address(nativeAssetLiquidity).balance;
        uint256 toBalanceBefore = _to.balance;

        // Call the mint function with unauthorized caller
        vm.prank(_caller);
        vm.expectRevert(LiquidityController.LiquidityController_Unauthorized.selector);
        liquidityController.mint(_to, _amount);

        // Assert recipient and NativeAssetLiquidity balances remain unchanged
        assertEq(_to.balance, toBalanceBefore);
        assertEq(address(nativeAssetLiquidity).balance, nativeAssetBalanceBefore);
    }

    /// @notice Tests that the mint function reverts when contract has insufficient balance.
    function test_mint_insufficientBalance_fails(address _minter) public {
        _authorizeMinter(_minter);
        // Try to mint more than available balance
        uint256 contractBalance = address(nativeAssetLiquidity).balance;
        uint256 amount = bound(contractBalance, contractBalance + 1, type(uint256).max);
        address to = makeAddr("recipient");

        // Call the mint function with insufficient balance
        vm.prank(_minter);
        // Should revert due to insufficient balance in NativeAssetLiquidity
        vm.expectRevert(NativeAssetLiquidity.NativeAssetLiquidity_InsufficientBalance.selector);

        liquidityController.mint(to, amount);

        // Assert recipient and NativeAssetLiquidity balances remain unchanged
        assertEq(to.balance, 0);
        assertEq(address(nativeAssetLiquidity).balance, contractBalance);
    }
}

/// @title LiquidityController_Burn_Test
/// @notice Tests the `burn` function of the `LiquidityController` contract.
contract LiquidityController_Burn_Test is LiquidityController_TestInit {
    /// @notice Tests that the burn function can be called by an authorized minter.
    function testFuzz_burn_fromAuthorizedMinter_succeeds(uint256 _amount, address _minter) public {
        vm.assume(_minter != Predeploys.NATIVE_ASSET_LIQUIDITY);

        _authorizeMinter(_minter);
        _amount = bound(_amount, 0, address(nativeAssetLiquidity).balance);

        // Deal the authorized minter with the amount to burn
        vm.deal(_minter, _amount);
        uint256 nativeAssetBalanceBefore = address(nativeAssetLiquidity).balance;
        uint256 minterBalanceBefore = _minter.balance;

        // Expect emit LiquidityDeposited event and call the burn function
        vm.expectEmit(address(nativeAssetLiquidity));
        emit LiquidityDeposited(address(liquidityController), _amount);
        // Expect emit LiquidityBurned event
        vm.expectEmit(address(liquidityController));
        emit LiquidityBurned(_minter, _amount);
        vm.prank(_minter);
        liquidityController.burn{ value: _amount }();

        // Assert minter and NativeAssetLiquidity balances are updated correctly
        assertEq(_minter.balance, minterBalanceBefore - _amount);
        assertEq(address(nativeAssetLiquidity).balance, nativeAssetBalanceBefore + _amount);
    }

    /// @notice Tests that the burn function reverts when called by unauthorized address.
    function testFuzz_burn_fromUnauthorizedCaller_fails(address _caller, uint256 _amount, address _minter) public {
        _authorizeMinter(_minter);
        vm.assume(_caller != _minter);
        _amount = bound(_amount, 0, address(nativeAssetLiquidity).balance);

        // Deal the unauthorized caller with the amount to burn
        vm.deal(_caller, _amount);
        uint256 nativeAssetBalanceBefore = address(nativeAssetLiquidity).balance;
        uint256 callerBalanceBefore = _caller.balance;

        // Call the burn function with unauthorized caller
        vm.prank(_caller);
        vm.expectRevert(LiquidityController.LiquidityController_Unauthorized.selector);
        liquidityController.burn{ value: _amount }();

        // Assert caller and NativeAssetLiquidity balances remain unchanged
        assertEq(_caller.balance, callerBalanceBefore);
        assertEq(address(nativeAssetLiquidity).balance, nativeAssetBalanceBefore);
    }
}

/// @title LiquidityController_Initialize_Test
/// @notice Tests the `initialize` function of the `LiquidityController` contract.
contract LiquidityController_Initialize_Test is LiquidityController_TestInit {
    /// @notice Tests that calling initialize on the implementation contract reverts.
    function testFuzz_initialize_implementation_reverts(address _owner) public {
        vm.assume(_owner != address(0));
        // Deploy a new implementation contract directly (not through proxy)
        LiquidityController implementation = new LiquidityController();

        // Try to initialize the implementation contract directly
        // This should revert because _disableInitializers() was called in the constructor
        vm.expectRevert("Initializable: contract is already initialized");
        implementation.initialize(_owner, "Test Token", "TEST");

        // Assert owner is set correctly
        assertNotEq(implementation.owner(), _owner);
    }
}
