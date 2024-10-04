// SPDX-License-Identifier: MIT
pragma solidity 0.8.25;

// Testing utilities
import { Test } from "forge-std/Test.sol";
import { EIP1967Helper } from "test/mocks/EIP1967Helper.sol";

// Libraries
import { Predeploys } from "src/libraries/Predeploys.sol";
import { IERC20 } from "@openzeppelin/contracts-v5/token/ERC20/IERC20.sol";
import { IL2ToL2CrossDomainMessenger } from "src/L2/interfaces/IL2ToL2CrossDomainMessenger.sol";
import { ERC1967Proxy } from "@openzeppelin/contracts-v5/proxy/ERC1967/ERC1967Proxy.sol";
import { Initializable } from "@openzeppelin/contracts-v5/proxy/utils/Initializable.sol";
import { IERC165 } from "@openzeppelin/contracts-v5/utils/introspection/IERC165.sol";
import { IBeacon } from "@openzeppelin/contracts-v5/proxy/beacon/IBeacon.sol";
import { BeaconProxy } from "@openzeppelin/contracts-v5/proxy/beacon/BeaconProxy.sol";

// Target contract
import { SuperchainERC20, ISuperchainERC20Extension } from "src/L2/SuperchainERC20.sol";
import { ISuperchainERC20Errors } from "src/L2/interfaces/ISuperchainERC20.sol";
import { SuperchainERC20Implementation_MockContract } from "test/mocks/SuperchainERC20Implementation.sol";

/// @title SuperchainERC20Test
/// @notice Contract for testing the SuperchainERC20 contract.
contract SuperchainERC20Test is Test {
    address internal constant ZERO_ADDRESS = address(0);
    address internal constant SUPERCHAIN_ERC20_BRIDGE = Predeploys.SUPERCHAIN_ERC20_BRIDGE;
    address internal constant MESSENGER = Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER;

    SuperchainERC20 public superchainERC20;

    /// @notice Sets up the test suite.
    function setUp() public {
        superchainERC20 = new SuperchainERC20Implementation_MockContract();
    }

    /// @notice Helper function to setup a mock and expect a call to it.
    function _mockAndExpect(address _receiver, bytes memory _calldata, bytes memory _returned) internal {
        vm.mockCall(_receiver, _calldata, _returned);
        vm.expectCall(_receiver, _calldata);
    }

    /// @notice Tests the `mint` function reverts when the caller is not the bridge.
    function testFuzz___superchainMint_callerNotBridge_reverts(address _caller, address _to, uint256 _amount) public {
        // Ensure the caller is not the bridge
        vm.assume(_caller != SUPERCHAIN_ERC20_BRIDGE);

        // Expect the revert with `OnlySuperchainERC20Bridge` selector
        vm.expectRevert(ISuperchainERC20Errors.OnlySuperchainERC20Bridge.selector);

        // Call the `mint` function with the non-bridge caller
        vm.prank(_caller);
        superchainERC20.__superchainMint(_to, _amount);
    }

    /// @notice Tests the `mint` succeeds and emits the `Mint` event.
    function testFuzz___superchainMint_succeeds(address _to, uint256 _amount) public {
        // Ensure `_to` is not the zero address
        vm.assume(_to != ZERO_ADDRESS);

        // Get the total supply and balance of `_to` before the mint to compare later on the assertions
        uint256 _totalSupplyBefore = superchainERC20.totalSupply();
        uint256 _toBalanceBefore = superchainERC20.balanceOf(_to);

        // Look for the emit of the `Transfer` event
        vm.expectEmit(address(superchainERC20));
        emit IERC20.Transfer(ZERO_ADDRESS, _to, _amount);

        // Look for the emit of the `SuperchainMinted` event
        vm.expectEmit(address(superchainERC20));
        emit ISuperchainERC20Extension.SuperchainMinted(_to, _amount);

        // Call the `mint` function with the bridge caller
        vm.prank(SUPERCHAIN_ERC20_BRIDGE);
        superchainERC20.__superchainMint(_to, _amount);

        // Check the total supply and balance of `_to` after the mint were updated correctly
        assertEq(superchainERC20.totalSupply(), _totalSupplyBefore + _amount);
        assertEq(superchainERC20.balanceOf(_to), _toBalanceBefore + _amount);
    }

    /// @notice Tests the `burn` function reverts when the caller is not the bridge.
    function testFuzz___superchainBurn_callerNotBridge_reverts(
        address _caller,
        address _from,
        uint256 _amount
    )
        public
    {
        // Ensure the caller is not the bridge
        vm.assume(_caller != SUPERCHAIN_ERC20_BRIDGE);

        // Expect the revert with `OnlySuperchainERC20Bridge` selector
        vm.expectRevert(ISuperchainERC20Errors.OnlySuperchainERC20Bridge.selector);

        // Call the `burn` function with the non-bridge caller
        vm.prank(_caller);
        superchainERC20.__superchainBurn(_from, _amount);
    }

    /// @notice Tests the `burn` burns the amount and emits the `SuperchainBurnt` event.
    function testFuzz___superchainBurn_succeeds(address _from, uint256 _amount) public {
        // Ensure `_from` is not the zero address
        vm.assume(_from != ZERO_ADDRESS);

        // Mint some tokens to `_from` so then they can be burned
        vm.prank(SUPERCHAIN_ERC20_BRIDGE);
        superchainERC20.__superchainMint(_from, _amount);

        // Get the total supply and balance of `_from` before the burn to compare later on the assertions
        uint256 _totalSupplyBefore = superchainERC20.totalSupply();
        uint256 _fromBalanceBefore = superchainERC20.balanceOf(_from);

        // Look for the emit of the `Transfer` event
        vm.expectEmit(address(superchainERC20));
        emit IERC20.Transfer(_from, ZERO_ADDRESS, _amount);

        // Look for the emit of the `SuperchainBurnt` event
        vm.expectEmit(address(superchainERC20));
        emit ISuperchainERC20Extension.SuperchainBurnt(_from, _amount);

        // Call the `burn` function with the bridge caller
        vm.prank(SUPERCHAIN_ERC20_BRIDGE);
        superchainERC20.__superchainBurn(_from, _amount);

        // Check the total supply and balance of `_from` after the burn were updated correctly
        assertEq(superchainERC20.totalSupply(), _totalSupplyBefore - _amount);
        assertEq(superchainERC20.balanceOf(_from), _fromBalanceBefore - _amount);
    }
}
