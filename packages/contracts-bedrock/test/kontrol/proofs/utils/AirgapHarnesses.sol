// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { AnchorStateRegistry } from "src/dispute/AnchorStateRegistry.sol";
import { Claim, GameStatus, GameType, Timestamp } from "src/dispute/lib/Types.sol";
import { IDisputeGame } from "interfaces/dispute/IDisputeGame.sol";
import { IDisputeGameFactory } from "interfaces/dispute/IDisputeGameFactory.sol";
import { ISystemConfig } from "interfaces/L1/ISystemConfig.sol";

contract AirgapGame_Harness {
    Timestamp public createdAt;
    Timestamp public resolvedAt;
    GameStatus public status;
    bool public wasRespectedGameTypeWhenCreated;

    function setState(uint64 _createdAt, uint64 _resolvedAt, GameStatus _status) external {
        createdAt = Timestamp.wrap(_createdAt);
        resolvedAt = Timestamp.wrap(_resolvedAt);
        status = _status;
        wasRespectedGameTypeWhenCreated = true;
    }

    function gameData() external pure returns (GameType, Claim, bytes memory) {
        return (GameType.wrap(1), Claim.wrap(bytes32(uint256(1))), bytes(""));
    }
}

contract AirgapFactory_Harness {
    IDisputeGame internal immutable GAME;

    constructor(IDisputeGame _game) {
        GAME = _game;
    }

    function games(GameType, Claim, bytes memory) external view returns (IDisputeGame, Timestamp) {
        return (GAME, Timestamp.wrap(1));
    }

    function gameAtIndex(uint256) external view returns (GameType, Timestamp, IDisputeGame) {
        return (GameType.wrap(1), Timestamp.wrap(1), GAME);
    }
}

contract AirgapSystemConfig_Harness {
    function paused() external pure returns (bool) {
        return false;
    }

    function isFeatureEnabled(bytes32) external pure returns (bool) {
        return false;
    }
}

contract AnchorStateRegistryAirgap_Harness is AnchorStateRegistry {
    constructor(uint256 _delay) AnchorStateRegistry(_delay) { }

    function configure(ISystemConfig _systemConfig, IDisputeGameFactory _factory) external {
        systemConfig = _systemConfig;
        disputeGameFactory = _factory;
        retirementTimestamp = 0;
    }
}
