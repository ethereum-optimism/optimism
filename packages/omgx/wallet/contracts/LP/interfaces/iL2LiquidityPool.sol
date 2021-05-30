// SPDX-License-Identifier: MIT
pragma solidity >0.5.0;
pragma experimental ABIEncoderV2;

/**
 * @title iL2LiquidityPool
 */
interface iL2LiquidityPool {

    /********************
     *       Event      *
     ********************/
    
    event AddLiquidity(
        address sender,
        uint256 amount,
        address tokenAddress
    );

    event OwnerRecoverFee(
        address sender,
        address receiver,
        uint256 amount,
        address tokenAddress
    );

    event ClientDepositL2(
        address sender,
        uint256 receivedAmount,
        uint256 userRewardFee,
        uint256 ownerRewardFee,
        uint256 totalFee,
        address tokenAddress
    );

    event ClientPayL2(
        address sender,
        uint256 amount,
        address tokenAddress
    );

    event WithdrawLiqudiity(
        address sender,
        address receiver,
        uint256 amount,
        address tokenAddress
    );

    event WithdrawReward(
        address sender,
        address receiver,
        uint256 amount,
        address tokenAddress
    );

    /*************************
     * Cross-chain Functions *
     *************************/

    function clientPayL2(
        address payable _to,
        uint256 _amount,
        address _tokenAddress
    )
        external;
}
