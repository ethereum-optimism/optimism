// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

// Testing
import { DisputeGameFactory_TestInit } from "test/dispute/DisputeGameFactory.t.sol";

// Libraries
import { GameStatus, GameTypes, Claim } from "src/dispute/lib/Types.sol";
import { BadAuth, BadExtraData, UnknownChainId } from "src/dispute/lib/Errors.sol";
import { Types } from "src/libraries/Types.sol";
import { Hashing } from "src/libraries/Hashing.sol";
import { Encoding } from "src/libraries/Encoding.sol";

// Interfaces
import { ISuperPermissionedDisputeGame } from "interfaces/dispute/ISuperPermissionedDisputeGame.sol";
import { IAnchorStateRegistry } from "interfaces/dispute/IAnchorStateRegistry.sol";

/// @title SuperPermissionedDisputeGame_TestInit
/// @notice Reusable test initialization for `SuperPermissionedDisputeGame` tests.
abstract contract SuperPermissionedDisputeGame_TestInit is DisputeGameFactory_TestInit {
    address internal constant PROPOSER = address(0xfacade9);

    ISuperPermissionedDisputeGame internal gameImpl;
    ISuperPermissionedDisputeGame internal gameProxy;
    Types.SuperRootProof internal superRootProof;
    Claim internal rootClaim;
    bytes internal extraData;
    uint256 internal validL2SequenceNumber;

    function setUp() public override {
        super.setUp();

        (, uint256 anchorSequenceNumber) = anchorStateRegistry.getAnchorRoot();
        validL2SequenceNumber = anchorSequenceNumber + 1;
        superRootProof.version = bytes1(uint8(1));
        superRootProof.timestamp = uint64(validL2SequenceNumber);
        superRootProof.outputRoots.push(Types.OutputRootWithChainId({ chainId: 5, root: keccak256("chain-5") }));
        superRootProof.outputRoots.push(Types.OutputRootWithChainId({ chainId: 6, root: keccak256("chain-6") }));
        rootClaim = Claim.wrap(Hashing.hashSuperRootProof(superRootProof));
        extraData = Encoding.encodeSuperRootProof(superRootProof);

        address impl = setupSuperPermissionedDisputeGame(PROPOSER);
        gameImpl = ISuperPermissionedDisputeGame(impl);

        vm.prank(superchainConfig.guardian());
        anchorStateRegistry.setRespectedGameType(GameTypes.SUPER_PERMISSIONED_CANNON);

        vm.prank(PROPOSER, PROPOSER);
        gameProxy = ISuperPermissionedDisputeGame(
            payable(address(disputeGameFactory.create(GameTypes.SUPER_PERMISSIONED_CANNON, rootClaim, extraData)))
        );
    }

    function _createGame(address _sender, Claim _claim, bytes memory _extraData) internal returns (address proxy_) {
        vm.prank(_sender, _sender);
        proxy_ = address(disputeGameFactory.create(GameTypes.SUPER_PERMISSIONED_CANNON, _claim, _extraData));
    }
}

contract SuperPermissionedDisputeGame_Version_Test is SuperPermissionedDisputeGame_TestInit {
    function test_version_works() public view {
        assertTrue(bytes(gameImpl.version()).length > 0);
    }
}

contract SuperPermissionedDisputeGame_Initialize_Test is SuperPermissionedDisputeGame_TestInit {
    function test_initialize_setsGameState_succeeds() public view {
        assertEq(gameProxy.createdAt().raw(), block.timestamp);
        assertEq(uint8(gameProxy.status()), uint8(GameStatus.DEFENDER_WINS));
        assertEq(gameProxy.resolvedAt().raw(), block.timestamp);
        assertTrue(gameProxy.wasRespectedGameTypeWhenCreated());
        assertEq(gameProxy.gameCreator(), PROPOSER);
        assertEq(gameProxy.gameType().raw(), GameTypes.SUPER_PERMISSIONED_CANNON.raw());
        assertEq(gameProxy.rootClaim().raw(), rootClaim.raw());
        assertEq(gameProxy.extraData(), extraData);
        assertEq(address(gameProxy.anchorStateRegistry()), address(anchorStateRegistry));
        assertEq(gameProxy.proposer(), PROPOSER);
    }

    function test_createGame_notProposer_reverts() public {
        Types.SuperRootProof memory authProof = superRootProof;
        authProof.timestamp = uint64(validL2SequenceNumber + 1);
        bytes memory authExtraData = Encoding.encodeSuperRootProof(authProof);

        vm.expectRevert(BadAuth.selector);
        _createGame(address(0xbeef), Claim.wrap(Hashing.hashSuperRootProof(authProof)), authExtraData);
    }

    function test_createGame_badRootClaim_reverts() public {
        vm.expectRevert(BadExtraData.selector);
        _createGame(PROPOSER, Claim.wrap(keccak256("bad-root")), extraData);
    }

    function test_createGame_oldSequenceNumber_reverts() public {
        Types.SuperRootProof memory oldProof;
        oldProof.version = bytes1(uint8(1));
        oldProof.timestamp = uint64(validL2SequenceNumber - 1);
        oldProof.outputRoots = new Types.OutputRootWithChainId[](1);
        oldProof.outputRoots[0] = Types.OutputRootWithChainId({ chainId: 5, root: keccak256("old") });
        bytes memory oldExtraData = Encoding.encodeSuperRootProof(oldProof);

        vm.expectRevert(BadExtraData.selector);
        _createGame(PROPOSER, Claim.wrap(Hashing.hashSuperRootProof(oldProof)), oldExtraData);
    }
}

contract SuperPermissionedDisputeGame_RootClaimByChainId_Test is SuperPermissionedDisputeGame_TestInit {
    function test_rootClaimByChainId_succeeds() public view {
        assertEq(gameProxy.rootClaimByChainId(5).raw(), keccak256("chain-5"));
        assertEq(gameProxy.rootClaimByChainId(6).raw(), keccak256("chain-6"));
    }

    function test_rootClaimByChainId_unknownChain_reverts() public {
        vm.expectRevert(UnknownChainId.selector);
        gameProxy.rootClaimByChainId(7);
    }
}

contract SuperPermissionedDisputeGame_Resolve_Test is SuperPermissionedDisputeGame_TestInit {
    function test_resolve_noop_succeeds() public {
        uint64 resolvedAt = gameProxy.resolvedAt().raw();

        vm.warp(block.timestamp + 1);
        assertEq(uint8(gameProxy.resolve()), uint8(GameStatus.DEFENDER_WINS));

        assertEq(uint8(gameProxy.status()), uint8(GameStatus.DEFENDER_WINS));
        assertEq(gameProxy.resolvedAt().raw(), resolvedAt);
    }

    function test_resolve_blacklistedGameIsNotClaimValid_succeeds() public {
        vm.prank(superchainConfig.guardian());
        anchorStateRegistry.blacklistDisputeGame(gameProxy);

        assertFalse(anchorStateRegistry.isGameClaimValid(gameProxy));
        vm.expectRevert(IAnchorStateRegistry.AnchorStateRegistry_InvalidAnchorGame.selector);
        anchorStateRegistry.setAnchorState(gameProxy);
    }
}
