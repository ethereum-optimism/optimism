// SPDX-License-Identifier: Apache-2.0
pragma solidity 0.8.24;

import { IKailuaTournament } from "interfaces/dispute/zk/IKailuaTournament.sol";
import { Claim, Duration, Hash, GameStatus, GameType, Timestamp } from "src/dispute/lib/Types.sol";

interface IKailuaGame is IKailuaTournament {
    function GENESIS_TIME_STAMP() external view returns (uint64);
    function L2_BLOCK_TIME() external view returns (uint64);
    function MAX_CLOCK_DURATION() external view returns (Duration);
    function duplicationCounter() external pure returns (uint64 duplicationCounter_);
    function parentGameIndex() external pure returns (uint64 parentGameIndex_);
}
