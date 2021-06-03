// SPDX-License-Identifier: MIT
pragma solidity >0.5.0;
pragma experimental ABIEncoderV2;

/**
 * @title iOVM_L2ERC20Bridge
 */
interface iOVM_L2ERC20Bridge {

    /**********
     * Events *
     **********/

    event WithdrawalInitiated(
        address indexed _from,
        address indexed _to,
        uint256 _amount,
        bytes _data
    );

    event DepositFinalized(
        address indexed _l1Token,
        address indexed _l2Token,
        address indexed _from,
        address _to,
        uint256 _amount,
        bytes _data
    );


    /********************
     * Public Functions *
     ********************/

    function withdraw(
        address _l2Token,
        uint _amount,
        uint32 _l1Gas,
        bytes calldata _data
    )
        external;

    function withdrawTo(
        address _l2Token,
        address _to,
        uint _amount,
        uint32 _l1Gas,
        bytes calldata _data
    )
        external;


    /*************************
     * Cross-chain Functions *
     *************************/

    function finalizeDeposit(
        address _l1Token,
        address _l2Token,
        address _from,
        address _to,
        uint _amount,
        bytes calldata _data
    )
        external;

}
