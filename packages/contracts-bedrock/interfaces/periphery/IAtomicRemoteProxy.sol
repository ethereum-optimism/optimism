// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

interface IAtomicRemoteProxy {
    function __constructor__(address _router, uint256 _chainId, address _target) external;
    fallback() external;
}
