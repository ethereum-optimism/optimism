// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

import "src/types/Types.sol";
import "src/types/Errors.sol";

import "forge-std/Test.sol";
import { DisputeGameFactory } from "src/DisputeGameFactory.sol";
import { IDisputeGame } from "src/interfaces/IDisputeGame.sol";
import { IBondManager } from "src/interfaces/IBondManager.sol";

import { BondManager } from "src/BondManager.sol";

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

    /// -------------------------------------------
    /// Test Bond Posting
    /// -------------------------------------------

    /// @notice Tests that posting a bond succeeds.
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

    /// @notice The bond manager should revert if the bond at the given id is already posted.
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

    /// @notice Posting with the zero address as the owner fails.
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

    /// @notice Posting zero value bonds should revert.
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

    /// -------------------------------------------
    /// Test Bond Seizing
    /// -------------------------------------------

    /// @notice Non-existing bonds shouldn't be seizable.
    function testFuzz_seize_missingBond_reverts(bytes32 bondId) public {
        vm.expectRevert("BondManager: The bond does not exist.");
        bm.seize(bondId);
    }

    /// @notice Bonds that expired cannot be seized.
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

    /// @notice Bonds cannot be seized by unauthorized parties.
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

    /// @notice Tests seizing a bond if the game resolves
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

    /// -------------------------------------------
    /// Test Bond Split Seizing
    /// -------------------------------------------

    /// @notice Tests seizing and splitting a bond if the game resolves
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

    /// -------------------------------------------
    /// Test Bond Reclaiming
    /// -------------------------------------------

    /// @notice Bonds can be reclaimed after the specified amount of time.
    function testFuzz_reclaim_succeeds(
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

/// @dev A mock dispute game for testing bond seizures.
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

    /// @dev Allow the contract to receive ether
    receive() external payable {}

    fallback() external payable {}

    /// @dev Resolve the game with a split
    function splitResolve() public {
        challengers = [address(1), address(2)];
        bm.seizeAndSplit(bondId, challengers);
    }

    /// -------------------------------------------
    /// IInitializable Functions
    /// -------------------------------------------

    function initialize() external {
        /* noop */
    }

    /// -------------------------------------------
    /// IVersioned Functions
    /// -------------------------------------------

    function version() external pure returns (string memory _version) {
        return "0.1.0";
    }

    /// -------------------------------------------
    /// IDisputeGame Functions
    /// -------------------------------------------

    /// @notice Returns the timestamp that the DisputeGame contract was created at.
    function createdAt() external pure override returns (Timestamp _createdAt) {
        return Timestamp.wrap(uint64(0));
    }

    /// @notice Returns the current status of the game.
    function status() external view override returns (GameStatus _status) {
        return gameStatus;
    }

    /// @notice Getter for the game type.
    /// @dev `clones-with-immutable-args` argument #1
    /// @dev The reference impl should be entirely different depending on the type (fault, validity)
    ///      i.e. The game type should indicate the security model.
    /// @return _gameType The type of proof system being used.
    function gameType() external pure returns (GameType _gameType) {
        return GameType.ATTESTATION;
    }

    /// @notice Getter for the root claim.
    /// @return _rootClaim The root claim of the DisputeGame.
    /// @dev `clones-with-immutable-args` argument #2
    function rootClaim() external view override returns (Claim _rootClaim) {
        return rc;
    }

    /// @notice Getter for the extra data.
    /// @dev `clones-with-immutable-args` argument #3
    /// @return _extraData Any extra data supplied to the dispute game contract by the creator.
    function extraData() external view returns (bytes memory _extraData) {
        return ed;
    }

    /// @notice Returns the address of the `BondManager` used
    function bondManager() external view override returns (IBondManager _bondManager) {
        return IBondManager(address(bm));
    }

    /// @notice If all necessary information has been gathered, this function should mark the game
    ///         status as either `CHALLENGER_WINS` or `DEFENDER_WINS` and return the status of
    ///         the resolved game. It is at this stage that the bonds should be awarded to the
    ///         necessary parties.
    /// @dev May only be called if the `status` is `IN_PROGRESS`.
    function resolve() external returns (GameStatus _status) {
        bm.seize(bondId);
        return gameStatus;
    }
}
