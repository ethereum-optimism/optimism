// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

interface ISystemConfigInterop {
    function addDependency(uint256 _chainId) external;
    function removeDependency(uint256 _chainId) external;
}
