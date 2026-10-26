// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Script } from "forge-std/Script.sol";

import { IFaultDisputeGame } from "interfaces/dispute/IFaultDisputeGame.sol";
import { IDisputeGameFactory } from "interfaces/dispute/IDisputeGameFactory.sol";
import { BaseDeployIO } from "scripts/deploy/BaseDeployIO.sol";
import { DeployState } from "scripts/libraries/DeployState.sol";
import { GameType } from "src/dispute/lib/Types.sol";
import { IAnchorStateRegistry } from "interfaces/dispute/IAnchorStateRegistry.sol";

contract SetDisputeGameImplInput is BaseDeployIO {
    IDisputeGameFactory internal _factory;
    IAnchorStateRegistry internal _anchorStateRegistry;
    IFaultDisputeGame internal _impl;
    uint32 internal _gameType;
    bytes internal _gameArgs;

    // Setter for address type
    function set(bytes4 _sel, address _addr) public {
        require(_addr != address(0), "SetDisputeGameImplInput: cannot set zero address");

        if (_sel == this.factory.selector) _factory = IDisputeGameFactory(_addr);
        else if (_sel == this.anchorStateRegistry.selector) _anchorStateRegistry = IAnchorStateRegistry(_addr);
        else if (_sel == this.impl.selector) _impl = IFaultDisputeGame(_addr);
        else revert("SetDisputeGameImplInput: unknown selector");
    }

    // Setter for GameType
    function set(bytes4 _sel, uint32 _type) public {
        if (_sel == this.gameType.selector) _gameType = _type;
        else revert("SetDisputeGameImplInput: unknown selector");
    }

    function set(bytes4 _sel, bytes memory _value) public {
        if (_sel == this.gameArgs.selector) _gameArgs = bytes(_value);
        else revert("SetDisputeGameImplInput: unknown selector");
    }

    function factory() public view returns (IDisputeGameFactory) {
        require(address(_factory) != address(0), "SetDisputeGameImplInput: not set");
        return _factory;
    }

    function anchorStateRegistry() public view returns (IAnchorStateRegistry) {
        return _anchorStateRegistry;
    }

    function impl() public view returns (IFaultDisputeGame) {
        require(address(_impl) != address(0), "SetDisputeGameImplInput: not set");
        return _impl;
    }

    function gameType() public view returns (uint32) {
        return _gameType;
    }

    function gameArgs() public view returns (bytes memory) {
        return _gameArgs;
    }
}

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

    function run(SetDisputeGameImplInput _input) public {
        run(
            Input({
                factory: _input.factory(),
                impl: _input.impl(),
                anchorStateRegistry: _input.anchorStateRegistry(),
                gameType: _input.gameType(),
                gameArgs: _input.gameArgs()
            })
        );
    }

    function run(Input memory _input) public {
        require(address(_input.factory) != address(0), "SetDisputeGameImpl: factory not set");
        require(address(_input.impl) != address(0), "SetDisputeGameImpl: impl not set");

        GameType gameType = GameType.wrap(_input.gameType);
        require(address(_input.factory.gameImpls(gameType)) == address(0), "SDGI-10");

        if (_input.gameArgs.length > 0) {
            vm.broadcast(msg.sender);
            _input.factory.setImplementation(gameType, _input.impl, _input.gameArgs);
        } else {
            vm.broadcast(msg.sender);
            _input.factory.setImplementation(gameType, _input.impl);
        }

        if (address(_input.anchorStateRegistry) != address(0)) {
            require(address(_input.anchorStateRegistry.disputeGameFactory()) == address(_input.factory), "SDGI-20");
            vm.broadcast(msg.sender);
            _input.anchorStateRegistry.setRespectedGameType(gameType);
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
