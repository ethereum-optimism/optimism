// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Test } from "forge-std/Test.sol";
import { Proxy } from "../universal/Proxy.sol";
import { ProxyAdmin } from "../universal/ProxyAdmin.sol";
import { SimpleStorage } from "./Proxy.t.sol";
import { L1ChugSplashProxy } from "../legacy/L1ChugSplashProxy.sol";
import { ResolvedDelegateProxy } from "../legacy/ResolvedDelegateProxy.sol";
import { AddressManager } from "../legacy/AddressManager.sol";

contract ProxyAdmin_Test is Test {
    address alice = address(64);

    Proxy proxy;
    L1ChugSplashProxy chugsplash;
    ResolvedDelegateProxy resolved;

    AddressManager addressManager;

    ProxyAdmin admin;

    SimpleStorage implementation;

    function setUp() external {
        // Deploy the proxy admin
        admin = new ProxyAdmin(alice);
        // Deploy the standard proxy
        proxy = new Proxy(address(admin));

        // Deploy the legacy L1ChugSplashProxy with the admin as the owner
        chugsplash = new L1ChugSplashProxy(address(admin));

        // Deploy the legacy AddressManager
        addressManager = new AddressManager();
        // The proxy admin must be the new owner of the address manager
        addressManager.transferOwnership(address(admin));
        // Deploy a legacy ResolvedDelegateProxy with the name `a`.
        // Whatever `a` is set to in AddressManager will be the address
        // that is used for the implementation.
        resolved = new ResolvedDelegateProxy(addressManager, "a");

        // Impersonate alice for setting up the admin.
        vm.startPrank(alice);
        // Set the address of the address manager in the admin so that it
        // can resolve the implementation address of legacy
        // ResolvedDelegateProxy based proxies.
        admin.setAddressManager(addressManager);
        // Set the reverse lookup of the ResolvedDelegateProxy
        // proxy
        admin.setImplementationName(address(resolved), "a");

        // Set the proxy types
        admin.setProxyType(address(proxy), ProxyAdmin.ProxyType.ERC1967);
        admin.setProxyType(address(chugsplash), ProxyAdmin.ProxyType.CHUGSPLASH);
        admin.setProxyType(address(resolved), ProxyAdmin.ProxyType.RESOLVED);
        vm.stopPrank();

        implementation = new SimpleStorage();
    }

    function test_setImplementationName() external {
        vm.prank(alice);
        admin.setImplementationName(address(1), "foo");
        assertEq(admin.implementationName(address(1)), "foo");
    }

    function test_onlyOwnerSetAddressManager() external {
        vm.expectRevert("Ownable: caller is not the owner");
        admin.setAddressManager(AddressManager((address(0))));
    }

    function test_onlyOwnerSetImplementationName() external {
        vm.expectRevert("Ownable: caller is not the owner");
        admin.setImplementationName(address(0), "foo");
    }

    function test_onlyOwnerSetProxyType() external {
        vm.expectRevert("Ownable: caller is not the owner");
        admin.setProxyType(address(0), ProxyAdmin.ProxyType.CHUGSPLASH);
    }

    function test_owner() external {
        assertEq(admin.owner(), alice);
    }

    function test_proxyType() external {
        assertEq(uint256(admin.proxyType(address(proxy))), uint256(ProxyAdmin.ProxyType.ERC1967));
        assertEq(
            uint256(admin.proxyType(address(chugsplash))),
            uint256(ProxyAdmin.ProxyType.CHUGSPLASH)
        );
        assertEq(
            uint256(admin.proxyType(address(resolved))),
            uint256(ProxyAdmin.ProxyType.RESOLVED)
        );
    }

    function test_erc1967GetProxyImplementation() external {
        getProxyImplementation(payable(proxy));
    }

    function test_chugsplashGetProxyImplementation() external {
        getProxyImplementation(payable(chugsplash));
    }

    function test_delegateResolvedGetProxyImplementation() external {
        getProxyImplementation(payable(resolved));
    }

    function getProxyImplementation(address payable _proxy) internal {
        {
            address impl = admin.getProxyImplementation(_proxy);
            assertEq(impl, address(0));
        }

        vm.prank(alice);
        admin.upgrade(_proxy, address(implementation));

        {
            address impl = admin.getProxyImplementation(_proxy);
            assertEq(impl, address(implementation));
        }
    }

    function test_erc1967GetProxyAdmin() external {
        getProxyAdmin(payable(proxy));
    }

    function test_chugsplashGetProxyAdmin() external {
        getProxyAdmin(payable(chugsplash));
    }

    function test_delegateResolvedGetProxyAdmin() external {
        getProxyAdmin(payable(resolved));
    }

    function getProxyAdmin(address payable _proxy) internal {
        address owner = admin.getProxyAdmin(_proxy);
        assertEq(owner, address(admin));
    }

    function test_erc1967ChangeProxyAdmin() external {
        changeProxyAdmin(payable(proxy));
    }

    function test_chugsplashChangeProxyAdmin() external {
        changeProxyAdmin(payable(chugsplash));
    }

    function test_delegateResolvedChangeProxyAdmin() external {
        changeProxyAdmin(payable(resolved));
    }

    function changeProxyAdmin(address payable _proxy) internal {
        ProxyAdmin.ProxyType proxyType = admin.proxyType(address(_proxy));

        vm.prank(alice);
        admin.changeProxyAdmin(_proxy, address(128));

        // The proxy is no longer the admin and can
        // no longer call the proxy interface except for
        // the ResolvedDelegate type on which anybody can
        // call the admin interface.
        if (proxyType == ProxyAdmin.ProxyType.ERC1967) {
            vm.expectRevert("Proxy: implementation not initialized");
            admin.getProxyAdmin(_proxy);
        } else if (proxyType == ProxyAdmin.ProxyType.CHUGSPLASH) {
            vm.expectRevert("L1ChugSplashProxy: implementation is not set yet");
            admin.getProxyAdmin(_proxy);
        } else if (proxyType == ProxyAdmin.ProxyType.RESOLVED) {
            // Just an empty block to show that all cases are covered
        } else {
            vm.expectRevert("ProxyAdmin: unknown proxy type");
        }

        // Call the proxy contract directly to get the admin.
        // Different proxy types have different interfaces.
        vm.prank(address(128));
        if (proxyType == ProxyAdmin.ProxyType.ERC1967) {
            assertEq(Proxy(payable(_proxy)).admin(), address(128));
        } else if (proxyType == ProxyAdmin.ProxyType.CHUGSPLASH) {
            assertEq(L1ChugSplashProxy(payable(_proxy)).getOwner(), address(128));
        } else if (proxyType == ProxyAdmin.ProxyType.RESOLVED) {
            assertEq(addressManager.owner(), address(128));
        } else {
            assert(false);
        }
    }

    function test_erc1967Upgrade() external {
        upgrade(payable(proxy));
    }

    function test_chugsplashUpgrade() external {
        upgrade(payable(chugsplash));
    }

    function test_delegateResolvedUpgrade() external {
        upgrade(payable(resolved));
    }

    function upgrade(address payable _proxy) internal {
        vm.prank(alice);
        admin.upgrade(_proxy, address(implementation));

        address impl = admin.getProxyImplementation(_proxy);
        assertEq(impl, address(implementation));
    }

    function test_erc1967UpgradeAndCall() external {
        upgradeAndCall(payable(proxy));
    }

    function test_chugsplashUpgradeAndCall() external {
        upgradeAndCall(payable(chugsplash));
    }

    function test_delegateResolvedUpgradeAndCall() external {
        upgradeAndCall(payable(resolved));
    }

    function upgradeAndCall(address payable _proxy) internal {
        vm.prank(alice);
        admin.upgradeAndCall(
            _proxy,
            address(implementation),
            abi.encodeWithSelector(SimpleStorage.set.selector, 1, 1)
        );

        address impl = admin.getProxyImplementation(_proxy);
        assertEq(impl, address(implementation));

        uint256 got = SimpleStorage(address(_proxy)).get(1);
        assertEq(got, 1);
    }

    function test_onlyOwner() external {
        vm.expectRevert("Ownable: caller is not the owner");
        admin.changeProxyAdmin(payable(proxy), address(0));

        vm.expectRevert("Ownable: caller is not the owner");
        admin.upgrade(payable(proxy), address(implementation));

        vm.expectRevert("Ownable: caller is not the owner");
        admin.upgradeAndCall(payable(proxy), address(implementation), hex"");
    }

    function test_isUpgrading() external {
        assertEq(false, admin.isUpgrading());

        vm.prank(alice);
        admin.setUpgrading(true);
        assertEq(true, admin.isUpgrading());
    }
}
