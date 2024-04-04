// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

import "src/libraries/DisputeTypes.sol";
import "src/libraries/DisputeErrors.sol";

import { Test } from "forge-std/Test.sol";
import { FaultDisputeGame_Init, _changeClaimStatus } from "test/dispute/FaultDisputeGame.t.sol";

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
    function test_initialize_succeeds() public {
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

contract AnchorStateRegistry_TryUpdateAnchorState_Test is AnchorStateRegistry_Init {
    /// @dev Tests that updating the anchor state succeeds when the game state is valid and newer.
    function test_tryUpdateAnchorState_validNewerState_succeeds() public {
        // Confirm that the anchor state is older than the game state.
        (Hash root, uint256 l2BlockNumber) = anchorStateRegistry.anchors(gameProxy.gameType());
        assert(l2BlockNumber < gameProxy.l2BlockNumber());

        // Mock the state that we want.
        vm.mockCall(
            address(gameProxy), abi.encodeWithSelector(gameProxy.status.selector), abi.encode(GameStatus.DEFENDER_WINS)
        );

        // Try to update the anchor state.
        vm.prank(address(gameProxy));
        anchorStateRegistry.tryUpdateAnchorState();

        // Confirm that the anchor state is now the same as the game state.
        (root, l2BlockNumber) = anchorStateRegistry.anchors(gameProxy.gameType());
        assertEq(l2BlockNumber, gameProxy.l2BlockNumber());
        assertEq(root.raw(), gameProxy.rootClaim().raw());
    }

    /// @dev Tests that updating the anchor state fails when the game state is valid but older.
    function test_tryUpdateAnchorState_validOlderState_fails() public {
        // Confirm that the anchor state is newer than the game state.
        vm.mockCall(address(gameProxy), abi.encodeWithSelector(gameProxy.l2BlockNumber.selector), abi.encode(0));
        (Hash root, uint256 l2BlockNumber) = anchorStateRegistry.anchors(gameProxy.gameType());
        assert(l2BlockNumber >= gameProxy.l2BlockNumber());

        // Mock the state that we want.
        vm.mockCall(address(gameProxy), abi.encodeWithSelector(gameProxy.l2BlockNumber.selector), abi.encode(0));
        vm.mockCall(
            address(gameProxy), abi.encodeWithSelector(gameProxy.status.selector), abi.encode(GameStatus.DEFENDER_WINS)
        );

        // Try to update the anchor state.
        vm.prank(address(gameProxy));
        anchorStateRegistry.tryUpdateAnchorState();

        // Confirm that the anchor state has not updated.
        (Hash updatedRoot, uint256 updatedL2BlockNumber) = anchorStateRegistry.anchors(gameProxy.gameType());
        assertEq(updatedL2BlockNumber, l2BlockNumber);
        assertEq(updatedRoot.raw(), root.raw());
    }

    /// @dev Tests that updating the anchor state fails when the game state is invalid.
    function test_tryUpdateAnchorState_invalidNewerState_fails() public {
        // Confirm that the anchor state is older than the game state.
        (Hash root, uint256 l2BlockNumber) = anchorStateRegistry.anchors(gameProxy.gameType());
        assert(l2BlockNumber < gameProxy.l2BlockNumber());

        // Mock the state that we want.
        vm.mockCall(
            address(gameProxy),
            abi.encodeWithSelector(gameProxy.status.selector),
            abi.encode(GameStatus.CHALLENGER_WINS)
        );

        // Try to update the anchor state.
        vm.prank(address(gameProxy));
        anchorStateRegistry.tryUpdateAnchorState();

        // Confirm that the anchor state has not updated.
        (Hash updatedRoot, uint256 updatedL2BlockNumber) = anchorStateRegistry.anchors(gameProxy.gameType());
        assertEq(updatedL2BlockNumber, l2BlockNumber);
        assertEq(updatedRoot.raw(), root.raw());
    }

    /// @dev Tests that updating the anchor state fails when the game is not registered with the factory.
    function test_tryUpdateAnchorState_invalidGame_fails() public {
        // Confirm that the anchor state is older than the game state.
        (Hash root, uint256 l2BlockNumber) = anchorStateRegistry.anchors(gameProxy.gameType());
        assert(l2BlockNumber < gameProxy.l2BlockNumber());

        // Mock the state that we want.
        vm.mockCall(
            address(disputeGameFactory),
            abi.encodeWithSelector(
                disputeGameFactory.games.selector, gameProxy.gameType(), gameProxy.rootClaim(), gameProxy.extraData()
            ),
            abi.encode(address(0), 0)
        );

        // Try to update the anchor state.
        vm.prank(address(gameProxy));
        vm.expectRevert("AnchorStateRegistry: fault dispute game not registered with factory");
        anchorStateRegistry.tryUpdateAnchorState();

        // Confirm that the anchor state has not updated.
        (Hash updatedRoot, uint256 updatedL2BlockNumber) = anchorStateRegistry.anchors(gameProxy.gameType());
        assertEq(updatedL2BlockNumber, l2BlockNumber);
        assertEq(updatedRoot.raw(), root.raw());
    }
}
