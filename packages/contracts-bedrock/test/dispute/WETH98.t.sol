pragma solidity 0.8.15;

import { Test } from "forge-std/Test.sol";
import { WETH98 } from "src/dispute/weth/WETH98.sol";

contract WETH98_Test is Test {
    WETH98 public weth;
    address alice;

    function setUp() public {
        weth = new WETH98();
        alice = makeAddr("alice");
        deal(alice, 1 ether);
    }

    function test_receive_succeeds() public {
        vm.prank(alice);
        (bool success,) = address(weth).call{ value: 1 ether }("");
        assertTrue(success);
        assertEq(weth.balanceOf(alice), 1 ether);
    }

    function test_fallback_succeeds() public {
        vm.prank(alice);
        (bool success,) = address(weth).call{ value: 1 ether }(hex"1234");
        assertTrue(success);
        assertEq(weth.balanceOf(alice), 1 ether);
    }

    function test_getName_succeeds() public {
        assertEq(weth.name(), "Wrapped Ether");
        assertEq(weth.symbol(), "WETH");
        assertEq(weth.decimals(), 18);
    }
}
