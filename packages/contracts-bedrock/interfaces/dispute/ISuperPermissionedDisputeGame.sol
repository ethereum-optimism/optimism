// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { IDisputeGame } from "interfaces/dispute/IDisputeGame.sol";
import { IAnchorStateRegistry } from "interfaces/dispute/IAnchorStateRegistry.sol";
import { ISuperFaultDisputeGame } from "interfaces/dispute/ISuperFaultDisputeGame.sol";
import { ISemver } from "interfaces/universal/ISemver.sol";
import { Claim, Duration } from "src/dispute/lib/Types.sol";

interface ISuperPermissionedDisputeGame is IDisputeGame, ISemver {
    error AlreadyInitialized();
    error BadAuth();
    error BadExtraData();
    error ClockNotExpired();
    error ClockTimeExceeded();
    error Encoding_EmptySuperRoot();
    error Encoding_InvalidSuperRootEncoding();
    error Encoding_InvalidSuperRootVersion();
    error GameNotInProgress();
    error UnknownChainId();

    function challenge() external;
    function anchorStateRegistry() external view returns (IAnchorStateRegistry registry_);
    function proposer() external pure returns (address proposer_);
    function challenger() external pure returns (address challenger_);
    function maxGameDepth() external view returns (uint256 maxGameDepth_);
    function splitDepth() external view returns (uint256 splitDepth_);
    function maxClockDuration() external view returns (Duration maxClockDuration_);
    function clockExtension() external view returns (Duration clockExtension_);
    function rootClaimByChainId(uint256 _chainId) external pure override returns (Claim outputRootClaim_);
    function __constructor__(ISuperFaultDisputeGame.GameConstructorParams memory _params) external;
}
