// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

import "forge-std/Test.sol";

import "../libraries/DisputeTypes.sol";

import { IDisputeGame } from "../dispute/IDisputeGame.sol";
import { IBondManager } from "../dispute/IBondManager.sol";

import { DisputeGameFactory } from "../dispute/DisputeGameFactory.sol";

import { BondManager } from "../dispute/BondManager.sol";

contract BondManager_Test is Test {
    DisputeGameFactory factory;
    BondManager bm;

    // DisputeGameFactory events
    event DisputeGameCreated(
        address indexed disputeProxy,
        GameType indexed gameType,
        Claim indexed rootClaim
    );

    // BondManager events
    event BondPosted(bytes32 bondId, address owner, uint256 expiration, uint256 amount);
    event BondSeized(bytes32 bondId, address owner, address seizer, uint256 amount);
    event BondReclaimed(bytes32 bondId, address claiment, uint256 amount);

    function setUp() public {
        factory = new DisputeGameFactory(address(this));
        bm = new BondManager(factory);
    }

    /**
     * -------------------------------------------
     * Test Bond Posting
     * -------------------------------------------
     */

    /**
     * @notice Tests that posting a bond succeeds.
     */
    function testFuzz_post_succeeds(
        bytes32 bondId,
        address owner,
        uint256 minClaimHold,
        uint256 amount
    ) public {
        vm.assume(owner != address(0));
        vm.assume(owner != address(bm));
        vm.assume(owner != address(this));
        // Create2Deployer
        vm.assume(owner != address(0x4e59b44847b379578588920cA78FbF26c0B4956C));
        vm.assume(amount != 0);
        unchecked {
            vm.assume(block.timestamp + minClaimHold > minClaimHold);
        }

        vm.deal(address(this), amount);

        vm.expectEmit(true, true, true, true);
        uint256 expiration = block.timestamp + minClaimHold;
        emit BondPosted(bondId, owner, expiration, amount);

        bm.post{ value: amount }(bondId, owner, minClaimHold);

        // Validate the bond
        (
            address newFetchedOwner,
            uint256 fetchedExpiration,
            bytes32 fetchedBondId,
            uint256 bondAmount
        ) = bm.bonds(bondId);
        assertEq(newFetchedOwner, owner);
        assertEq(fetchedExpiration, block.timestamp + minClaimHold);
        assertEq(fetchedBondId, bondId);
        assertEq(bondAmount, amount);
    }

    /**
     * @notice Tests that posting a bond with the same id twice reverts.
     */
    function testFuzz_post_duplicates_reverts(
        bytes32 bondId,
        address owner,
        uint256 minClaimHold,
        uint256 amount
    ) public {
        vm.assume(owner != address(0));
        amount = amount / 2;
        vm.assume(amount != 0);
        unchecked {
            vm.assume(block.timestamp + minClaimHold > minClaimHold);
        }

        vm.deal(address(this), amount);
        bm.post{ value: amount }(bondId, owner, minClaimHold);

        vm.deal(address(this), amount);
        vm.expectRevert("BondManager: BondId already posted.");
        bm.post{ value: amount }(bondId, owner, minClaimHold);
    }

    /**
     * @notice Posting with the zero address as the owner fails.
     */
    function testFuzz_post_zeroAddress_reverts(
        bytes32 bondId,
        uint256 minClaimHold,
        uint256 amount
    ) public {
        address owner = address(0);
        vm.deal(address(this), amount);
        vm.expectRevert("BondManager: Owner cannot be the zero address.");
        bm.post{ value: amount }(bondId, owner, minClaimHold);
    }

    /**
     * @notice Posting zero value bonds should revert.
     */
    function testFuzz_post_zeroAddress_reverts(
        bytes32 bondId,
        address owner,
        uint256 minClaimHold
    ) public {
        vm.assume(owner != address(0));
        uint256 amount = 0;
        vm.deal(address(this), amount);
        vm.expectRevert("BondManager: Value must be non-zero.");
        bm.post{ value: amount }(bondId, owner, minClaimHold);
    }

    /**
     * -------------------------------------------
     * Test Bond Seizing
     * -------------------------------------------
     */

    /**
     * @notice Non-existing bonds shouldn't be seizable.
     */
    function testFuzz_seize_missingBond_reverts(bytes32 bondId) public {
        vm.expectRevert("BondManager: The bond does not exist.");
        bm.seize(bondId);
    }

    /**
     * @notice Bonds that expired cannot be seized.
     */
    function testFuzz_seize_expired_reverts(
        bytes32 bondId,
        address owner,
        uint256 minClaimHold,
        uint256 amount
    ) public {
        vm.assume(owner != address(0));
        vm.assume(owner != address(bm));
        vm.assume(owner != address(this));
        vm.assume(amount != 0);
        unchecked {
            vm.assume(block.timestamp + minClaimHold + 1 > minClaimHold);
        }
        vm.deal(address(this), amount);
        bm.post{ value: amount }(bondId, owner, minClaimHold);

        vm.warp(block.timestamp + minClaimHold + 1);
        vm.expectRevert("BondManager: Bond expired.");
        bm.seize(bondId);
    }

    /**
     * @notice Bonds cannot be seized by unauthorized parties.
     */
    function testFuzz_seize_unauthorized_reverts(
        bytes32 bondId,
        address owner,
        uint256 minClaimHold,
        uint256 amount
    ) public {
        vm.assume(owner != address(0));
        vm.assume(owner != address(bm));
        vm.assume(owner != address(this));
        vm.assume(amount != 0);
        unchecked {
            vm.assume(block.timestamp + minClaimHold > minClaimHold);
        }
        vm.deal(address(this), amount);
        bm.post{ value: amount }(bondId, owner, minClaimHold);

        MockAttestationDisputeGame game = new MockAttestationDisputeGame();
        vm.prank(address(game));
        vm.expectRevert("BondManager: Unauthorized seizure.");
        bm.seize(bondId);
    }

    /**
     * @notice Seizing a bond should succeed if the game resolves.
     */
    function testFuzz_seize_succeeds(
        bytes32 bondId,
        uint256 minClaimHold,
        bytes calldata extraData
    ) public {
        unchecked {
            vm.assume(block.timestamp + minClaimHold > minClaimHold);
        }

        vm.deal(address(this), 1 ether);
        bm.post{ value: 1 ether }(bondId, address(0xba5ed), minClaimHold);

        // Create a mock dispute game in the factory
        IDisputeGame proxy;
        Claim rootClaim;
        bytes memory ed = extraData;
        {
            rootClaim = Claim.wrap(bytes32(""));
            MockAttestationDisputeGame implementation = new MockAttestationDisputeGame();
            GameType gt = GameType.ATTESTATION;
            factory.setImplementation(gt, IDisputeGame(address(implementation)));
            vm.expectEmit(false, true, true, false);
            emit DisputeGameCreated(address(0), gt, rootClaim);
            proxy = factory.create(gt, rootClaim, extraData);
            assertEq(address(factory.games(gt, rootClaim, extraData)), address(proxy));
        }

        // Update the game fields
        MockAttestationDisputeGame spawned = MockAttestationDisputeGame(payable(address(proxy)));
        spawned.setBondManager(bm);
        spawned.setRootClaim(rootClaim);
        spawned.setGameStatus(GameStatus.CHALLENGER_WINS);
        spawned.setBondId(bondId);
        spawned.setExtraData(ed);

        // Seize the bond by calling resolve
        vm.expectEmit(true, true, true, true);
        emit BondSeized(bondId, address(0xba5ed), address(spawned), 1 ether);
        spawned.resolve();
        assertEq(address(spawned).balance, 1 ether);

        // Validate that the bond was deleted
        (address newFetchedOwner, , , ) = bm.bonds(bondId);
        assertEq(newFetchedOwner, address(0));
    }

    /**
     * -------------------------------------------
     * Test Bond Split and Seizing
     * -------------------------------------------
     */

    /**
     * @notice Seizing and splitting a bond should succeed if the game resolves.
     */
    function testFuzz_seizeAndSplit_succeeds(
        bytes32 bondId,
        uint256 minClaimHold,
        bytes calldata extraData
    ) public {
        unchecked {
            vm.assume(block.timestamp + minClaimHold > minClaimHold);
        }

        vm.deal(address(this), 1 ether);
        bm.post{ value: 1 ether }(bondId, address(0xba5ed), minClaimHold);

        // Create a mock dispute game in the factory
        IDisputeGame proxy;
        Claim rootClaim;
        bytes memory ed = extraData;
        {
            rootClaim = Claim.wrap(bytes32(""));
            MockAttestationDisputeGame implementation = new MockAttestationDisputeGame();
            GameType gt = GameType.ATTESTATION;
            factory.setImplementation(gt, IDisputeGame(address(implementation)));
            vm.expectEmit(false, true, true, false);
            emit DisputeGameCreated(address(0), gt, rootClaim);
            proxy = factory.create(gt, rootClaim, extraData);
            assertEq(address(factory.games(gt, rootClaim, extraData)), address(proxy));
        }

        // Update the game fields
        MockAttestationDisputeGame spawned = MockAttestationDisputeGame(payable(address(proxy)));
        spawned.setBondManager(bm);
        spawned.setRootClaim(rootClaim);
        spawned.setGameStatus(GameStatus.CHALLENGER_WINS);
        spawned.setBondId(bondId);
        spawned.setExtraData(ed);

        // Seize the bond by calling resolve
        vm.expectEmit(true, true, true, true);
        emit BondSeized(bondId, address(0xba5ed), address(spawned), 1 ether);
        spawned.splitResolve();
        assertEq(address(spawned).balance, 0);
        address[] memory challengers = spawned.getChallengers();
        uint256 proportionalAmount = 1 ether / challengers.length;
        for (uint256 i = 0; i < challengers.length; i++) {
            assertEq(address(challengers[i]).balance, proportionalAmount);
        }

        // Validate that the bond was deleted
        (address newFetchedOwner, , , ) = bm.bonds(bondId);
        assertEq(newFetchedOwner, address(0));
    }

    /**
     * -------------------------------------------
     * Test Bond Reclaiming
     * -------------------------------------------
     */

    /**
     * @notice Bonds can be reclaimed after the specified amount of time.
     */
    function testFuzz_reclaim_succeeds(
        bytes32 bondId,
        address owner,
        uint256 minClaimHold,
        uint256 amount
    ) public {
        vm.assume(owner != address(factory));
        vm.assume(owner != address(bm));
        vm.assume(owner != address(this));
        vm.assume(owner != address(0));
        vm.assume(owner.code.length == 0);
        vm.assume(amount != 0);
        unchecked {
            vm.assume(block.timestamp + minClaimHold > minClaimHold);
        }
        assumeNoPrecompiles(owner);

        // Post the bond
        vm.deal(address(this), amount);
        bm.post{ value: amount }(bondId, owner, minClaimHold);

        // We can't claim if the block.timestamp is less than the bond expiration.
        (, uint256 expiration, , ) = bm.bonds(bondId);
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

/**
 * @title MockAttestationDisputeGame
 * @dev A mock dispute game for testing bond seizures.
 */
contract MockAttestationDisputeGame is IDisputeGame {
    GameStatus internal gameStatus;
    BondManager bm;
    Claim internal rc;
    bytes internal ed;
    bytes32 internal bondId;

    address[] internal challengers;

    function getChallengers() public view returns (address[] memory) {
        return challengers;
    }

    function setBondId(bytes32 bid) external {
        bondId = bid;
    }

    function setBondManager(BondManager _bm) external {
        bm = _bm;
    }

    function setGameStatus(GameStatus _gs) external {
        gameStatus = _gs;
    }

    function setRootClaim(Claim _rc) external {
        rc = _rc;
    }

    function setExtraData(bytes memory _ed) external {
        ed = _ed;
    }

    receive() external payable {}

    fallback() external payable {}

    function splitResolve() public {
        challengers = [address(1), address(2)];
        bm.seizeAndSplit(bondId, challengers);
    }

    /**
     * -------------------------------------------
     * Initializable Functions
     * -------------------------------------------
     */

    function initialize() external {
        /* noop */
    }

    /**
     * -------------------------------------------
     * IVersioned Functions
     * -------------------------------------------
     */

    function version() external pure returns (string memory _version) {
        return "0.1.0";
    }

    /**
     * -------------------------------------------
     * IDisputeGame Functions
     * -------------------------------------------
     */

    function createdAt() external pure override returns (Timestamp _createdAt) {
        return Timestamp.wrap(uint64(0));
    }

    function status() external view override returns (GameStatus _status) {
        return gameStatus;
    }

    function gameType() external pure returns (GameType _gameType) {
        return GameType.ATTESTATION;
    }

    function rootClaim() external view override returns (Claim _rootClaim) {
        return rc;
    }

    function extraData() external view returns (bytes memory _extraData) {
        return ed;
    }

    function bondManager() external view override returns (IBondManager _bondManager) {
        return IBondManager(address(bm));
    }

    function resolve() external returns (GameStatus _status) {
        bm.seize(bondId);
        return gameStatus;
    }
}
