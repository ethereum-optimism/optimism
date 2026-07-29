// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { IOptimismMintableERC721FactoryLegacyMapping } from
    "interfaces/L2/IOptimismMintableERC721FactoryLegacyMapping.sol";

import { IProxyAdminOwnedBase } from "interfaces/universal/IProxyAdminOwnedBase.sol";

interface IOptimismMintableERC721Factory is IOptimismMintableERC721FactoryLegacyMapping, IProxyAdminOwnedBase {
    event OptimismMintableERC721Created(address indexed localToken, address indexed remoteToken, address deployer);

    event Initialized(uint8 version);

    function BRIDGE() external view returns (address);
    function REMOTE_CHAIN_ID() external view returns (uint256);
    function bridge() external view returns (address);
    function createOptimismMintableERC721(
        address _remoteToken,
        string memory _name,
        string memory _symbol
    )
        external
        returns (address);
    function remoteChainID() external view returns (uint256);
    function version() external view returns (string memory);

    function initialize(address _bridge, uint256 _remoteChainID) external;
}
