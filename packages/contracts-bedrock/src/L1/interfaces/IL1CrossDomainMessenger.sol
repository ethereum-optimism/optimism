// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { ICrossDomainMessenger } from "src/universal/interfaces/ICrossDomainMessenger.sol";
import { ISemver } from "src/universal/interfaces/ISemver.sol";

/// @title IL1CrossDomainMessenger
/// @notice Interface for the L1CrossDomainMessenger contract.
interface IL1CrossDomainMessenger is ICrossDomainMessenger, ISemver {
    function PORTAL() external view returns (address);
    function initialize(address _superchainConfig, address _portal, address _systemConfig) external;
    function portal() external view returns (address);
    function superchainConfig() external view returns (address);
    function systemConfig() external view returns (address);
    function version() external view returns (string memory);
}
