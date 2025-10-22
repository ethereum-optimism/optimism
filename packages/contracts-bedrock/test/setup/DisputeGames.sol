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
import { LibGameArgs } from "src/dispute/lib/LibGameArgs.sol";

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

    enum GameArg {
        PRESTATE,
        VM,
        ASR,
        WETH,
        L2_CHAIN_ID,
        PROPOSER,
        CHALLENGER
    }

    /// @notice Thrown when an unsupported game arg is provided
    error DisputeGames_UnsupportedGameArg(GameArg gameArg);

    function gameArgsOffset(GameArg _gameArg) internal pure returns (uint256) {
        if (_gameArg == GameArg.PRESTATE) return 0;
        if (_gameArg == GameArg.VM) return 32;
        if (_gameArg == GameArg.ASR) return 52;
        if (_gameArg == GameArg.WETH) return 72;
        if (_gameArg == GameArg.L2_CHAIN_ID) return 92;
        if (_gameArg == GameArg.PROPOSER) return 124;
        if (_gameArg == GameArg.CHALLENGER) return 144;

        revert DisputeGames_UnsupportedGameArg(_gameArg);
    }

    function permissionedGameChallenger(IDisputeGameFactory _dgf) internal returns (address challenger_) {
        GameType gameType = GameTypes.PERMISSIONED_CANNON;
        (bool gameArgsExist, bytes memory gameArgsData) = _getGameArgs(_dgf, gameType);
        if (gameArgsExist) {
            LibGameArgs.GameArgs memory gameArgs = LibGameArgs.decode(gameArgsData);
            challenger_ = gameArgs.challenger;
        } else {
            challenger_ = IPermissionedDisputeGame(address(_dgf.gameImpls(gameType))).challenger();
        }
    }

    function permissionedGameProposer(IDisputeGameFactory _dgf) internal returns (address proposer_) {
        GameType gameType = GameTypes.PERMISSIONED_CANNON;
        (bool gameArgsExist, bytes memory gameArgsData) = _getGameArgs(_dgf, gameType);
        if (gameArgsExist) {
            LibGameArgs.GameArgs memory gameArgs = LibGameArgs.decode(gameArgsData);
            proposer_ = gameArgs.proposer;
        } else {
            proposer_ = IPermissionedDisputeGame(address(_dgf.gameImpls(gameType))).proposer();
        }
    }

    function mockGameImplPrestate(IDisputeGameFactory _dgf, GameType _gameType, bytes32 _prestate) internal {
        if (isDevFeatureEnabled(DevFeatures.DEPLOY_V2_DISPUTE_GAMES)) {
            bytes memory value = abi.encodePacked(_prestate);
            _mockGameArg(_dgf, _gameType, GameArg.PRESTATE, value);
        } else {
            address gameAddr = address(_dgf.gameImpls(_gameType));
            vm.mockCall(gameAddr, abi.encodeCall(IFaultDisputeGame.absolutePrestate, ()), abi.encode(_prestate));
        }
    }

    function mockGameImplVM(IDisputeGameFactory _dgf, GameType _gameType, address _vm) internal {
        if (isDevFeatureEnabled(DevFeatures.DEPLOY_V2_DISPUTE_GAMES)) {
            bytes memory value = abi.encodePacked(_vm);
            _mockGameArg(_dgf, _gameType, GameArg.VM, value);
        } else {
            address gameAddr = address(_dgf.gameImpls(_gameType));
            vm.mockCall(gameAddr, abi.encodeCall(IFaultDisputeGame.vm, ()), abi.encode(_vm));
        }
    }

    function mockGameImplASR(IDisputeGameFactory _dgf, GameType _gameType, address _asr) internal {
        if (isDevFeatureEnabled(DevFeatures.DEPLOY_V2_DISPUTE_GAMES)) {
            bytes memory value = abi.encodePacked(_asr);
            _mockGameArg(_dgf, _gameType, GameArg.ASR, value);
        } else {
            address gameAddr = address(_dgf.gameImpls(_gameType));
            vm.mockCall(gameAddr, abi.encodeCall(IFaultDisputeGame.anchorStateRegistry, ()), abi.encode(_asr));
        }
    }

    function mockGameImplWeth(IDisputeGameFactory _dgf, GameType _gameType, address _weth) internal {
        if (isDevFeatureEnabled(DevFeatures.DEPLOY_V2_DISPUTE_GAMES)) {
            bytes memory value = abi.encodePacked(_weth);
            _mockGameArg(_dgf, _gameType, GameArg.WETH, value);
        } else {
            address gameAddr = address(_dgf.gameImpls(_gameType));
            vm.mockCall(gameAddr, abi.encodeCall(IFaultDisputeGame.weth, ()), abi.encode(_weth));
        }
    }

    function mockGameImplL2ChainId(IDisputeGameFactory _dgf, GameType _gameType, uint256 _chainId) internal {
        if (isDevFeatureEnabled(DevFeatures.DEPLOY_V2_DISPUTE_GAMES)) {
            bytes memory value = abi.encodePacked(_chainId);
            _mockGameArg(_dgf, _gameType, GameArg.L2_CHAIN_ID, value);
        } else {
            address gameAddr = address(_dgf.gameImpls(_gameType));
            vm.mockCall(gameAddr, abi.encodeCall(IFaultDisputeGame.l2ChainId, ()), abi.encode(_chainId));
        }
    }

    function mockGameImplProposer(IDisputeGameFactory _dgf, GameType _gameType, address _proposer) internal {
        if (isDevFeatureEnabled(DevFeatures.DEPLOY_V2_DISPUTE_GAMES)) {
            bytes memory value = abi.encodePacked(_proposer);
            _mockGameArg(_dgf, _gameType, GameArg.PROPOSER, value);
        } else {
            address gameAddr = address(_dgf.gameImpls(_gameType));
            vm.mockCall(gameAddr, abi.encodeCall(IPermissionedDisputeGame.proposer, ()), abi.encode(_proposer));
        }
    }

    function mockGameImplChallenger(IDisputeGameFactory _dgf, GameType _gameType, address _challenger) internal {
        if (isDevFeatureEnabled(DevFeatures.DEPLOY_V2_DISPUTE_GAMES)) {
            bytes memory value = abi.encodePacked(_challenger);
            _mockGameArg(_dgf, _gameType, GameArg.CHALLENGER, value);
        } else {
            address gameAddr = address(_dgf.gameImpls(_gameType));
            vm.mockCall(gameAddr, abi.encodeCall(IPermissionedDisputeGame.challenger, ()), abi.encode(_challenger));
        }
    }

    function _getGameArgs(
        IDisputeGameFactory _dgf,
        GameType _gameType
    )
        private
        returns (bool gameArgsExist_, bytes memory gameArgs_)
    {
        if (!isDevFeatureEnabled(DevFeatures.DEPLOY_V2_DISPUTE_GAMES)) {
            return (false, gameArgs_);
        }

        bytes memory gameArgsCallData = abi.encodeCall(IDisputeGameFactory.gameArgs, (_gameType));
        (bool success, bytes memory gameArgs) = address(_dgf).call(gameArgsCallData);

        gameArgsExist_ = success && gameArgs.length > 0;
        gameArgs_ = gameArgsExist_ ? gameArgs : bytes("");
    }

    function _mockGameArg(
        IDisputeGameFactory _dgf,
        GameType _gameType,
        GameArg _gameArg,
        bytes memory _value
    )
        private
    {
        bytes memory modifiedGameArgs = _dgf.gameArgs(_gameType);
        uint256 offset = gameArgsOffset(_gameArg);
        modifiedGameArgs.overwriteAtOffset(offset, _value);

        vm.mockCall(
            address(_dgf), abi.encodeCall(IDisputeGameFactory.gameArgs, (_gameType)), abi.encode(modifiedGameArgs)
        );
    }
}
