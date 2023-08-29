// SPDX-License-Identifier: LGPL-3.0-only
pragma solidity ^0.8.15;

interface IFreezer {
    function isFrozen(address) external view returns (bool);
}
