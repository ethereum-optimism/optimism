// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { IProxyAdmin } from "interfaces/universal/IProxyAdmin.sol";

interface IProxyAdminOwnedBase {
    function proxyAdmin() external view returns (IProxyAdmin);
    function proxyAdminOwner() external view returns (address);
}
