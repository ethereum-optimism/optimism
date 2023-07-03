// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { ERC1155 } from "@openzeppelin/contracts/token/ERC1155/ERC1155.sol";
import { IERC165 } from "@openzeppelin/contracts/utils/introspection/IERC165.sol";
import { IOptimismMintableERC1155 } from "./IOptimismMintableERC1155.sol";
import { Semver } from "../universal/Semver.sol";

/// @title OptimismMintableERC1155
/// @notice This contract is the remote representation for some token that lives on another network,
///         typically an Optimism representation of an Ethereum-based token. Standard reference
///         implementation that can be extended or modified according to your needs.
contract OptimismMintableERC1155 is ERC1155, IOptimismMintableERC1155, Semver {
    /// @inheritdoc IOptimismMintableERC1155
    uint256 public immutable REMOTE_CHAIN_ID;

    /// @inheritdoc IOptimismMintableERC1155
    address public immutable REMOTE_TOKEN;

    /// @inheritdoc IOptimismMintableERC1155
    address public immutable BRIDGE;

    /// @notice Modifier that prevents callers other than the bridge from calling the function.
    modifier onlyBridge() {
        require(
            msg.sender == BRIDGE,
            "OptimismMintableERC1155: only bridge can call this function"
        );
        _;
    }

    /// @custom:semver 1.0.0
    /// @param _bridge        Address of the bridge on this network.
    /// @param _remoteChainId Chain ID where the remote token is deployed.
    /// @param _remoteToken   Address of the corresponding token on the other network.
    /// @param _uri           ERC1155 uri.
    constructor(
        address _bridge,
        uint256 _remoteChainId,
        address _remoteToken,
        string memory _uri
    ) ERC1155(_uri) Semver(1, 0, 0) {
        require(_bridge != address(0), "OptimismMintableERC1155: bridge cannot be address(0)");
        require(_remoteChainId != 0, "OptimismMintableERC1155: remote chain id cannot be zero");
        require(
            _remoteToken != address(0),
            "OptimismMintableERC1155: remote token cannot be address(0)"
        );

        REMOTE_CHAIN_ID = _remoteChainId;
        REMOTE_TOKEN = _remoteToken;
        BRIDGE = _bridge;
    }

    /// @inheritdoc IOptimismMintableERC1155
    function remoteChainId() external view returns (uint256) {
        return REMOTE_CHAIN_ID;
    }

    /// @inheritdoc IOptimismMintableERC1155
    function remoteToken() external view returns (address) {
        return REMOTE_TOKEN;
    }

    /// @inheritdoc IOptimismMintableERC1155
    function bridge() external view returns (address) {
        return BRIDGE;
    }

    /// @inheritdoc IOptimismMintableERC1155
    function mint(address _to, uint256 _id, uint256 _amount) external virtual onlyBridge {
        _mint(_to, _id, _amount, "");

        emit Mint(_to, _id, _amount);
    }

    /// @inheritdoc IOptimismMintableERC1155
    function mintBatch(
        address _to,
        uint256[] memory _ids,
        uint256[] memory _amounts
    ) external virtual onlyBridge {
        _mintBatch(_to, _ids, _amounts, "");

        emit MintBatch(_to, _ids, _amounts);
    }

    /// @inheritdoc IOptimismMintableERC1155
    function burn(address _from, uint256 _id, uint256 _amount) external virtual onlyBridge {
        _burn(_from, _id, _amount);

        emit Burn(_from, _id, _amount);
    }

    /// @inheritdoc IOptimismMintableERC1155
    function burnBatch(
        address _from,
        uint256[] memory _ids,
        uint256[] memory _amounts
    ) external virtual onlyBridge {
        _burnBatch(_from, _ids, _amounts);

        emit BurnBatch(_from, _ids, _amounts);
    }

    /// @notice Checks if a given interface ID is supported by this contract.
    /// @param _interfaceId The interface ID to check.
    /// @return True if the interface ID is supported, false otherwise.
    function supportsInterface(
        bytes4 _interfaceId
    ) public view override(ERC1155, IERC165) returns (bool) {
        bytes4 iface = type(IOptimismMintableERC1155).interfaceId;
        return _interfaceId == iface || super.supportsInterface(_interfaceId);
    }
}
