// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Test } from "forge-std/Test.sol";

import { Permit2Lib, IAllowanceTransfer } from "src/libraries/Permit2Lib.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";
import { IERC20 } from "@openzeppelin/contracts/token/ERC20/IERC20.sol";

import { console } from "forge-std/console.sol";

/// @notice Simple testing contract that implements
contract AllowanceTransfer is IAllowanceTransfer {
    bool shouldRevert;

    address from;
    address to;
    uint160 amount;
    address token;

    function setShouldRevert(bool _shouldRevert) external {
        shouldRevert = _shouldRevert;
    }

    // @notice Implements the permit2 transferFrom interface
    function transferFrom(address _from, address _to, uint160 _amount, address _token) external {
        if (shouldRevert) {
            require(false);
        }

        from = _from;
        to = _to;
        amount = _amount;
        token = _token;
    }
}

/// @notice Non compliant ERC20 token that does not return anything when
///         `transferFrom` is called.
contract ERC20NonCompliant {
    function transferFrom(address, address, uint256) external {}
}

contract Permit2Lib_Test is Test {
    IERC20 weth = IERC20(Predeploys.WETH9);
    address alice;
    address bob;

    /// @notice Sets up the test environment with WETH9
    ///         to use as an ERC20 token. Also create addresses
    ///         for alice and bob.
    function setUp() external {
        bytes memory code = vm.getDeployedCode("WETH9.sol:WETH9");
        vm.etch(Predeploys.WETH9, code);

        alice = makeAddr("alice");
        bob = makeAddr("bob");

        vm.deal(alice, 2 ether);

        vm.prank(alice);
        (bool success, ) = address(weth).call{ value: 1 ether }(hex"");
        assertTrue(success);

        deployCodeTo("Permit2Lib.t.sol:AllowanceTransfer", address(Permit2Lib.PERMIT2));
    }

    /// @notice Simple transferFrom test. Approve bob as alice
    ///         and then have bob trasferFrom
    function test_transferFrom2_succeeds() external {
        uint256 bobBalance = weth.balanceOf(bob);
        uint256 aliceBalance = weth.balanceOf(alice);

        vm.prank(alice);
        weth.approve(bob, 1 ether);

        assertEq(weth.allowance(alice, bob), 1 ether);

        vm.prank(bob);
        Permit2Lib.safeTransferFrom2({
            _token: address(weth),
            _from: alice,
            _to: bob,
            _amount: 1 ether
        });

        assertEq(weth.balanceOf(bob), bobBalance + 1 ether);
        assertEq(weth.balanceOf(alice), aliceBalance - 1 ether);
    }

    /// @notice Ensure that transferFrom2 will revert when
    ///         the token address has no code.
    function test_transferFrom2_noCode_reverts() external {
        assertEq(address(0).code.length, 0);

        vm.expectRevert(Permit2Lib.NoCode.selector);
        Permit2Lib.safeTransferFrom2({
            _token: address(0),
            _from: alice,
            _to: bob,
            _amount: 1 ether
        });
    }

    function test_transferFrom2_unsafeCast_reverts() external {
        vm.expectRevert(Permit2Lib.UnsafeCast.selector);
        Permit2Lib._safeTransferFrom2({
            _token: address(weth),
            _from: alice,
            _to: bob,
            _amount: uint256(type(uint160).max) + 1
        });
    }

    /// @notice
    function test_transferFrom_nonCompliantReturnValueEmpty_succeeds() external {
        address erc20 = address(0x40);
        deployCodeTo("Permit2Lib.t.sol:ERC20NonCompliant", erc20);

        bool success = Permit2Lib.safeTransferFrom({
            _token: address(erc20),
            _from: alice,
            _to: bob,
            _amount: 1 ether
        });

        assertTrue(success);
    }

    function test_transferFrom_noCode_reverts() external {
        assertEq(address(0).code.length, 0);

        vm.expectRevert(Permit2Lib.NoCode.selector);
        Permit2Lib.safeTransferFrom({
            _token: address(0),
            _from: alice,
            _to: bob,
            _amount: 1 ether
        });
    }

    function test_transferFrom2_call_succeeds() public {
        vm.expectCall(
            address(Permit2Lib.PERMIT2),
            abi.encodeCall(Permit2Lib.PERMIT2.transferFrom, (alice, bob, uint160(1 ether), address(weth)))
        );

        Permit2Lib.safeTransferFrom2({
            _token: address(weth),
            _from: alice,
            _to: bob,
            _amount: 1 ether
        });
    }

    function test_transferFrom2_call_reverts() external {
        AllowanceTransfer(address(Permit2Lib.PERMIT2)).setShouldRevert(true);


        vm.expectRevert();
        Permit2Lib._safeTransferFrom2({
            _token: address(weth),
            _from: alice,
            _to: bob,
            _amount: 1 ether
        });
    }
}
