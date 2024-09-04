// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

/// @title IL1ChugSplashProxy
/// @notice Interface for the L1ChugSplashProxy contract.
interface IL1ChugSplashProxy {
    fallback() external payable;

    receive() external payable;

    function getImplementation() external returns (address);
    function getOwner() external returns (address);
    function setCode(bytes memory _code) external;
    function setOwner(address _owner) external;
    function setStorage(bytes32 _key, bytes32 _value) external;
}
