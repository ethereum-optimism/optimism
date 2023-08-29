// SPDX-License-Identifier: LGPL-3.0-only
pragma solidity ^0.8.15;

import "../FixidityLib.sol";

interface IFeeHandler {
    // sets the portion of the fee that should be burned.
    function setBurnFraction(uint256 fraction) external;

    function addToken(address tokenAddress, address handlerAddress) external;
    function removeToken(address tokenAddress) external;

    function setHandler(address tokenAddress, address handlerAddress) external;

    // marks token to be handled in "handleAll())
    function activateToken(address tokenAddress) external;
    function deactivateToken(address tokenAddress) external;

    function sell(address tokenAddress) external;

    // calls exchange(tokenAddress), and distribute(tokenAddress)
    function handle(address tokenAddress) external;

    // main entrypoint for a burn, iterates over token and calles handle
    function handleAll() external;

    // Sends the balance of token at tokenAddress to feesBeneficiary,
    // according to the entry tokensToDistribute[tokenAddress]
    function distribute(address tokenAddress) external;

    // burns the balance of Celo in the contract minus the entry of tokensToDistribute[CeloAddress]
    function burnCelo() external;

    // calls distribute for all the nonCeloTokens
    function distributeAll() external;

    // in case some funds need to be returned or moved to another contract
    function transfer(address token, address recipient, uint256 value) external returns (bool);
}
