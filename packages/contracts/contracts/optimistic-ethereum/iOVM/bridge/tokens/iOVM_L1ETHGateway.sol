// SPDX-License-Identifier: MIT
pragma solidity >0.5.0;
pragma experimental ABIEncoderV2;

/**
 * @title iOVM_L1ETHGateway
 */
interface iOVM_L1ETHGateway {

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
        uint32 _l2Gas,
        bytes calldata _data
    )
        external
        payable;

    function depositTo(
        address _to,
        uint32 _l2Gas,
        bytes calldata _data
    )
        external
        payable;

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

    function getFinalizeDepositL2Gas()
        external
        view
        returns(
            uint32
        );
}
