// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Test } from "forge-std/Test.sol";
import { AddressManager } from "../legacy/AddressManager.sol";
import { ResolvedDelegateProxy } from "../legacy/ResolvedDelegateProxy.sol";

contract ResolvedDelegateProxy_Test is Test {
    AddressManager internal addressManager;
    SimpleImplementation internal impl;
    SimpleImplementation internal proxy;

    function setUp() public {
        // Set up the address manager.
        addressManager = new AddressManager();
        impl = new SimpleImplementation();
        addressManager.setAddress("SimpleImplementation", address(impl));

        // Set up the proxy.
        proxy = SimpleImplementation(
            address(new ResolvedDelegateProxy(addressManager, "SimpleImplementation"))
        );
    }

    // Tests that the proxy properly bubbles up returndata when the delegatecall succeeds.
    function testFuzz_fallback_delegateCallFoo_succeeds(uint256 x) public {
        vm.expectCall(address(impl), abi.encodeWithSelector(impl.foo.selector, x));
        assertEq(proxy.foo(x), x);
    }

    // Tests that the proxy properly bubbles up returndata when the delegatecall reverts.
    function test_fallback_delegateCallBar_reverts() public {
        vm.expectRevert("SimpleImplementation: revert");
        vm.expectCall(address(impl), abi.encodeWithSelector(impl.bar.selector));
        proxy.bar();
    }

    // Tests that the proxy fallback reverts as expected if the implementation within the
    // address manager is not set.
    function test_fallback_addressManagerNotSet_reverts() public {
        AddressManager am = new AddressManager();
        SimpleImplementation p = SimpleImplementation(
            address(new ResolvedDelegateProxy(am, "SimpleImplementation"))
        );

        vm.expectRevert("ResolvedDelegateProxy: target address must be initialized");
        p.foo(0);
    }
}

contract SimpleImplementation {
    function foo(uint256 _x) public pure returns (uint256) {
        return _x;
    }

    function bar() public pure {
        revert("SimpleImplementation: revert");
    }
}
