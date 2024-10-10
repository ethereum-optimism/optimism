// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { ProxyAdmin } from "src/universal/ProxyAdmin.sol";
import { Constants } from "src/libraries/Constants.sol";

contract L2ProxyAdmin is ProxyAdmin {
    constructor() ProxyAdmin(Constants.DEPOSITOR_ACCOUNT) { }
}
