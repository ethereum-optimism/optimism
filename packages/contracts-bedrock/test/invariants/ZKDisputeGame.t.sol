// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { Vm } from "forge-std/Vm.sol";
import { StdUtils } from "forge-std/StdUtils.sol";
import { ZKDisputeGame_TestInit } from "test/dispute/zk/ZKDisputeGame.t.sol";

// Libraries
import "src/dispute/lib/Types.sol";
import "src/dispute/lib/Errors.sol";

// Contracts
import { ZKDisputeGame } from "src/dispute/zk/ZKDisputeGame.sol";

contract ZKDisputeGame_Solvency_Invariant is ZKDisputeGame_TestInit {
    RandomZKActor internal actor;

    function setUp() public override {
        super.setUp();

        actor = new RandomZKActor(game, vm, proposer);

        targetContract(address(actor));
    }

    /// @custom:invariant ZKDisputeGame always returns all bonded ETH on resolution
    ///
    /// Whatever sequence of challenge, prove and time-advance the actor performs, driving the game to
    /// resolution and then closing it must return every wei bonded into the game to a participant.
    /// The game itself must never retain ETH, and no credit branch in `resolve()` may create or
    /// destroy value.
    function invariant_zkDisputeGame_solvency() public {
        // Snapshot balances after the actor's sequence, before the game is driven to completion. The
        // actor deals itself the challenger bond as it goes, so deltas from this point measure only
        // what the game pays back out.
        uint256 proposerBalanceBefore = proposer.balance;
        uint256 actorBalanceBefore = address(actor).balance;
        uint256 totalBonds = game.totalBonds();

        // Push past every deadline so the game is over no matter what the actor did.
        vm.warp(block.timestamp + maxChallengeDuration.raw() + maxProveDuration.raw() + 1 seconds);

        if (game.status() == GameStatus.IN_PROGRESS) {
            game.resolve();
        }

        // Wait out the finality airgap, then close the game to fix the bond distribution mode.
        vm.warp(game.resolvedAt().raw() + anchorStateRegistry.disputeGameFinalityDelaySeconds() + 1 seconds);
        game.closeGame();

        // Phase 1: unlock whatever credit each participant is owed.
        _unlockClaim(proposer);
        _unlockClaim(address(actor));

        // Phase 2: wait out the DelayedWETH delay, then withdraw.
        vm.warp(block.timestamp + delayedWeth.delay() + 1 seconds);
        _withdrawClaim(proposer);
        _withdrawClaim(address(actor));

        // The game must never hold ETH: bonds live in DelayedWETH from the moment they are posted.
        assertEq(address(game).balance, 0, "game must not hold ETH");

        // Every wei bonded into the game must land with a participant. The parent of this game
        // resolved DEFENDER_WINS, so no credit is burned to address(0) and the sum must be exact.
        uint256 totalReturned =
            (proposer.balance - proposerBalanceBefore) + (address(actor).balance - actorBalanceBefore);
        assertEq(totalReturned, totalBonds, "all bonded ETH must be returned to participants");
    }

    /// @notice Runs the unlock phase of a two-phase claim, skipping recipients owed nothing. A
    ///         recipient with no credit would revert with `NoCreditToClaim` once the game is closed;
    ///         that path belongs to the unit tests, not to this solvency check.
    function _unlockClaim(address _recipient) internal {
        if (game.credit(_recipient) == 0) return;
        game.claimCredit(_recipient);
    }

    /// @notice Runs the withdraw phase of a two-phase claim, skipping recipients with nothing
    ///         pending in DelayedWETH.
    function _withdrawClaim(address _recipient) internal {
        (uint256 amount,) = delayedWeth.withdrawals(address(game), _recipient);
        if (amount == 0) return;
        game.claimCredit(_recipient);
    }
}

/// @notice Drives a ZKDisputeGame through arbitrary orderings of the actions available to an
///         unprivileged participant. Every action is guarded so the actor never reverts; the fuzzer
///         explores which actions happen and in what order, not whether they succeed.
contract RandomZKActor is StdUtils {
    ZKDisputeGame internal immutable GAME;
    Vm internal immutable VM;
    address internal immutable PROPOSER;

    /// @notice Total ETH the actor has bonded into the game.
    uint256 public totalBonded;

    constructor(ZKDisputeGame _game, Vm _vm, address _proposer) {
        GAME = _game;
        VM = _vm;
        PROPOSER = _proposer;
    }

    /// @notice Challenges the game if it is still challengeable.
    function challenge() public {
        if (GAME.gameOver()) return;
        (, ZKDisputeGame.ProposalStatus status,,,,) = GAME.claimData();
        if (status != ZKDisputeGame.ProposalStatus.Unchallenged) return;

        uint256 bond = GAME.challengerBond();
        VM.deal(address(this), address(this).balance + bond);
        totalBonded += bond;
        GAME.challenge{ value: bond }();
    }

    /// @notice Proves the game as a third party, so the prover differs from the game creator.
    function prove() public {
        if (!_provable()) return;
        GAME.prove(bytes(""));
    }

    /// @notice Proves the game as the proposer, exercising the `prover == gameCreator` credit branch.
    function proveAsProposer() public {
        if (!_provable()) return;
        VM.prank(PROPOSER);
        GAME.prove(bytes(""));
    }

    /// @notice Advances time, so deadlines can expire mid-sequence and the unproven branches become
    ///         reachable.
    function warpForward(uint256 _seconds) public {
        VM.warp(block.timestamp + bound(_seconds, 1, 2 days));
    }

    /// @notice Whether `prove()` would currently succeed.
    function _provable() internal view returns (bool) {
        return GAME.status() == GameStatus.IN_PROGRESS && !GAME.gameOver();
    }

    fallback() external payable { }

    receive() external payable { }
}
