// SPDX-License-Identifier: GPL-3
pragma solidity ^0.8.24;

import { ProtocolHandler } from "../handlers/Protocol.t.sol";
import { EnumerableMap } from "@openzeppelin/contracts/utils/structs/EnumerableMap.sol";

contract HandlerGetters is ProtocolHandler {
    using EnumerableMap for EnumerableMap.Bytes32ToUintMap;

    function deploySaltsLength() external view returns (uint256 length) {
        return ghost_totalSupplyAcrossChains.length();
    }

    function totalSupplyAcrossChainsAtIndex(uint256 index) external view returns (bytes32 salt, uint256 supply) {
        return ghost_totalSupplyAcrossChains.at(index);
    }

    function tokensInTransitForDeploySalt(bytes32 salt) external view returns (uint256 amount) {
        (, amount) = ghost_tokensInTransit.tryGet(salt);
        return amount;
    }
}
