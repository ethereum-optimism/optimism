// SPDX-License-Identifier: MIT
pragma solidity >0.5.0;
pragma experimental ABIEncoderV2;

/**
 * @title iL1LiquidityPool
 */
interface iL1LiquidityPool {

    /********************
     *       Events     *
     ********************/

    event ownerAddERC20Liquidity_EVENT(
        address sender,
        uint256 amount,
        address erc20ContractAddress
    );

    event ownerRecoverFee_EVENT(
        address sender,
        address receiver,
        address erc20ContractAddress,
        uint256 amount
    );

    event clientDepositL1_EVENT(
        address sender,
        uint256 amount,
        uint256 fee,
        address erc20ContractL1Address,
        address erc20ContractL2Address
    );

    event clientPayL1_EVENT(
        address sender,
        uint256 amount,
        address erc20ContractAddress
    );

    /*************************
     * Cross-chain Functions *
     *************************/

    function clientPayL1(
        address payable _to,
        uint256 _amount,
        address _erc20ContractAddress
    )
        external;
}
