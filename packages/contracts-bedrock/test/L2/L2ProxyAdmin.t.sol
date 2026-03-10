// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { CommonTest } from "test/setup/CommonTest.sol";
import { MockHelper } from "test/utils/MockHelper.sol";
import {
    ProxyAdmin_SetProxyType_Test,
    ProxyAdmin_SetImplementationName_Test,
    ProxyAdmin_SetAddressManager_Test,
    ProxyAdmin_IsUpgrading_Test,
    ProxyAdmin_GetProxyImplementation_Test,
    ProxyAdmin_GetProxyAdmin_Test,
    ProxyAdmin_ChangeProxyAdmin_Test,
    ProxyAdmin_Upgrade_Test,
    ProxyAdmin_UpgradeAndCall_Test,
    ProxyAdmin_Uncategorized_Test
} from "test/universal/ProxyAdmin.t.sol";

// Libraries
import { Constants } from "src/libraries/Constants.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";

// Interfaces
import { IL2ProxyAdmin } from "interfaces/L2/IL2ProxyAdmin.sol";
import { IProxyAdmin } from "interfaces/universal/IProxyAdmin.sol";

// Contracts
import { L2ProxyAdmin } from "src/L2/L2ProxyAdmin.sol";
import { IL2ContractsManager } from "interfaces/L2/IL2ContractsManager.sol";

/// @title L2ProxyAdmin_TestInit
/// @notice Reusable test initialization for `L2ProxyAdmin` tests.
abstract contract L2ProxyAdmin_TestInit is CommonTest, MockHelper {
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
}

/// @title L2ProxyAdmin_Constructor_Test
/// @notice Tests the `constructor` function of the `L2ProxyAdmin` contract.
contract L2ProxyAdmin_Constructor_Test is L2ProxyAdmin_TestInit {
    /// @notice Tests that the `constructor` function succeeds.
    function test_constructor_succeeds() public {
        // Deploy the L2ProxyAdmin contract
        l2ProxyAdmin = IL2ProxyAdmin(address(new L2ProxyAdmin()));
        // It sets the owner to address(0)
        assertEq(l2ProxyAdmin.owner(), address(0));
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

// Backwards-compatibility: rerun all ProxyAdmin tests against L2ProxyAdmin
// by overriding _createAdmin to deploy L2ProxyAdmin instead.

/// @title L2ProxyAdmin_SetProxyType_Test
/// @notice Tests the `setProxyType` function of the `L2ProxyAdmin` contract for backwards compatibility.
contract L2ProxyAdmin_SetProxyType_Test is ProxyAdmin_SetProxyType_Test {
    function _createAdmin(address _owner) internal override returns (IProxyAdmin) {
        IProxyAdmin admin = IProxyAdmin(address(new L2ProxyAdmin()));
        // Manually set the owner in the storage slot 0.
        vm.store(address(admin), bytes32(0), bytes32(uint256(uint160(_owner))));
        return admin;
    }
}

/// @title L2ProxyAdmin_SetImplementationName_Test
/// @notice Tests the `setImplementationName` function of the `L2ProxyAdmin` contract for backwards compatibility.
contract L2ProxyAdmin_SetImplementationName_Test is ProxyAdmin_SetImplementationName_Test {
    function _createAdmin(address _owner) internal override returns (IProxyAdmin) {
        IProxyAdmin admin = IProxyAdmin(address(new L2ProxyAdmin()));
        // Manually set the owner in the storage slot 0.
        vm.store(address(admin), bytes32(0), bytes32(uint256(uint160(_owner))));
        return admin;
    }
}

/// @title L2ProxyAdmin_SetAddressManager_Test
/// @notice Tests the `setAddressManager` function of the `L2ProxyAdmin` contract for backwards compatibility.
contract L2ProxyAdmin_SetAddressManager_Test is ProxyAdmin_SetAddressManager_Test {
    function _createAdmin(address _owner) internal override returns (IProxyAdmin) {
        IProxyAdmin admin = IProxyAdmin(address(new L2ProxyAdmin()));
        // Manually set the owner in the storage slot 0.
        vm.store(address(admin), bytes32(0), bytes32(uint256(uint160(_owner))));
        return admin;
    }
}

/// @title L2ProxyAdmin_IsUpgrading_Test
/// @notice Tests the `isUpgrading` function of the `L2ProxyAdmin` contract for backwards compatibility.
contract L2ProxyAdmin_IsUpgrading_Test is ProxyAdmin_IsUpgrading_Test {
    function _createAdmin(address _owner) internal override returns (IProxyAdmin) {
        IProxyAdmin admin = IProxyAdmin(address(new L2ProxyAdmin()));
        // Manually set the owner in the storage slot 0.
        vm.store(address(admin), bytes32(0), bytes32(uint256(uint160(_owner))));
        return admin;
    }
}

/// @title L2ProxyAdmin_GetProxyImplementation_Test
/// @notice Tests the `getProxyImplementation` function of the `L2ProxyAdmin` contract for backwards compatibility.
contract L2ProxyAdmin_GetProxyImplementation_Test is ProxyAdmin_GetProxyImplementation_Test {
    function _createAdmin(address _owner) internal override returns (IProxyAdmin) {
        IProxyAdmin admin = IProxyAdmin(address(new L2ProxyAdmin()));
        // Manually set the owner in the storage slot 0.
        vm.store(address(admin), bytes32(0), bytes32(uint256(uint160(_owner))));
        return admin;
    }
}

/// @title L2ProxyAdmin_GetProxyAdmin_Test
/// @notice Tests the `getProxyAdmin` function of the `L2ProxyAdmin` contract for backwards compatibility.
contract L2ProxyAdmin_GetProxyAdmin_Test is ProxyAdmin_GetProxyAdmin_Test {
    function _createAdmin(address _owner) internal override returns (IProxyAdmin) {
        IProxyAdmin admin = IProxyAdmin(address(new L2ProxyAdmin()));
        // Manually set the owner in the storage slot 0.
        vm.store(address(admin), bytes32(0), bytes32(uint256(uint160(_owner))));
        return admin;
    }
}

/// @title L2ProxyAdmin_ChangeProxyAdmin_Test
/// @notice Tests the `changeProxyAdmin` function of the `L2ProxyAdmin` contract for backwards compatibility.
contract L2ProxyAdmin_ChangeProxyAdmin_Test is ProxyAdmin_ChangeProxyAdmin_Test {
    function _createAdmin(address _owner) internal override returns (IProxyAdmin) {
        IProxyAdmin admin = IProxyAdmin(address(new L2ProxyAdmin()));
        // Manually set the owner in the storage slot 0.
        vm.store(address(admin), bytes32(0), bytes32(uint256(uint160(_owner))));
        return admin;
    }
}

/// @title L2ProxyAdmin_Upgrade_Test
/// @notice Tests the `upgrade` function of the `L2ProxyAdmin` contract for backwards compatibility.
contract L2ProxyAdmin_Upgrade_Test is ProxyAdmin_Upgrade_Test {
    function _createAdmin(address _owner) internal override returns (IProxyAdmin) {
        IProxyAdmin admin = IProxyAdmin(address(new L2ProxyAdmin()));
        // Manually set the owner in the storage slot 0.
        vm.store(address(admin), bytes32(0), bytes32(uint256(uint160(_owner))));
        return admin;
    }
}

/// @title L2ProxyAdmin_UpgradeAndCall_Test
/// @notice Tests the `upgradeAndCall` function of the `L2ProxyAdmin` contract for backwards compatibility.
contract L2ProxyAdmin_UpgradeAndCall_Test is ProxyAdmin_UpgradeAndCall_Test {
    function _createAdmin(address _owner) internal override returns (IProxyAdmin) {
        IProxyAdmin admin = IProxyAdmin(address(new L2ProxyAdmin()));
        // Manually set the owner in the storage slot 0.
        vm.store(address(admin), bytes32(0), bytes32(uint256(uint160(_owner))));
        return admin;
    }
}

/// @title L2ProxyAdmin_Uncategorized_Test
/// @notice General backwards-compatibility tests for the `L2ProxyAdmin` contract.
contract L2ProxyAdmin_Uncategorized_Test is ProxyAdmin_Uncategorized_Test {
    function _createAdmin(address _owner) internal override returns (IProxyAdmin) {
        IProxyAdmin admin = IProxyAdmin(address(new L2ProxyAdmin()));
        // Manually set the owner in the storage slot 0.
        vm.store(address(admin), bytes32(0), bytes32(uint256(uint160(_owner))));
        return admin;
    }
}
