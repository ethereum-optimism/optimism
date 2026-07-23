// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { CommonTest } from "test/setup/CommonTest.sol";
import { DisputeGames } from "test/setup/DisputeGames.sol";

// Libraries
import { GameType, GameTypes } from "src/dispute/lib/Types.sol";

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

contract DisputeGames_GameType_Test is CommonTest {
    function test_permissionedGameType_legacyEnabled_succeeds() public {
        _disableGameTypes(GameTypes.PERMISSIONED_CANNON, GameTypes.SUPER_PERMISSIONED);
        _enableGameType(GameTypes.PERMISSIONED_CANNON);

        assertEq(
            GameType.unwrap(DisputeGames.permissionedGameType(disputeGameFactory)),
            GameType.unwrap(GameTypes.PERMISSIONED_CANNON)
        );
    }

    function test_permissionedGameType_superEnabled_succeeds() public {
        _disableGameTypes(GameTypes.PERMISSIONED_CANNON, GameTypes.SUPER_PERMISSIONED);
        _enableGameType(GameTypes.PERMISSIONED_CANNON);
        _enableGameType(GameTypes.SUPER_PERMISSIONED);

        assertEq(
            GameType.unwrap(DisputeGames.permissionedGameType(disputeGameFactory)),
            GameType.unwrap(GameTypes.SUPER_PERMISSIONED)
        );
    }

    function test_permissionedGameType_superDisabledWithStaleArgs_succeeds() public {
        _disableGameTypes(GameTypes.PERMISSIONED_CANNON, GameTypes.SUPER_PERMISSIONED);
        _enableGameType(GameTypes.PERMISSIONED_CANNON);
        _enableGameType(GameTypes.SUPER_PERMISSIONED);

        vm.prank(disputeGameFactory.owner());
        disputeGameFactory.setImplementation(GameTypes.SUPER_PERMISSIONED, IDisputeGame(address(0)));

        assertEq(
            GameType.unwrap(DisputeGames.permissionedGameType(disputeGameFactory)),
            GameType.unwrap(GameTypes.PERMISSIONED_CANNON)
        );
    }

    function test_permissionlessGameType_legacyEnabled_succeeds() public {
        _disableGameTypes(GameTypes.CANNON_KONA, GameTypes.SUPER_CANNON_KONA);
        _enableGameType(GameTypes.CANNON_KONA);

        assertEq(
            GameType.unwrap(DisputeGames.permissionlessGameType(disputeGameFactory)),
            GameType.unwrap(GameTypes.CANNON_KONA)
        );
    }

    function test_permissionlessGameType_superEnabled_succeeds() public {
        _disableGameTypes(GameTypes.CANNON_KONA, GameTypes.SUPER_CANNON_KONA);
        _enableGameType(GameTypes.CANNON_KONA);
        _enableGameType(GameTypes.SUPER_CANNON_KONA);

        assertEq(
            GameType.unwrap(DisputeGames.permissionlessGameType(disputeGameFactory)),
            GameType.unwrap(GameTypes.SUPER_CANNON_KONA)
        );
    }

    function test_permissionlessGameType_superDisabledWithStaleArgs_succeeds() public {
        _disableGameTypes(GameTypes.CANNON_KONA, GameTypes.SUPER_CANNON_KONA);
        _enableGameType(GameTypes.CANNON_KONA);
        _enableGameType(GameTypes.SUPER_CANNON_KONA);

        vm.prank(disputeGameFactory.owner());
        disputeGameFactory.setImplementation(GameTypes.SUPER_CANNON_KONA, IDisputeGame(address(0)));

        assertEq(
            GameType.unwrap(DisputeGames.permissionlessGameType(disputeGameFactory)),
            GameType.unwrap(GameTypes.CANNON_KONA)
        );
    }

    function _disableGameTypes(GameType _legacyGameType, GameType _superGameType) internal {
        address owner = disputeGameFactory.owner();
        vm.startPrank(owner);
        disputeGameFactory.setImplementation(_legacyGameType, IDisputeGame(address(0)), hex"");
        disputeGameFactory.setImplementation(_superGameType, IDisputeGame(address(0)), hex"");
        vm.stopPrank();
    }

    function _enableGameType(GameType _gameType) internal {
        vm.prank(disputeGameFactory.owner());
        disputeGameFactory.setImplementation(_gameType, IDisputeGame(address(1)), hex"01");
    }
}
