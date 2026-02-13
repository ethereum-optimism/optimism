// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { ICrossDomainMessenger } from "interfaces/universal/ICrossDomainMessenger.sol";
import { IProxyAdminOwnedBase } from "interfaces/universal/IProxyAdminOwnedBase.sol";

interface IL2CrossDomainMessenger is ICrossDomainMessenger, IProxyAdminOwnedBase {
    function MESSAGE_VERSION() external view returns (uint16);
    function initialize(ICrossDomainMessenger _l1CrossDomainMessenger) external;
    function l1CrossDomainMessenger() external view returns (ICrossDomainMessenger);
    function version() external view returns (string memory);

    function __constructor__() external;
}
