// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { Test } from "forge-std/Test.sol";
import { Constants } from "src/libraries/Constants.sol";

// Contracts
import { ProxyAdminOwnedBase } from "src/L1/ProxyAdminOwnedBase.sol";
import { ProxyAdmin } from "src/universal/ProxyAdmin.sol";

// Interfaces
import { IOwnable } from "interfaces/universal/IOwnable.sol";

/// @notice Contract implementing the abstract `ProxyAdminOwnedBase` contract so unit tests can be written
contract ProxyAdminOwned is ProxyAdminOwnedBase {
    /// @notice For testing purposes, we expose the `_sameProxyAdminOwner` function
    function forTest_sameProxyAdminOwner(address _proxy) public view returns (bool) {
        return _sameProxyAdminOwner(_proxy);
    }
}

contract ProxyAdminOwnedBaseTest is Test {
    ProxyAdminOwned public proxyAdminOwned;
    ProxyAdmin public proxyAdmin;

    address public owner = makeAddr("owner");

    function setUp() public {
        proxyAdmin = new ProxyAdmin(owner);
        proxyAdminOwned = new ProxyAdminOwned();

        vm.store(
            address(proxyAdminOwned),
            bytes32(Constants.PROXY_OWNER_ADDRESS),
            bytes32(uint256(uint160(address(proxyAdmin))))
        );
    }

    function _mockAndExpect(address _target, bytes memory _call, bytes memory _return) internal {
        vm.mockCall(_target, _call, _return);
        vm.expectCall(_target, _call);
    }

    // Test that the `proxyAdminOwner` function returns the correct owner
    function test_proxyAdminOwner_succeeds() public {
        vm.expectCall(address(proxyAdmin), abi.encodeCall(IOwnable.owner, ()));
        assertEq(proxyAdminOwned.proxyAdminOwner(), owner);
    }

    // Test that the `_sameProxyAdminOwner` function returns true if the proxy admin owner owner
    // of both proxies is the same
    function test_sameProxyAdminOwner_sameOwner_succeeds(address _proxy) public {
        assumeNotForgeAddress(_proxy);
        _mockAndExpect(_proxy, abi.encodeCall(ProxyAdminOwnedBase.proxyAdminOwner, ()), abi.encode(owner));
        assertEq(proxyAdminOwned.forTest_sameProxyAdminOwner(_proxy), true);
    }

    // Test that the `_sameProxyAdminOwner` function returns false if the proxy admin owner of both
    // proxies is different
    function test_sameProxyAdminOwner_differentOwner_fails(address _proxy, address _otherProxyOwner) public {
        assumeNotForgeAddress(_proxy);
        assumeNotForgeAddress(_otherProxyOwner);
        vm.assume(_otherProxyOwner != owner);
        _mockAndExpect(_proxy, abi.encodeCall(ProxyAdminOwnedBase.proxyAdminOwner, ()), abi.encode(_otherProxyOwner));
        assertEq(proxyAdminOwned.forTest_sameProxyAdminOwner(_proxy), false);
    }
}
