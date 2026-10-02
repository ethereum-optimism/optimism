// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { ICrossDomainMessenger } from "interfaces/universal/ICrossDomainMessenger.sol";
import { IProxyAdminOwnedBase } from "interfaces/universal/IProxyAdminOwnedBase.sol";

interface IL2CrossDomainMessenger is ICrossDomainMessenger, IProxyAdminOwnedBase {
    function MESSAGE_VERSION() external view returns (uint16);
    function LOCAL_RELAY_GAS_CHECK_BUFFER() external view returns (uint64);
    function LOCAL_RELAY_RESERVED_GAS() external view returns (uint64);
    function RELAY_CALL_OVERHEAD() external view returns (uint64);
    function RELAY_CONSTANT_OVERHEAD() external view returns (uint64);
    function RELAY_GAS_CHECK_BUFFER() external view returns (uint64);
    function RELAY_RESERVED_GAS() external view returns (uint64);
    function REMOTE_RELAY_CALL_OVERHEAD() external view returns (uint64);
    function REMOTE_RELAY_CONSTANT_OVERHEAD() external view returns (uint64);
    function REMOTE_RELAY_GAS_CHECK_BUFFER() external view returns (uint64);
    function REMOTE_RELAY_RESERVED_GAS() external view returns (uint64);
    function initialize(ICrossDomainMessenger _l1CrossDomainMessenger) external;
    function l1CrossDomainMessenger() external view returns (ICrossDomainMessenger);
    function version() external view returns (string memory);

    function __constructor__() external;
}
