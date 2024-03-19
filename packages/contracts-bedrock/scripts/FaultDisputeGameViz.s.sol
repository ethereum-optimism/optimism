// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

import { Script } from "forge-std/Script.sol";
import { console2 as console } from "forge-std/console2.sol";

import { FaultDisputeGame_Init } from "test/dispute/FaultDisputeGame.t.sol";
import { DisputeGameFactory } from "src/dispute/DisputeGameFactory.sol";
import { FaultDisputeGame } from "src/dispute/FaultDisputeGame.sol";
import { IFaultDisputeGame } from "src/dispute/interfaces/IFaultDisputeGame.sol";

import "src/libraries/DisputeTypes.sol";
import "src/libraries/DisputeErrors.sol";
import { LibPosition } from "src/dispute/lib/LibPosition.sol";

/**
 * @title FaultDisputeGameViz
 * @dev To run this script, make sure to install the `dagviz` & `eth_abi` python packages.
 */
contract FaultDisputeGameViz is Script, FaultDisputeGame_Init {
    /// @dev The root claim of the game.
    Claim internal constant ROOT_CLAIM = Claim.wrap(bytes32(uint256(1)));
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

    /**
     * @dev Entry point
     */
    function local() public {
        // Construct the game by performing attacks, defenses, and steps.
        // ...

        buildGraph();
        console.log("Saved graph to `./dispute_game.svg");
    }

    /**
     * @dev Entry point
     */
    function remote(address _addr) public {
        gameProxy = FaultDisputeGame(_addr);
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
                address countered,
                address claimant,
                uint128 bond,
                Claim claim,
                Position position,
                Clock clock
            ) = gameProxy.claimData(i);

            gameData[i] = IFaultDisputeGame.ClaimData({
                parentIndex: parentIndex,
                counteredBy: countered,
                claimant: claimant,
                bond: bond,
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
