// SPDX-License-Identifier: MIT
pragma solidity ^0.8.9;

import { Test } from "forge-std/Test.sol";
import { Proxy } from "../universal/Proxy.sol";
import { ProxyAdmin } from "../universal/ProxyAdmin.sol";
import { SimpleStorage } from "./Proxy.t.sol";

contract ProxyAdmin_Test is Test {
    address alice = address(64);

    Proxy proxy;
    ProxyAdmin admin;

    SimpleStorage implementation;

    function setUp() external {
        admin = new ProxyAdmin(alice);
        proxy = new Proxy(address(admin));

        implementation = new SimpleStorage();
    }

    function test_getProxyImplementation() external {
        {
            address impl = admin.getProxyImplementation(proxy);
            assertEq(impl, address(0));
        }

        vm.prank(alice);
        admin.upgrade(proxy, address(implementation));

        {
            address impl = admin.getProxyImplementation(proxy);
            assertEq(impl, address(implementation));
        }
    }

    function test_getProxyAdmin() external {
        address owner = admin.getProxyAdmin(proxy);
        assertEq(owner, address(admin));
    }

    function test_changeProxyAdmin() external {
        vm.prank(alice);
        admin.changeProxyAdmin(proxy, address(128));

        // The proxy is not longer the admin and can
        // no longer call the proxy interface
        vm.expectRevert();
        admin.getProxyAdmin(proxy);

        // The new admin is the owner
        vm.prank(address(128));
        assertEq(proxy.admin(), address(128));
    }

    function test_upgrade() external {
        vm.prank(alice);
        admin.upgrade(proxy, address(implementation));

        address impl = admin.getProxyImplementation(proxy);
        assertEq(impl, address(implementation));
    }

    function test_upgradeAndCall() external {
        vm.prank(alice);
        admin.upgradeAndCall(
            proxy,
            address(implementation),
            abi.encodeWithSelector(SimpleStorage.set.selector, 1, 1)
        );

        address impl = admin.getProxyImplementation(proxy);
        assertEq(impl, address(implementation));

        uint256 got = SimpleStorage(address(proxy)).get(1);
        assertEq(got, 1);
    }

    function test_onlyOwner() external {
        vm.expectRevert("UNAUTHORIZED");
        admin.changeProxyAdmin(proxy, address(0));

        vm.expectRevert("UNAUTHORIZED");
        admin.upgrade(proxy, address(implementation));

        vm.expectRevert("UNAUTHORIZED");
        admin.upgradeAndCall(proxy, address(implementation), hex"");
    }

    function test_isUpgrading() external {
        assertEq(false, admin.isUpgrading());
    }
}
