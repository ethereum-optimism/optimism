// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import {
    ERC721Enumerable
} from "@openzeppelin/contracts/token/ERC721/extensions/ERC721Enumerable.sol";
import { ERC721 } from "@openzeppelin/contracts/token/ERC721/ERC721.sol";
import { IERC165 } from "@openzeppelin/contracts/utils/introspection/IERC165.sol";
import { Strings } from "@openzeppelin/contracts/utils/Strings.sol";
import { IOptimismMintableERC721 } from "./IOptimismMintableERC721.sol";
import { Semver } from "../universal/Semver.sol";

/**
 * @title OptimismMintableERC721
 * @notice This contract is the remote representation for some token that lives on another network,
 *         typically an Optimism representation of an Ethereum-based token. Standard reference
 *         implementation that can be extended or modified according to your needs.
 */
contract OptimismMintableERC721 is ERC721Enumerable, IOptimismMintableERC721, Semver {
    /**
     * @inheritdoc IOptimismMintableERC721
     */
    uint256 public immutable REMOTE_CHAIN_ID;

    /**
     * @inheritdoc IOptimismMintableERC721
     */
    address public immutable REMOTE_TOKEN;

    /**
     * @inheritdoc IOptimismMintableERC721
     */
    address public immutable BRIDGE;

    /**
     * @notice Base token URI for this token.
     */
    string public baseTokenURI;

    /**
     * @notice Modifier that prevents callers other than the bridge from calling the function.
     */
    modifier onlyBridge() {
        require(msg.sender == BRIDGE, "OptimismMintableERC721: only bridge can call this function");
        _;
    }

    /**
     * @custom:semver 1.1.0
     *
     * @param _bridge        Address of the bridge on this network.
     * @param _remoteChainId Chain ID where the remote token is deployed.
     * @param _remoteToken   Address of the corresponding token on the other network.
     * @param _name          ERC721 name.
     * @param _symbol        ERC721 symbol.
     */
    constructor(
        address _bridge,
        uint256 _remoteChainId,
        address _remoteToken,
        string memory _name,
        string memory _symbol
    ) ERC721(_name, _symbol) Semver(1, 1, 0) {
        require(_bridge != address(0), "OptimismMintableERC721: bridge cannot be address(0)");
        require(_remoteChainId != 0, "OptimismMintableERC721: remote chain id cannot be zero");
        require(
            _remoteToken != address(0),
            "OptimismMintableERC721: remote token cannot be address(0)"
        );

        REMOTE_CHAIN_ID = _remoteChainId;
        REMOTE_TOKEN = _remoteToken;
        BRIDGE = _bridge;

        // Creates a base URI in the format specified by EIP-681:
        // https://eips.ethereum.org/EIPS/eip-681
        baseTokenURI = string(
            abi.encodePacked(
                "ethereum:",
                Strings.toHexString(uint160(_remoteToken), 20),
                "@",
                Strings.toString(_remoteChainId),
                "/tokenURI?uint256="
            )
        );
    }

    /**
     * @inheritdoc IOptimismMintableERC721
     */
    function remoteChainId() external view returns (uint256) {
        return REMOTE_CHAIN_ID;
    }

    /**
     * @inheritdoc IOptimismMintableERC721
     */
    function remoteToken() external view returns (address) {
        return REMOTE_TOKEN;
    }

    /**
     * @inheritdoc IOptimismMintableERC721
     */
    function bridge() external view returns (address) {
        return BRIDGE;
    }

    /**
     * @inheritdoc IOptimismMintableERC721
     */
    function safeMint(address _to, uint256 _tokenId) external virtual onlyBridge {
        _safeMint(_to, _tokenId);

        emit Mint(_to, _tokenId);
    }

    /**
     * @inheritdoc IOptimismMintableERC721
     */
    function burn(address _from, uint256 _tokenId) external virtual onlyBridge {
        _burn(_tokenId);

        emit Burn(_from, _tokenId);
    }

    /**
     * @notice Checks if a given interface ID is supported by this contract.
     *
     * @param _interfaceId The interface ID to check.
     *
     * @return True if the interface ID is supported, false otherwise.
     */
    function supportsInterface(bytes4 _interfaceId)
        public
        view
        override(ERC721Enumerable, IERC165)
        returns (bool)
    {
        bytes4 iface = type(IOptimismMintableERC721).interfaceId;
        return _interfaceId == iface || super.supportsInterface(_interfaceId);
    }

    /**
     * @notice Returns the base token URI.
     *
     * @return Base token URI.
     */
    function _baseURI() internal view virtual override returns (string memory) {
        return baseTokenURI;
    }
}
