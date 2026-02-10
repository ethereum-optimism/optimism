// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { ByteUtils } from "./ByteUtils.sol";
import { Vm } from "forge-std/Vm.sol";
import { console2 as console } from "forge-std/console2.sol";

// Libraries
import { GameType, Claim } from "src/dispute/lib/LibUDT.sol";
import { GameTypes } from "src/dispute/lib/Types.sol";
import { LibGameArgs } from "src/dispute/lib/LibGameArgs.sol";

// Interfaces
import "../../interfaces/dispute/IDisputeGame.sol";
import "../../interfaces/dispute/IDisputeGameFactory.sol";
import { IFaultDisputeGame } from "../../interfaces/dispute/IFaultDisputeGame.sol";
import { IPermissionedDisputeGame } from "../../interfaces/dispute/IPermissionedDisputeGame.sol";

library DisputeGames {
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
        return createGame(_factory, _gameType, _proposer, _claim, abi.encode(bytes32(_l2BlockNumber)));
    }

    /// @notice Helper function to create a permissioned game through the factory
    function createGame(
        IDisputeGameFactory _factory,
        GameType _gameType,
        address _proposer,
        Claim _claim,
        bytes memory _extraData
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

        IDisputeGame gameProxy = _factory.create{ value: initBond }(_gameType, _claim, _extraData);

        vm.stopPrank();

        return address(gameProxy);
    }

    function isGamePermissioned(GameType _gameType) internal pure returns (bool) {
        return _gameType.raw() == GameTypes.PERMISSIONED_CANNON.raw()
            || _gameType.raw() == GameTypes.SUPER_PERMISSIONED_CANNON.raw();
    }

    /// @notice Checks if the game type is a super game type
    function isSuperGame(GameType _gameType) internal pure returns (bool) {
        return _gameType.raw() == GameTypes.SUPER_PERMISSIONED_CANNON.raw()
            || _gameType.raw() == GameTypes.SUPER_CANNON.raw() || _gameType.raw() == GameTypes.SUPER_CANNON_KONA.raw();
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

    function permissionedGameChallenger(IDisputeGameFactory _dgf) internal view returns (address challenger_) {
        GameType gameType = GameTypes.PERMISSIONED_CANNON;
        (bool gameArgsExist, bytes memory gameArgsData) = _getGameArgs(_dgf, gameType);
        if (gameArgsExist) {
            LibGameArgs.GameArgs memory gameArgs = LibGameArgs.decode(gameArgsData);
            challenger_ = gameArgs.challenger;
        } else {
            challenger_ = IPermissionedDisputeGame(address(_dgf.gameImpls(gameType))).challenger();
        }
    }

    function permissionedGameProposer(IDisputeGameFactory _dgf) internal view returns (address proposer_) {
        GameType gameType = GameTypes.PERMISSIONED_CANNON;
        (bool gameArgsExist, bytes memory gameArgsData) = _getGameArgs(_dgf, gameType);
        if (gameArgsExist) {
            LibGameArgs.GameArgs memory gameArgs = LibGameArgs.decode(gameArgsData);
            proposer_ = gameArgs.proposer;
        } else {
            proposer_ = IPermissionedDisputeGame(address(_dgf.gameImpls(gameType))).proposer();
        }
    }

    /// @notice Gets the absolute prestate for a game type, handling both v1 and v2 dispute games.
    ///         V1 games store the prestate on the game implementation, v2 games store it in gameArgs.
    ///         Returns Claim.wrap(bytes32(0)) if no implementation exists for the game type.
    /// @param _dgf The dispute game factory.
    /// @param _gameType The game type to get the prestate for.
    /// @return prestate_ The absolute prestate claim.
    function getGameImplPrestate(
        IDisputeGameFactory _dgf,
        GameType _gameType
    )
        internal
        view
        returns (Claim prestate_)
    {
        // Return zero if no implementation exists for this game type
        address gameImpl = address(_dgf.gameImpls(_gameType));
        if (gameImpl == address(0)) {
            return Claim.wrap(bytes32(0));
        }

        (bool gameArgsExist, bytes memory gameArgsData) = _getGameArgs(_dgf, _gameType);
        if (gameArgsExist) {
            LibGameArgs.GameArgs memory gameArgs = LibGameArgs.decode(gameArgsData);
            prestate_ = Claim.wrap(gameArgs.absolutePrestate);
        } else {
            prestate_ = IFaultDisputeGame(gameImpl).absolutePrestate();
        }
    }

    function mockGameImplPrestate(IDisputeGameFactory _dgf, GameType _gameType, bytes32 _prestate) internal {
        bytes memory value = abi.encodePacked(_prestate);
        _mockGameArg(_dgf, _gameType, GameArg.PRESTATE, value);
    }

    function mockGameImplVM(IDisputeGameFactory _dgf, GameType _gameType, address _vm) internal {
        bytes memory value = abi.encodePacked(_vm);
        _mockGameArg(_dgf, _gameType, GameArg.VM, value);
    }

    function mockGameImplASR(IDisputeGameFactory _dgf, GameType _gameType, address _asr) internal {
        bytes memory value = abi.encodePacked(_asr);
        _mockGameArg(_dgf, _gameType, GameArg.ASR, value);
    }

    function mockGameImplWeth(IDisputeGameFactory _dgf, GameType _gameType, address _weth) internal {
        bytes memory value = abi.encodePacked(_weth);
        _mockGameArg(_dgf, _gameType, GameArg.WETH, value);
    }

    function mockGameImplL2ChainId(IDisputeGameFactory _dgf, GameType _gameType, uint256 _chainId) internal {
        bytes memory value = abi.encodePacked(_chainId);
        _mockGameArg(_dgf, _gameType, GameArg.L2_CHAIN_ID, value);
    }

    function mockGameImplProposer(IDisputeGameFactory _dgf, GameType _gameType, address _proposer) internal {
        bytes memory value = abi.encodePacked(_proposer);
        _mockGameArg(_dgf, _gameType, GameArg.PROPOSER, value);
    }

    function mockGameImplChallenger(IDisputeGameFactory _dgf, GameType _gameType, address _challenger) internal {
        bytes memory value = abi.encodePacked(_challenger);
        _mockGameArg(_dgf, _gameType, GameArg.CHALLENGER, value);
    }

    function _getGameArgs(
        IDisputeGameFactory _dgf,
        GameType _gameType
    )
        private
        view
        returns (bool gameArgsExist_, bytes memory gameArgs_)
    {
        // Safe from issues with EIP150 since this is only used in the testing environment.
        // eip150-safe
        try _dgf.gameArgs(_gameType) returns (bytes memory gameArgsRet_) {
            gameArgsExist_ = gameArgsRet_.length > 0;
            gameArgs_ = gameArgsRet_;
        } catch {
            gameArgsExist_ = false;
            gameArgs_ = bytes("");
        }
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
