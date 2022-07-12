// SPDX-License-Identifier: MIT
pragma solidity ^0.8.9;

import {
    OwnableUpgradeable
} from "@openzeppelin/contracts-upgradeable/access/OwnableUpgradeable.sol";
import { OptimismMintableERC721 } from "./OptimismMintableERC721.sol";
import { Semver } from "@eth-optimism/contracts-bedrock/contracts/universal/Semver.sol";

/**
 * @title OptimismMintableERC721Factory
 * @notice Factory contract for creating OptimismMintableERC721 contracts.
 */
contract OptimismMintableERC721Factory is Semver, OwnableUpgradeable {
    /**
     * @notice Emitted whenever a new OptimismMintableERC721 contract is created.
     *
     * @param remoteToken Address of the token on the remote domain.
     * @param localToken  Address of the token on the this domain.
     */
    event OptimismMintableERC721Created(address indexed remoteToken, address indexed localToken);

    /**
     * @notice Address of the ERC721 bridge on this network.
     */
    address public bridge;

    /**
     * @notice Chain ID for the remote network.
     */
    uint256 public remoteChainId;

    /**
     * @notice Tracks addresses created by this factory.
     */
    mapping(address => bool) public isStandardOptimismMintableERC721;

    /**
     * @custom:semver 0.0.1
     *
     * @param _bridge Address of the ERC721 bridge on this network.
     */
    constructor(address _bridge, uint256 _remoteChainId) Semver(0, 0, 1) {
        initialize(_bridge, _remoteChainId);
    }

    /**
     * @notice Initializes the factory.
     *
     * @param _bridge Address of the ERC721 bridge on this network.
     */
    function initialize(address _bridge, uint256 _remoteChainId) public initializer {
        bridge = _bridge;
        remoteChainId = _remoteChainId;

        // Initialize upgradable OZ contracts
        __Ownable_init();
    }

    /**
     * @notice Creates an instance of the standard ERC721.
     *
     * @param _remoteToken Address of the corresponding token on the other domain.
     * @param _name        ERC721 name.
     * @param _symbol      ERC721 symbol.
     */
    function createStandardOptimismMintableERC721(
        address _remoteToken,
        string memory _name,
        string memory _symbol
    ) external {
        require(
            _remoteToken != address(0),
            "OptimismMintableERC721Factory: L1 token address cannot be address(0)"
        );

        require(
            bridge != address(0),
            "OptimismMintableERC721Factory: bridge address must be initialized"
        );

        OptimismMintableERC721 localToken = new OptimismMintableERC721(
            bridge,
            remoteChainId,
            _remoteToken,
            _name,
            _symbol
        );

        isStandardOptimismMintableERC721[address(localToken)] = true;
        emit OptimismMintableERC721Created(_remoteToken, address(localToken));
    }
}
