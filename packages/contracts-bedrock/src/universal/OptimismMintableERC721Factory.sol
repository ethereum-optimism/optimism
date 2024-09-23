// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { OptimismMintableERC721 } from "src/universal/OptimismMintableERC721.sol";
import { ISemver } from "src/universal/interfaces/ISemver.sol";
import { Initializable } from "@openzeppelin/contracts/proxy/utils/Initializable.sol";

/// @title OptimismMintableERC721Factory
/// @notice Factory contract for creating OptimismMintableERC721 contracts.
/// @custom:legacy true
///         While this contract is in the "universal" directory, it is not maintained as part
///         of the L1 deployments. This contract needs to be modified to read from storage
///         rather than use immutables to be deployed on L1 safely due to allow for all
///         L2 networks to use the same implementation.
contract OptimismMintableERC721Factory is ISemver {
    /// @notice Address of the ERC721 bridge on this network.
    address internal bridge;

    /// @custom:legacy true
    /// @notice Chain ID for the remote network.
    uint256 internal remoteChainId;

    /// @notice Tracks addresses created by this factory.
    mapping(address => bool) public isOptimismMintableERC721;

    /// @notice Emitted whenever a new OptimismMintableERC721 contract is created.
    /// @param localToken  Address of the token on the this domain.
    /// @param remoteToken Address of the token on the remote domain.
    /// @param deployer    Address of the initiator of the deployment
    event OptimismMintableERC721Created(address indexed localToken, address indexed remoteToken, address deployer);

    /// @notice Semantic version.
    /// @custom:semver 1.4.1-beta.3
    string public constant version = "1.4.1-beta.3";

    /// @notice The semver MUST be bumped any time that there is a change in
    ///         the OptimismMintableERC721 token contract since this contract
    ///         is responsible for deploying OptimismMintableERC721 contracts.
    /// @param _bridge Address of the ERC721 bridge on this network.
    /// @param _remoteChainId Chain ID for the remote network.
    constructor() {
        __disableInitializers();
    }

    function initialize(address _bridge, uint256 _remoteChainId) public initializer {
        bridge = _bridge;
        remoteChainId = _remoteChainId;
    }

    /// @notice
    function REMOTE_CHAIN_ID() external view returns (uint256) {
        return remoteChainId;
    }

    /// @notice
    function remoteChainId() external virtual view returns (uint256) {
        return remoteChainId;
    }

    /// @notice
    function BRIDGE() external virtual view returns (address) {
        return bridge;
    }

    /// @notice
    function bridge() external virtual view returns (address) {
        return bridge;
    }

    /// @notice Address of the ERC721 bridge on this network.
    function bridge() external view returns (address) {
        return BRIDGE;
    }

    /// @notice Chain ID for the remote network.
    function remoteChainID() external view returns (uint256) {
        return REMOTE_CHAIN_ID;
    }

    /// @notice Creates an instance of the standard ERC721.
    /// @param _remoteToken Address of the corresponding token on the other domain.
    /// @param _name        ERC721 name.
    /// @param _symbol      ERC721 symbol.
    function createOptimismMintableERC721(
        address _remoteToken,
        string memory _name,
        string memory _symbol
    )
        external
        returns (address)
    {
        require(_remoteToken != address(0), "OptimismMintableERC721Factory: L1 token address cannot be address(0)");

        bytes32 salt = keccak256(abi.encode(_remoteToken, _name, _symbol));
        address localToken =
            address(new OptimismMintableERC721{ salt: salt }(BRIDGE, REMOTE_CHAIN_ID, _remoteToken, _name, _symbol));

        isOptimismMintableERC721[localToken] = true;
        emit OptimismMintableERC721Created(localToken, _remoteToken, msg.sender);

        return localToken;
    }
}
