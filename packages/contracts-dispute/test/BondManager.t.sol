// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

import "src/types/Types.sol";
import "src/types/Errors.sol";

import "forge-std/Test.sol";
import { DisputeGameFactory } from "src/DisputeGameFactory.sol";
import { IDisputeGame } from "src/interfaces/IDisputeGame.sol";

import { BondManager } from "src/BondManager.sol";

contract BondManager_Test is Test {
    DisputeGameFactory factory;
    BondManager bm;

    // DisputeGameFactory events
    event DisputeGameCreated(address indexed disputeProxy, GameType indexed gameType, Claim indexed rootClaim);

    // BondManager events
    event BondPosted(bytes32 bondId, address owner, uint64 expiration, uint256 amount);
    event BondSeized(bytes32 bondId, address owner, address seizer, uint256 amount);
    event BondReclaimed(bytes32 bondId, address claiment, uint256 amount);

    function setUp() public {
        factory = new DisputeGameFactory(address(this));
        bm = new BondManager(factory);
    }

    /// @notice Tests that posting a bond succeeds.
    function testFuzz_post_succeeds(bytes32 bondId, address owner, uint256 minClaimHold, uint256 amount) public {
        vm.assume(owner != address(0));
        vm.assume(owner != address(bm));
        vm.assume(owner != address(this));
        unchecked {
            vm.assume(block.timestamp + minClaimHold > minClaimHold);
        }
        console2.log("Block timestamp:", block.timestamp);

        // Make sure the bond doesn't already exist
        (address fetchedOwner,,,) = bm.bonds(bondId);
        vm.assume(fetchedOwner == address(0));

        // Cannot have a 0 value bond
        vm.assume(amount != 0);

        // Post
        vm.deal(address(this), amount);
        bm.post{value: amount}(bondId, owner, minClaimHold);

        // Validate the bond
        (address newFetchedOwner, uint256 fetchedExpiration, bytes32 fetchedBondId, uint256 bondAmount) = bm.bonds(bondId);
        assertEq(newFetchedOwner, owner);
        assertEq(fetchedExpiration, block.timestamp + minClaimHold);
        assertEq(fetchedBondId, bondId);
        assertEq(bondAmount, amount);
    }

}
