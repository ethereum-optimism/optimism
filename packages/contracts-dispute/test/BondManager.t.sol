// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

import "src/types/Types.sol";
import "src/types/Errors.sol";

import { Test } from "forge-std/Test.sol";
import { DisputeGameFactory } from "src/DisputeGameFactory.sol";
import { IDisputeGame } from "src/interfaces/IDisputeGame.sol";

import { BondManager } from "src/BondManager.sol";

contract BondManager_Test is Test {
    DisputeGameFactory factory;
    BondManager bm;

    // DisputeGameFactory events
    event DisputeGameCreated(address indexed disputeProxy, GameType indexed gameType, Claim indexed rootClaim);

    // BondManager events
    event BondPosted(bytes32 bondId, address owner, uint256 expiration, uint256 amount);
    event BondSeized(bytes32 bondId, address owner, address seizer, uint256 amount);
    event BondReclaimed(bytes32 bondId, address claiment, uint256 amount);

    function setUp() public {
        factory = new DisputeGameFactory(address(this));
        bm = new BondManager(factory);
    }

    /// -------------------------------------------
    /// Test Bond Posting
    /// -------------------------------------------

    /// @notice Tests that posting a bond succeeds.
    function testFuzz_post_succeeds(bytes32 bondId, address owner, uint256 minClaimHold, uint256 amount) public {
        vm.assume(owner != address(0));
        vm.assume(owner != address(bm));
        vm.assume(owner != address(this));
        vm.assume(amount != 0);
        unchecked {
            vm.assume(block.timestamp + minClaimHold > minClaimHold);
        }

        vm.deal(address(this), amount);

        vm.expectEmit(true, true, true, true);
        uint256 expiration = block.timestamp + minClaimHold;
        emit BondPosted(bondId, owner, expiration, amount);

        bm.post{value: amount}(bondId, owner, minClaimHold);

        // Validate the bond
        (address newFetchedOwner, uint256 fetchedExpiration, bytes32 fetchedBondId, uint256 bondAmount) = bm.bonds(bondId);
        assertEq(newFetchedOwner, owner);
        assertEq(fetchedExpiration, block.timestamp + minClaimHold);
        assertEq(fetchedBondId, bondId);
        assertEq(bondAmount, amount);
    }

    /// @notice The bond manager should revert if the bond at the given id is already posted.
    function testFuzz_post_duplicates_reverts(bytes32 bondId, address owner, uint256 minClaimHold, uint256 amount) public {
        vm.assume(owner != address(0));
        amount = amount / 2;
        vm.assume(amount != 0);
        unchecked {
            vm.assume(block.timestamp + minClaimHold > minClaimHold);
        }

        vm.deal(address(this), amount);
        bm.post{value: amount}(bondId, owner, minClaimHold);

        vm.deal(address(this), amount);
        vm.expectRevert("BondManager: BondId already posted.");
        bm.post{value: amount}(bondId, owner, minClaimHold);
    }

    /// @notice Posting with the zero address as the owner fails.
    function testFuzz_post_zeroAddres_reverts(bytes32 bondId, uint256 minClaimHold, uint256 amount) public {
        address owner = address(0);
        vm.deal(address(this), amount);
        vm.expectRevert("BondManager: Owner cannot be the zero address.");
        bm.post{value: amount}(bondId, owner, minClaimHold);
    }

    /// @notice Posting zero value bonds should revert.
    function testFuzz_post_zeroAddres_reverts(bytes32 bondId, address owner, uint256 minClaimHold) public {
        vm.assume(owner != address(0));
        uint256 amount = 0;
        vm.deal(address(this), amount);
        vm.expectRevert("BondManager: Value must be non-zero.");
        bm.post{value: amount}(bondId, owner, minClaimHold);
    }

    /// -------------------------------------------
    /// Test Bond Seizing
    /// -------------------------------------------

    // TODO: This will require mocking the DG in the DGF

    /// -------------------------------------------
    /// Test Bond Split Seizing
    /// -------------------------------------------

    // TODO: This will require mocking the DG in the DGF

    /// -------------------------------------------
    /// Test Bond Reclaiming
    /// -------------------------------------------

    /// @notice Bonds can be reclaimed after the specified amount of time.
    function testFuzz_reclaim_succeeds(bytes32 bondId, address owner, uint256 minClaimHold, uint256 amount) public {
        vm.assume(owner != address(0));
        vm.assume(owner != address(bm));
        vm.assume(owner != address(this));
        vm.assume(amount != 0);
        unchecked {
            vm.assume(block.timestamp + minClaimHold > minClaimHold);
        }

        // Post the bond
        vm.deal(address(this), amount);
        bm.post{value: amount}(bondId, owner, minClaimHold);

        // We can't claim if the block.timestamp is less than the bond expiration.
        (, uint256 expiration,,) = bm.bonds(bondId);
        if (expiration > block.timestamp) {
            vm.prank(owner);
            vm.expectRevert("BondManager: Bond isn't claimable yet.");
            bm.reclaim(bondId);
        }

        // Past expiration, the owner can reclaim
        vm.warp(expiration);
        vm.prank(owner);
        bm.reclaim(bondId);
        assertEq(owner.balance, amount);
    }
}
