// SPDX-License-Identifier: MIT
pragma solidity ^0.8.25;

interface IReinitializableBase {
    error ReinitializableBase_ZeroInitVersion();

    function initVersion() external view returns (uint8);

    // ReinitializerBase is abstract, so it has no constructor in its interface.
    function __constructor__() external;
}
