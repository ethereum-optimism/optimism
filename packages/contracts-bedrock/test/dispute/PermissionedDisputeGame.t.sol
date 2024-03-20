// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

import { Test } from "forge-std/Test.sol";
import { Vm } from "forge-std/Vm.sol";
import { DisputeGameFactory_Init } from "test/dispute/DisputeGameFactory.t.sol";
import { DisputeGameFactory } from "src/dispute/DisputeGameFactory.sol";
import { PermissionedDisputeGame } from "src/dispute/PermissionedDisputeGame.sol";
import { DelayedWETH } from "src/dispute/weth/DelayedWETH.sol";
import { L2OutputOracle } from "src/L1/L2OutputOracle.sol";
import { PreimageOracle } from "src/cannon/PreimageOracle.sol";
import { PreimageKeyLib } from "src/cannon/PreimageKeyLib.sol";

import "src/libraries/DisputeTypes.sol";
import "src/libraries/DisputeErrors.sol";
import { Types } from "src/libraries/Types.sol";
import { LibClock } from "src/dispute/lib/LibUDT.sol";
import { LibPosition } from "src/dispute/lib/LibPosition.sol";
import { IBigStepper, IPreimageOracle } from "src/dispute/interfaces/IBigStepper.sol";
import { AlphabetVM } from "test/mocks/AlphabetVM.sol";

import { DisputeActor, HonestDisputeActor } from "test/actors/FaultDisputeActors.sol";

contract PermissionedDisputeGame_Init is DisputeGameFactory_Init {
    /// @dev The type of the game being tested.
    GameType internal constant GAME_TYPE = GameType.wrap(1);
    /// @dev Mock proposer key
    address internal constant PROPOSER = address(0xfacade9);
    /// @dev Mock challenger key
    address internal constant CHALLENGER = address(0xfacadec);

    /// @dev The implementation of the game.
    PermissionedDisputeGame internal gameImpl;
    /// @dev The `Clone` proxy of the game.
    PermissionedDisputeGame internal gameProxy;

    /// @dev The extra data passed to the game for initialization.
    bytes internal extraData;

    event Move(uint256 indexed parentIndex, Claim indexed pivot, address indexed claimant);

    function init(Claim rootClaim, Claim absolutePrestate, uint256 l2BlockNumber) public {
        // Set the time to a realistic date.
        vm.warp(1690906994);

        // Set the extra data for the game creation
        extraData = abi.encode(l2BlockNumber);

        AlphabetVM _vm = new AlphabetVM(absolutePrestate, new PreimageOracle(0, 0));

        // Use a 7 day delayed WETH to simulate withdrawals.
        DelayedWETH _weth = new DelayedWETH(7 days);

        // Deploy an implementation of the fault game
        gameImpl = new PermissionedDisputeGame({
            _gameType: GAME_TYPE,
            _absolutePrestate: absolutePrestate,
            _maxGameDepth: 2 ** 3,
            _splitDepth: 2 ** 2,
            _gameDuration: Duration.wrap(7 days),
            _vm: _vm,
            _weth: _weth,
            _anchorStateRegistry: anchorStateRegistry,
            _l2ChainId: 10,
            _proposer: PROPOSER,
            _challenger: CHALLENGER
        });
        // Register the game implementation with the factory.
        disputeGameFactory.setImplementation(GAME_TYPE, gameImpl);
        // Create a new game.
        vm.prank(PROPOSER, PROPOSER);
        gameProxy =
            PermissionedDisputeGame(payable(address(disputeGameFactory.create(GAME_TYPE, rootClaim, extraData))));

        // Check immutables
        assertEq(gameProxy.gameType().raw(), GAME_TYPE.raw());
        assertEq(gameProxy.absolutePrestate().raw(), absolutePrestate.raw());
        assertEq(gameProxy.maxGameDepth(), 2 ** 3);
        assertEq(gameProxy.splitDepth(), 2 ** 2);
        assertEq(gameProxy.gameDuration().raw(), 7 days);
        assertEq(address(gameProxy.vm()), address(_vm));

        // Label the proxy
        vm.label(address(gameProxy), "FaultDisputeGame_Clone");
    }

    fallback() external payable { }

    receive() external payable { }
}

contract PermissionedDisputeGame_Test is PermissionedDisputeGame_Init {
    /// @dev The root claim of the game.
    Claim internal constant ROOT_CLAIM = Claim.wrap(bytes32((uint256(1) << 248) | uint256(10)));
    /// @dev Minimum bond value that covers all possible moves.
    uint256 internal constant MIN_BOND = 50 ether;

    /// @dev The preimage of the absolute prestate claim
    bytes internal absolutePrestateData;
    /// @dev The absolute prestate of the trace.
    Claim internal absolutePrestate;

    function setUp() public override {
        absolutePrestateData = abi.encode(0);
        absolutePrestate = _changeClaimStatus(Claim.wrap(keccak256(absolutePrestateData)), VMStatuses.UNFINISHED);

        super.setUp();
        super.init({ rootClaim: ROOT_CLAIM, absolutePrestate: absolutePrestate, l2BlockNumber: 0x10 });
    }

    /// @dev Tests that the proposer can create a permissioned dispute game.
    function test_createGame_proposer_succeeds() public {
        vm.prank(PROPOSER, PROPOSER);
        disputeGameFactory.create(GAME_TYPE, ROOT_CLAIM, abi.encode(0x420));
    }

    /// @dev Tests that the permissioned game cannot be created by any address other than the proposer.
    function testFuzz_createGame_notProposer_reverts(address _p) public {
        vm.assume(_p != PROPOSER);

        vm.prank(_p, _p);
        vm.expectRevert(BadAuth.selector);
        disputeGameFactory.create(GAME_TYPE, ROOT_CLAIM, abi.encode(0x420));
    }

    /// @dev Tests that the challenger can participate in a permissioned dispute game.
    function test_participateInGame_challenger_succeeds() public {
        vm.startPrank(CHALLENGER, CHALLENGER);
        vm.deal(CHALLENGER, MIN_BOND * 3);
        gameProxy.attack{ value: MIN_BOND }(0, Claim.wrap(0));
        gameProxy.defend{ value: MIN_BOND }(1, Claim.wrap(0));
        gameProxy.move{ value: MIN_BOND }(2, Claim.wrap(0), true);
        vm.stopPrank();
    }

    /// @dev Tests that the proposer can participate in a permissioned dispute game.
    function test_participateInGame_proposer_succeeds() public {
        vm.startPrank(PROPOSER, PROPOSER);
        vm.deal(PROPOSER, MIN_BOND * 3);
        gameProxy.attack{ value: MIN_BOND }(0, Claim.wrap(0));
        gameProxy.defend{ value: MIN_BOND }(1, Claim.wrap(0));
        gameProxy.move{ value: MIN_BOND }(2, Claim.wrap(0), true);
        vm.stopPrank();
    }

    /// @dev Tests that addresses that are not the proposer or challenger cannot participate in a permissioned dispute
    ///      game.
    function test_participateInGame_notAuthorized_reverts(address _p) public {
        vm.assume(_p != PROPOSER && _p != CHALLENGER);

        vm.startPrank(_p, _p);
        vm.expectRevert(BadAuth.selector);
        gameProxy.attack(0, Claim.wrap(0));
        vm.expectRevert(BadAuth.selector);
        gameProxy.defend(1, Claim.wrap(0));
        vm.expectRevert(BadAuth.selector);
        gameProxy.move(2, Claim.wrap(0), true);
        vm.stopPrank();
    }
}

/// @dev Helper to change the VM status byte of a claim.
function _changeClaimStatus(Claim _claim, VMStatus _status) pure returns (Claim out_) {
    assembly {
        out_ := or(and(not(shl(248, 0xFF)), _claim), shl(248, _status))
    }
}
