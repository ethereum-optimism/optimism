// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { Test } from "forge-std/Test.sol";
import { SimpleStorage } from "test/universal/Proxy.t.sol";

// Interfaces
import { IAddressManager } from "interfaces/legacy/IAddressManager.sol";
import { IL1ChugSplashProxy } from "interfaces/legacy/IL1ChugSplashProxy.sol";
import { IResolvedDelegateProxy } from "interfaces/legacy/IResolvedDelegateProxy.sol";
import { IProxy } from "interfaces/universal/IProxy.sol";
import { IProxyAdmin } from "interfaces/universal/IProxyAdmin.sol";

import { DeployUtils } from "scripts/libraries/DeployUtils.sol";

/// @title ProxyAdmin_TestInit
/// @notice Reusable test initialization for `ProxyAdmin` tests.
contract ProxyAdmin_TestInit is Test {
    address alice = address(64);

    IProxy proxy;
    IL1ChugSplashProxy chugsplash;
    IResolvedDelegateProxy resolved;

    IAddressManager addressManager;

    IProxyAdmin admin;

    SimpleStorage implementation;

    function setUp() external {
        // Deploy the proxy admin
        admin = IProxyAdmin(
            DeployUtils.create1({
                _name: "ProxyAdmin",
                _args: DeployUtils.encodeConstructor(abi.encodeCall(IProxyAdmin.__constructor__, (alice)))
            })
        );

        // Deploy the standard proxy
        proxy = IProxy(
            DeployUtils.create1({
                _name: "Proxy",
                _args: DeployUtils.encodeConstructor(abi.encodeCall(IProxy.__constructor__, (address(admin))))
            })
        );

        // Deploy the legacy L1ChugSplashProxy with the admin as the owner
        chugsplash = IL1ChugSplashProxy(
            DeployUtils.create1({
                _name: "L1ChugSplashProxy",
                _args: DeployUtils.encodeConstructor(abi.encodeCall(IL1ChugSplashProxy.__constructor__, (address(admin))))
            })
        );

        // Deploy the legacy AddressManager
        addressManager = IAddressManager(
            DeployUtils.create1({
                _name: "AddressManager",
                _args: DeployUtils.encodeConstructor(abi.encodeCall(IAddressManager.__constructor__, ()))
            })
        );
        // The proxy admin must be the new owner of the address manager
        addressManager.transferOwnership(address(admin));
        // Deploy a legacy ResolvedDelegateProxy with the name `a`. Whatever `a` is set to in
        // AddressManager will be the address that is used for the implementation.
        resolved = IResolvedDelegateProxy(
            DeployUtils.create1({
                _name: "ResolvedDelegateProxy",
                _args: DeployUtils.encodeConstructor(
                    abi.encodeCall(IResolvedDelegateProxy.__constructor__, (addressManager, "a"))
                )
            })
        );
        // Impersonate alice for setting up the admin.
        vm.startPrank(alice);
        // Set the address of the address manager in the admin so that it can resolve the
        // implementation address of legacy ResolvedDelegateProxy based proxies.
        admin.setAddressManager(IAddressManager(address(addressManager)));
        // Set the reverse lookup of the ResolvedDelegateProxy proxy
        admin.setImplementationName(address(resolved), "a");

        // Set the proxy types
        admin.setProxyType(address(proxy), IProxyAdmin.ProxyType.ERC1967);
        admin.setProxyType(address(chugsplash), IProxyAdmin.ProxyType.CHUGSPLASH);
        admin.setProxyType(address(resolved), IProxyAdmin.ProxyType.RESOLVED);
        vm.stopPrank();

        implementation = new SimpleStorage();
    }
}

/// @title ProxyAdmin_SetProxyType_Test
/// @notice Tests the `setProxyType` function of the `ProxyAdmin` contract.
contract ProxyAdmin_SetProxyType_Test is ProxyAdmin_TestInit {
    function test_setProxyType_notOwner_reverts() external {
        vm.expectRevert("Ownable: caller is not the owner");
        admin.setProxyType(address(0), IProxyAdmin.ProxyType.CHUGSPLASH);
    }
}

/// @title ProxyAdmin_SetImplementationName_Test
/// @notice Tests the `setImplementationName` function of the `ProxyAdmin` contract.
contract ProxyAdmin_SetImplementationName_Test is ProxyAdmin_TestInit {
    function test_setImplementationName_succeeds() external {
        vm.prank(alice);
        admin.setImplementationName(address(1), "foo");
        assertEq(admin.implementationName(address(1)), "foo");
    }

    function test_setImplementationName_notOwner_reverts() external {
        vm.expectRevert("Ownable: caller is not the owner");
        admin.setImplementationName(address(0), "foo");
    }
}

/// @title ProxyAdmin_SetAddressManager_Test
/// @notice Tests the `setAddressManager` function of the `ProxyAdmin` contract.
contract ProxyAdmin_SetAddressManager_Test is ProxyAdmin_TestInit {
    function test_setAddressManager_notOwner_reverts() external {
        vm.expectRevert("Ownable: caller is not the owner");
        admin.setAddressManager(IAddressManager((address(0))));
    }
}

/// @title ProxyAdmin_IsUpgrading_Test
/// @notice Tests the `isUpgrading` function of the `ProxyAdmin` contract.
contract ProxyAdmin_IsUpgrading_Test is ProxyAdmin_TestInit {
    function test_isUpgrading_succeeds() external {
        assertEq(false, admin.isUpgrading());

        vm.prank(alice);
        admin.setUpgrading(true);
        assertEq(true, admin.isUpgrading());
    }
}

/// @title ProxyAdmin_GetProxyImplementation_Test
/// @notice Tests the `getProxyImplementation` function of the `ProxyAdmin` contract.
contract ProxyAdmin_GetProxyImplementation_Test is ProxyAdmin_TestInit {
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

    function test_getProxyImplementation_erc1967_succeeds() external {
        getProxyImplementation(payable(proxy));
    }

    function test_getProxyImplementation_chugsplash_succeeds() external {
        getProxyImplementation(payable(chugsplash));
    }

    function test_getProxyImplementation_resolved_succeeds() external {
        getProxyImplementation(payable(resolved));
    }
}

/// @title ProxyAdmin_GetProxyAdmin_Test
/// @notice Tests the `getProxyAdmin` function of the `ProxyAdmin` contract.
contract ProxyAdmin_GetProxyAdmin_Test is ProxyAdmin_TestInit {
    function getProxyAdmin(address payable _proxy) internal view {
        address owner = admin.getProxyAdmin(_proxy);
        assertEq(owner, address(admin));
    }

    function test_getProxyAdmin_erc1967_succeeds() external view {
        getProxyAdmin(payable(proxy));
    }

    function test_getProxyAdmin_chugsplash_succeeds() external view {
        getProxyAdmin(payable(chugsplash));
    }

    function test_getProxyAdmin_resolved_succeeds() external view {
        getProxyAdmin(payable(resolved));
    }
}

/// @title ProxyAdmin_ChangeProxyAdmin_Test
/// @notice Tests the `changeProxyAdmin` function of the `ProxyAdmin` contract.
contract ProxyAdmin_ChangeProxyAdmin_Test is ProxyAdmin_TestInit {
    function changeProxyAdmin(address payable _proxy) internal {
        IProxyAdmin.ProxyType proxyType = admin.proxyType(address(_proxy));

        vm.prank(alice);
        admin.changeProxyAdmin(_proxy, address(128));

        // The proxy is no longer the admin and can
        // no longer call the proxy interface except for
        // the ResolvedDelegate type on which anybody can
        // call the admin interface.
        if (proxyType == IProxyAdmin.ProxyType.ERC1967) {
            vm.expectRevert("Proxy: implementation not initialized");
            admin.getProxyAdmin(_proxy);
        } else if (proxyType == IProxyAdmin.ProxyType.CHUGSPLASH) {
            vm.expectRevert("L1ChugSplashProxy: implementation is not set yet");
            admin.getProxyAdmin(_proxy);
        } else if (proxyType == IProxyAdmin.ProxyType.RESOLVED) {
            // Just an empty block to show that all cases are covered
        } else {
            vm.expectRevert("ProxyAdmin: unknown proxy type");
        }

        // Call the proxy contract directly to get the admin.
        // Different proxy types have different interfaces.
        vm.prank(address(128));
        if (proxyType == IProxyAdmin.ProxyType.ERC1967) {
            assertEq(IProxy(payable(_proxy)).admin(), address(128));
        } else if (proxyType == IProxyAdmin.ProxyType.CHUGSPLASH) {
            assertEq(IL1ChugSplashProxy(payable(_proxy)).getOwner(), address(128));
        } else if (proxyType == IProxyAdmin.ProxyType.RESOLVED) {
            assertEq(addressManager.owner(), address(128));
        } else {
            assert(false);
        }
    }

    function test_changeProxyAdmin_erc1967_succeeds() external {
        changeProxyAdmin(payable(proxy));
    }

    function test_changeProxyAdmin_chugsplash_succeeds() external {
        changeProxyAdmin(payable(chugsplash));
    }

    function test_changeProxyAdmin_resolved_succeeds() external {
        changeProxyAdmin(payable(resolved));
    }
}

/// @title ProxyAdmin_Upgrade_Test
/// @notice Tests the `upgrade` function of the `ProxyAdmin` contract.
contract ProxyAdmin_Upgrade_Test is ProxyAdmin_TestInit {
    function upgrade(address payable _proxy) internal {
        vm.prank(alice);
        admin.upgrade(_proxy, address(implementation));

        address impl = admin.getProxyImplementation(_proxy);
        assertEq(impl, address(implementation));
    }

    function test_upgrade_erc1967_succeeds() external {
        upgrade(payable(proxy));
    }

    function test_upgrade_chugsplash_succeeds() external {
        upgrade(payable(chugsplash));
    }

    function test_upgrade_resolved_succeeds() external {
        upgrade(payable(resolved));
    }
}

/// @title ProxyAdmin_UpgradeAndCall_Test
/// @notice Tests the `upgradeAndCall` function of the `ProxyAdmin` contract.
contract ProxyAdmin_UpgradeAndCall_Test is ProxyAdmin_TestInit {
    function upgradeAndCall(address payable _proxy) internal {
        vm.prank(alice);
        admin.upgradeAndCall(_proxy, address(implementation), abi.encodeCall(SimpleStorage.set, (1, 1)));

        address impl = admin.getProxyImplementation(_proxy);
        assertEq(impl, address(implementation));

        uint256 got = SimpleStorage(address(_proxy)).get(1);
        assertEq(got, 1);
    }

    function test_erc1967UpgradeAndCall_succeeds() external {
        upgradeAndCall(payable(proxy));
    }

    function test_chugsplashUpgradeAndCall_succeeds() external {
        upgradeAndCall(payable(chugsplash));
    }

    function test_delegateResolvedUpgradeAndCall_succeeds() external {
        upgradeAndCall(payable(resolved));
    }
}

/// @title ProxyAdmin_Unclassified_Test
/// @notice General tests that are not testing any function directly or that test multiple
///         functions of the `ProxyAdmin` contract.
contract ProxyAdmin_Unclassified_Test is ProxyAdmin_TestInit {
    function test_owner_succeeds() external view {
        assertEq(admin.owner(), alice);
    }

    function test_proxyType_succeeds() external view {
        assertEq(uint256(admin.proxyType(address(proxy))), uint256(IProxyAdmin.ProxyType.ERC1967));
        assertEq(uint256(admin.proxyType(address(chugsplash))), uint256(IProxyAdmin.ProxyType.CHUGSPLASH));
        assertEq(uint256(admin.proxyType(address(resolved))), uint256(IProxyAdmin.ProxyType.RESOLVED));
    }

    function test_onlyOwner_notOwner_reverts() external {
        vm.expectRevert("Ownable: caller is not the owner");
        admin.changeProxyAdmin(payable(proxy), address(0));

        vm.expectRevert("Ownable: caller is not the owner");
        admin.upgrade(payable(proxy), address(implementation));

        vm.expectRevert("Ownable: caller is not the owner");
        admin.upgradeAndCall(payable(proxy), address(implementation), hex"");
    }
}
