// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { IERC1155 } from "@openzeppelin/contracts/token/ERC1155/IERC1155.sol";

/// @title IOptimismMintableERC1155
/// @notice Interface for contracts that are compatible with the OptimismMintableERC1155 standard.
///         Tokens that follow this standard can be easily transferred across the ERC1155 bridge.
interface IOptimismMintableERC1155 is IERC1155 {
    /// @notice Emitted when tokens are minted.
    /// @param account Address of the account the token were minted to.
    /// @param id      Type ID of the minted tokens.
    /// @param value   Amount of tokens minted.
    event Mint(address indexed account, uint256 id, uint256 value);

    /// @notice Equivalent to multiple Mint events for the same account.
    /// @param account Address of the account the tokens were minted to.
    /// @param ids     Type IDs of the minted tokens.
    /// @param values  Amounts of tokens minted.
    event MintBatch(address indexed account, uint256[] ids, uint256[] values);

    /// @notice Emitted when tokens are burned.
    /// @param account Address of the account the tokens were burned from.
    /// @param id      Type ID of the burned tokens.
    /// @param value   Amount of tokens burned.
    event Burn(address indexed account, uint256 id, uint256 value);

    /// @notice Equivalent to multiple Burn events for the same account.
    /// @param account Address of the account the tokens were burned from.
    /// @param ids     Type IDs of the burned tokens.
    /// @param values  Amounts of tokens burned.
    event BurnBatch(address indexed account, uint256[] ids, uint256[] values);

    /// @notice Mints an amount of a token type to a user
    /// @param _to     Address of the user to mint the token to.
    /// @param _id     Type ID of the tokens to mint.
    /// @param _amount Amount of tokens to mint.
    function mint(address _to, uint256 _id, uint256 _amount) external;

    /// @notice Batch version of mint. Mints multiple amounts of multiple token types to a user
    /// @param _to      Address of the user to mint the token to.
    /// @param _ids     Type IDs of the tokens to mint.
    /// @param _amounts Amounts of tokens to mint.
    function mintBatch(address _to, uint256[] memory _ids, uint256[] memory _amounts) external;

    /// @notice Burns an amount of a token type from a user
    /// @param _from  Address of the user to burn the token from.
    /// @param _id    Type ID of the tokens to burn.
    /// @param _amount Amount of tokens to burn.
    function burn(address _from, uint256 _id, uint256 _amount) external;

    /// @notice Batch version of Burn. Burns multiple amounts of multiple token types from a user
    /// @param _from  Address of the user to burn the token from.
    /// @param _ids    Type IDs of the tokens to burn.
    /// @param _amounts Amounts of tokens to burn.
    function burnBatch(address _from, uint256[] memory _ids, uint256[] memory _amounts) external;

    /// @notice Chain ID of the chain where the remote token is deployed.
    function REMOTE_CHAIN_ID() external view returns (uint256);

    /// @notice Address of the token on the remote domain.
    function REMOTE_TOKEN() external view returns (address);

    /// @notice Address of the ERC1155 bridge on this network.
    function BRIDGE() external view returns (address);

    /// @notice Chain ID of the chain where the remote token is deployed.
    function remoteChainId() external view returns (uint256);

    /// @notice Address of the token on the remote domain.
    function remoteToken() external view returns (address);

    /// @notice Address of the ERC1155 bridge on this network.
    function bridge() external view returns (address);
}
