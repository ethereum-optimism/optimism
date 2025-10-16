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
    /// @notice Slot 0, used to test ResolvedDelegateProxy behavior.
    mapping(address => string) public slot0;

    /// @notice Slot 1, used to test ResolvedDelegateProxy behavior.
    mapping(address => address) public slot1;

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
    function assertOnlyProxyAdminOrProxyAdminOwner() public view {
        _assertOnlyProxyAdminOrProxyAdminOwner();
    }

    /// @notice Set the value of slot 0 for the provided address.
    function setSlot0(address _address, string memory _value) public {
        slot0[_address] = _value;
    }

    /// @notice Set the value of slot 1 for the provided address.
    function setSlot1(address _address, address _value) public {
        slot1[_address] = _value;
    }
}

abstract contract ProxyAdminOwnedBase_TestInit is CommonTest {
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

    /// @notice Tests that the proxyAdmin function returns the correct proxy when the current
    ///         contract is a full ResolvedDelegateProxy.
    function test_proxyAdmin_fullResolvedDelegateProxy_succeeds() public {
        // Unset the standard proxy owner slot.
        vm.store(address(harness), bytes32(Constants.PROXY_OWNER_ADDRESS), bytes32(0));

        // Store the string "OVM_L1CrossDomainMessenger" in slot 0.
        harness.setSlot0(address(harness), "OVM_L1CrossDomainMessenger");

        // Store the address of the proxyAdmin in slot 1.
        harness.setSlot1(address(harness), address(addressManager));

        // Expect no revert.
        assertEq(address(harness.proxyAdmin()), address(proxyAdmin));
    }

    /// @notice Tests that the proxyAdmin function reverts if the current contract is not a
    ///         ResolvedDelegateProxy.
    /// @param _slot0Value The value to store in slot 0.
    function test_proxyAdmin_notResolvedDelegateProxy_reverts(string memory _slot0Value) public {
        // Assume the slot 0 value is not "OVM_L1CrossDomainMessenger".
        vm.assume(keccak256(abi.encode(_slot0Value)) != keccak256(abi.encode("OVM_L1CrossDomainMessenger")));

        // Unset the standard proxy owner slot.
        vm.store(address(harness), bytes32(Constants.PROXY_OWNER_ADDRESS), bytes32(0));

        // Store the slot 0 value.
        harness.setSlot0(address(harness), _slot0Value);

        // Expect a revert.
        vm.expectRevert(ProxyAdminOwnedBase.ProxyAdminOwnedBase_NotResolvedDelegateProxy.selector);
        harness.proxyAdmin();
    }

    /// @notice Tests that the proxyAdmin function reverts if the proxy admin is not found.
    function test_proxyAdmin_proxyAdminNotFound_reverts() public {
        // Unset the standard proxy owner slot.
        vm.store(address(harness), bytes32(Constants.PROXY_OWNER_ADDRESS), bytes32(0));

        // Store the string "OVM_L1CrossDomainMessenger" in slot 0.
        harness.setSlot0(address(harness), "OVM_L1CrossDomainMessenger");

        // Store address(0) in slot 1.
        harness.setSlot1(address(harness), address(0));

        // Expect a revert.
        vm.expectRevert(ProxyAdminOwnedBase.ProxyAdminOwnedBase_ProxyAdminNotFound.selector);
        harness.proxyAdmin();
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

contract ProxyAdminOwnedBase_assertOnlyProxyAdminOrProxyAdminOwner_Test is ProxyAdminOwnedBase_TestInit {
    /// @notice Tests that the assertOnlyProxyAdminOrProxyAdminOwner function does not revert if
    ///         the caller is the ProxyAdmin or the ProxyAdmin owner.
    function test_assertOnlyProxyAdminOrProxyAdminOwner_proxyAdmin_succeeds() public {
        // Prank as the ProxyAdmin.
        vm.prank(address(proxyAdmin));

        // Expect no revert.
        harness.assertOnlyProxyAdminOrProxyAdminOwner();
    }

    /// @notice Tests that the assertOnlyProxyAdminOrProxyAdminOwner function does not revert if
    ///         the caller is the ProxyAdmin owner.
    function test_assertOnlyProxyAdminOrProxyAdminOwner_proxyAdminOwner_succeeds() public {
        // Prank as the ProxyAdmin owner.
        vm.prank(proxyAdminOwner);

        // Expect no revert.
        harness.assertOnlyProxyAdminOrProxyAdminOwner();
    }

    /// @notice Tests that the assertOnlyProxyAdminOrProxyAdminOwner function reverts if the caller
    ///         is not the ProxyAdmin or the ProxyAdmin owner.
    /// @param _sender The address of the sender to test.
    function test_assertOnlyProxyAdminOrProxyAdminOwner_notProxyAdminOrProxyAdminOwner_reverts(address _sender)
        public
    {
        // Prank as the not ProxyAdmin or ProxyAdmin owner.
        vm.assume(_sender != address(proxyAdmin) && _sender != proxyAdminOwner);
        vm.prank(_sender);

        // Expect a revert.
        vm.expectRevert(ProxyAdminOwnedBase.ProxyAdminOwnedBase_NotProxyAdminOrProxyAdminOwner.selector);
        harness.assertOnlyProxyAdminOrProxyAdminOwner();
    }
}
