// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { OptimismMintableERC721 } from "./OptimismMintableERC721.sol";
import { Semver } from "./Semver.sol";

/**
 * @title OptimismMintableERC721Factory
 * @notice Factory contract for creating OptimismMintableERC721 contracts.
 */
contract OptimismMintableERC721Factory is Semver {
    /**
     * @notice Emitted whenever a new OptimismMintableERC721 contract is created.
     *
     * @param localToken  Address of the token on the this domain.
     * @param remoteToken Address of the token on the remote domain.
     * @param deployer    Address of the initiator of the deployment
     */
    event OptimismMintableERC721Created(
        address indexed localToken,
        address indexed remoteToken,
        address deployer
    );

    /**
     * @notice Address of the ERC721 bridge on this network.
     */
    address public immutable bridge;

    /**
     * @notice Chain ID for the remote network.
     */
    uint256 public immutable remoteChainId;

    /**
     * @notice Tracks addresses created by this factory.
     */
    mapping(address => bool) public isOptimismMintableERC721;

    /**
     * @custom:semver 1.0.0
     *
     * @param _bridge Address of the ERC721 bridge on this network.
     */
    constructor(address _bridge, uint256 _remoteChainId) Semver(1, 0, 0) {
        require(
            _bridge != address(0),
            "OptimismMintableERC721Factory: bridge cannot be address(0)"
        );
        require(
            _remoteChainId != 0,
            "OptimismMintableERC721Factory: remote chain id cannot be zero"
        );

        bridge = _bridge;
        remoteChainId = _remoteChainId;
    }

    /**
     * @notice Creates an instance of the standard ERC721.
     *
     * @param _remoteToken Address of the corresponding token on the other domain.
     * @param _name        ERC721 name.
     * @param _symbol      ERC721 symbol.
     */
    function createOptimismMintableERC721(
        address _remoteToken,
        string memory _name,
        string memory _symbol
    ) external returns (address) {
        require(
            _remoteToken != address(0),
            "OptimismMintableERC721Factory: L1 token address cannot be address(0)"
        );

        address localToken = address(
            new OptimismMintableERC721(bridge, remoteChainId, _remoteToken, _name, _symbol)
        );

        isOptimismMintableERC721[localToken] = true;
        emit OptimismMintableERC721Created(localToken, _remoteToken, msg.sender);

        return localToken;
    }
}
