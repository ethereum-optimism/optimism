// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { BaseSuperFaultDisputeGame_TestInit } from "test/dispute/SuperFaultDisputeGame.t.sol";
import { IFaultDisputeGame } from "interfaces/dispute/IFaultDisputeGame.sol";
import { RandomClaimActor } from "test/invariants/FaultDisputeGame.t.sol";

// Libraries
import "src/dispute/lib/Types.sol";
import "src/dispute/lib/Errors.sol";

contract SuperFaultDisputeGame_Solvency_Invariant is BaseSuperFaultDisputeGame_TestInit {
    Claim internal constant ROOT_CLAIM = Claim.wrap(bytes32(uint256(10)));
    Claim internal constant ABSOLUTE_PRESTATE = Claim.wrap(bytes32((uint256(3) << 248) | uint256(0)));

    RandomClaimActor internal actor;
    uint256 internal defaultSenderBalance;

    function setUp() public override {
        super.setUp();
        super.init({ _rootClaim: ROOT_CLAIM, _absolutePrestate: ABSOLUTE_PRESTATE, _l2SequenceNumber: 0x10 });

        actor = new RandomClaimActor(IFaultDisputeGame(address(gameProxy)), vm);

        targetContract(address(actor));
        vm.startPrank(address(actor));
    }

    /// @custom:invariant SuperFaultDisputeGame always returns all ETH on total resolution
    ///
    /// The SuperFaultDisputeGame contract should always return all ETH in the contract to the correct recipients upon
    /// resolution of all outstanding claims. There may never be any ETH left in the contract after a full resolution.
    function invariant_faultDisputeGame_solvency() public {
        vm.warp(block.timestamp + 7 days + 1 seconds);

        (,,, uint256 rootBond,,,) = gameProxy.claimData(0);

        // Ensure the game creator has locked up the root bond.
        assertEq(address(this).balance, type(uint96).max - rootBond);

        for (uint256 i = gameProxy.claimDataLen(); i > 0; i--) {
            (bool success,) = address(gameProxy).call(abi.encodeCall(gameProxy.resolveClaim, (i - 1, 0)));
            assertTrue(success);
        }
        gameProxy.resolve();

        // Wait for finalization delay
        vm.warp(block.timestamp + 3.5 days + 1 seconds);

        // Close the game.
        gameProxy.closeGame();

        // Claim credit once to trigger unlock period.
        gameProxy.claimCredit(address(this));
        gameProxy.claimCredit(address(actor));

        // Wait for the withdrawal delay.
        vm.warp(block.timestamp + 7 days + 1 seconds);

        if (gameProxy.credit(address(this)) == 0) {
            vm.expectRevert(NoCreditToClaim.selector);
            gameProxy.claimCredit(address(this));
        } else {
            gameProxy.claimCredit(address(this));
        }

        if (gameProxy.credit(address(actor)) == 0) {
            vm.expectRevert(NoCreditToClaim.selector);
            gameProxy.claimCredit(address(actor));
        } else {
            gameProxy.claimCredit(address(actor));
        }

        if (gameProxy.status() == GameStatus.DEFENDER_WINS) {
            // In the event that the defender wins, they receive their bond back. The root claim is never paid out
            // bonds from claims below it, so the actor that has challenged the root claim (and potentially their)
            // own receives all of their bonds back.
            assertEq(address(this).balance, type(uint96).max);
            assertEq(address(actor).balance, actor.totalBonded());
        } else if (gameProxy.status() == GameStatus.CHALLENGER_WINS) {
            // If the defender wins, the game creator loses the root bond and the actor receives it. The actor also
            // is the only party that may have challenged their own claims, so we expect them to receive all of them
            // back.
            assertEq(address(this).balance, type(uint96).max - rootBond);
            assertEq(address(actor).balance, actor.totalBonded() + rootBond);
        } else {
            revert("SuperFaultDisputeGame_Solvency_Invariant: unreachable");
        }

        assertEq(address(gameProxy).balance, 0);
    }
}
