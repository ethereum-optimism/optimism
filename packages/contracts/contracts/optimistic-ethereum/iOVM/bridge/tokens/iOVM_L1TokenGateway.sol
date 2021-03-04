// SPDX-License-Identifier: MIT
pragma solidity >0.5.0;
pragma experimental ABIEncoderV2;

/**
 * @title iOVM_L1TokenGateway
 */
interface iOVM_L1TokenGateway {

    /**********
     * Events *
     **********/

    event DepositInitiated(
        address indexed _from,
        address _to,
        uint256 _amount
    );
  
    event WithdrawalFinalized(
        address indexed _to,
        uint256 _amount
    );


    /********************
     * Public Functions *
     ********************/

    function deposit(
        uint _amount
    )
        external;

    function depositTo(
        address _to,
        uint _amount
    )
        external;


    /*************************
     * Cross-chain Functions *
     *************************/

    function finalizeWithdrawal(
        address _to,
        uint _amount
    )
        external;
}
