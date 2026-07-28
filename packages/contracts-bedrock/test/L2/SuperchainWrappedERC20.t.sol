// SPDX-License-Identifier: MIT
pragma solidity 0.8.25;

// Testing
import { SuperchainERC20Factory_TestInit } from "test/L2/SuperchainERC20Factory.t.sol";

// Libraries
import { Predeploys } from "src/libraries/Predeploys.sol";
import { Unauthorized } from "src/libraries/errors/CommonErrors.sol";

// Target contract
import { ISuperchainWrappedERC20 } from "interfaces/L2/ISuperchainWrappedERC20.sol";
import { IERC7802 } from "interfaces/L2/IERC7802.sol";

/// @title SuperchainWrappedERC20_TestInit
/// @notice Reusable test initialization for `SuperchainWrappedERC20` tests.
abstract contract SuperchainWrappedERC20_TestInit is SuperchainERC20Factory_TestInit {
    ISuperchainWrappedERC20 public wrapped;

    /// @notice Sets up the test suite.
    function setUp() public override {
        super.setUp();
        wrapped = _deployWrappedToken();
    }
}

/// @title SuperchainWrappedERC20_Mint_Test
/// @notice Tests the `mint` function of the `SuperchainWrappedERC20` contract.
contract SuperchainWrappedERC20_Mint_Test is SuperchainWrappedERC20_TestInit {
    /// @notice Tests the `mint` function reverts when the caller is not the factory.
    function testFuzz_mint_callerNotFactory_reverts(address _caller, address _to, uint256 _amount) public {
        vm.assume(_caller != address(factory));

        vm.expectRevert(Unauthorized.selector);
        vm.prank(_caller);
        wrapped.mint(_to, _amount);
    }
}

/// @title SuperchainWrappedERC20_Burn_Test
/// @notice Tests the `burn` function of the `SuperchainWrappedERC20` contract.
contract SuperchainWrappedERC20_Burn_Test is SuperchainWrappedERC20_TestInit {
    /// @notice Tests the `burn` function reverts when the caller is not the factory.
    function testFuzz_burn_callerNotFactory_reverts(address _caller, address _from, uint256 _amount) public {
        vm.assume(_caller != address(factory));

        vm.expectRevert(Unauthorized.selector);
        vm.prank(_caller);
        wrapped.burn(_from, _amount);
    }
}

/// @title SuperchainWrappedERC20_CrosschainMint_Test
/// @notice Tests the `crosschainMint` function of the `SuperchainWrappedERC20` contract.
contract SuperchainWrappedERC20_CrosschainMint_Test is SuperchainWrappedERC20_TestInit {
    /// @notice Tests the SuperchainTokenBridge can mint and burn the wrapped token, which is what
    ///         moves it across chains.
    function testFuzz_crosschainMint_bridge_succeeds(address _to, uint256 _amount) public {
        vm.assume(_to != address(0));

        vm.prank(Predeploys.SUPERCHAIN_TOKEN_BRIDGE);
        wrapped.crosschainMint(_to, _amount);
        assertEq(wrapped.balanceOf(_to), _amount);

        vm.prank(Predeploys.SUPERCHAIN_TOKEN_BRIDGE);
        wrapped.crosschainBurn(_to, _amount);
        assertEq(wrapped.balanceOf(_to), 0);
    }

    /// @notice Tests the `crosschainMint` function reverts when the caller is not the bridge.
    function testFuzz_crosschainMint_callerNotBridge_reverts(address _caller, address _to, uint256 _amount) public {
        vm.assume(_caller != Predeploys.SUPERCHAIN_TOKEN_BRIDGE);

        vm.expectRevert(Unauthorized.selector);
        vm.prank(_caller);
        wrapped.crosschainMint(_to, _amount);
    }
}

/// @title SuperchainWrappedERC20_SupportsInterface_Test
/// @notice Tests the `supportsInterface` function of the `SuperchainWrappedERC20` contract.
contract SuperchainWrappedERC20_SupportsInterface_Test is SuperchainWrappedERC20_TestInit {
    /// @notice Tests the wrapped token supports the IERC7802 interface.
    function test_supportsInterface_succeeds() public view {
        assertTrue(wrapped.supportsInterface(type(IERC7802).interfaceId));
    }
}
