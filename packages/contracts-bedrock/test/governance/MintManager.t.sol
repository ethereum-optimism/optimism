// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { CommonTest } from "test/setup/CommonTest.sol";

// Interfaces
import { IGovernanceToken } from "interfaces/governance/IGovernanceToken.sol";
import { IMintManager } from "interfaces/governance/IMintManager.sol";
import { DeployUtils } from "scripts/libraries/DeployUtils.sol";

/// @title MintManager_TestInit
/// @notice Reusable test initialization for `MintManager` tests.
abstract contract MintManager_TestInit is CommonTest {
    address constant owner = address(0x1234);
    address constant rando = address(0x5678);
    IGovernanceToken internal gov;
    IMintManager internal manager;

    /// @notice Sets up the test suite.
    function setUp() public virtual override {
        super.setUp();

        vm.prank(owner);
        gov = IGovernanceToken(
            DeployUtils.create1({
                _name: "GovernanceToken",
                _args: DeployUtils.encodeConstructor(abi.encodeCall(IGovernanceToken.__constructor__, ()))
            })
        );

        vm.prank(owner);
        manager = IMintManager(
            DeployUtils.create1({
                _name: "MintManager",
                _args: DeployUtils.encodeConstructor(abi.encodeCall(IMintManager.__constructor__, (owner, address(gov))))
            })
        );

        vm.prank(owner);
        gov.transferOwnership(address(manager));
    }
}

/// @title MintManager_Constructor_Test
/// @notice Tests the constructor of the `MintManager` contract.
contract MintManager_Constructor_Test is MintManager_TestInit {
    /// @notice Tests that the constructor properly configures the contract.
    function test_constructor_succeeds() external view {
        assertEq(manager.owner(), owner);
        assertEq(address(manager.governanceToken()), address(gov));
    }
}

/// @title MintManager_Mint_Test
/// @notice Tests the `mint` function of the `MintManager` contract.
contract MintManager_Mint_Test is MintManager_TestInit {
    /// @notice Tests that the first mint can be any amount since no cap applies.
    function testFuzz_mint_firstMint_succeeds(uint256 _amount) external {
        _amount = bound(_amount, 0, type(uint192).max);

        vm.prank(owner);
        manager.mint(owner, _amount);

        assertEq(gov.balanceOf(owner), _amount);
    }

    /// @notice Tests that the mint function reverts when called by a non-owner.
    function test_mint_fromNotOwner_reverts() external {
        // Mint from rando fails.
        vm.prank(rando);
        vm.expectRevert("Ownable: caller is not the owner");
        manager.mint(owner, 100);
    }

    /// @notice Tests that subsequent mints succeed when within cap after period elapsed.
    function testFuzz_mint_afterPeriodElapsed_succeeds(uint256 _initialAmount, uint256 _secondAmount) external {
        _initialAmount = bound(_initialAmount, 1, type(uint192).max);

        vm.prank(owner);
        manager.mint(owner, _initialAmount);

        assertEq(gov.balanceOf(owner), _initialAmount);

        uint256 maxMint = (_initialAmount * manager.MINT_CAP()) / manager.DENOMINATOR();
        _secondAmount = bound(_secondAmount, 0, maxMint);

        vm.warp(block.timestamp + manager.MINT_PERIOD() + 1);
        vm.prank(owner);
        manager.mint(owner, _secondAmount);

        assertEq(gov.balanceOf(owner), _initialAmount + _secondAmount);
    }

    /// @notice Tests that the mint function always reverts when called before the mint period has
    ///         elapsed, even if the caller is the owner.
    function test_mint_beforePeriodElapsed_reverts() external {
        // Mint once.
        vm.prank(owner);
        manager.mint(owner, 100);

        // Token balance increases.
        assertEq(gov.balanceOf(owner), 100);

        // Mint again.
        vm.prank(owner);
        vm.expectRevert("MintManager: minting not permitted yet");
        manager.mint(owner, 100);

        // Token balance does not increase.
        assertEq(gov.balanceOf(owner), 100);
    }

    /// @notice Tests that the owner cannot mint more than the mint cap.
    function test_mint_moreThanCap_reverts() external {
        // Mint once.
        vm.prank(owner);
        manager.mint(owner, 100);

        // Token balance increases.
        assertEq(gov.balanceOf(owner), 100);

        // Mint again (greater than 2% max).
        vm.warp(block.timestamp + manager.MINT_PERIOD() + 1);
        vm.prank(owner);
        vm.expectRevert("MintManager: mint amount exceeds cap");
        manager.mint(owner, 3);

        // Token balance does not increase.
        assertEq(gov.balanceOf(owner), 100);
    }
}

/// @title MintManager_Upgrade_Test
/// @notice Tests the `upgrade` function of the `MintManager` contract.
contract MintManager_Upgrade_Test is MintManager_TestInit {
    /// @notice Tests that the owner can upgrade to any non-zero address.
    function testFuzz_upgrade_fromOwner_succeeds(address _newManager) external {
        vm.assume(_newManager != address(0));

        vm.prank(owner);
        manager.upgrade(_newManager);

        assertEq(gov.owner(), _newManager);
    }

    /// @notice Tests that the upgrade function reverts when called by a non-owner.
    function test_upgrade_fromNotOwner_reverts() external {
        // Upgrade from rando fails.
        vm.prank(rando);
        vm.expectRevert("Ownable: caller is not the owner");
        manager.upgrade(rando);
    }

    /// @notice Tests that the upgrade function reverts when attempting to update to the zero
    ///         address, even if the caller is the owner.
    function test_upgrade_toZeroAddress_reverts() external {
        // Upgrade to zero address fails.
        vm.prank(owner);
        vm.expectRevert("MintManager: mint manager cannot be the zero address");
        manager.upgrade(address(0));
    }
}
