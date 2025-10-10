// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { Vm } from "forge-std/Vm.sol";
import { console2 as console } from "forge-std/console2.sol";

// Libraries
import { GameType, Claim } from "src/dispute/lib/LibUDT.sol";

// Interfaces
import "../../interfaces/dispute/IDisputeGame.sol";
import "../../interfaces/dispute/IDisputeGameFactory.sol";

contract DisputeGames {
    /// @notice The address of the foundry Vm contract.
    Vm private constant vm = Vm(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);

    /// @notice Helper function to create a permissioned game through the factory
    function createGame(
        IDisputeGameFactory _factory,
        GameType _gameType,
        address _proposer,
        Claim _claim,
        uint256 _l2BlockNumber
    )
        internal
        returns (address)
    {
        // Check if there's an init bond required for the game type
        uint256 initBond = _factory.initBonds(_gameType);
        console.log("Init bond", initBond);

        // Fund the proposer if needed
        if (initBond > 0) {
            vm.deal(_proposer, initBond);
        }

        // We use vm.startPrank to set both msg.sender and tx.origin to the proposer
        vm.startPrank(_proposer, _proposer);

        IDisputeGame gameProxy =
            _factory.create{ value: initBond }(_gameType, _claim, abi.encode(bytes32(_l2BlockNumber)));

        vm.stopPrank();

        return address(gameProxy);
    }
}
