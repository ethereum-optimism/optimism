// SPDX-License-Identifier: MIT
pragma solidity ^0.8.25;

import { IOPContractsManagerUtils } from "interfaces/L1/opcm/IOPContractsManagerUtils.sol";

interface IOPContractsManagerUtilsCaller {
    function __constructor__(IOPContractsManagerUtils _utils) external;
    function opcmUtils() external view returns (IOPContractsManagerUtils);
}
