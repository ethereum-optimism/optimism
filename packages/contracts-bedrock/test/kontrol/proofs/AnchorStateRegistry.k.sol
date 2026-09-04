// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { GameStatus } from "src/dispute/lib/Types.sol";
import { IDisputeGame } from "interfaces/dispute/IDisputeGame.sol";
import { IDisputeGameFactory } from "interfaces/dispute/IDisputeGameFactory.sol";
import { ISystemConfig } from "interfaces/L1/ISystemConfig.sol";
import { KontrolUtils } from "./utils/KontrolUtils.sol";
import {
    AirgapFactory_Harness,
    AirgapGame_Harness,
    AirgapSystemConfig_Harness,
    AnchorStateRegistryAirgap_Harness
} from "./utils/AirgapHarnesses.sol";

/// @notice Proves the AnchorStateRegistry portion of the withdrawal air gap using the production
///         implementation. The harness fixes only registration, respect, retirement, blacklist,
///         and pause to their valid values; status, resolution time, and L1 time remain symbolic.
contract AnchorStateRegistryKontrol is KontrolUtils {
    uint256 internal constant GAME_FINALITY_DELAY = 7 days;

    function prove_validClaimRequiresDefenderAndMatureResolution(
        uint64 _now,
        uint64 _resolvedAt,
        uint8 _status
    )
        external
    {
        vm.assume(_now >= _resolvedAt);
        vm.assume(_status <= uint8(GameStatus.DEFENDER_WINS));

        (AnchorStateRegistryAirgap_Harness registry, AirgapGame_Harness game) = _setup(_resolvedAt, GameStatus(_status));
        vm.warp(_now);

        bool valid = registry.isGameClaimValid(IDisputeGame(address(game)));
        if (valid) {
            assert(_status == uint8(GameStatus.DEFENDER_WINS));
            assert(_resolvedAt != 0);
            assert(uint256(_now) - uint256(_resolvedAt) > GAME_FINALITY_DELAY);
        }
    }

    function prove_challengerClaimNeverValid(uint64 _now, uint64 _resolvedAt) external {
        vm.assume(_resolvedAt > 0);
        vm.assume(_now >= _resolvedAt);
        (AnchorStateRegistryAirgap_Harness registry, AirgapGame_Harness game) =
            _setup(_resolvedAt, GameStatus.CHALLENGER_WINS);
        vm.warp(_now);

        assert(!registry.isGameClaimValid(IDisputeGame(address(game))));
    }

    function _setup(
        uint64 _resolvedAt,
        GameStatus _status
    )
        internal
        returns (AnchorStateRegistryAirgap_Harness registry_, AirgapGame_Harness game_)
    {
        game_ = new AirgapGame_Harness();
        game_.setState(1, _resolvedAt, _status);
        AirgapFactory_Harness factory = new AirgapFactory_Harness(IDisputeGame(address(game_)));
        AirgapSystemConfig_Harness systemConfig = new AirgapSystemConfig_Harness();
        registry_ = new AnchorStateRegistryAirgap_Harness(GAME_FINALITY_DELAY);
        registry_.configure(ISystemConfig(address(systemConfig)), IDisputeGameFactory(address(factory)));
    }
}
