// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { CommonTest } from "test/setup/CommonTest.sol";
import { Safe } from "safe-contracts/Safe.sol";
import "test/safe-tools/SafeTestTools.sol";

import { IDisputeGame } from "src/dispute/interfaces/IDisputeGame.sol";
import { DeputyGuardianModule } from "src/Safe/DeputyGuardianModule.sol";

import "src/libraries/DisputeTypes.sol";

contract DeputyGuardianModule_TestInit is CommonTest, SafeTestTools {
    using SafeTestLib for SafeInstance;

    DeputyGuardianModule deputyGuardianModule;
    SafeInstance safeInstance;
    address deputyGuardian;

    /// @dev Sets up the test environment
    function setUp() public virtual override {
        super.enableFaultProofs();
        super.setUp();

        // Create a Safe with 10 owners
        (, uint256[] memory keys) = SafeTestLib.makeAddrsAndKeys("moduleTest", 10);
        safeInstance = _setupSafe(keys, 10);

        // Set the Safe as the Guardian of the SuperchainConfig
        vm.store(
            address(superchainConfig),
            superchainConfig.GUARDIAN_SLOT(),
            bytes32(uint256(uint160(address(safeInstance.safe))))
        );

        deputyGuardian = makeAddr("deputyGuardian");

        deputyGuardianModule = new DeputyGuardianModule({
            _safe: safeInstance.safe,
            _superchainConfig: superchainConfig,
            _deputyGuardian: deputyGuardian
        });
        safeInstance.enableModule(address(deputyGuardianModule));
    }
}

contract DeputyGuardianModule_Getters_Test is DeputyGuardianModule_TestInit {
    /// @dev Tests that the constructor sets the correct values
    function test_getters_works() external {
        assertEq(address(deputyGuardianModule.safe()), address(safeInstance.safe));
        assertEq(address(deputyGuardianModule.deputyGuardian()), address(deputyGuardian));
        assertEq(address(deputyGuardianModule.superchainConfig()), address(superchainConfig));
    }
}

contract DeputyGuardianModule_Pause_Test is DeputyGuardianModule_TestInit {
    /// @dev Tests that `pause` successfully pauses when called by the deputy guardian.
    function test_pause_succeeds() external {
        // Pause the SuperchainConfig contract
        vm.prank(address(deputyGuardian));
        deputyGuardianModule.pause();
        assertEq(superchainConfig.paused(), true);
    }
}

contract DeputyGuardianModule_Pause_TestFail is DeputyGuardianModule_TestInit {
    /// @dev Tests that `pause` reverts when called by a non deputy guardian.
    function test_pause_notDeputyGuardian_reverts() external {
        vm.expectRevert("DeputyGuardianModule: Only the deputy guardian can pause.");
        deputyGuardianModule.pause();
    }
}

contract DeputyGuardianModule_Unpause_Test is DeputyGuardianModule_TestInit {
    /// @dev Sets up the test environment with the SuperchainConfig paused
    function setUp() public override {
        super.setUp();
        vm.prank(address(deputyGuardian));
        deputyGuardianModule.pause();
        assertEq(superchainConfig.paused(), true);
    }

    /// @dev Tests that `unpause` successfully unpauses when called by the deputy guardian.
    function test_unpause_succeeds() external {
        assertEq(superchainConfig.paused(), true);

        vm.prank(address(deputyGuardian));
        deputyGuardianModule.unpause();
        assertEq(superchainConfig.paused(), false);
    }
}

contract DeputyGuardianModule_Unpause_TestFail is DeputyGuardianModule_Unpause_Test {
    /// @dev Tests that `unpause` reverts when called by a non deputy guardian.
    function test_pause_notDeputyGuardian_reverts() external {
        assertEq(superchainConfig.paused(), true);

        vm.expectRevert("DeputyGuardianModule: Only the deputy guardian can unpause.");
        deputyGuardianModule.unpause();
    }
}

contract DeputyGuardianModule_BlacklistDisputeGame_Test is DeputyGuardianModule_TestInit {
    /// @dev Tests that `blacklistDisputeGame` successfully blacklists a dispute game when called by the deputy
    /// guardian.
    function test_blacklistDisputeGame_succeeds() external {
        IDisputeGame game = IDisputeGame(makeAddr("game"));
        vm.prank(address(deputyGuardian));
        deputyGuardianModule.blacklistDisputeGame(optimismPortal2, game);
        assertTrue(optimismPortal2.disputeGameBlacklist(game));
    }
}

contract DeputyGuardianModule_BlacklistDisputeGame_TestFail is DeputyGuardianModule_TestInit {
    /// @dev Tests that `blacklistDisputeGame` reverts when called by a non deputy guardian.
    function test_blacklistDisputeGame_notDeputyGuardian_reverts() external {
        IDisputeGame game = IDisputeGame(makeAddr("game"));
        vm.expectRevert("DeputyGuardianModule: Only the deputy guardian can blacklist dispute games.");
        deputyGuardianModule.blacklistDisputeGame(optimismPortal2, game);
        assertFalse(optimismPortal2.disputeGameBlacklist(game));
    }
}

contract DeputyGuardianModule_setRespectedGameType_Test is DeputyGuardianModule_TestInit {
    /// @dev Tests that `setRespectedGameType` successfully updates the respected game type when called by the deputy
    /// guardian.
    function testFuzz_setRespectedGameType_succeeds(GameType _gameType) external {
        vm.prank(address(deputyGuardian));
        deputyGuardianModule.setRespectedGameType(optimismPortal2, _gameType);
        assertEq(GameType.unwrap(optimismPortal2.respectedGameType()), GameType.unwrap(_gameType));
        assertEq(optimismPortal2.respectedGameTypeUpdatedAt(), uint64(block.timestamp));
    }
}

contract DeputyGuardianModule_setRespectedGameType_TestFail is DeputyGuardianModule_TestInit {
    /// @dev Tests that `setRespectedGameType` when called by a non deputy guardian.
    function testFuzz_setRespectedGameType_notDeputyGuardian_reverts(GameType _gameType) external {
        vm.assume(GameType.unwrap(optimismPortal2.respectedGameType()) != GameType.unwrap(_gameType));
        vm.expectRevert("DeputyGuardianModule: Only the deputy guardian can set the respected game type.");
        deputyGuardianModule.setRespectedGameType(optimismPortal2, _gameType);
        assertNotEq(GameType.unwrap(optimismPortal2.respectedGameType()), GameType.unwrap(_gameType));
    }
}
