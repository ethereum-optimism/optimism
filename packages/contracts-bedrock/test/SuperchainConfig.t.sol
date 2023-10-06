// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing utilities
import { SuperchainConfig_Initializer } from "./CommonTest.t.sol";

// Libraries
import { Constants } from "src/libraries/Constants.sol";
import { Types } from "src/libraries/Types.sol";
import { Hashing } from "src/libraries/Hashing.sol";

// Target contract dependencies
import { Proxy } from "src/universal/Proxy.sol";

// Target contract
import { SuperchainConfig } from "src/L1/SuperchainConfig.sol";

contract SuperchainConfig_Init_Test is SuperchainConfig_Initializer {
    /// @dev Tests that initialization sets the correct values. These are defined in CommonTest.sol.
    function test_initialize_values_succeeds() external {
        assertEq(supConf.systemOwner(), systemOwner);
        assertEq(supConf.initiator(), initiator);
        assertEq(supConf.vetoer(), vetoer);
        assertEq(supConf.guardian(), guardian);
        assertEq(supConf.delay(), delay);
        assertEq(supConf.maxPause(), maxPause);
        assertFalse(supConf.paused());
        bytes32 sequencerHash = Hashing.hashSequencerKeyPair(dummySequencer);
        assertEq(supConf.allowedSequencers(sequencerHash), true);
        assertEq(supConf.isAllowedSequencer(dummySequencer), true);
    }
}

contract SuperchainConfig_Pause_TestFail is SuperchainConfig_Initializer {
    /// @dev Tests that `pause` reverts when called by a non-GUARDIAN.
    function test_pause_notGuardian_reverts() external {
        assertFalse(supConf.paused());

        assertTrue(supConf.guardian() != alice);
        vm.expectRevert("SuperchainConfig: only guardian can pause");
        vm.prank(alice);
        supConf.pause(100, "identifier");

        assertFalse(supConf.paused());
    }

    /// @dev Tests that `pause` reverts when the duration is greater than the max pause.
    function test_pause_durationGreaterThanMaxPause_reverts() external {
        vm.expectRevert("SuperchainConfig: duration exceeds maxPause");
        vm.prank(guardian);
        supConf.pause(maxPause + 1, "identifier");

        assertFalse(supConf.paused());
    }
}

contract SuperchainConfig_Pause_Test is SuperchainConfig_Initializer {
    /// @dev Tests that `pause` successfully pauses
    ///      when called by the GUARDIAN.
    function test_pause_succeeds() external {
        assertFalse(supConf.paused());

        vm.expectEmit(address(supConf));
        emit Paused(100, "identifier");

        vm.prank(guardian);
        supConf.pause(100, "identifier");

        assertTrue(supConf.paused());
        assertEq(supConf.pausedUntil(), block.timestamp + 100);
    }

    /// @dev Tests that `pause` automatially unpauses after the duration has passed
    function test_pause_thaws_succeeds() external {
        _pause();

        vm.warp(block.timestamp + 100);
        assertFalse(supConf.paused());
    }
}

contract SuperchainConfig_Unpause_TestFail is SuperchainConfig_Initializer {
    /// @dev Tests that `unpause` reverts when called by a non-GUARDIAN.
    function test_unpause_notGuardian_reverts() external {
        _pause();

        assertTrue(supConf.guardian() != alice);
        vm.expectRevert("SuperchainConfig: only guardian can unpause");
        vm.prank(alice);
        supConf.unpause();

        assertTrue(supConf.paused());
    }
}

contract SuperchainConfig_Unpause_Test is SuperchainConfig_Initializer {
    /// @dev Tests that `unpause` successfully unpauses
    ///      when called by the GUARDIAN.
    function test_unpause_succeeds() external {
        _pause();

        vm.expectEmit(address(supConf));
        emit Unpaused();
        vm.prank(guardian);
        supConf.unpause();

        assertFalse(supConf.paused());
    }
}

contract SuperchainConfig_ExtendPause_TestFail is SuperchainConfig_Initializer {
    /// @dev Tests that `extendPause` reverts when called by a non-GUARDIAN.
    function test_pause_extendingPausenotGuardian_reverts() external {
        _pause();

        assertTrue(supConf.guardian() != alice);
        vm.expectRevert("SuperchainConfig: only guardian can pause");
        vm.prank(alice);
        supConf.pause(100, "identifier");

        assertTrue(supConf.paused());
    }

    /// @dev Tests that `extendPause` reverts when the duration is greater than the max pause.
    function test_pause_extendingPauseDurationGreaterThanMaxPause_reverts() external {
        _pause();

        vm.expectRevert("SuperchainConfig: duration exceeds maxPause");
        vm.prank(guardian);
        supConf.pause(maxPause + 1, "identifier");

        assertTrue(supConf.paused());
    }
}

contract SuperchainConfig_ExtendPause_Test is SuperchainConfig_Initializer {
    /// @dev Tests that `extendPause` successfully extends the pause by the duration
    ///      when called by the GUARDIAN.
    function test_extendPause_succeeds() external {
        _pause();

        uint256 pausedUntilBefore = supConf.pausedUntil();
        vm.expectEmit(address(supConf));
        emit PauseExtended(200, "identifier");

        vm.prank(guardian);
        supConf.pause(200, "identifier");

        assertTrue(supConf.paused());
        assertEq(pausedUntilBefore + 200, supConf.pausedUntil());
    }
}

contract SuperchainConfig_AddSequencer_TestFail is SuperchainConfig_Initializer {
    /// @dev Tests that `addSequencer` successfully adds a sequencer
    function testFuzz_addSequencer_notOwner_reverts() external {
        vm.expectRevert("SuperchainConfig: only initiator can add sequencer");
        supConf.addSequencer(dummySequencer);
    }
}

contract SuperchainConfig_AddSequencer_Test is SuperchainConfig_Initializer {
    /// @dev Tests that `addSequencer` successfully adds a sequencer
    function testFuzz_addSequencer_succeeds(Types.SequencerKeyPair calldata sequencer) external {
        // Add to the allowed sequencers list
        vm.expectEmit(true, true, true, true);
        emit ConfigUpdate(SuperchainConfig.UpdateType.ADD_SEQUENCER, abi.encode(sequencer));

        vm.prank(supConf.initiator());
        supConf.addSequencer(sequencer);
        bytes32 sequencerHash = Hashing.hashSequencerKeyPair(sequencer);
        assertTrue(supConf.allowedSequencers(sequencerHash));
    }
}

contract SuperchainConfig_RemoveSequencer_TestFail is SuperchainConfig_Initializer {
    /// @dev Tests that `removeSequencer` successfully removes a sequencer
    function testFuzz_removeSequencer_notOwner_reverts() external {
        vm.expectRevert("SuperchainConfig: only systemOwner can remove a sequencer");
        supConf.removeSequencer(dummySequencer);
    }
}

contract SuperchainConfig_RemoveSequencer_Test is SuperchainConfig_Initializer {
    /// @dev Tests that `removeSequencer` successfully removes a sequencer
    function testFuzz_removeSequencer_succeeds(Types.SequencerKeyPair calldata sequencer) external {
        // Remove from the allowed sequencers list
        vm.expectEmit(true, true, true, true);
        emit ConfigUpdate(SuperchainConfig.UpdateType.REMOVE_SEQUENCER, abi.encode(sequencer));

        vm.prank(supConf.systemOwner());
        supConf.removeSequencer(sequencer);
        bytes32 sequencerHash = Hashing.hashSequencerKeyPair(sequencer);
        assertFalse(supConf.allowedSequencers(sequencerHash));
    }
}
