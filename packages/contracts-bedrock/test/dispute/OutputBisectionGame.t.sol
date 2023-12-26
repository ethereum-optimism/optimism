// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

import { Test } from "forge-std/Test.sol";
import { Vm } from "forge-std/Vm.sol";
import { DisputeGameFactory_Init } from "test/dispute/DisputeGameFactory.t.sol";
import { DisputeGameFactory } from "src/dispute/DisputeGameFactory.sol";
import { OutputBisectionGame } from "src/dispute/OutputBisectionGame.sol";
import { L2OutputOracle } from "src/L1/L2OutputOracle.sol";
import { BlockOracle } from "src/dispute/BlockOracle.sol";
import { PreimageOracle } from "src/cannon/PreimageOracle.sol";
import { PreimageKeyLib } from "src/cannon/PreimageKeyLib.sol";

import "src/libraries/DisputeTypes.sol";
import "src/libraries/DisputeErrors.sol";
import { Types } from "src/libraries/Types.sol";
import { LibClock } from "src/dispute/lib/LibClock.sol";
import { LibPosition } from "src/dispute/lib/LibPosition.sol";
import { IBigStepper, IPreimageOracle } from "src/dispute/interfaces/IBigStepper.sol";
import { AlphabetVM2 } from "test/mocks/AlphabetVM2.sol";

import { DisputeActor, HonestDisputeActor } from "test/actors/OutputBisectionActors.sol";

contract OutputBisectionGame_Init is DisputeGameFactory_Init {
    /// @dev The type of the game being tested.
    GameType internal constant GAME_TYPE = GameType.wrap(0);

    /// @dev The implementation of the game.
    OutputBisectionGame internal gameImpl;
    /// @dev The `Clone` proxy of the game.
    OutputBisectionGame internal gameProxy;

    /// @dev The extra data passed to the game for initialization.
    bytes internal extraData;

    event Move(uint256 indexed parentIndex, Claim indexed pivot, address indexed claimant);

    function init(
        Claim rootClaim,
        Claim absolutePrestate,
        uint256 l2BlockNumber,
        uint256 genesisBlockNumber,
        Hash genesisOutputRoot
    )
        public
    {
        // Set the time to a realistic date.
        vm.warp(1690906994);

        // Set the extra data for the game creation
        extraData = abi.encode(l2BlockNumber);

        AlphabetVM2 _vm = new AlphabetVM2(absolutePrestate);

        // Deploy an implementation of the fault game
        gameImpl = new OutputBisectionGame({
            _gameType: GAME_TYPE,
            _absolutePrestate: absolutePrestate,
            _genesisBlockNumber: genesisBlockNumber,
            _genesisOutputRoot: genesisOutputRoot,
            _maxGameDepth: 2 ** 3,
            _splitDepth: 2 ** 2,
            _gameDuration: Duration.wrap(7 days),
            _vm: _vm
        });
        // Register the game implementation with the factory.
        factory.setImplementation(GAME_TYPE, gameImpl);
        // Create a new game.
        gameProxy = OutputBisectionGame(address(factory.create(GAME_TYPE, rootClaim, extraData)));

        // Check immutables
        assertEq(GameType.unwrap(gameProxy.gameType()), GameType.unwrap(GAME_TYPE));
        assertEq(Claim.unwrap(gameProxy.absolutePrestate()), Claim.unwrap(absolutePrestate));
        assertEq(gameProxy.genesisBlockNumber(), genesisBlockNumber);
        assertEq(Hash.unwrap(gameProxy.genesisOutputRoot()), Hash.unwrap(genesisOutputRoot));
        assertEq(gameProxy.maxGameDepth(), 2 ** 3);
        assertEq(gameProxy.splitDepth(), 2 ** 2);
        assertEq(Duration.unwrap(gameProxy.gameDuration()), 7 days);
        assertEq(address(gameProxy.vm()), address(_vm));

        // Label the proxy
        vm.label(address(gameProxy), "OutputBisectionGame_Clone");
    }
}

contract OutputBisectionGame_Test is OutputBisectionGame_Init {
    /// @dev The root claim of the game.
    Claim internal constant ROOT_CLAIM = Claim.wrap(bytes32((uint256(1) << 248) | uint256(10)));
    /// @dev The absolute prestate of the trace.
    Claim internal constant ABSOLUTE_PRESTATE = Claim.wrap(bytes32((uint256(3) << 248) | uint256(0)));

    function setUp() public override {
        super.setUp();
        super.init({
            rootClaim: ROOT_CLAIM,
            absolutePrestate: ABSOLUTE_PRESTATE,
            l2BlockNumber: 0x10,
            genesisBlockNumber: 0,
            genesisOutputRoot: Hash.wrap(bytes32(0))
        });
    }

    ////////////////////////////////////////////////////////////////
    //            `IDisputeGame` Implementation Tests             //
    ////////////////////////////////////////////////////////////////

    /// @dev Tests that the constructor of the `OutputBisectionGame` reverts when the `_splitDepth`
    ///      parameter is greater than or equal to the `MAX_GAME_DEPTH`
    function test_constructor_wrongArgs_reverts(uint256 _splitDepth) public {
        AlphabetVM2 alphabetVM = new AlphabetVM2(ABSOLUTE_PRESTATE);

        // Test that the constructor reverts when the `_splitDepth` parameter is greater than or equal
        // to the `MAX_GAME_DEPTH` parameter.
        _splitDepth = bound(_splitDepth, 2 ** 3, type(uint256).max);
        vm.expectRevert(InvalidSplitDepth.selector);
        new OutputBisectionGame({
            _gameType: GAME_TYPE,
            _absolutePrestate: ABSOLUTE_PRESTATE,
            _genesisBlockNumber: 0,
            _genesisOutputRoot: Hash.wrap(bytes32(0)),
            _maxGameDepth: 2 ** 3,
            _splitDepth: _splitDepth,
            _gameDuration: Duration.wrap(7 days),
            _vm: alphabetVM
        });
    }

    /// @dev Tests that the game's root claim is set correctly.
    function test_rootClaim_succeeds() public {
        assertEq(Claim.unwrap(gameProxy.rootClaim()), Claim.unwrap(ROOT_CLAIM));
    }

    /// @dev Tests that the game's extra data is set correctly.
    function test_extraData_succeeds() public {
        assertEq(gameProxy.extraData(), extraData);
    }

    /// @dev Tests that the game's starting timestamp is set correctly.
    function test_createdAt_succeeds() public {
        assertEq(Timestamp.unwrap(gameProxy.createdAt()), block.timestamp);
    }

    /// @dev Tests that the game's type is set correctly.
    function test_gameType_succeeds() public {
        assertEq(GameType.unwrap(gameProxy.gameType()), GameType.unwrap(GAME_TYPE));
    }

    /// @dev Tests that the game's data is set correctly.
    function test_gameData_succeeds() public {
        (GameType gameType, Claim rootClaim, bytes memory _extraData) = gameProxy.gameData();

        assertEq(GameType.unwrap(gameType), GameType.unwrap(GAME_TYPE));
        assertEq(Claim.unwrap(rootClaim), Claim.unwrap(ROOT_CLAIM));
        assertEq(_extraData, extraData);
    }

    ////////////////////////////////////////////////////////////////
    //          `IOutputBisectionGame` Implementation Tests       //
    ////////////////////////////////////////////////////////////////

    /// @dev Tests that the game cannot be initialized with an output root that commits to <= the configured genesis
    ///      block number
    function testFuzz_initialize_cannotProposeGenesis_reverts(uint256 _blockNumber) public {
        _blockNumber = bound(_blockNumber, 0, gameProxy.genesisBlockNumber());

        Claim claim = _dummyClaim();
        vm.expectRevert(abi.encodeWithSelector(UnexpectedRootClaim.selector, claim));
        gameProxy = OutputBisectionGame(address(factory.create(GAME_TYPE, claim, abi.encode(_blockNumber))));
    }

    /// @dev Tests that the game cannot be initialized with extra data > 64 bytes long (root claim + l2 block number
    ///      concatenated)
    function testFuzz_initialize_extraDataTooLong_reverts(uint256 _extraDataLen) public {
        // The `DisputeGameFactory` will pack the root claim and the extra data into a single array, which is enforced
        // to be at least 64 bytes long.
        // We bound the upper end to 23.5KB to ensure that the minimal proxy never surpasses the contract size limit
        // in this test, as CWIA proxies store the immutable args in their bytecode.
        // [33 bytes, 23.5 KB]
        _extraDataLen = bound(_extraDataLen, 33, 23_500);
        bytes memory _extraData = new bytes(_extraDataLen);

        // Assign the first 32 bytes in `extraData` to a valid L2 block number passed genesis.
        uint256 genesisBlockNumber = gameProxy.genesisBlockNumber();
        assembly {
            mstore(add(_extraData, 0x20), add(genesisBlockNumber, 1))
        }

        Claim claim = _dummyClaim();
        vm.expectRevert(abi.encodeWithSelector(ExtraDataTooLong.selector));
        gameProxy = OutputBisectionGame(address(factory.create(GAME_TYPE, claim, _extraData)));
    }

    /// @dev Tests that the game is initialized with the correct data.
    function test_initialize_correctData_succeeds() public {
        // Assert that the root claim is initialized correctly.
        (uint32 parentIndex, bool countered, Claim claim, Position position, Clock clock) = gameProxy.claimData(0);
        assertEq(parentIndex, type(uint32).max);
        assertEq(countered, false);
        assertEq(Claim.unwrap(claim), Claim.unwrap(ROOT_CLAIM));
        assertEq(Position.unwrap(position), 1);
        assertEq(
            Clock.unwrap(clock), Clock.unwrap(LibClock.wrap(Duration.wrap(0), Timestamp.wrap(uint64(block.timestamp))))
        );

        // Assert that the `createdAt` timestamp is correct.
        assertEq(Timestamp.unwrap(gameProxy.createdAt()), block.timestamp);

        // Assert that the blockhash provided is correct.
        assertEq(Hash.unwrap(gameProxy.l1Head()), blockhash(block.number - 1));
    }

    /// @dev Tests that a move while the game status is not `IN_PROGRESS` causes the call to revert
    ///      with the `GameNotInProgress` error
    function test_move_gameNotInProgress_reverts() public {
        uint256 chalWins = uint256(GameStatus.CHALLENGER_WINS);

        // Replace the game status in storage. It exists in slot 0 at offset 16.
        uint256 slot = uint256(vm.load(address(gameProxy), bytes32(0)));
        uint256 offset = 16 << 3;
        uint256 mask = 0xFF << offset;
        // Replace the byte in the slot value with the challenger wins status.
        slot = (slot & ~mask) | (chalWins << offset);
        vm.store(address(gameProxy), bytes32(0), bytes32(slot));

        // Ensure that the game status was properly updated.
        GameStatus status = gameProxy.status();
        assertEq(uint256(status), chalWins);

        // Attempt to make a move. Should revert.
        vm.expectRevert(GameNotInProgress.selector);
        gameProxy.attack(0, Claim.wrap(0));
    }

    /// @dev Tests that an attempt to defend the root claim reverts with the `CannotDefendRootClaim` error.
    function test_move_defendRoot_reverts() public {
        vm.expectRevert(CannotDefendRootClaim.selector);
        gameProxy.defend(0, _dummyClaim());
    }

    /// @dev Tests that an attempt to move against a claim that does not exist reverts with the
    ///      `ParentDoesNotExist` error.
    function test_move_nonExistentParent_reverts() public {
        Claim claim = _dummyClaim();

        // Expect an out of bounds revert for an attack
        vm.expectRevert(abi.encodeWithSignature("Panic(uint256)", 0x32));
        gameProxy.attack(1, claim);

        // Expect an out of bounds revert for a defense
        vm.expectRevert(abi.encodeWithSignature("Panic(uint256)", 0x32));
        gameProxy.defend(1, claim);
    }

    /// @dev Tests that an attempt to move at the maximum game depth reverts with the
    ///      `GameDepthExceeded` error.
    function test_move_gameDepthExceeded_reverts() public {
        Claim claim = _changeClaimStatus(_dummyClaim(), VMStatuses.PANIC);

        uint256 maxDepth = gameProxy.maxGameDepth();

        for (uint256 i = 0; i <= maxDepth; i++) {
            // At the max game depth, the `_move` function should revert with
            // the `GameDepthExceeded` error.
            if (i == maxDepth) {
                vm.expectRevert(GameDepthExceeded.selector);
            }
            gameProxy.attack(i, claim);
        }
    }

    /// @dev Tests that a move made after the clock time has exceeded reverts with the
    ///      `ClockTimeExceeded` error.
    function test_move_clockTimeExceeded_reverts() public {
        // Warp ahead past the clock time for the first move (3 1/2 days)
        vm.warp(block.timestamp + 3 days + 12 hours + 1);
        vm.expectRevert(ClockTimeExceeded.selector);
        gameProxy.attack(0, _dummyClaim());
    }

    /// @notice Static unit test for the correctness of the chess clock incrementation.
    function test_move_clockCorrectness_succeeds() public {
        (,,,, Clock clock) = gameProxy.claimData(0);
        assertEq(
            Clock.unwrap(clock), Clock.unwrap(LibClock.wrap(Duration.wrap(0), Timestamp.wrap(uint64(block.timestamp))))
        );

        Claim claim = _dummyClaim();

        vm.warp(block.timestamp + 15);
        gameProxy.attack(0, claim);
        (,,,, clock) = gameProxy.claimData(1);
        assertEq(
            Clock.unwrap(clock), Clock.unwrap(LibClock.wrap(Duration.wrap(15), Timestamp.wrap(uint64(block.timestamp))))
        );

        vm.warp(block.timestamp + 10);
        gameProxy.attack(1, claim);
        (,,,, clock) = gameProxy.claimData(2);
        assertEq(
            Clock.unwrap(clock), Clock.unwrap(LibClock.wrap(Duration.wrap(10), Timestamp.wrap(uint64(block.timestamp))))
        );

        // We are at the split depth, so we need to set the status byte of the claim
        // for the next move.
        claim = _changeClaimStatus(claim, VMStatuses.PANIC);

        vm.warp(block.timestamp + 10);
        gameProxy.attack(2, claim);
        (,,,, clock) = gameProxy.claimData(3);
        assertEq(
            Clock.unwrap(clock), Clock.unwrap(LibClock.wrap(Duration.wrap(25), Timestamp.wrap(uint64(block.timestamp))))
        );

        vm.warp(block.timestamp + 10);
        gameProxy.attack(3, claim);
        (,,,, clock) = gameProxy.claimData(4);
        assertEq(
            Clock.unwrap(clock), Clock.unwrap(LibClock.wrap(Duration.wrap(20), Timestamp.wrap(uint64(block.timestamp))))
        );
    }

    /// @dev Tests that an identical claim cannot be made twice. The duplicate claim attempt should
    ///      revert with the `ClaimAlreadyExists` error.
    function test_move_duplicateClaim_reverts() public {
        Claim claim = _dummyClaim();

        // Make the first move. This should succeed.
        gameProxy.attack(0, claim);

        // Attempt to make the same move again.
        vm.expectRevert(ClaimAlreadyExists.selector);
        gameProxy.attack(0, claim);
    }

    /// @dev Static unit test asserting that identical claims at the same position can be made in different subgames.
    function test_move_duplicateClaimsDifferentSubgames_succeeds() public {
        Claim claimA = _dummyClaim();
        Claim claimB = _dummyClaim();

        // Make the first moves. This should succeed.
        gameProxy.attack(0, claimA);
        gameProxy.attack(0, claimB);

        // Perform an attack at the same position with the same claim value in both subgames.
        // These both should succeed.
        gameProxy.attack(1, claimA);
        gameProxy.attack(2, claimA);
    }

    /// @dev Static unit test for the correctness of an opening attack.
    function test_move_simpleAttack_succeeds() public {
        // Warp ahead 5 seconds.
        vm.warp(block.timestamp + 5);

        Claim counter = _dummyClaim();

        // Perform the attack.
        vm.expectEmit(true, true, true, false);
        emit Move(0, counter, address(this));
        gameProxy.attack(0, counter);

        // Grab the claim data of the attack.
        (uint32 parentIndex, bool countered, Claim claim, Position position, Clock clock) = gameProxy.claimData(1);

        // Assert correctness of the attack claim's data.
        assertEq(parentIndex, 0);
        assertEq(countered, false);
        assertEq(Claim.unwrap(claim), Claim.unwrap(counter));
        assertEq(Position.unwrap(position), Position.unwrap(Position.wrap(1).move(true)));
        assertEq(
            Clock.unwrap(clock), Clock.unwrap(LibClock.wrap(Duration.wrap(5), Timestamp.wrap(uint64(block.timestamp))))
        );

        // Grab the claim data of the parent.
        (parentIndex, countered, claim, position, clock) = gameProxy.claimData(0);

        // Assert correctness of the parent claim's data.
        assertEq(parentIndex, type(uint32).max);
        assertEq(countered, true);
        assertEq(Claim.unwrap(claim), Claim.unwrap(ROOT_CLAIM));
        assertEq(Position.unwrap(position), 1);
        assertEq(
            Clock.unwrap(clock),
            Clock.unwrap(LibClock.wrap(Duration.wrap(0), Timestamp.wrap(uint64(block.timestamp - 5))))
        );
    }

    /// @dev Tests that making a claim at the execution trace bisection root level with an invalid status
    ///      byte reverts with the `UnexpectedRootClaim` error.
    function test_move_incorrectStatusExecRoot_reverts() public {
        for (uint256 i; i < 4; i++) {
            gameProxy.attack(i, _dummyClaim());
        }

        vm.expectRevert(abi.encodeWithSelector(UnexpectedRootClaim.selector, bytes32(0)));
        gameProxy.attack(4, Claim.wrap(bytes32(0)));
    }

    /// @dev Tests that making a claim at the execution trace bisection root level with a valid status
    ///      byte succeeds.
    function test_move_correctStatusExecRoot_succeeds() public {
        for (uint256 i; i < 4; i++) {
            gameProxy.attack(i, _dummyClaim());
        }
        gameProxy.attack(4, _changeClaimStatus(_dummyClaim(), VMStatuses.PANIC));
    }

    /// @dev Static unit test for the correctness an uncontested root resolution.
    function test_resolve_rootUncontested_succeeds() public {
        vm.warp(block.timestamp + 3 days + 12 hours + 1 seconds);
        gameProxy.resolveClaim(0);
        assertEq(uint8(gameProxy.resolve()), uint8(GameStatus.DEFENDER_WINS));
    }

    /// @dev Static unit test for the correctness an uncontested root resolution.
    function test_resolve_rootUncontestedClockNotExpired_succeeds() public {
        vm.warp(block.timestamp + 3 days + 12 hours);
        vm.expectRevert(ClockNotExpired.selector);
        gameProxy.resolveClaim(0);
    }

    /// @dev Static unit test asserting that resolve reverts when the absolute root
    ///      subgame has not been resolved.
    function test_resolve_rootUncontestedButUnresolved_reverts() public {
        vm.warp(block.timestamp + 3 days + 12 hours + 1 seconds);
        vm.expectRevert(OutOfOrderResolution.selector);
        gameProxy.resolve();
    }

    /// @dev Static unit test asserting that resolve reverts when the game state is
    ///      not in progress.
    function test_resolve_notInProgress_reverts() public {
        uint256 chalWins = uint256(GameStatus.CHALLENGER_WINS);

        // Replace the game status in storage. It exists in slot 0 at offset 16.
        uint256 slot = uint256(vm.load(address(gameProxy), bytes32(0)));
        uint256 offset = 16 << 3;
        uint256 mask = 0xFF << offset;
        // Replace the byte in the slot value with the challenger wins status.
        slot = (slot & ~mask) | (chalWins << offset);

        vm.store(address(gameProxy), bytes32(uint256(0)), bytes32(slot));
        vm.expectRevert(GameNotInProgress.selector);
        gameProxy.resolveClaim(0);
    }

    /// @dev Static unit test for the correctness of resolving a single attack game state.
    function test_resolve_rootContested_succeeds() public {
        gameProxy.attack(0, _dummyClaim());

        vm.warp(block.timestamp + 3 days + 12 hours + 1 seconds);

        gameProxy.resolveClaim(0);
        assertEq(uint8(gameProxy.resolve()), uint8(GameStatus.CHALLENGER_WINS));
    }

    /// @dev Static unit test for the correctness of resolving a game with a contested challenge claim.
    function test_resolve_challengeContested_succeeds() public {
        gameProxy.attack(0, _dummyClaim());
        gameProxy.defend(1, _dummyClaim());

        vm.warp(block.timestamp + 3 days + 12 hours + 1 seconds);

        gameProxy.resolveClaim(1);
        gameProxy.resolveClaim(0);
        assertEq(uint8(gameProxy.resolve()), uint8(GameStatus.DEFENDER_WINS));
    }

    /// @dev Static unit test for the correctness of resolving a game with multiplayer moves.
    function test_resolve_teamDeathmatch_succeeds() public {
        gameProxy.attack(0, _dummyClaim());
        gameProxy.attack(0, _dummyClaim());
        gameProxy.defend(1, _dummyClaim());
        gameProxy.defend(1, _dummyClaim());

        vm.warp(block.timestamp + 3 days + 12 hours + 1 seconds);

        gameProxy.resolveClaim(1);
        gameProxy.resolveClaim(0);
        assertEq(uint8(gameProxy.resolve()), uint8(GameStatus.CHALLENGER_WINS));
    }

    /// @dev Static unit test for the correctness of resolving a game that reaches max game depth.
    function test_resolve_stepReached_succeeds() public {
        Claim claim = _dummyClaim();
        for (uint256 i; i < gameProxy.splitDepth(); i++) {
            gameProxy.attack(i, claim);
        }

        claim = _changeClaimStatus(claim, VMStatuses.PANIC);
        for (uint256 i = gameProxy.claimDataLen() - 1; i < gameProxy.maxGameDepth(); i++) {
            gameProxy.attack(i, claim);
        }

        vm.warp(block.timestamp + 3 days + 12 hours + 1 seconds);

        // resolving claim at 8 isn't necessary
        for (uint256 i = 8; i > 0; i--) {
            gameProxy.resolveClaim(i - 1);
        }
        assertEq(uint8(gameProxy.resolve()), uint8(GameStatus.DEFENDER_WINS));
    }

    /// @dev Static unit test asserting that resolve reverts when attempting to resolve a subgame multiple times
    function test_resolve_claimAlreadyResolved_reverts() public {
        Claim claim = _dummyClaim();
        gameProxy.attack(0, claim);
        gameProxy.attack(1, claim);

        vm.warp(block.timestamp + 3 days + 12 hours + 1 seconds);

        gameProxy.resolveClaim(1);
        vm.expectRevert(ClaimAlreadyResolved.selector);
        gameProxy.resolveClaim(1);
    }

    /// @dev Static unit test asserting that resolve reverts when attempting to resolve a subgame at max depth
    function test_resolve_claimAtMaxDepthAlreadyResolved_reverts() public {
        Claim claim = _dummyClaim();
        for (uint256 i; i < gameProxy.splitDepth(); i++) {
            gameProxy.attack(i, claim);
        }

        claim = _changeClaimStatus(claim, VMStatuses.PANIC);
        for (uint256 i = gameProxy.claimDataLen() - 1; i < gameProxy.maxGameDepth(); i++) {
            gameProxy.attack(i, claim);
        }

        vm.warp(block.timestamp + 3 days + 12 hours + 1 seconds);

        vm.expectRevert(ClaimAlreadyResolved.selector);
        gameProxy.resolveClaim(8);
    }

    /// @dev Static unit test asserting that resolve reverts when attempting to resolve subgames out of order
    function test_resolve_outOfOrderResolution_reverts() public {
        gameProxy.attack(0, _dummyClaim());
        gameProxy.attack(1, _dummyClaim());

        vm.warp(block.timestamp + 3 days + 12 hours + 1 seconds);

        vm.expectRevert(OutOfOrderResolution.selector);
        gameProxy.resolveClaim(0);
    }

    /// @dev Tests that adding local data with an out of bounds identifier reverts.
    function testFuzz_addLocalData_oob_reverts(uint256 _ident) public {
        // Get a claim below the split depth so that we can add local data for an execution trace subgame.
        for (uint256 i; i < 4; i++) {
            gameProxy.attack(i, _dummyClaim());
        }
        gameProxy.attack(4, _changeClaimStatus(_dummyClaim(), VMStatuses.PANIC));

        // [1, 5] are valid local data identifiers.
        if (_ident <= 5) _ident = 0;

        vm.expectRevert(InvalidLocalIdent.selector);
        gameProxy.addLocalData(_ident, 5, 0);
    }

    /// @dev Tests that local data is loaded into the preimage oracle correctly in the subgame
    ///      that is disputing the transition from `GENESIS -> GENESIS + 1`
    function test_addLocalDataGenesisTransition_static_succeeds() public {
        IPreimageOracle oracle = IPreimageOracle(address(gameProxy.vm().oracle()));

        // Get a claim below the split depth so that we can add local data for an execution trace subgame.
        for (uint256 i; i < 4; i++) {
            gameProxy.attack(i, Claim.wrap(bytes32(i)));
        }
        gameProxy.attack(4, _changeClaimStatus(_dummyClaim(), VMStatuses.PANIC));

        // Expected start/disputed claims
        bytes32 startingClaim = Hash.unwrap(gameProxy.genesisOutputRoot());
        bytes32 disputedClaim = bytes32(uint256(3));
        Position disputedPos = LibPosition.wrap(4, 0);

        // Expected local data
        bytes32[5] memory data =
            [Hash.unwrap(gameProxy.l1Head()), startingClaim, disputedClaim, bytes32(0), bytes32(block.chainid << 0xC0)];

        for (uint256 i = 1; i <= 5; i++) {
            uint256 expectedLen = i > 3 ? 8 : 32;
            bytes32 key = _getKey(i, keccak256(abi.encode(disputedClaim, disputedPos)));

            gameProxy.addLocalData(i, 5, 0);
            (bytes32 dat, uint256 datLen) = oracle.readPreimage(key, 0);
            assertEq(dat >> 0xC0, bytes32(expectedLen));
            // Account for the length prefix if i > 3 (the data stored
            // at identifiers i <= 3 are 32 bytes long, so the expected
            // length is already correct. If i > 3, the data is only 8
            // bytes long, so the length prefix + the data is 16 bytes
            // total.)
            assertEq(datLen, expectedLen + (i > 3 ? 8 : 0));

            gameProxy.addLocalData(i, 5, 8);
            (dat, datLen) = oracle.readPreimage(key, 8);
            assertEq(dat, data[i - 1]);
            assertEq(datLen, expectedLen);
        }
    }

    /// @dev Tests that local data is loaded into the preimage oracle correctly.
    function test_addLocalDataMiddle_static_succeeds() public {
        IPreimageOracle oracle = IPreimageOracle(address(gameProxy.vm().oracle()));

        // Get a claim below the split depth so that we can add local data for an execution trace subgame.
        for (uint256 i; i < 4; i++) {
            gameProxy.attack(i, Claim.wrap(bytes32(i)));
        }
        gameProxy.defend(4, _changeClaimStatus(ROOT_CLAIM, VMStatuses.VALID));

        // Expected start/disputed claims
        bytes32 startingClaim = bytes32(uint256(3));
        Position startingPos = LibPosition.wrap(4, 0);
        bytes32 disputedClaim = bytes32(uint256(2));
        Position disputedPos = LibPosition.wrap(3, 0);

        // Expected local data
        bytes32[5] memory data = [
            Hash.unwrap(gameProxy.l1Head()),
            startingClaim,
            disputedClaim,
            bytes32(uint256(1) << 0xC0),
            bytes32(block.chainid << 0xC0)
        ];

        for (uint256 i = 1; i <= 5; i++) {
            uint256 expectedLen = i > 3 ? 8 : 32;
            bytes32 key = _getKey(i, keccak256(abi.encode(startingClaim, startingPos, disputedClaim, disputedPos)));

            gameProxy.addLocalData(i, 5, 0);
            (bytes32 dat, uint256 datLen) = oracle.readPreimage(key, 0);
            assertEq(dat >> 0xC0, bytes32(expectedLen));
            // Account for the length prefix if i > 3 (the data stored
            // at identifiers i <= 3 are 32 bytes long, so the expected
            // length is already correct. If i > 3, the data is only 8
            // bytes long, so the length prefix + the data is 16 bytes
            // total.)
            assertEq(datLen, expectedLen + (i > 3 ? 8 : 0));

            gameProxy.addLocalData(i, 5, 8);
            (dat, datLen) = oracle.readPreimage(key, 8);
            assertEq(dat, data[i - 1]);
            assertEq(datLen, expectedLen);
        }
    }

    /// @dev Helper to return a pseudo-random claim
    function _dummyClaim() internal view returns (Claim) {
        return Claim.wrap(keccak256(abi.encode(gasleft())));
    }

    /// @dev Helper to get the localized key for an identifier in the context of the game proxy.
    function _getKey(uint256 _ident, bytes32 _localContext) internal view returns (bytes32) {
        bytes32 h = keccak256(abi.encode(_ident | (1 << 248), address(gameProxy), _localContext));
        return bytes32((uint256(h) & ~uint256(0xFF << 248)) | (1 << 248));
    }
}

contract OutputBisection_1v1_Actors_Test is OutputBisectionGame_Init {
    /// @dev The honest actor
    DisputeActor internal honest;
    /// @dev The dishonest actor
    DisputeActor internal dishonest;

    function setUp() public override {
        // Setup the `OutputBisectionGame`
        super.setUp();
    }

    /// @notice Fuzz test for a 1v1 output bisection dispute.
    /// @dev The alphabet game has a constant status byte, and is not safe from someone being dishonest in
    ///      output bisection and then posting a correct execution trace bisection root claim. This test
    ///      does not cover this case (i.e. root claim of output bisection is dishonest, root claim of
    ///      execution trace bisection is made by the dishonest actor but is honest, honest actor cannot
    ///      attack it without risk of losing).
    function testFuzz_outputBisection1v1honestRoot_succeeds(uint8 _divergeOutput, uint8 _divergeStep) public {
        uint256[] memory honestL2Outputs = new uint256[](16);
        for (uint256 i; i < honestL2Outputs.length; i++) {
            honestL2Outputs[i] = i + 1;
        }
        bytes memory honestTrace = new bytes(256);
        for (uint256 i; i < honestTrace.length; i++) {
            honestTrace[i] = bytes1(uint8(i));
        }

        uint256 divergeAtOutput = bound(_divergeOutput, 0, 15);
        uint256 divergeAtStep = bound(_divergeStep, 0, 7);
        uint256 divergeStepOffset = (divergeAtOutput << 4) + divergeAtStep;

        uint256[] memory dishonestL2Outputs = new uint256[](16);
        for (uint256 i; i < dishonestL2Outputs.length; i++) {
            dishonestL2Outputs[i] = i >= divergeAtOutput ? 0xFF : i + 1;
        }
        bytes memory dishonestTrace = new bytes(256);
        for (uint256 i; i < dishonestTrace.length; i++) {
            dishonestTrace[i] = i >= divergeStepOffset ? bytes1(uint8(0xFF)) : bytes1(uint8(i));
        }

        // Run the actor test
        _actorTest({
            _rootClaim: 16,
            _absolutePrestateData: 0,
            _honestTrace: honestTrace,
            _honestL2Outputs: honestL2Outputs,
            _dishonestTrace: dishonestTrace,
            _dishonestL2Outputs: dishonestL2Outputs,
            _expectedStatus: GameStatus.DEFENDER_WINS
        });
    }

    /// @notice Static unit test for a 1v1 output bisection dispute.
    function test_static_1v1honestRootGenesisAbsolutePrestate_succeeds() public {
        // The honest l2 outputs are from [1, 16] in this game.
        uint256[] memory honestL2Outputs = new uint256[](16);
        for (uint256 i; i < honestL2Outputs.length; i++) {
            honestL2Outputs[i] = i + 1;
        }
        // The honest trace covers all block -> block + 1 transitions, and is 256 bytes long, consisting
        // of bytes [0, 255].
        bytes memory honestTrace = new bytes(256);
        for (uint256 i; i < honestTrace.length; i++) {
            honestTrace[i] = bytes1(uint8(i));
        }

        // The dishonest l2 outputs are from [2, 17] in this game.
        uint256[] memory dishonestL2Outputs = new uint256[](16);
        for (uint256 i; i < dishonestL2Outputs.length; i++) {
            dishonestL2Outputs[i] = i + 2;
        }
        // The dishonest trace covers all block -> block + 1 transitions, and is 256 bytes long, consisting
        // of all set bits.
        bytes memory dishonestTrace = new bytes(256);
        for (uint256 i; i < dishonestTrace.length; i++) {
            dishonestTrace[i] = bytes1(0xFF);
        }

        // Run the actor test
        _actorTest({
            _rootClaim: 16,
            _absolutePrestateData: 0,
            _honestTrace: honestTrace,
            _honestL2Outputs: honestL2Outputs,
            _dishonestTrace: dishonestTrace,
            _dishonestL2Outputs: dishonestL2Outputs,
            _expectedStatus: GameStatus.DEFENDER_WINS
        });
    }

    /// @notice Static unit test for a 1v1 output bisection dispute.
    function test_static_1v1dishonestRootGenesisAbsolutePrestate_succeeds() public {
        // The honest l2 outputs are from [1, 16] in this game.
        uint256[] memory honestL2Outputs = new uint256[](16);
        for (uint256 i; i < honestL2Outputs.length; i++) {
            honestL2Outputs[i] = i + 1;
        }
        // The honest trace covers all block -> block + 1 transitions, and is 256 bytes long, consisting
        // of bytes [0, 255].
        bytes memory honestTrace = new bytes(256);
        for (uint256 i; i < honestTrace.length; i++) {
            honestTrace[i] = bytes1(uint8(i));
        }

        // The dishonest l2 outputs are from [2, 17] in this game.
        uint256[] memory dishonestL2Outputs = new uint256[](16);
        for (uint256 i; i < dishonestL2Outputs.length; i++) {
            dishonestL2Outputs[i] = i + 2;
        }
        // The dishonest trace covers all block -> block + 1 transitions, and is 256 bytes long, consisting
        // of all set bits.
        bytes memory dishonestTrace = new bytes(256);
        for (uint256 i; i < dishonestTrace.length; i++) {
            dishonestTrace[i] = bytes1(0xFF);
        }

        // Run the actor test
        _actorTest({
            _rootClaim: 17,
            _absolutePrestateData: 0,
            _honestTrace: honestTrace,
            _honestL2Outputs: honestL2Outputs,
            _dishonestTrace: dishonestTrace,
            _dishonestL2Outputs: dishonestL2Outputs,
            _expectedStatus: GameStatus.CHALLENGER_WINS
        });
    }

    /// @notice Static unit test for a 1v1 output bisection dispute.
    function test_static_1v1honestRoot_succeeds() public {
        // The honest l2 outputs are from [1, 16] in this game.
        uint256[] memory honestL2Outputs = new uint256[](16);
        for (uint256 i; i < honestL2Outputs.length; i++) {
            honestL2Outputs[i] = i + 1;
        }
        // The honest trace covers all block -> block + 1 transitions, and is 256 bytes long, consisting
        // of bytes [0, 255].
        bytes memory honestTrace = new bytes(256);
        for (uint256 i; i < honestTrace.length; i++) {
            honestTrace[i] = bytes1(uint8(i));
        }

        // The dishonest l2 outputs are from [2, 17] in this game.
        uint256[] memory dishonestL2Outputs = new uint256[](16);
        for (uint256 i; i < dishonestL2Outputs.length; i++) {
            dishonestL2Outputs[i] = i + 2;
        }
        // The dishonest trace covers all block -> block + 1 transitions, and is 256 bytes long, consisting
        // of all zeros.
        bytes memory dishonestTrace = new bytes(256);

        // Run the actor test
        _actorTest({
            _rootClaim: 16,
            _absolutePrestateData: 0,
            _honestTrace: honestTrace,
            _honestL2Outputs: honestL2Outputs,
            _dishonestTrace: dishonestTrace,
            _dishonestL2Outputs: dishonestL2Outputs,
            _expectedStatus: GameStatus.DEFENDER_WINS
        });
    }

    /// @notice Static unit test for a 1v1 output bisection dispute.
    function test_static_1v1dishonestRoot_succeeds() public {
        // The honest l2 outputs are from [1, 16] in this game.
        uint256[] memory honestL2Outputs = new uint256[](16);
        for (uint256 i; i < honestL2Outputs.length; i++) {
            honestL2Outputs[i] = i + 1;
        }
        // The honest trace covers all block -> block + 1 transitions, and is 256 bytes long, consisting
        // of bytes [0, 255].
        bytes memory honestTrace = new bytes(256);
        for (uint256 i; i < honestTrace.length; i++) {
            honestTrace[i] = bytes1(uint8(i));
        }

        // The dishonest l2 outputs are from [2, 17] in this game.
        uint256[] memory dishonestL2Outputs = new uint256[](16);
        for (uint256 i; i < dishonestL2Outputs.length; i++) {
            dishonestL2Outputs[i] = i + 2;
        }
        // The dishonest trace covers all block -> block + 1 transitions, and is 256 bytes long, consisting
        // of all zeros.
        bytes memory dishonestTrace = new bytes(256);

        // Run the actor test
        _actorTest({
            _rootClaim: 17,
            _absolutePrestateData: 0,
            _honestTrace: honestTrace,
            _honestL2Outputs: honestL2Outputs,
            _dishonestTrace: dishonestTrace,
            _dishonestL2Outputs: dishonestL2Outputs,
            _expectedStatus: GameStatus.CHALLENGER_WINS
        });
    }

    /// @notice Static unit test for a 1v1 output bisection dispute.
    function test_static_1v1correctRootHalfWay_succeeds() public {
        // The honest l2 outputs are from [1, 16] in this game.
        uint256[] memory honestL2Outputs = new uint256[](16);
        for (uint256 i; i < honestL2Outputs.length; i++) {
            honestL2Outputs[i] = i + 1;
        }
        // The honest trace covers all block -> block + 1 transitions, and is 256 bytes long, consisting
        // of bytes [0, 255].
        bytes memory honestTrace = new bytes(256);
        for (uint256 i; i < honestTrace.length; i++) {
            honestTrace[i] = bytes1(uint8(i));
        }

        // The dishonest l2 outputs are half correct, half incorrect.
        uint256[] memory dishonestL2Outputs = new uint256[](16);
        for (uint256 i; i < dishonestL2Outputs.length; i++) {
            dishonestL2Outputs[i] = i > 7 ? 0xFF : i + 1;
        }
        // The dishonest trace is half correct, half incorrect.
        bytes memory dishonestTrace = new bytes(256);
        for (uint256 i; i < dishonestTrace.length; i++) {
            dishonestTrace[i] = i > (127 + 4) ? bytes1(0xFF) : bytes1(uint8(i));
        }

        // Run the actor test
        _actorTest({
            _rootClaim: 16,
            _absolutePrestateData: 0,
            _honestTrace: honestTrace,
            _honestL2Outputs: honestL2Outputs,
            _dishonestTrace: dishonestTrace,
            _dishonestL2Outputs: dishonestL2Outputs,
            _expectedStatus: GameStatus.DEFENDER_WINS
        });
    }

    /// @notice Static unit test for a 1v1 output bisection dispute.
    function test_static_1v1dishonestRootHalfWay_succeeds() public {
        // The honest l2 outputs are from [1, 16] in this game.
        uint256[] memory honestL2Outputs = new uint256[](16);
        for (uint256 i; i < honestL2Outputs.length; i++) {
            honestL2Outputs[i] = i + 1;
        }
        // The honest trace covers all block -> block + 1 transitions, and is 256 bytes long, consisting
        // of bytes [0, 255].
        bytes memory honestTrace = new bytes(256);
        for (uint256 i; i < honestTrace.length; i++) {
            honestTrace[i] = bytes1(uint8(i));
        }

        // The dishonest l2 outputs are half correct, half incorrect.
        uint256[] memory dishonestL2Outputs = new uint256[](16);
        for (uint256 i; i < dishonestL2Outputs.length; i++) {
            dishonestL2Outputs[i] = i > 7 ? 0xFF : i + 1;
        }
        // The dishonest trace is half correct, half incorrect.
        bytes memory dishonestTrace = new bytes(256);
        for (uint256 i; i < dishonestTrace.length; i++) {
            dishonestTrace[i] = i > (127 + 4) ? bytes1(0xFF) : bytes1(uint8(i));
        }

        // Run the actor test
        _actorTest({
            _rootClaim: 0xFF,
            _absolutePrestateData: 0,
            _honestTrace: honestTrace,
            _honestL2Outputs: honestL2Outputs,
            _dishonestTrace: dishonestTrace,
            _dishonestL2Outputs: dishonestL2Outputs,
            _expectedStatus: GameStatus.CHALLENGER_WINS
        });
    }

    /// @notice Static unit test for a 1v1 output bisection dispute.
    function test_static_1v1correctAbsolutePrestate_succeeds() public {
        // The honest l2 outputs are from [1, 16] in this game.
        uint256[] memory honestL2Outputs = new uint256[](16);
        for (uint256 i; i < honestL2Outputs.length; i++) {
            honestL2Outputs[i] = i + 1;
        }
        // The honest trace covers all block -> block + 1 transitions, and is 256 bytes long, consisting
        // of bytes [0, 255].
        bytes memory honestTrace = new bytes(256);
        for (uint256 i; i < honestTrace.length; i++) {
            honestTrace[i] = bytes1(uint8(i));
        }

        // The dishonest l2 outputs are half correct, half incorrect.
        uint256[] memory dishonestL2Outputs = new uint256[](16);
        for (uint256 i; i < dishonestL2Outputs.length; i++) {
            dishonestL2Outputs[i] = i > 7 ? 0xFF : i + 1;
        }
        // The dishonest trace correct is half correct, half incorrect.
        bytes memory dishonestTrace = new bytes(256);
        for (uint256 i; i < dishonestTrace.length; i++) {
            dishonestTrace[i] = i > 127 ? bytes1(0xFF) : bytes1(uint8(i));
        }

        // Run the actor test
        _actorTest({
            _rootClaim: 16,
            _absolutePrestateData: 0,
            _honestTrace: honestTrace,
            _honestL2Outputs: honestL2Outputs,
            _dishonestTrace: dishonestTrace,
            _dishonestL2Outputs: dishonestL2Outputs,
            _expectedStatus: GameStatus.DEFENDER_WINS
        });
    }

    /// @notice Static unit test for a 1v1 output bisection dispute.
    function test_static_1v1dishonestAbsolutePrestate_succeeds() public {
        // The honest l2 outputs are from [1, 16] in this game.
        uint256[] memory honestL2Outputs = new uint256[](16);
        for (uint256 i; i < honestL2Outputs.length; i++) {
            honestL2Outputs[i] = i + 1;
        }
        // The honest trace covers all block -> block + 1 transitions, and is 256 bytes long, consisting
        // of bytes [0, 255].
        bytes memory honestTrace = new bytes(256);
        for (uint256 i; i < honestTrace.length; i++) {
            honestTrace[i] = bytes1(uint8(i));
        }

        // The dishonest l2 outputs are half correct, half incorrect.
        uint256[] memory dishonestL2Outputs = new uint256[](16);
        for (uint256 i; i < dishonestL2Outputs.length; i++) {
            dishonestL2Outputs[i] = i > 7 ? 0xFF : i + 1;
        }
        // The dishonest trace correct is half correct, half incorrect.
        bytes memory dishonestTrace = new bytes(256);
        for (uint256 i; i < dishonestTrace.length; i++) {
            dishonestTrace[i] = i > 127 ? bytes1(0xFF) : bytes1(uint8(i));
        }

        // Run the actor test
        _actorTest({
            _rootClaim: 0xFF,
            _absolutePrestateData: 0,
            _honestTrace: honestTrace,
            _honestL2Outputs: honestL2Outputs,
            _dishonestTrace: dishonestTrace,
            _dishonestL2Outputs: dishonestL2Outputs,
            _expectedStatus: GameStatus.CHALLENGER_WINS
        });
    }

    /// @notice Static unit test for a 1v1 output bisection dispute.
    function test_static_1v1honestRootFinalInstruction_succeeds() public {
        // The honest l2 outputs are from [1, 16] in this game.
        uint256[] memory honestL2Outputs = new uint256[](16);
        for (uint256 i; i < honestL2Outputs.length; i++) {
            honestL2Outputs[i] = i + 1;
        }
        // The honest trace covers all block -> block + 1 transitions, and is 256 bytes long, consisting
        // of bytes [0, 255].
        bytes memory honestTrace = new bytes(256);
        for (uint256 i; i < honestTrace.length; i++) {
            honestTrace[i] = bytes1(uint8(i));
        }

        // The dishonest l2 outputs are half correct, half incorrect.
        uint256[] memory dishonestL2Outputs = new uint256[](16);
        for (uint256 i; i < dishonestL2Outputs.length; i++) {
            dishonestL2Outputs[i] = i > 7 ? 0xFF : i + 1;
        }
        // The dishonest trace is half correct, and correct all the way up to the final instruction of the exec
        // subgame.
        bytes memory dishonestTrace = new bytes(256);
        for (uint256 i; i < dishonestTrace.length; i++) {
            dishonestTrace[i] = i > (127 + 7) ? bytes1(0xFF) : bytes1(uint8(i));
        }

        // Run the actor test
        _actorTest({
            _rootClaim: 16,
            _absolutePrestateData: 0,
            _honestTrace: honestTrace,
            _honestL2Outputs: honestL2Outputs,
            _dishonestTrace: dishonestTrace,
            _dishonestL2Outputs: dishonestL2Outputs,
            _expectedStatus: GameStatus.DEFENDER_WINS
        });
    }

    /// @notice Static unit test for a 1v1 output bisection dispute.
    function test_static_1v1dishonestRootFinalInstruction_succeeds() public {
        // The honest l2 outputs are from [1, 16] in this game.
        uint256[] memory honestL2Outputs = new uint256[](16);
        for (uint256 i; i < honestL2Outputs.length; i++) {
            honestL2Outputs[i] = i + 1;
        }
        // The honest trace covers all block -> block + 1 transitions, and is 256 bytes long, consisting
        // of bytes [0, 255].
        bytes memory honestTrace = new bytes(256);
        for (uint256 i; i < honestTrace.length; i++) {
            honestTrace[i] = bytes1(uint8(i));
        }

        // The dishonest l2 outputs are half correct, half incorrect.
        uint256[] memory dishonestL2Outputs = new uint256[](16);
        for (uint256 i; i < dishonestL2Outputs.length; i++) {
            dishonestL2Outputs[i] = i > 7 ? 0xFF : i + 1;
        }
        // The dishonest trace is half correct, and correct all the way up to the final instruction of the exec
        // subgame.
        bytes memory dishonestTrace = new bytes(256);
        for (uint256 i; i < dishonestTrace.length; i++) {
            dishonestTrace[i] = i > (127 + 7) ? bytes1(0xFF) : bytes1(uint8(i));
        }

        // Run the actor test
        _actorTest({
            _rootClaim: 0xFF,
            _absolutePrestateData: 0,
            _honestTrace: honestTrace,
            _honestL2Outputs: honestL2Outputs,
            _dishonestTrace: dishonestTrace,
            _dishonestL2Outputs: dishonestL2Outputs,
            _expectedStatus: GameStatus.CHALLENGER_WINS
        });
    }

    ////////////////////////////////////////////////////////////////
    //                          HELPERS                           //
    ////////////////////////////////////////////////////////////////

    /// @dev Helper to run a 1v1 actor test
    function _actorTest(
        uint256 _rootClaim,
        uint256 _absolutePrestateData,
        bytes memory _honestTrace,
        uint256[] memory _honestL2Outputs,
        bytes memory _dishonestTrace,
        uint256[] memory _dishonestL2Outputs,
        GameStatus _expectedStatus
    )
        internal
    {
        // Setup the environment
        bytes memory absolutePrestateData =
            _setup({ _absolutePrestateData: _absolutePrestateData, _rootClaim: _rootClaim });

        // Create actors
        _createActors({
            _honestTrace: _honestTrace,
            _honestPreStateData: absolutePrestateData,
            _honestL2Outputs: _honestL2Outputs,
            _dishonestTrace: _dishonestTrace,
            _dishonestPreStateData: absolutePrestateData,
            _dishonestL2Outputs: _dishonestL2Outputs
        });

        // Exhaust all moves from both actors
        _exhaustMoves();

        // Resolve the game and assert that the defender won
        _warpAndResolve();
        assertEq(uint8(gameProxy.status()), uint8(_expectedStatus));
    }

    /// @dev Helper to setup the 1v1 test
    function _setup(
        uint256 _absolutePrestateData,
        uint256 _rootClaim
    )
        internal
        returns (bytes memory absolutePrestateData_)
    {
        absolutePrestateData_ = abi.encode(_absolutePrestateData);
        Claim absolutePrestateExec =
            _changeClaimStatus(Claim.wrap(keccak256(absolutePrestateData_)), VMStatuses.UNFINISHED);
        Claim rootClaim = Claim.wrap(bytes32(uint256(_rootClaim)));
        super.init({
            rootClaim: rootClaim,
            absolutePrestate: absolutePrestateExec,
            l2BlockNumber: _rootClaim,
            genesisBlockNumber: 0,
            genesisOutputRoot: Hash.wrap(bytes32(0))
        });
    }

    /// @dev Helper to create actors for the 1v1 dispute.
    function _createActors(
        bytes memory _honestTrace,
        bytes memory _honestPreStateData,
        uint256[] memory _honestL2Outputs,
        bytes memory _dishonestTrace,
        bytes memory _dishonestPreStateData,
        uint256[] memory _dishonestL2Outputs
    )
        internal
    {
        honest = new HonestDisputeActor({
            _gameProxy: gameProxy,
            _l2Outputs: _honestL2Outputs,
            _trace: _honestTrace,
            _preStateData: _honestPreStateData
        });
        dishonest = new HonestDisputeActor({
            _gameProxy: gameProxy,
            _l2Outputs: _dishonestL2Outputs,
            _trace: _dishonestTrace,
            _preStateData: _dishonestPreStateData
        });

        vm.label(address(honest), "HonestActor");
        vm.label(address(dishonest), "DishonestActor");
    }

    /// @dev Helper to exhaust all moves from both actors.
    function _exhaustMoves() internal {
        while (true) {
            // Allow the dishonest actor to make their moves, and then the honest actor.
            (uint256 numMovesA,) = dishonest.move();
            (uint256 numMovesB, bool success) = honest.move();

            require(success, "Honest actor's moves should always be successful");

            // If both actors have run out of moves, we're done.
            if (numMovesA == 0 && numMovesB == 0) break;
        }
    }

    /// @dev Helper to warp past the chess clock and resolve all claims within the dispute game.
    function _warpAndResolve() internal {
        // Warp past the chess clock
        vm.warp(block.timestamp + 3 days + 12 hours + 1 seconds);

        // Resolve all claims in reverse order. We allow `resolveClaim` calls to fail due to
        // the check that prevents claims with no subgames attached from being passed to
        // `resolveClaim`. There's also a check in `resolve` to ensure all children have been
        // resolved before global resolution, which catches any unresolved subgames here.
        for (uint256 i = gameProxy.claimDataLen(); i > 0; i--) {
            (bool success,) = address(gameProxy).call(abi.encodeCall(gameProxy.resolveClaim, (i - 1)));
            success;
        }
        gameProxy.resolve();
    }
}

/// @dev Helper to change the VM status byte of a claim.
function _changeClaimStatus(Claim _claim, VMStatus _status) pure returns (Claim out_) {
    assembly {
        out_ := or(and(not(shl(248, 0xFF)), _claim), shl(248, _status))
    }
}
