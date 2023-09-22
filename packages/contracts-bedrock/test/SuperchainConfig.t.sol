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

contract SuperchainConfig_Pause_TestFail is SuperchainConfig_Initializer {
    /// @dev Tests that `pause` reverts when called by a non-GUARDIAN.
    function test_pause_onlyGuardian_reverts() external {
        assertEq(supConf.paused(), false);

        assertTrue(supConf.guardian() != alice);
        vm.expectRevert("SuperchainConfig: only guardian can pause");
        vm.prank(alice);
        supConf.pause();

        assertEq(supConf.paused(), false);
    }

    /// @dev Tests that `unpause` reverts when called by a non-GUARDIAN.
    function test_unpause_onlyGuardian_reverts() external {
        vm.prank(guardian);
        supConf.pause();
        assertEq(supConf.paused(), true);

        assertTrue(supConf.guardian() != alice);
        vm.expectRevert("SuperchainConfig: only guardian can unpause");
        vm.prank(alice);
        supConf.unpause();

        assertEq(supConf.paused(), true);
    }
}

contract SuperchainConfig_Pause_Test is SuperchainConfig_Initializer {
    /// @dev Tests that `pause` successfully pauses
    ///      when called by the GUARDIAN.
    function test_pause_succeeds() external {
        assertEq(supConf.paused(), false);

        vm.expectEmit(true, true, true, true, address(supConf));
        emit Paused();

        vm.prank(guardian);
        supConf.pause();

        assertEq(supConf.paused(), true);
    }

    /// @dev Tests that `unpause` successfully unpauses
    ///      when called by the GUARDIAN.
    function test_unpause_succeeds() external {
        vm.prank(guardian);
        supConf.pause();
        assertEq(supConf.paused(), true);

        vm.expectEmit(true, true, true, true, address(supConf));
        emit Unpaused();
        vm.prank(guardian);
        supConf.unpause();

        assertEq(supConf.paused(), false);
    }
}

contract SuperchainConfig_AddSequencer_TestFail is SuperchainConfig_Initializer {
    /// @dev Tests that `addSequencer` successfully adds a sequencer
    function testFuzz_addSequencer_notOwner_reverts() external {
        vm.expectRevert("Ownable: caller is not the owner");
        supConf.addSequencer(dummySequencer);
    }
}

contract SuperchainConfig_AddSequencer_Test is SuperchainConfig_Initializer {
    /// @dev Tests that `addSequencer` successfully adds a sequencer
    function testFuzz_addSequencer_succeeds(Types.SequencerKeys calldata sequencer) external {
        // Add to the allowed sequencers list
        vm.expectEmit(true, true, true, true);
        emit ConfigUpdate(0, SuperchainConfig.UpdateType.ADD_SEQUENCER, abi.encode(sequencer));

        vm.prank(supConf.owner());
        supConf.addSequencer(sequencer);
        bytes32 sequencerHash = Hashing.hashSequencerKeys(sequencer);
        assertTrue(supConf.allowedSequencers(sequencerHash));
    }
}

contract SuperchainConfig_RemoveSequencer_TestFail is SuperchainConfig_Initializer {
    /// @dev Tests that `removeSequencer` successfully removes a sequencer
    function testFuzz_removeSequencer_notOwner_reverts() external {
        vm.expectRevert("Ownable: caller is not the owner");
        supConf.removeSequencer(dummySequencer);
    }
}

contract SuperchainConfig_RemoveSequencer_Test is SuperchainConfig_Initializer {
    /// @dev Tests that `removeSequencer` successfully removes a sequencer
    function testFuzz_removeSequencer_succeeds(Types.SequencerKeys calldata sequencer) external {
        // Remove from the allowed sequencers list
        vm.expectEmit(true, true, true, true);
        emit ConfigUpdate(0, SuperchainConfig.UpdateType.REMOVE_SEQUENCER, abi.encode(sequencer));

        vm.prank(supConf.owner());
        supConf.removeSequencer(sequencer);
        bytes32 sequencerHash = Hashing.hashSequencerKeys(sequencer);
        assertFalse(supConf.allowedSequencers(sequencerHash));
    }
}


