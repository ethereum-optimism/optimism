// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { CommonTest } from "test/setup/CommonTest.sol";

// Libraries
import { Constants } from "src/libraries/Constants.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";

// Interfaces
import { IL2ProxyAdmin } from "interfaces/L2/IL2ProxyAdmin.sol";

// Contracts
import { L2ProxyAdmin } from "src/L2/L2ProxyAdmin.sol";
import { IL2ContractsManager } from "interfaces/L2/IL2ContractsManager.sol";

/// @title L2ProxyAdmin_TestInit
/// @notice Reusable test initialization for `L2ProxyAdmin` tests.
abstract contract L2ProxyAdmin_TestInit is CommonTest {
    IL2ProxyAdmin public l2ProxyAdmin;
    address public owner;

    // Events
    event PredeploysUpgraded(address indexed l2ContractsManager);

    /// @notice Test setup.
    function setUp() public virtual override {
        super.setUp();
        l2ProxyAdmin = IL2ProxyAdmin(Predeploys.PROXY_ADMIN);
        owner = l2ProxyAdmin.owner();
    }

    /// @notice Helper function to setup a mock and expect a call to it.
    function _mockAndExpect(address _receiver, bytes memory _calldata, bytes memory _returned) internal {
        vm.mockCall(_receiver, _calldata, _returned);
        vm.expectCall(_receiver, _calldata);
    }
}

/// @title L2ProxyAdmin_Constructor_Test
/// @notice Tests the `constructor` function of the `L2ProxyAdmin` contract.
contract L2ProxyAdmin_Constructor_Test is L2ProxyAdmin_TestInit {
    /// @notice Tests that the `constructor` function succeeds.
    function test_constructor_succeeds(address _owner) public {
        // Deploy the L2ProxyAdmin contract
        l2ProxyAdmin = IL2ProxyAdmin(address(new L2ProxyAdmin(_owner)));
        // It sets the owner to the correct address
        assertEq(l2ProxyAdmin.owner(), _owner);
    }
}

/// @title L2ProxyAdmin_Version_Test
/// @notice Tests the `version` function of the `L2ProxyAdmin` contract.
contract L2ProxyAdmin_Version_Test is L2ProxyAdmin_TestInit {
    /// @notice Tests that the `version` function returns a non-empty string.
    function test_version_succeeds() public view {
        assertGt(bytes(l2ProxyAdmin.version()).length, 0, "Version should be non-empty");
    }
}

/// @title L2ProxyAdmin_UpgradePredeploys_Test
/// @notice Tests the `upgradePredeploys` function of the `L2ProxyAdmin` contract.
contract L2ProxyAdmin_UpgradePredeploys_Test is L2ProxyAdmin_TestInit {
    /// @notice Tests that upgradePredeploys reverts when called by unauthorized caller.
    function testFuzz_upgradePredeploys_unauthorizedCaller_reverts(
        address _caller,
        address _l2ContractsManager
    )
        public
    {
        vm.assume(_caller != Constants.DEPOSITOR_ACCOUNT);

        // Expect the revert with L2ProxyAdmin__Unauthorized
        vm.expectRevert(L2ProxyAdmin.L2ProxyAdmin__Unauthorized.selector);

        // Call upgradePredeploys with unauthorized caller
        vm.prank(_caller);
        l2ProxyAdmin.upgradePredeploys(_l2ContractsManager);
    }

    /// @notice Tests that upgradePredeploys succeeds when called by DEPOSITOR_ACCOUNT.
    function testFuzz_upgradePredeploys_succeeds(address _l2ContractsManager) public {
        assumeAddressIsNot(_l2ContractsManager, AddressType.Precompile, AddressType.ForgeAddress);

        // Mock the delegatecall to return success
        _mockAndExpect(_l2ContractsManager, abi.encodeCall(IL2ContractsManager.upgrade, ()), abi.encode());

        // Expect the PredeploysUpgraded event
        vm.expectEmit(address(l2ProxyAdmin));
        emit PredeploysUpgraded(_l2ContractsManager);

        // Call upgradePredeploys with authorized caller
        vm.prank(Constants.DEPOSITOR_ACCOUNT);
        l2ProxyAdmin.upgradePredeploys(_l2ContractsManager);
    }

    /// @notice Tests that upgradePredeploys reverts when delegatecall fails.
    function testFuzz_upgradePredeploys_delegatecallFails_reverts(address _l2ContractsManager) public {
        assumeAddressIsNot(_l2ContractsManager, AddressType.Precompile, AddressType.ForgeAddress);

        // Mock the delegatecall to return failure
        vm.mockCallRevert(_l2ContractsManager, abi.encodeCall(IL2ContractsManager.upgrade, ()), bytes("error"));

        // Expect the revert with L2ProxyAdmin__UpgradeFailed
        vm.expectRevert(abi.encodeWithSelector(L2ProxyAdmin.L2ProxyAdmin__UpgradeFailed.selector, bytes("error")));

        // Call upgradePredeploys with authorized caller
        vm.prank(Constants.DEPOSITOR_ACCOUNT);
        l2ProxyAdmin.upgradePredeploys(_l2ContractsManager);
    }
}
