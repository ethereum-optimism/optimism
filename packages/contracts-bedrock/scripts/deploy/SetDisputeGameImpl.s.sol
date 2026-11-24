// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Script } from "forge-std/Script.sol";

import { IFaultDisputeGame } from "interfaces/dispute/IFaultDisputeGame.sol";
import { IDisputeGameFactory } from "interfaces/dispute/IDisputeGameFactory.sol";
import { DeployState } from "scripts/libraries/DeployState.sol";
import { GameType } from "src/dispute/lib/Types.sol";
import { IAnchorStateRegistry } from "interfaces/dispute/IAnchorStateRegistry.sol";

contract SetDisputeGameImpl is Script {
    struct Input {
        IDisputeGameFactory factory;
        IFaultDisputeGame impl;
        IAnchorStateRegistry anchorStateRegistry;
        uint32 gameType;
        bytes gameArgs;
    }

    function runWithBytes(bytes memory _input) public returns (bytes memory) {
        DeployState.load();
        run(abi.decode(_input, (Input)));
        DeployState.dump(msg.sender);
        return "";
    }

    function run(Input memory _input) public {
        require(address(_input.factory) != address(0), "SetDisputeGameImpl: factory not set");
        require(address(_input.impl) != address(0), "SetDisputeGameImpl: impl not set");

        // A rerun after a partial deployment finds its own registration, and sends nothing for it.
        GameType gameType = GameType.wrap(_input.gameType);
        address current = address(_input.factory.gameImpls(gameType));
        if (current == address(0)) {
            if (_input.gameArgs.length > 0) {
                vm.broadcast(msg.sender);
                _input.factory.setImplementation(gameType, _input.impl, _input.gameArgs);
            } else {
                vm.broadcast(msg.sender);
                _input.factory.setImplementation(gameType, _input.impl);
            }
        } else {
            require(
                current == address(_input.impl)
                    && keccak256(_input.factory.gameArgs(gameType)) == keccak256(_input.gameArgs),
                "SDGI-10"
            );
        }

        if (address(_input.anchorStateRegistry) != address(0)) {
            require(address(_input.anchorStateRegistry.disputeGameFactory()) == address(_input.factory), "SDGI-20");
            if (GameType.unwrap(_input.anchorStateRegistry.respectedGameType()) != _input.gameType) {
                vm.broadcast(msg.sender);
                _input.anchorStateRegistry.setRespectedGameType(gameType);
            }
        }

        assertValid(_input);
    }

    function assertValid(Input memory _input) public view {
        GameType gameType = GameType.wrap(_input.gameType);
        require(address(_input.factory.gameImpls(gameType)) == address(_input.impl), "SDGI-30");

        if (address(_input.anchorStateRegistry) != address(0)) {
            require(address(_input.anchorStateRegistry.disputeGameFactory()) == address(_input.factory), "SDGI-40");
            require(
                GameType.unwrap(_input.anchorStateRegistry.respectedGameType()) == GameType.unwrap(gameType), "SDGI-50"
            );
        }
    }
}
