// SPDX-License-Identifier: MIT
pragma solidity ^0.8.17;

import "forge-std/Test.sol";

import { ClaimAlreadyExists } from "src/types/Errors.sol";
import { CannotDefendRootClaim } from "src/types/Errors.sol";

import { Clock } from "src/types/Types.sol";
import { Claim } from "src/types/Types.sol";
import { Position } from "src/types/Types.sol";
import { Duration } from "src/types/Types.sol";
import { GameType } from "src/types/Types.sol";
import { Timestamp } from "src/types/Types.sol";
import { ClaimHash } from "src/types/Types.sol";

import { LibClock } from "src/lib/LibClock.sol";
import { LibHashing } from "src/lib/LibHashing.sol";
import { LibPosition } from "src/lib/LibPosition.sol";

import { FaultDisputeGame } from "src/FaultDisputeGame.sol";
import { IDisputeGameFactory } from "src/interfaces/IDisputeGameFactory.sol";

/// @title FaultDisputeGame_Test
// contract FaultDisputeGame_Test is Test {
//     /// @notice The current semantic version of the [FaultDisputeGame] contract.
//     /// @dev This should be updated whenever the contract is changed.
//     string constant VERSION = "0.0.1";
//
//     /// @notice The game type for a [FaultDisputeGame] contract.
//     GameType constant FAULT_GAME_TYPE = LibGames.FaultGameType;
//
//     /// @notice The root claim of the test case game.
//     Claim constant ROOT_CLAIM = Claim.wrap(bytes32(uint256(256)));
//
//     /// @notice The position of the root claim.
//     Position constant ROOT_POSITION = Position.wrap(0);
//
//     /// @notice A clone of the [FaultDisputeGame] contract being tested.
//     FaultDisputeGame internal disputeGame;
//
//     /// @notice Emitted when a new dispute game is created by the [DisputeGameFactory]
//     event DisputeGameCreated(address indexed disputeProxy, GameType indexed gameType, Claim indexed rootClaim);
//
//     /// @notice Emitted when a subclaim is disagreed upon by `claimant`
//     /// @dev Disagreeing with a subclaim is akin to attacking it.
//     /// @param claimHash The unique ClaimHash that is being disagreed upon
//     /// @param pivot The claim for the following pivot (disagreement = go left)
//     /// @param claimant The address of the claimant
//     event Attack(ClaimHash indexed claimHash, Claim indexed pivot, address indexed claimant);
//
//     /// @notice Emitted when a subclaim is agreed upon by `claimant`
//     /// @dev Agreeing with a subclaim is akin to defending it.
//     /// @param claimHash The unique ClaimHash that is being agreed upon
//     /// @param pivot The claim for the following pivot (agreement = go right)
//     /// @param claimant The address of the claimant
//     event Defend(ClaimHash indexed claimHash, Claim indexed pivot, address indexed claimant);
//
//     function setUp() public {
//         // Deploy the reference [FaultDisputeGame] contract.
//         // This contract will be delegatecalled by all clone proxies.
//         disputeGame = new FaultDisputeGame();
//
//         // Create a clone factory to create [FaultDisputeGame] clones with.
//         IDisputeGameFactory factory = IDisputeGameFactory(
//             HuffDeployer.config().with_args(abi.encode(address(disputeGame))).with_addr_constant(
//                 "IMPL_ADDR", address(disputeGame)
//             ).deploy("DisputeGameFactory")
//         );
//
//         // Assert that the `DisputeGameCreated` event was emitted (sans checking the `disputeProxy` address)
//         vm.expectEmit(false, true, true, false);
//         emit DisputeGameCreated(address(0), FAULT_GAME_TYPE, ROOT_CLAIM);
//
//         // Deploy a clone of `disputeGame`.
//         bytes memory _extraData = bytes("");
//         disputeGame = FaultDisputeGame(address(factory.create(FAULT_GAME_TYPE, ROOT_CLAIM, _extraData)));
//
//         // Label the dispute game proxy
//         vm.label(address(disputeGame), "FaultDisputeGame_Proxy");
//     }
//
//     /// @notice Asserts that the [FaultDisputeGame] proxy was initialized.
//     function test_initialized_succeeds() public {
//         assertTrue(disputeGame.initialized());
//
//         // TODO: Assert correctness of the `FaultDisputeGame` proxy's state after initialization.
//     }
//
//     /// @notice Asserts that the [FaultDisputeGame] proxy has the correct version.
//     function test_version_correctness() public {
//         assertEq(disputeGame.version(), VERSION);
//     }
//
//     /// @notice Asserts that the [FaultDisputeGame] proxy has the correct GameType.
//     function test_gameType_correctness() public {
//         assertEq(GameType.unwrap(disputeGame.gameType()), GameType.unwrap(FAULT_GAME_TYPE));
//     }
//
//     /// @notice Asserts that the [FaultDisputeGame] proxy has the correct rootClaim.
//     function test_rootClaim_correctness() public {
//         assertEq(Claim.unwrap(disputeGame.rootClaim()), bytes32(uint256(256)));
//     }
//
//     ////////////////////////////////////////////////////////////////
//     //                       `attack` Tests                       //
//     ////////////////////////////////////////////////////////////////
//
//     /// @notice Tests that an initial attack against the root claim succeeds and that the state was set properly.
//     function test_attack_initialAttack_suceeds() public {
//         // Fast forward 1 hour from the start of the game.
//         vm.warp(Timestamp.unwrap(disputeGame.gameStart()) + 1 hours);
//
//         ClaimHash claimHash = LibHashing.hashClaimPos(ROOT_CLAIM, ROOT_POSITION);
//         Claim pivot = Claim.wrap(bytes32(uint256(128)));
//
//         // Compute the pivot claim hash
//         Position attackPos = LibPosition.attack(ROOT_POSITION);
//         ClaimHash pivotClaimHash = LibHashing.hashClaimPos(pivot, attackPos);
//
//         // Ensure that the `Attack` event is emitted
//         vm.expectEmit(true, true, true, false);
//         emit Attack(pivotClaimHash, pivot, address(this));
//
//         // The initial attack on the root claim should succeed.
//         disputeGame.attack(claimHash, pivot);
//
//         // Ensure that the preimage claim to the `pivotClaimHash` was properly set to `pivot`.
//         assertEq(Claim.unwrap(disputeGame.claims(pivotClaimHash)), Claim.unwrap(pivot));
//
//         // Ensure that the `pivotClaimHash`'s position was properly set to `attackPos`.
//         assertEq(Position.unwrap(disputeGame.positions(pivotClaimHash)), Position.unwrap(attackPos));
//
//         // Ensure that the `pivotClaimHash`'s parent was properly set to `claimHash`.
//         assertEq(ClaimHash.unwrap(disputeGame.parents(pivotClaimHash)), ClaimHash.unwrap(claimHash));
//
//         // Ensure that the `claimHash` was marked as countered.
//         assertTrue(disputeGame.countered(claimHash));
//
//         // Ensure that the `claimHash`'s reference counter was incremented by one.
//         assertEq(disputeGame.rc(claimHash), 1);
//
//         // Ensure the clock was set properly
//         Clock pivotClock = disputeGame.clocks(pivotClaimHash);
//         assertEq(Duration.unwrap(LibClock.duration(pivotClock)), 3.5 days - 1 hours);
//         assertEq(Timestamp.unwrap(LibClock.timestamp(pivotClock)), block.timestamp);
//     }
//
//     /// @notice Tests that the same attack cannot be performed twice.
//     function test_attack_cannotAttackTwice_reverts() public {
//         disputeGame.attack(LibHashing.hashClaimPos(ROOT_CLAIM, ROOT_POSITION), Claim.wrap(bytes32(uint256(128))));
//         vm.expectRevert(ClaimAlreadyExists.selector);
//         disputeGame.attack(LibHashing.hashClaimPos(ROOT_CLAIM, ROOT_POSITION), Claim.wrap(bytes32(uint256(128))));
//     }
//
//     ////////////////////////////////////////////////////////////////
//     //                       `defend` Tests                       //
//     ////////////////////////////////////////////////////////////////
//
//     /// @notice Tests that a defense against an existing counter claim succeeds and the state is updated properly.
//     function test_defend_suceeds() public {
//         Claim attackPivot = Claim.wrap(bytes32(uint256(128)));
//         Claim defendPivot = Claim.wrap(bytes32(uint256(196)));
//
//         // Perform an attack against the root claim
//         test_attack_initialAttack_suceeds();
//
//         // Compute the pivot claim hash
//         Position attackPos = LibPosition.attack(ROOT_POSITION);
//         ClaimHash attackPivotClaimHash = LibHashing.hashClaimPos(attackPivot, attackPos);
//
//         // Compute the defense claim hash
//         Position defendPos = LibPosition.defend(attackPos);
//         ClaimHash defendPivotClaimHash = LibHashing.hashClaimPos(defendPivot, defendPos);
//
//         // Warp ahead 2 hours
//         vm.warp(block.timestamp + 2 hours);
//
//         vm.expectEmit(true, true, true, false);
//         emit Defend(defendPivotClaimHash, defendPivot, address(this));
//
//         disputeGame.defend(attackPivotClaimHash, defendPivot);
//
//         // Ensure that the preimage claim to the `defendPivotClaimHash` was properly set to `defendPivot`.
//         assertEq(Claim.unwrap(disputeGame.claims(defendPivotClaimHash)), Claim.unwrap(defendPivot));
//
//         // Ensure that the `defendPivotClaimHash`'s position was properly set to `defendPos`.
//         assertEq(Position.unwrap(disputeGame.positions(defendPivotClaimHash)), Position.unwrap(defendPos));
//
//         // Ensure that the `defendPivotClaimHash`'s parent was properly set to `attackPivotClaimHash`.
//         assertEq(ClaimHash.unwrap(disputeGame.parents(defendPivotClaimHash)), ClaimHash.unwrap(attackPivotClaimHash));
//
//         // Ensure that the `attackPivotClaimHash` was marked as countered.
//         assertTrue(disputeGame.countered(attackPivotClaimHash));
//
//         // Ensure that the `attackPivotClaimHash`'s reference counter was incremented by one.
//         assertEq(disputeGame.rc(attackPivotClaimHash), 1);
//
//         // Ensure the clock was set properly
//         Clock pivotClock = disputeGame.clocks(defendPivotClaimHash);
//         assertEq(Duration.unwrap(LibClock.duration(pivotClock)), 3.5 days - 2 hours);
//         assertEq(Timestamp.unwrap(LibClock.timestamp(pivotClock)), block.timestamp);
//     }
//
//     function test_defend_cannotDefendRootClaim_reverts() public {
//         ClaimHash rootClaimHash = LibHashing.hashClaimPos(ROOT_CLAIM, ROOT_POSITION);
//         Claim pivot = Claim.wrap(bytes32(uint256(128)));
//
//         // A defense against the root claim should fail.
//         vm.expectRevert(CannotDefendRootClaim.selector);
//         disputeGame.defend(rootClaimHash, pivot);
//     }
//
//     /// @notice Tests that the same defense cannot be performed twice.
//     function test_defend_cannotDefendTwice_reverts() public {
//         Claim attackPivot = Claim.wrap(bytes32(uint256(128)));
//         Claim defendPivot = Claim.wrap(bytes32(uint256(196)));
//
//         // Perform an attack against the root claim
//         test_attack_initialAttack_suceeds();
//
//         // Compute the pivot claim hash
//         Position attackPos = LibPosition.attack(ROOT_POSITION);
//         ClaimHash attackPivotClaimHash = LibHashing.hashClaimPos(attackPivot, attackPos);
//
//         // The first defense should succeed, but the second should revert.
//         disputeGame.defend(attackPivotClaimHash, defendPivot);
//         vm.expectRevert(ClaimAlreadyExists.selector);
//         disputeGame.defend(attackPivotClaimHash, defendPivot);
//     }
// }
