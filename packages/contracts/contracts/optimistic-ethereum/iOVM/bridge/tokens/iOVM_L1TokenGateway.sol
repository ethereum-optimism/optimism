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
        address indexed _to,
        uint256 _amount,
        bytes _data
    );

    event WithdrawalFinalized(
        address indexed _from,
        address indexed _to,
        uint256 _amount,
        bytes _data
    );


    /********************
     * Public Functions *
     ********************/

    function deposit(
        uint _amount,
        bytes calldata _data
    )
        external;

    function depositTo(
        address _to,
        uint _amount,
        bytes calldata _data
    )
        external;


    /*************************
     * Cross-chain Functions *
     *************************/

    function finalizeWithdrawal(
        address _from,
        address _to,
        uint _amount,
        bytes calldata _data
    )
        external;
}
