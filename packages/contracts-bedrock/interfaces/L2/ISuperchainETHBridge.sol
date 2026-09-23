// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { ISemver } from "interfaces/universal/ISemver.sol";

interface ISuperchainETHBridge is ISemver {
    error Unauthorized();
    error InvalidCrossDomainSender();
    error ZeroAddress();
    error SuperchainETHBridge_SendChainNotAllowed();
    error SuperchainETHBridge_RelayChainNotAllowed();

    event ChainPermissionsUpdated(uint256 indexed chainId, bool allowSend, bool allowRelay);

    function allowedSendChain(uint256) external view returns (bool);
    function allowedRelayChain(uint256) external view returns (bool);
    function setChainPermissions(uint256 _chainId, bool _allowSend, bool _allowRelay) external;

    event SendETH(address indexed from, address indexed to, uint256 amount, uint256 destination);

    event RelayETH(address indexed from, address indexed to, uint256 amount, uint256 source);

    function sendETH(address _to, uint256 _chainId) external payable returns (bytes32 msgHash_);
    function relayETH(address _from, address _to, uint256 _amount) external;

    function __constructor__() external;
}
