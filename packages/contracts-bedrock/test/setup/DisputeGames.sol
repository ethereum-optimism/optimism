// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { FeatureFlags } from "./FeatureFlags.sol";
import { ByteUtils } from "./ByteUtils.sol";
import { Vm } from "forge-std/Vm.sol";
import { console2 as console } from "forge-std/console2.sol";

// Libraries
import { GameType, Claim } from "src/dispute/lib/LibUDT.sol";
import { GameTypes } from "src/dispute/lib/Types.sol";
import { DevFeatures } from "src/libraries/DevFeatures.sol";

// Interfaces
import "../../interfaces/dispute/IDisputeGame.sol";
import "../../interfaces/dispute/IDisputeGameFactory.sol";
import { IFaultDisputeGame } from "../../interfaces/dispute/IFaultDisputeGame.sol";
import { IPermissionedDisputeGame } from "../../interfaces/dispute/IPermissionedDisputeGame.sol";

contract DisputeGames is FeatureFlags {
    using ByteUtils for bytes;

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

    function mockGameImplPrestate(IDisputeGameFactory _dgf, GameType _gameType, bytes32 _prestate) internal {
        if (isDevFeatureEnabled(DevFeatures.DEPLOY_V2_DISPUTE_GAMES)) {
            uint256 offset = 0;
            bytes memory value = abi.encodePacked(_prestate);
            _mockGameArg(_dgf, _gameType, offset, value);
        } else {
            address gameAddr = address(_dgf.gameImpls(_gameType));
            vm.mockCall(gameAddr, abi.encodeCall(IFaultDisputeGame.absolutePrestate, ()), abi.encode(_prestate));
        }
    }

    function mockGameImplVM(IDisputeGameFactory _dgf, GameType _gameType, address _vm) internal {
        if (isDevFeatureEnabled(DevFeatures.DEPLOY_V2_DISPUTE_GAMES)) {
            uint256 offset = 32;
            bytes memory value = abi.encodePacked(_vm);
            _mockGameArg(_dgf, _gameType, offset, value);
        } else {
            address gameAddr = address(_dgf.gameImpls(_gameType));
            vm.mockCall(gameAddr, abi.encodeCall(IFaultDisputeGame.vm, ()), abi.encode(_vm));
        }
    }

    function mockGameImplASR(IDisputeGameFactory _dgf, GameType _gameType, address _asr) internal {
        if (isDevFeatureEnabled(DevFeatures.DEPLOY_V2_DISPUTE_GAMES)) {
            uint256 offset = 52;
            bytes memory value = abi.encodePacked(_asr);
            _mockGameArg(_dgf, _gameType, offset, value);
        } else {
            address gameAddr = address(_dgf.gameImpls(_gameType));
            vm.mockCall(gameAddr, abi.encodeCall(IFaultDisputeGame.anchorStateRegistry(), ()), abi.encode(_asr));
        }
    }

    function mockGameImplWeth(IDisputeGameFactory _dgf, GameType _gameType, address _weth) internal {
        if (isDevFeatureEnabled(DevFeatures.DEPLOY_V2_DISPUTE_GAMES)) {
            uint256 offset = 72;
            bytes memory value = abi.encodePacked(_weth);
            _mockGameArg(_dgf, _gameType, offset, value);
        } else {
            address gameAddr = address(_dgf.gameImpls(_gameType));
            vm.mockCall(gameAddr, abi.encodeCall(IFaultDisputeGame.weth(), ()), abi.encode(_weth));
        }
    }

    function mockGameImplL2ChainId(IDisputeGameFactory _dgf, GameType _gameType, uint256 _chainId) internal {
        if (isDevFeatureEnabled(DevFeatures.DEPLOY_V2_DISPUTE_GAMES)) {
            uint256 offset = 92;
            bytes memory value = abi.encodePacked(_chainId);
            _mockGameArg(_dgf, _gameType, offset, value);
        } else {
            address gameAddr = address(_dgf.gameImpls(_gameType));
            vm.mockCall(gameAddr, abi.encodeCall(IFaultDisputeGame.l2ChainId, ()), abi.encode(_chainId));
        }
    }

    function mockGameImplProposer(IDisputeGameFactory _dgf, GameType _gameType, address _proposer) internal {
        if (isDevFeatureEnabled(DevFeatures.DEPLOY_V2_DISPUTE_GAMES)) {
            uint256 offset = 124;
            bytes memory value = abi.encodePacked(_proposer);
            _mockGameArg(_dgf, _gameType, offset, value);
        } else {
            address gameAddr = address(_dgf.gameImpls(_gameType));
            vm.mockCall(gameAddr, abi.encodeCall(IPermissionedDisputeGame.proposer, ()), abi.encode(_proposer));
        }
    }

    function mockGameImplChallenger(IDisputeGameFactory _dgf, GameType _gameType, address _challenger) internal {
        if (isDevFeatureEnabled(DevFeatures.DEPLOY_V2_DISPUTE_GAMES)) {
            uint256 offset = 144;
            bytes memory value = abi.encodePacked(_challenger);
            _mockGameArg(_dgf, _gameType, offset, value);
        } else {
            address gameAddr = address(_dgf.gameImpls(_gameType));
            vm.mockCall(gameAddr, abi.encodeCall(IPermissionedDisputeGame.challenger, ()), abi.encode(_challenger));
        }
    }

    function _mockGameArg(
        IDisputeGameFactory _dgf,
        GameType _gameType,
        uint256 _gameArgsOffset,
        bytes memory _value
    )
        private
    {
        bytes memory modifiedGameArgs = _dgf.gameArgs(_gameType);
        modifiedGameArgs.overwriteAtOffset(_gameArgsOffset, _value);
        vm.mockCall(
            address(_dgf), abi.encodeCall(IDisputeGameFactory.gameArgs, (_gameType)), abi.encode(modifiedGameArgs)
        );
    }
}
