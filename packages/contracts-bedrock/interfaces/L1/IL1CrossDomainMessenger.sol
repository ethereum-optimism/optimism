// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { ICrossDomainMessenger } from "interfaces/universal/ICrossDomainMessenger.sol";
import { ISystemConfig } from "interfaces/L1/ISystemConfig.sol";
import { IOptimismPortal2 as IOptimismPortal } from "interfaces/L1/IOptimismPortal2.sol";
import { ISuperchainConfig } from "interfaces/L1/ISuperchainConfig.sol";
import { IProxyAdminOwnedBase } from "interfaces/universal/IProxyAdminOwnedBase.sol";

interface IL1CrossDomainMessenger is ICrossDomainMessenger, IProxyAdminOwnedBase {
    error ReinitializableBase_ZeroInitVersion();
    error L1CrossDomainMessenger_InteropNotEnabled();
    error L1CrossDomainMessenger_NotInteropMessenger();
    error L1CrossDomainMessenger_Paused();

    function PORTAL() external view returns (IOptimismPortal);
    function initialize(ISystemConfig _systemConfig, IOptimismPortal _portal) external;
    function initVersion() external view returns (uint8);
    function portal() external view returns (IOptimismPortal);
    function systemConfig() external view returns (ISystemConfig);
    function version() external view returns (string memory);
    function superchainConfig() external view returns (ISuperchainConfig);
    function relayUndeliveredMessage(bytes32 _messageHash, uint256 _undeliveredAt) external;

    function __constructor__() external;
}
