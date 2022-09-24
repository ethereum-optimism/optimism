// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import {
    OwnableUpgradeable
} from "@openzeppelin/contracts-upgradeable/access/OwnableUpgradeable.sol";
import { OptimismMintableERC721 } from "./OptimismMintableERC721.sol";
import { Semver } from "./Semver.sol";

/**
 * @title OptimismMintableERC721Factory
 * @notice Factory contract for creating OptimismMintableERC721 contracts.
 */
contract OptimismMintableERC721Factory is Semver, OwnableUpgradeable {
    /**
     * @notice Emitted whenever a new OptimismMintableERC721 contract is created.
     *
     * @param localToken  Address of the token on the this domain.
     * @param remoteToken Address of the token on the remote domain.
     */
    event OptimismMintableERC721Created(address indexed localToken, address indexed remoteToken);

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
     * @custom:semver 1.0.0
     *
     * @param _bridge Address of the ERC721 bridge on this network.
     */
    constructor(address _bridge, uint256 _remoteChainId) Semver(1, 0, 0) {
        initialize(_bridge, _remoteChainId);
    }

    /**
     * @notice Initializes the factory.
     *
     * @param _bridge Address of the ERC721 bridge on this network.
     */
    function initialize(address _bridge, uint256 _remoteChainId) public initializer {
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
        emit OptimismMintableERC721Created(address(localToken), _remoteToken);
    }
}
