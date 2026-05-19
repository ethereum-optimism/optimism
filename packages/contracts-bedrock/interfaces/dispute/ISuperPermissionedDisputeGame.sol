// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { IDisputeGame } from "interfaces/dispute/IDisputeGame.sol";
import { IAnchorStateRegistry } from "interfaces/dispute/IAnchorStateRegistry.sol";
import { ISemver } from "interfaces/universal/ISemver.sol";
import { Claim } from "src/dispute/lib/Types.sol";

interface ISuperPermissionedDisputeGame is IDisputeGame, ISemver {
    error AlreadyInitialized();
    error BadAuth();
    error BadExtraData();
    error Encoding_EmptySuperRoot();
    error Encoding_InvalidSuperRootEncoding();
    error Encoding_InvalidSuperRootVersion();
    error UnknownChainId();

    function anchorStateRegistry() external view returns (IAnchorStateRegistry registry_);
    function proposer() external pure returns (address proposer_);
    function rootClaimByChainId(uint256 _chainId) external pure override returns (Claim outputRootClaim_);
    function __constructor__() external;
}
