// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

import "src/dispute/lib/Types.sol";
import "src/dispute/lib/Errors.sol";

import { Test } from "forge-std/Test.sol";
import { FaultDisputeGame_Init, _changeClaimStatus } from "test/dispute/FaultDisputeGame.t.sol";
import { FaultDisputeGame, IFaultDisputeGame, IDisputeGame } from "src/dispute/FaultDisputeGame.sol";
import { DisputeGameFactory } from "src/dispute/DisputeGameFactory.sol";
import { Types } from "src/libraries/Types.sol";
import { Hashing } from "src/libraries/Hashing.sol";
import { RLPWriter } from "src/libraries/rlp/RLPWriter.sol";

contract AnchorStateRegistry_Init is FaultDisputeGame_Init {
    function setUp() public virtual override {
        // Duplicating the initialization/setup logic of FaultDisputeGame_Test.
        // See that test for more information, actual values here not really important.
        Claim rootClaim = Claim.wrap(bytes32((uint256(1) << 248) | uint256(10)));
        bytes memory absolutePrestateData = abi.encode(0);
        Claim absolutePrestate = _changeClaimStatus(Claim.wrap(keccak256(absolutePrestateData)), VMStatuses.UNFINISHED);

        super.setUp();
        super.init({ rootClaim: rootClaim, absolutePrestate: absolutePrestate, l2BlockNumber: 0x10 });
    }
}

contract AnchorStateRegistry_Initialize_Test is AnchorStateRegistry_Init {
    /// @dev Tests that initialization is successful.
    function test_initialize_succeeds() public view {
        (Hash cannonRoot, uint256 cannonL2BlockNumber) = anchorStateRegistry.anchors(GameTypes.CANNON);
        (Hash permissionedCannonRoot, uint256 permissionedCannonL2BlockNumber) =
            anchorStateRegistry.anchors(GameTypes.PERMISSIONED_CANNON);
        (Hash asteriscRoot, uint256 asteriscL2BlockNumber) = anchorStateRegistry.anchors(GameTypes.ASTERISC);
        (Hash alphabetRoot, uint256 alphabetL2BlockNumber) = anchorStateRegistry.anchors(GameTypes.ALPHABET);
        assertEq(cannonRoot.raw(), 0xDEADBEEFDEADBEEFDEADBEEFDEADBEEFDEADBEEFDEADBEEFDEADBEEFDEADBEEF);
        assertEq(cannonL2BlockNumber, 0);
        assertEq(permissionedCannonRoot.raw(), 0xDEADBEEFDEADBEEFDEADBEEFDEADBEEFDEADBEEFDEADBEEFDEADBEEFDEADBEEF);
        assertEq(permissionedCannonL2BlockNumber, 0);
        assertEq(asteriscRoot.raw(), 0xDEADBEEFDEADBEEFDEADBEEFDEADBEEFDEADBEEFDEADBEEFDEADBEEFDEADBEEF);
        assertEq(asteriscL2BlockNumber, 0);
        assertEq(alphabetRoot.raw(), 0xDEADBEEFDEADBEEFDEADBEEFDEADBEEFDEADBEEFDEADBEEFDEADBEEFDEADBEEF);
        assertEq(alphabetL2BlockNumber, 0);
    }
}

contract AnchorStateRegistry_VerifyAnchor_Test is AnchorStateRegistry_Init {
    address mockGameAddress;

    function setUp() public virtual override {
        super.setUp();
        mockGameAddress = vm.addr(0xFACADE);
    }

    /// @dev Tests that verifying an output root succeeds when the root is valid.
    function test_staticVerifyAnchor_validRoot_succeeds() public {
        // Raw header RLP
        bytes memory headerRLP =
            hex"f901fca0102de6ffb001480cc9b8b548fd05c34cd4f46ae4aa91759393db90ea0409887da01dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347944200000000000000000000000000000000000011a04c0f6050836fd6788a0a15f3569b192def4cb8d2734399a31e4ad6be421eb3dca04df094c413f499eaaabf48b96fd1d83c23b30b5cec95a5eff9c96b2d5d2ee875a03c715dd96d2597ccd46fde046da5e4b13e0a5b7d0a2ff60c3ee6c92fee9600eab901000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000080018401c9c38082f9f58464d6dbae80a032e4469959675ceed3d1a9f43709a8055d689d712844f820aa34dbc9f49e286c880000000000000000843a699d00";

        // Output root
        bytes32 bh = 0xe460dd641f493c0184f2544c9bcfe3b4dcfe69cfa8054f8aed291b0ddda0025e;
        bytes32 wdr = 0x8ed4baae3a927be3dea54996b4d5899f8c01e7594bf50b17dc1e741388ce3d12;
        bytes32 sr = 0x4c0f6050836fd6788a0a15f3569b192def4cb8d2734399a31e4ad6be421eb3dc;
        Types.OutputRootProof memory outputRootProof =
            Types.OutputRootProof({ version: 0, stateRoot: sr, messagePasserStorageRoot: wdr, latestBlockhash: bh });
        bytes32 outputRoot = Hashing.hashOutputRootProof(outputRootProof);

        // Create a mock dispute game.
        IDisputeGame game = disputeGameFactory.create(GameTypes.CANNON, Claim.wrap(outputRoot), abi.encode(1));
        FaultDisputeGame prox = FaultDisputeGame(address(game));

        // Warp past the clock time and resolve.
        vm.warp(block.timestamp + prox.maxClockDuration().raw() + 1 seconds);
        prox.resolveClaim(0, 0);
        prox.resolve();
        // Warp past the finalization delay.
        vm.warp(block.timestamp + prox.maxClockDuration().raw() + 1 seconds);

        anchorStateRegistry.verifyAnchor(IFaultDisputeGame(address(prox)), outputRootProof, headerRLP);
    }

    /// @dev Tests that verifying an output root succeeds when the root is valid.
    function testFuzz_verifyAnchor_validRoot_succeeds(
        bytes32 withdrawalRoot,
        bytes32 storageRoot,
        uint64 l2BlockNumber
    )
        public
    {
        l2BlockNumber = uint64(bound(l2BlockNumber, 1, type(uint64).max));

        // L2 Block header
        bytes[] memory rawHeaderRLP = new bytes[](12);
        rawHeaderRLP[0] = hex"83FACADE";
        rawHeaderRLP[1] = hex"83FACADE";
        rawHeaderRLP[2] = hex"83FACADE";
        rawHeaderRLP[3] = hex"83FACADE";
        rawHeaderRLP[4] = hex"83FACADE";
        rawHeaderRLP[5] = hex"83FACADE";
        rawHeaderRLP[6] = hex"83FACADE";
        rawHeaderRLP[7] = hex"83FACADE";
        rawHeaderRLP[8] = RLPWriter.writeBytes(abi.encodePacked(l2BlockNumber));
        rawHeaderRLP[9] = hex"83FACADE";
        rawHeaderRLP[10] = hex"83FACADE";
        rawHeaderRLP[11] = hex"83FACADE";
        bytes memory headerRLP = RLPWriter.writeList(rawHeaderRLP);

        // Output root
        Types.OutputRootProof memory outputRootProof = Types.OutputRootProof({
            version: 0,
            stateRoot: storageRoot,
            messagePasserStorageRoot: withdrawalRoot,
            latestBlockhash: keccak256(headerRLP)
        });
        bytes32 outputRoot = Hashing.hashOutputRootProof(outputRootProof);

        // Create a mock dispute game.
        IDisputeGame game =
            disputeGameFactory.create(GameTypes.CANNON, Claim.wrap(outputRoot), abi.encode(l2BlockNumber));
        FaultDisputeGame prox = FaultDisputeGame(address(game));

        // Warp past the clock time and resolve.
        vm.warp(block.timestamp + prox.maxClockDuration().raw() + 1 seconds);
        prox.resolveClaim(0, 0);
        prox.resolve();
        // Warp past the finalization delay.
        vm.warp(block.timestamp + prox.maxClockDuration().raw() + 1 seconds);

        anchorStateRegistry.verifyAnchor(IFaultDisputeGame(address(prox)), outputRootProof, headerRLP);
    }

    /// @dev Tests that verifying an output root fails when the game is not registered.
    function test_verifyAnchor_gameNotValid_reverts() public {
        vm.mockCall(
            mockGameAddress,
            abi.encodeCall(IDisputeGame.gameData, ()),
            abi.encode(GameType.wrap(0), Claim.wrap(0), abi.encode(0))
        );

        vm.expectRevert("AnchorStateRegistry: fault dispute game not registered with factory");
        anchorStateRegistry.verifyAnchor(
            IFaultDisputeGame(mockGameAddress),
            Types.OutputRootProof({ version: 0, stateRoot: 0, messagePasserStorageRoot: 0, latestBlockhash: 0 }),
            ""
        );
    }

    /// @dev Tests that verifying an output root fails when the game is not proposing an output that advances the
    /// anchor.
    function test_verifyAnchor_gameDoesntAdvanceAnchor_reverts() public {
        vm.mockCall(
            mockGameAddress,
            abi.encodeCall(IDisputeGame.gameData, ()),
            abi.encode(GameType.wrap(0), Claim.wrap(0), abi.encode(0))
        );
        vm.mockCall(
            address(disputeGameFactory),
            abi.encodeCall(DisputeGameFactory.games, (GameType.wrap(0), Claim.wrap(0), abi.encode(0))),
            abi.encode(mockGameAddress, false)
        );
        _setAnchor(GameTypes.CANNON, Hash.wrap(0), 0xFF);

        vm.expectRevert("AnchorStateRegistry: block number of proposal does not advance anchor");
        anchorStateRegistry.verifyAnchor(
            IFaultDisputeGame(mockGameAddress),
            Types.OutputRootProof({ version: 0, stateRoot: 0, messagePasserStorageRoot: 0, latestBlockhash: 0 }),
            ""
        );
    }

    /// @dev Tests that verifying an output root fails when the game is not resolved yet.
    function test_verifyAnchor_gameNotResolved_reverts() public {
        vm.mockCall(
            mockGameAddress,
            abi.encodeCall(IDisputeGame.gameData, ()),
            abi.encode(GameType.wrap(0), Claim.wrap(0), abi.encode(1))
        );
        vm.mockCall(mockGameAddress, abi.encodeCall(IDisputeGame.status, ()), abi.encode(GameStatus.IN_PROGRESS));
        vm.mockCall(
            address(disputeGameFactory),
            abi.encodeCall(DisputeGameFactory.games, (GameType.wrap(0), Claim.wrap(0), abi.encode(1))),
            abi.encode(mockGameAddress, false)
        );

        vm.expectRevert("AnchorStateRegistry: status of proposal is not DEFENDER_WINS");
        anchorStateRegistry.verifyAnchor(
            IFaultDisputeGame(mockGameAddress),
            Types.OutputRootProof({ version: 0, stateRoot: 0, messagePasserStorageRoot: 0, latestBlockhash: 0 }),
            ""
        );
    }

    /// @dev Tests that verifying an output root fails when the game is not resolved yet.
    function test_verifyAnchor_gameNotFinalized_reverts() public {
        vm.mockCall(
            mockGameAddress,
            abi.encodeCall(IDisputeGame.gameData, ()),
            abi.encode(GameType.wrap(0), Claim.wrap(0), abi.encode(1))
        );
        vm.mockCall(mockGameAddress, abi.encodeCall(IDisputeGame.status, ()), abi.encode(GameStatus.DEFENDER_WINS));
        vm.mockCall(mockGameAddress, abi.encodeCall(IDisputeGame.resolvedAt, ()), abi.encode(block.timestamp));
        vm.mockCall(
            address(disputeGameFactory),
            abi.encodeCall(DisputeGameFactory.games, (GameType.wrap(0), Claim.wrap(0), abi.encode(1))),
            abi.encode(mockGameAddress, false)
        );

        vm.expectRevert("AnchorStateRegistry: proposal not finalized");
        anchorStateRegistry.verifyAnchor(
            IFaultDisputeGame(mockGameAddress),
            Types.OutputRootProof({ version: 0, stateRoot: 0, messagePasserStorageRoot: 0, latestBlockhash: 0 }),
            ""
        );
    }

    /// @dev Tests that verifying an output root fails when the output root proof is invalid.
    function test_verifyAnchor_badOutputRootProof_reverts() public {
        vm.mockCall(
            mockGameAddress,
            abi.encodeCall(IDisputeGame.gameData, ()),
            abi.encode(GameType.wrap(0), Claim.wrap(0), abi.encode(1))
        );
        vm.mockCall(mockGameAddress, abi.encodeCall(IDisputeGame.status, ()), abi.encode(GameStatus.DEFENDER_WINS));
        vm.mockCall(mockGameAddress, abi.encodeCall(IDisputeGame.rootClaim, ()), abi.encode(0));
        vm.mockCall(
            mockGameAddress, abi.encodeCall(IDisputeGame.resolvedAt, ()), abi.encode(block.timestamp - 3.5 days)
        );
        vm.mockCall(
            address(disputeGameFactory),
            abi.encodeCall(DisputeGameFactory.games, (GameType.wrap(0), Claim.wrap(0), abi.encode(1))),
            abi.encode(mockGameAddress, false)
        );

        vm.expectRevert("AnchorStateRegistry: output root proof invalid");
        anchorStateRegistry.verifyAnchor(
            IFaultDisputeGame(mockGameAddress),
            Types.OutputRootProof({ version: 0, stateRoot: 0, messagePasserStorageRoot: 0, latestBlockhash: 0 }),
            ""
        );
    }

    /// @dev Tests that verifying an output root fails when the header RLP is invalid with respect to the output root's
    ///      latest blockhash.
    function test_verifyAnchor_badHeaderRLP_reverts() public {
        Types.OutputRootProof memory outputRootProof =
            Types.OutputRootProof({ version: 0, stateRoot: 0, messagePasserStorageRoot: 0, latestBlockhash: 0 });
        bytes32 outputRoot = Hashing.hashOutputRootProof(outputRootProof);

        vm.mockCall(
            mockGameAddress,
            abi.encodeCall(IDisputeGame.gameData, ()),
            abi.encode(GameType.wrap(0), Claim.wrap(0), abi.encode(1))
        );
        vm.mockCall(mockGameAddress, abi.encodeCall(IDisputeGame.status, ()), abi.encode(GameStatus.DEFENDER_WINS));
        vm.mockCall(mockGameAddress, abi.encodeCall(IDisputeGame.rootClaim, ()), abi.encode(outputRoot));
        vm.mockCall(
            mockGameAddress, abi.encodeCall(IDisputeGame.resolvedAt, ()), abi.encode(block.timestamp - 3.5 days)
        );
        vm.mockCall(
            address(disputeGameFactory),
            abi.encodeCall(DisputeGameFactory.games, (GameType.wrap(0), Claim.wrap(0), abi.encode(1))),
            abi.encode(mockGameAddress, false)
        );

        vm.expectRevert("AnchorStateRegistry: header rlp invalid");
        anchorStateRegistry.verifyAnchor(IFaultDisputeGame(mockGameAddress), outputRootProof, "");
    }

    /// @dev Tests that verifying an output root fails when the header timestamp is incorrectly formatted.
    function test_verifyAnchor_badBlockHeaderTimestamp_reverts() public {
        // L2 Block header
        bytes[] memory rawHeaderRLP = new bytes[](9);
        rawHeaderRLP[0] = hex"83FACADE";
        rawHeaderRLP[1] = hex"83FACADE";
        rawHeaderRLP[2] = hex"83FACADE";
        rawHeaderRLP[3] = hex"83FACADE";
        rawHeaderRLP[4] = hex"83FACADE";
        rawHeaderRLP[5] = hex"83FACADE";
        rawHeaderRLP[6] = hex"83FACADE";
        rawHeaderRLP[7] = hex"83FACADE";
        rawHeaderRLP[8] = RLPWriter.writeBytes(new bytes(0xFF));
        bytes memory headerRLP = RLPWriter.writeList(rawHeaderRLP);

        // output root
        Types.OutputRootProof memory outputRootProof = Types.OutputRootProof({
            version: 0,
            stateRoot: 0,
            messagePasserStorageRoot: 0,
            latestBlockhash: keccak256(headerRLP)
        });
        bytes32 outputRoot = Hashing.hashOutputRootProof(outputRootProof);

        vm.mockCall(
            mockGameAddress,
            abi.encodeCall(IDisputeGame.gameData, ()),
            abi.encode(GameType.wrap(0), Claim.wrap(0), abi.encode(1))
        );
        vm.mockCall(mockGameAddress, abi.encodeCall(IDisputeGame.status, ()), abi.encode(GameStatus.DEFENDER_WINS));
        vm.mockCall(mockGameAddress, abi.encodeCall(IDisputeGame.rootClaim, ()), abi.encode(outputRoot));
        vm.mockCall(
            mockGameAddress, abi.encodeCall(IDisputeGame.resolvedAt, ()), abi.encode(block.timestamp - 3.5 days)
        );
        vm.mockCall(
            address(disputeGameFactory),
            abi.encodeCall(DisputeGameFactory.games, (GameType.wrap(0), Claim.wrap(0), abi.encode(1))),
            abi.encode(mockGameAddress, false)
        );

        vm.expectRevert("AnchorStateRegistry: bad block header timestamp");
        anchorStateRegistry.verifyAnchor(IFaultDisputeGame(mockGameAddress), outputRootProof, headerRLP);
    }

    /// @dev Tests that verifying an output root fails when the block in the output root does not correspond to the
    ///     claimed L2 block number.
    function test_verifyAnchor_badBlockNumber_reverts() public {
        // L2 Block header
        bytes[] memory rawHeaderRLP = new bytes[](9);
        rawHeaderRLP[0] = hex"83FACADE";
        rawHeaderRLP[1] = hex"83FACADE";
        rawHeaderRLP[2] = hex"83FACADE";
        rawHeaderRLP[3] = hex"83FACADE";
        rawHeaderRLP[4] = hex"83FACADE";
        rawHeaderRLP[5] = hex"83FACADE";
        rawHeaderRLP[6] = hex"83FACADE";
        rawHeaderRLP[7] = hex"83FACADE";
        rawHeaderRLP[8] = RLPWriter.writeBytes(abi.encode(0));
        bytes memory headerRLP = RLPWriter.writeList(rawHeaderRLP);

        // output root
        Types.OutputRootProof memory outputRootProof = Types.OutputRootProof({
            version: 0,
            stateRoot: 0,
            messagePasserStorageRoot: 0,
            latestBlockhash: keccak256(headerRLP)
        });
        bytes32 outputRoot = Hashing.hashOutputRootProof(outputRootProof);

        vm.mockCall(
            mockGameAddress,
            abi.encodeCall(IDisputeGame.gameData, ()),
            abi.encode(GameType.wrap(0), Claim.wrap(0), abi.encode(1))
        );
        vm.mockCall(mockGameAddress, abi.encodeCall(IDisputeGame.status, ()), abi.encode(GameStatus.DEFENDER_WINS));
        vm.mockCall(mockGameAddress, abi.encodeCall(IDisputeGame.rootClaim, ()), abi.encode(outputRoot));
        vm.mockCall(
            mockGameAddress, abi.encodeCall(IDisputeGame.resolvedAt, ()), abi.encode(block.timestamp - 3.5 days)
        );
        vm.mockCall(
            address(disputeGameFactory),
            abi.encodeCall(DisputeGameFactory.games, (GameType.wrap(0), Claim.wrap(0), abi.encode(1))),
            abi.encode(mockGameAddress, false)
        );

        vm.expectRevert("AnchorStateRegistry: block number mismatch");
        anchorStateRegistry.verifyAnchor(IFaultDisputeGame(mockGameAddress), outputRootProof, headerRLP);
    }

    /// @dev Tests that the `_setAnchor` helper function works as expected.
    function testFuzz_setAnchor_succeeds(GameType gameType, Hash root, uint256 l2BlockNumber) public {
        _setAnchor(gameType, root, l2BlockNumber);

        (Hash _root, uint256 _l2BlockNumber) = anchorStateRegistry.anchors(gameType);
        assertEq(_root.raw(), root.raw());
        assertEq(_l2BlockNumber, l2BlockNumber);
    }

    function _setAnchor(GameType gameType, Hash root, uint256 l2BlockNumber) internal {
        bytes32 startMapSlot = keccak256(abi.encode(gameType, uint256(1)));
        vm.store(address(anchorStateRegistry), startMapSlot, root.raw());
        vm.store(address(anchorStateRegistry), bytes32(uint256(startMapSlot) + 1), bytes32(l2BlockNumber));
    }
}
