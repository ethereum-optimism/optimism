// SPDX-License-Identifier: MIT
pragma solidity ^0.8.9;

import {
    OwnableUpgradeable
} from "@openzeppelin/contracts-upgradeable/access/OwnableUpgradeable.sol";
import { OptimismMintableERC721 } from "./OptimismMintableERC721.sol";

/**
 * @title OptimismMintableERC721Factory
 * @notice Factory contract for creating OptimismMintableERC721 contracts.
 */
contract OptimismMintableERC721Factory is OwnableUpgradeable {
    /**
     * @notice Contract version number.
     */
    uint8 public constant VERSION = 1;

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
     * @notice Tracks addresses created by this factory.
     */
    mapping(address => bool) public isStandardOptimismMintableERC721;

    /**
     * @param _bridge Address of the ERC721 bridge on this network.
     */
    constructor(address _bridge) {
        initialize(_bridge);
    }

    /**
     * @notice Initializes the factory.
     *
     * @param _bridge Address of the ERC721 bridge on this network.
     */
    function initialize(address _bridge) public reinitializer(VERSION) {
        bridge = _bridge;

        // Initialize upgradable OZ contracts
        __Ownable_init();
    }

    /**
     * @notice Creates an instance of the standard ERC721.
     *
     * @param _remoteToken Address of the corresponding token on the other domain.
     * @param _name ERC721 name.
     * @param _symbol ERC721 symbol.
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
            _remoteToken,
            _name,
            _symbol
        );

        isStandardOptimismMintableERC721[address(localToken)] = true;
        emit OptimismMintableERC721Created(_remoteToken, address(localToken));
    }
}
