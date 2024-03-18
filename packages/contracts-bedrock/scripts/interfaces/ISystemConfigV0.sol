// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { ResourceMetering } from "src/L1/ResourceMetering.sol";

/// @title ISystemConfigV0
/// @notice Minimal interface of the Legacy SystemConfig containing only getters.
///         Based on
/// https://github.com/ethereum-optimism/optimism/blob/f54a2234f2f350795552011f35f704a3feb56a08/packages/contracts-bedrock/src/L1/SystemConfig.sol
interface ISystemConfigV0 {
    function owner() external view returns (address);
    function VERSION() external view returns (uint256);
    function overhead() external view returns (uint256);
    function scalar() external view returns (uint256);
    function batcherHash() external view returns (bytes32);
    function gasLimit() external view returns (uint64);
    function resourceConfig() external view returns (ResourceMetering.ResourceConfig memory);
    function unsafeBlockSigner() external view returns (address);
}
