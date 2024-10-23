// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { IWETH } from "src/universal/interfaces/IWETH.sol";
import { ICrosschainERC20 } from "src/L2/interfaces/ICrosschainERC20.sol";
import { ISemver } from "src/universal/interfaces/ISemver.sol";

interface ISuperchainWETH is IWETH, ICrosschainERC20, ISemver {
    error Unauthorized();
    error NotCustomGasToken();

    function __constructor__() external;
}
