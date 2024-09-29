// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { OptimismMintableERC721 } from "src/universal/OptimismMintableERC721.sol";
import { ISemver } from "src/universal/interfaces/ISemver.sol";
import { Initializable } from "@openzeppelin/contracts/proxy/utils/Initializable.sol";
import { IL1Block } from "src/L2/interfaces/IL1Block.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";

/// @title OptimismMintableERC721Factory
/// @notice Factory contract for creating OptimismMintableERC721 contracts.
///         This contract could in theory live on both L1 and L2 but it is not widely
///         used and is therefore set up to work on L2. This could be abstracted in the
///         future to be deployable on L1 as well.
contract OptimismMintableERC721Factory is ISemver {
    // TODO: check storage layout
    /// @notice Tracks addresses created by this factory.
    mapping(address => bool) public isOptimismMintableERC721;

    /// @notice Emitted whenever a new OptimismMintableERC721 contract is created.
    /// @param localToken  Address of the token on the this domain.
    /// @param remoteToken Address of the token on the remote domain.
    /// @param deployer    Address of the initiator of the deployment
    event OptimismMintableERC721Created(address indexed localToken, address indexed remoteToken, address deployer);

    /// @notice Semantic version.
    /// @custom:semver 1.4.1-beta.3
    ///         The semver MUST be bumped any time that there is a change in
    ///         the OptimismMintableERC721 token contract since this contract
    ///         is responsible for deploying OptimismMintableERC721 contracts.
    /// @custom:semver 1.4.1-beta.2
    string public constant version = "1.4.1-beta.3";

    /// @notice TODO: call L1Block
    function REMOTE_CHAIN_ID() external view returns (uint256) {
        return remoteChainId();
    }

    /// @notice TODO: call L1Block
    function remoteChainId() public view returns (uint256) {
        return IL1Block(Predeploys.L1_BLOCK_ATTRIBUTES).remoteChainId();
    }

    /// @notice TODO: call L1Block
    function BRIDGE() external view returns (address) {
        return bridge();
    }

    /// @notice TODO: call L1Block
    function bridge() public view returns (address) {
        return IL1Block(Predeploys.L1_BLOCK_ATTRIBUTES).l1ERC721Bridge();
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
            address(new OptimismMintableERC721{ salt: salt }(bridge(), remoteChainId(), _remoteToken, _name, _symbol));

        isOptimismMintableERC721[localToken] = true;
        emit OptimismMintableERC721Created(localToken, _remoteToken, msg.sender);

        return localToken;
    }
}
