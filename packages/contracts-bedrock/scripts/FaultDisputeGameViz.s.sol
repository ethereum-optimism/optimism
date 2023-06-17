pragma solidity ^0.8.15;

import { Script } from "forge-std/Script.sol";
import { console2 as console } from "forge-std/console2.sol";

import { DisputeGameFactory_Init } from "../contracts/test/DisputeGameFactory.t.sol";
import { DisputeGameFactory } from "../contracts/dispute/DisputeGameFactory.sol";
import { FaultDisputeGame } from "../contracts/dispute/FaultDisputeGame.sol";
import { IFaultDisputeGame } from "../contracts/dispute/interfaces/IFaultDisputeGame.sol";

import "../contracts/libraries/DisputeTypes.sol";
import "../contracts/libraries/DisputeErrors.sol";
import { LibClock } from "../contracts/dispute/lib/LibClock.sol";
import { LibPosition } from "../contracts/dispute/lib/LibPosition.sol";

/**
 * @title FaultDisputeGameViz
 * @dev To run this script, make sure to install the `dagviz` & `eth_abi` python packages.
 */
contract FaultDisputeGameViz is Script, DisputeGameFactory_Init {
    /**
     * @dev The root claim of the game.
     */
    Claim internal constant ROOT_CLAIM = Claim.wrap(bytes32(uint256(10)));
    /**
     * @dev The extra data passed to the game for initialization.
     */
    bytes internal constant EXTRA_DATA = abi.encode(1);
    /**
     * @dev The type of the game being tested.
     */
    GameType internal constant GAME_TYPE = GameType.wrap(0);
    /**
     * @dev The implementation of the game.
     */
    FaultDisputeGame internal gameImpl;
    /**
     * @dev The `Clone` proxy of the game.
     */
    FaultDisputeGame internal gameProxy;

    function setUp() public override {
        super.setUp();
        // Deploy an implementation of the fault game
        gameImpl = new FaultDisputeGame();
        // Register the game implementation with the factory.
        factory.setImplementation(GAME_TYPE, gameImpl);
        // Create a new game.
        gameProxy = FaultDisputeGame(address(factory.create(GAME_TYPE, ROOT_CLAIM, EXTRA_DATA)));

        // Label the proxy
        vm.label(address(gameProxy), "FaultDisputeGame_Clone");
    }

    /**
     * @dev Entry point
     */
    function run() public {
        // Construct the game by performing attacks, defenses, and steps.
        // ...

        buildGraph();
        console.log("Saved graph to `./dispute_game.svg");
    }

    /**
     * @dev Uses the `dag-viz` python script to generate a visual model of the game state.
     */
    function buildGraph() internal {
        uint256 numClaims = uint256(vm.load(address(gameProxy), bytes32(uint256(1))));
        IFaultDisputeGame.ClaimData[] memory gameData = new IFaultDisputeGame.ClaimData[](numClaims);
        for (uint256 i = 0; i < numClaims; i++) {
            (
                uint32 parentIndex,
                bool countered,
                Claim claim,
                Position position,
                Clock clock
            ) = gameProxy.claimData(i);

            gameData[i] = IFaultDisputeGame.ClaimData({
                parentIndex: parentIndex,
                countered: countered,
                claim: claim,
                position: position,
                clock: clock
            });
        }

        string[] memory commands = new string[](3);
        commands[0] = "python3";
        commands[1] = "scripts/dag-viz.py";
        commands[2] = vm.toString(abi.encode(gameData));
        vm.ffi(commands);
    }
}
