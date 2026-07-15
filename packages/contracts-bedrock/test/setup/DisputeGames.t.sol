// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { CommonTest } from "test/setup/CommonTest.sol";
import { DisputeGames } from "test/setup/DisputeGames.sol";

// Libraries
import { GameTypes } from "src/dispute/lib/Types.sol";

// Interfaces
import { IDisputeGame } from "interfaces/dispute/IDisputeGame.sol";

/// @title DisputeGames_PermissionlessGameInitBondForUpgrade_Test
/// @notice Tests for the `DisputeGames.permissionlessGameInitBondForUpgrade` helper.
contract DisputeGames_PermissionlessGameInitBondForUpgrade_Test is CommonTest {
    /// @notice Non-default bond used in assertions.
    uint256 internal constant STALE_INIT_BOND = 1 ether;

    /// @notice Sets the CANNON_KONA implementation and bond.
    function _setFactoryState(address _impl, uint256 _initBond) internal {
        vm.startPrank(disputeGameFactory.owner());
        disputeGameFactory.setImplementation(GameTypes.CANNON_KONA, IDisputeGame(_impl));
        disputeGameFactory.setInitBond(GameTypes.CANNON_KONA, _initBond);
        vm.stopPrank();
    }

    /// @notice Uses the default when a stale bond has no implementation.
    function test_permissionlessGameInitBondForUpgrade_staleBondWithoutImpl_succeeds() public {
        _setFactoryState(address(0), STALE_INIT_BOND);
        uint256 bond = DisputeGames.permissionlessGameInitBondForUpgrade(
            disputeGameFactory, GameTypes.CANNON_KONA, DEFAULT_DISPUTE_GAME_INIT_BOND
        );
        assertEq(bond, DEFAULT_DISPUTE_GAME_INIT_BOND, "stale bond must not override the default");
    }

    /// @notice Uses the default when a registered game's bond is zero.
    function test_permissionlessGameInitBondForUpgrade_zeroBondWithImpl_succeeds() public {
        _setFactoryState(address(0xdead), 0);
        uint256 bond = DisputeGames.permissionlessGameInitBondForUpgrade(
            disputeGameFactory, GameTypes.CANNON_KONA, DEFAULT_DISPUTE_GAME_INIT_BOND
        );
        assertEq(bond, DEFAULT_DISPUTE_GAME_INIT_BOND, "zero bond must fall back to the default");
    }

    /// @notice Uses the live bond when a registered game's bond is nonzero.
    function test_permissionlessGameInitBondForUpgrade_liveBondWithImpl_succeeds() public {
        _setFactoryState(address(0xdead), STALE_INIT_BOND);
        uint256 bond = DisputeGames.permissionlessGameInitBondForUpgrade(
            disputeGameFactory, GameTypes.CANNON_KONA, DEFAULT_DISPUTE_GAME_INIT_BOND
        );
        assertEq(bond, STALE_INIT_BOND, "live bond must be used when the game is registered");
    }
}
