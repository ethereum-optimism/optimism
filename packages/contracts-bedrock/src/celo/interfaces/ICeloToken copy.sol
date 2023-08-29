// SPDX-License-Identifier: LGPL-3.0-only
pragma solidity ^0.8.15;

/**
 * @title This interface describes the non- ERC20 shared interface for all Celo Tokens, and
 * in the absence of interface inheritance is intended as a companion to IERC20.sol.
 */
interface ICeloToken {
    function transferWithComment(address, uint256, string calldata) external returns (bool);
    function name() external view returns (string memory);
    function symbol() external view returns (string memory);
    function decimals() external view returns (uint8);
    function burn(uint256 value) external returns (bool);
}
