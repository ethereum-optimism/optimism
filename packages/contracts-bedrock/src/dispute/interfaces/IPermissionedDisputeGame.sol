// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { IFaultDisputeGame } from "src/dispute/interfaces/IFaultDisputeGame.sol";

interface IPermissionedDisputeGame is IFaultDisputeGame {
    error BadAuth();

    function proposer() external view returns (address proposer_);
    function challenger() external view returns (address challenger_);
}
