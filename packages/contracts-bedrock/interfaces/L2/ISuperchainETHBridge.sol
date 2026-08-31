// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { ISemver } from "interfaces/universal/ISemver.sol";
import { IProxyAdminOwnedBase } from "interfaces/universal/IProxyAdminOwnedBase.sol";

interface ISuperchainETHBridge is ISemver, IProxyAdminOwnedBase {
    error Unauthorized();
    error InvalidCrossDomainSender();
    error ZeroAddress();
    error SuperchainETHBridge_BannedSource();

    event SendETH(address indexed from, address indexed to, uint256 amount, uint256 destination);

    event RelayETH(address indexed from, address indexed to, uint256 amount, uint256 source);

    event BannedSourceSet(uint256 indexed chainId, bool banned);

    function bannedSources(uint256) external view returns (bool);
    function setBannedSource(uint256 _chainId, bool _banned) external;
    function sendETH(address _to, uint256 _chainId) external payable returns (bytes32 msgHash_);
    function relayETH(address _from, address _to, uint256 _amount) external;

    function __constructor__() external;
}
