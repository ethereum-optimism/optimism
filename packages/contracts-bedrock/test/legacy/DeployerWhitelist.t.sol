// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing utilities
import { Test } from "forge-std/Test.sol";

// Target contract
import { IDeployerWhitelist } from "interfaces/legacy/IDeployerWhitelist.sol";
import { DeployUtils } from "scripts/libraries/DeployUtils.sol";

/// @title DeployerWhitelist_TestInit
/// @notice Reusable test initialization for `DeployerWhitelist` tests.
abstract contract DeployerWhitelist_TestInit is Test {
    IDeployerWhitelist list;
    address owner = address(12345);

    event OwnerChanged(address oldOwner, address newOwner);
    event WhitelistDisabled(address oldOwner);
    event WhitelistStatusChanged(address deployer, bool whitelisted);

    /// @notice Sets up the test suite.
    function setUp() public {
        list = IDeployerWhitelist(
            DeployUtils.create1({
                _name: "DeployerWhitelist",
                _args: DeployUtils.encodeConstructor(abi.encodeCall(IDeployerWhitelist.__constructor__, ()))
            })
        );
    }
}

/// @title DeployerWhitelist_Owner_Test
/// @notice Tests the `owner` function of the `DeployerWhitelist` contract.
contract DeployerWhitelist_Owner_Test is DeployerWhitelist_TestInit {
    /// @notice Tests that `owner` is initialized to the zero address.
    function test_owner_succeeds() external view {
        assertEq(list.owner(), address(0));
    }
}

/// @title DeployerWhitelist_SetWhitelistedDeployer_Test
/// @notice Tests the `setWhitelistedDeployer` function of the `DeployerWhitelist` contract.
contract DeployerWhitelist_SetWhitelistedDeployer_Test is DeployerWhitelist_TestInit {
    /// @notice Tests that `setWhitelistedDeployer` correctly sets the whitelist status of a
    ///         deployer.
    function test_setWhitelistedDeployer_succeeds(address _deployer, bool _isWhitelisted) external {
        vm.store(address(list), bytes32(uint256(0)), bytes32(uint256(uint160(owner))));
        assertEq(list.owner(), owner);

        vm.prank(owner);
        vm.expectEmit(true, true, true, true);
        emit WhitelistStatusChanged(_deployer, _isWhitelisted);
        list.setWhitelistedDeployer(_deployer, _isWhitelisted);

        assertEq(list.whitelist(_deployer), _isWhitelisted);

        // _deployer is whitelisted or not (and arbitrary contract deployment is not enabled)
        assertNotEq(list.owner(), address(0));
        assertEq(list.isDeployerAllowed(_deployer), _isWhitelisted);
    }

    /// @notice Tests that `setWhitelistedDeployer` reverts when the caller is not the owner.
    function test_setWhitelistedDeployer_callerNotOwner_reverts(
        address _caller,
        address _deployer,
        bool _isWhitelisted
    )
        external
    {
        vm.store(address(list), bytes32(uint256(0)), bytes32(uint256(uint160(owner))));
        assertEq(list.owner(), owner);

        vm.assume(_caller != owner);

        vm.prank(_caller);
        vm.expectRevert(bytes("DeployerWhitelist: function can only be called by the owner of this contract"));
        list.setWhitelistedDeployer(_deployer, _isWhitelisted);
    }
}

/// @title DeployerWhitelist_SetOwner_Test
/// @notice Tests the `setOwner` function of the `DeployerWhitelist` contract.
contract DeployerWhitelist_SetOwner_Test is DeployerWhitelist_TestInit {
    /// @notice Tests that `setOwner` correctly sets the contract owner.
    function test_setOwner_succeeds(address _owner) external {
        vm.store(address(list), bytes32(uint256(0)), bytes32(uint256(uint160(owner))));
        assertEq(list.owner(), owner);
        _owner = address(uint160(bound(uint160(_owner), 1, type(uint160).max)));

        vm.prank(owner);
        vm.expectEmit(true, true, true, true);
        emit OwnerChanged(owner, _owner);
        list.setOwner(_owner);

        assertEq(list.owner(), _owner);
    }

    /// @notice Tests that `setOwner` reverts when the caller is not the owner.
    function test_setOwner_callerNotOwner_reverts(address _caller, address _owner) external {
        vm.store(address(list), bytes32(uint256(0)), bytes32(uint256(uint160(owner))));
        assertEq(list.owner(), owner);

        vm.assume(_caller != owner);

        vm.prank(_caller);
        vm.expectRevert(bytes("DeployerWhitelist: function can only be called by the owner of this contract"));
        list.setOwner(_owner);
    }

    /// @notice Tests that `setOwner` reverts when the new owner is the zero address.
    function test_setOwner_zeroAddress_reverts() external {
        vm.store(address(list), bytes32(uint256(0)), bytes32(uint256(uint160(owner))));
        assertEq(list.owner(), owner);

        vm.prank(owner);
        vm.expectRevert(bytes("DeployerWhitelist: can only be disabled via enableArbitraryContractDeployment"));
        list.setOwner(address(0));
    }
}

/// @title DeployerWhitelist_EnableArbitraryContractDeployment_Test
/// @notice Tests the `enableArbitraryContractDeployment` function of the `DeployerWhitelist`
///         contract.
contract DeployerWhitelist_EnableArbitraryContractDeployment_Test is DeployerWhitelist_TestInit {
    /// @notice Tests that `enableArbitraryContractDeployment` correctly disables the whitelist.
    function test_enableArbitraryContractDeployment_succeeds() external {
        vm.store(address(list), bytes32(uint256(0)), bytes32(uint256(uint160(owner))));
        assertEq(list.owner(), owner);

        vm.prank(owner);
        vm.expectEmit(true, true, true, true);
        emit WhitelistDisabled(owner);
        list.enableArbitraryContractDeployment();

        assertEq(list.owner(), address(0));

        // Any address is allowed to deploy contracts even if they are not whitelisted
        assertEq(list.whitelist(address(1)), false);
        assertEq(list.isDeployerAllowed(address(1)), true);
    }

    /// @notice Tests that `enableArbitraryContractDeployment` reverts when the caller is not the
    ///         owner.
    function test_enableArbitraryContractDeployment_callerNotOwner_reverts(address _caller) external {
        vm.store(address(list), bytes32(uint256(0)), bytes32(uint256(uint160(owner))));
        assertEq(list.owner(), owner);

        vm.assume(_caller != owner);

        vm.prank(_caller);
        vm.expectRevert(bytes("DeployerWhitelist: function can only be called by the owner of this contract"));
        list.enableArbitraryContractDeployment();
    }
}
