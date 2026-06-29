// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

// Testing
import { Test } from "test/setup/Test.sol";

// Libraries
import { GameTypes } from "src/dispute/lib/Types.sol";

/// @title Types_IsSuperGame_Test
/// @notice Unit tests for the GameTypes.isSuperGame() function.
contract Types_IsSuperGame_Test is Test {
    /// @notice Tests that ZK_DISPUTE_GAME is recognized as a super game.
    function test_isSuperGame_zkDisputeGame_succeeds() public pure {
        assertTrue(GameTypes.isSuperGame(GameTypes.ZK_DISPUTE_GAME), "ZK_DISPUTE_GAME must be a super game");
    }

    /// @notice Tests that non-super game types are not recognized as super games.
    function test_isSuperGame_nonSuperTypes_succeeds() public pure {
        assertFalse(GameTypes.isSuperGame(GameTypes.CANNON), "CANNON must not be a super game");
        assertFalse(
            GameTypes.isSuperGame(GameTypes.PERMISSIONED_CANNON), "PERMISSIONED_CANNON must not be a super game"
        );
        assertFalse(GameTypes.isSuperGame(GameTypes.CANNON_KONA), "CANNON_KONA must not be a super game");
    }

    /// @notice Tests that all expected super game types are recognized.
    function test_isSuperGame_allSuperTypes_succeeds() public pure {
        assertTrue(GameTypes.isSuperGame(GameTypes.SUPER_CANNON), "SUPER_CANNON must be a super game");
        assertTrue(GameTypes.isSuperGame(GameTypes.SUPER_PERMISSIONED), "SUPER_PERMISSIONED must be a super game");
        assertTrue(GameTypes.isSuperGame(GameTypes.SUPER_ASTERISC_KONA), "SUPER_ASTERISC_KONA must be a super game");
        assertTrue(GameTypes.isSuperGame(GameTypes.SUPER_CANNON_KONA), "SUPER_CANNON_KONA must be a super game");
        assertTrue(GameTypes.isSuperGame(GameTypes.ZK_DISPUTE_GAME), "ZK_DISPUTE_GAME must be a super game");
    }
}
