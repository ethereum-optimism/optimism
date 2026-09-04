// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

interface ISuperchainETHBridgePinned {
    error InsufficientNetFlow();
    error InvalidCrossDomainSender();
    error NotHomeChain();
    error Unauthorized();
    error ZeroAddress();

    event RelayETH(address indexed from, address indexed to, uint256 amount, uint256 source);
    event SendETH(address indexed from, address indexed to, uint256 amount, uint256 destination);

    function homeChainId() external view returns (uint256);
    function netSent(uint256) external view returns (uint256);
    function relayETH(address _from, address _to, uint256 _amount) external;
    function sendETH(address _to, uint256 _chainId) external payable returns (bytes32 msgHash_);
    function version() external view returns (string memory);

    function __constructor__() external;
}
