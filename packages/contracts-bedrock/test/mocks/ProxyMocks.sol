// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

/// @title MockEIP1967Proxy
/// @dev Mock implementation of an EIP1967 proxy.
contract MockEIP1967Proxy {
    /// @notice Address of the implementation.
    address public immutable implementationAddr;

    constructor(address _impl) {
        implementationAddr = _impl;
    }

    /// @notice Returns the implementation address.
    function implementation() external view returns (address) {
        return implementationAddr;
    }
}

/// @title MockL1ChugSplashProxy
/// @dev Mock implementation of an L1ChugSplash proxy.
contract MockL1ChugSplashProxy {
    /// @notice Address of the implementation.
    address public immutable implementationAddr;

    constructor(address _impl) {
        implementationAddr = _impl;
    }

    /// @notice Returns the implementation address.
    function getImplementation() external view returns (address) {
        return implementationAddr;
    }
}
