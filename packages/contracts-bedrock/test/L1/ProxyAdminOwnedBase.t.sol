// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { CommonTest } from "test/setup/CommonTest.sol";
import { Constants } from "src/libraries/Constants.sol";

// Contracts
import { ProxyAdminOwnedBase } from "src/L1/ProxyAdminOwnedBase.sol";

/// @title ProxyAdminOwnedBase_Harness
/// @notice Contract implementing the abstract `ProxyAdminOwnedBase` contract so we can write unit
///         tests for the `ProxyAdminOwnedBase` contract.
contract ProxyAdminOwnedBase_Harness is ProxyAdminOwnedBase {
    /// @notice Assert that the proxy admin owner of the current contract is the same as the proxy
    ///         admin owner of the other Proxy address provided.
    function assertSharedProxyAdminOwner(address _proxy) public view {
        _assertSharedProxyAdminOwner(_proxy);
    }

    /// @notice Assert that the caller is the ProxyAdmin.
    function assertOnlyProxyAdmin() public view {
        _assertOnlyProxyAdmin();
    }

    /// @notice Assert that the caller is the ProxyAdmin owner.
    function assertOnlyProxyAdminOwner() public view {
        _assertOnlyProxyAdminOwner();
    }

    /// @notice Assert that the caller is the ProxyAdmin or the ProxyAdmin owner.
    function assertOnlyProxyAdminOrOwner() public view {
        _assertOnlyProxyAdminOrOwner();
    }
}

contract ProxyAdminOwnedBase_TestInit is CommonTest {
    /// @notice Harness for the `ProxyAdminOwnedBase` contract.
    ProxyAdminOwnedBase_Harness public harness;

    /// @notice Sets up the test.
    function setUp() public override {
        super.setUp();

        // Create a new harness
        harness = new ProxyAdminOwnedBase_Harness();

        // Set the owner of the harness to the ProxyAdmin contract.
        vm.store(
            address(harness), bytes32(Constants.PROXY_OWNER_ADDRESS), bytes32(uint256(uint160(address(proxyAdmin))))
        );
    }
}

contract ProxyAdminOwnedBase_proxyAdminOwner_Test is ProxyAdminOwnedBase_TestInit {
    /// @notice Tests that the proxyAdminOwner function returns the correct owner.
    function test_proxyAdminOwner_succeeds() public view {
        assertEq(harness.proxyAdminOwner(), proxyAdminOwner);
    }
}

contract ProxyAdminOwnedBase_proxyAdmin_Test is ProxyAdminOwnedBase_TestInit {
    /// @notice Tests that the proxyAdmin function returns the correct proxy.
    function test_proxyAdmin_succeeds() public view {
        assertEq(address(harness.proxyAdmin()), address(proxyAdmin));
    }
}

contract ProxyAdminOwnedBase_assertSharedProxyAdminOwner_Test is ProxyAdminOwnedBase_TestInit {
    /// @notice Tests that the assertSharedProxyAdminOwner function does not revert if the provided
    ///         proxy has the same owner as the current contract.
    function test_assertSharedProxyAdminOwner_sameOwner_succeeds(address _proxy) public {
        // Assume the provided proxy is not a forge address.
        assumeNotForgeAddress(_proxy);

        // Mock the proxyAdminOwner function to return the same owner as the current contract.
        vm.mockCall(_proxy, abi.encodeCall(ProxyAdminOwnedBase.proxyAdminOwner, ()), abi.encode(proxyAdminOwner));

        // Expect no revert.
        harness.assertSharedProxyAdminOwner(_proxy);
    }

    /// @notice Tests that the assertSharedProxyAdminOwner function reverts if the proxy admin
    ///         owner of both proxies is different.
    function testFuzz_assertSharedProxyAdminOwner_differentOwner_reverts(
        address _proxy,
        address _otherProxyOwner
    )
        public
    {
        // Assume the provided proxy is not a forge address.
        assumeNotForgeAddress(_proxy);
        assumeNotForgeAddress(_otherProxyOwner);

        // Assume the other proxy owner is not the same as the current owner.
        vm.assume(_otherProxyOwner != proxyAdminOwner);

        // Mock the proxyAdminOwner function to return the other proxy owner.
        vm.mockCall(_proxy, abi.encodeCall(ProxyAdminOwnedBase.proxyAdminOwner, ()), abi.encode(_otherProxyOwner));

        // Expect a revert.
        vm.expectRevert(ProxyAdminOwnedBase.ProxyAdminOwnedBase_NotSharedProxyAdminOwner.selector);
        harness.assertSharedProxyAdminOwner(_proxy);
    }
}

contract ProxyAdminOwnedBase_assertOnlyProxyAdmin_Test is ProxyAdminOwnedBase_TestInit {
    /// @notice Tests that the assertOnlyProxyAdmin function does not revert if the caller is the
    ///         ProxyAdmin.
    function test_assertOnlyProxyAdmin_proxyAdmin_succeeds() public {
        // Prank as the ProxyAdmin.
        vm.prank(address(proxyAdmin));

        // Expect no revert.
        harness.assertOnlyProxyAdmin();
    }

    /// @notice Tests that the assertOnlyProxyAdmin function reverts if the caller is not the
    ///         ProxyAdmin.
    /// @param _sender The address of the sender to test.
    function test_assertOnlyProxyAdmin_notProxyAdmin_reverts(address _sender) public {
        // Prank as the not ProxyAdmin.
        vm.assume(_sender != address(proxyAdmin));
        vm.prank(_sender);

        // Expect a revert.
        vm.expectRevert(ProxyAdminOwnedBase.ProxyAdminOwnedBase_NotProxyAdmin.selector);
        harness.assertOnlyProxyAdmin();
    }
}

contract ProxyAdminOwnedBase_assertOnlyProxyAdminOwner_Test is ProxyAdminOwnedBase_TestInit {
    /// @notice Tests that the assertOnlyProxyAdminOwner function does not revert if the caller is
    ///         the ProxyAdmin owner.
    function test_assertOnlyProxyAdminOwner_proxyAdminOwner_succeeds() public {
        // Prank as the ProxyAdmin owner.
        vm.prank(proxyAdminOwner);

        // Expect no revert.
        harness.assertOnlyProxyAdminOwner();
    }

    /// @notice Tests that the assertOnlyProxyAdminOwner function reverts if the caller is not the
    ///         ProxyAdmin owner.
    /// @param _sender The address of the sender to test.
    function test_assertOnlyProxyAdminOwner_notProxyAdminOwner_reverts(address _sender) public {
        // Prank as the not ProxyAdmin owner.
        vm.assume(_sender != proxyAdminOwner);
        vm.prank(_sender);

        // Expect a revert.
        vm.expectRevert(ProxyAdminOwnedBase.ProxyAdminOwnedBase_NotProxyAdminOwner.selector);
        harness.assertOnlyProxyAdminOwner();
    }
}

contract ProxyAdminOwnedBase_assertOnlyProxyAdminOrOwner_Test is ProxyAdminOwnedBase_TestInit {
    /// @notice Tests that the assertOnlyProxyAdminOrOwner function does not revert if the caller
    ///         is the ProxyAdmin or the ProxyAdmin owner.
    function test_assertOnlyProxyAdminOrOwner_proxyAdmin_succeeds() public {
        // Prank as the ProxyAdmin.
        vm.prank(address(proxyAdmin));

        // Expect no revert.
        harness.assertOnlyProxyAdminOrOwner();
    }

    /// @notice Tests that the assertOnlyProxyAdminOrOwner function does not revert if the caller
    ///         is the ProxyAdmin owner.
    function test_assertOnlyProxyAdminOrOwner_proxyAdminOwner_succeeds() public {
        // Prank as the ProxyAdmin owner.
        vm.prank(proxyAdminOwner);

        // Expect no revert.
        harness.assertOnlyProxyAdminOrOwner();
    }

    /// @notice Tests that the assertOnlyProxyAdminOrOwner function reverts if the caller is not
    ///         the ProxyAdmin or the ProxyAdmin owner.
    /// @param _sender The address of the sender to test.
    function test_assertOnlyProxyAdminOrOwner_notProxyAdminOrOwner_reverts(address _sender) public {
        // Prank as the not ProxyAdmin or ProxyAdmin owner.
        vm.assume(_sender != address(proxyAdmin) && _sender != proxyAdminOwner);
        vm.prank(_sender);

        // Expect a revert.
        vm.expectRevert(ProxyAdminOwnedBase.ProxyAdminOwnedBase_NotProxyAdminOrOwner.selector);
        harness.assertOnlyProxyAdminOrOwner();
    }
}
