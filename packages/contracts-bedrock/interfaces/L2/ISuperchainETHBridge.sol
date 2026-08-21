// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { ISemver } from "interfaces/universal/ISemver.sol";

interface ISuperchainETHBridge is ISemver {
    error Unauthorized();
    error InvalidCrossDomainSender();
    error ZeroAddress();
    error NoPendingSend();

    event SendETH(address indexed from, address indexed to, uint256 amount, uint256 destination);

    event RelayETH(address indexed from, address indexed to, uint256 amount, uint256 source);

    event RefundETH(address indexed from, uint256 amount, bytes32 indexed msgHash);

    function sendETH(address _to, uint256 _chainId) external payable returns (bytes32 msgHash_);
    function relayETH(address _from, address _to, uint256 _amount) external;
    function pendingETHSends(bytes32) external view returns (address from, uint256 amount);
    function onMessageExpired(bytes32 _msgHash, uint256, uint256) external;

    function __constructor__() external;
}
