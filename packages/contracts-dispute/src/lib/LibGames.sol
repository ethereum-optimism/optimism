// SPDX-License-Identifier: MIT
pragma solidity ^0.8.17;

import { GameType } from "src/types/Types.sol";

/// @title LibGames
/// @author refcell <https://github.com/refcell>
/// @notice This library contains constants for the different game types.
library LibGames {
    /// @notice A FaultGameType is a dispute game that uses a fault proof to verify claims.
    GameType constant FaultGameType = GameType.wrap(bytes32(abi.encodePacked("Fault")));

    /// @notice A ValidityGameType uses a validity proof to verify claims.
    GameType constant ValidityGameType = GameType.wrap(bytes32(abi.encodePacked("Validity")));

    /// @notice An AttestationGameType is a permissioned set of attestors who verify claims.
    GameType constant AttestationGameType = GameType.wrap(bytes32(abi.encodePacked("Attestation")));
}
