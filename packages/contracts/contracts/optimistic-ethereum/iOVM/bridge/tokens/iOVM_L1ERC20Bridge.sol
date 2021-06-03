// SPDX-License-Identifier: MIT
pragma solidity >0.5.0;
pragma experimental ABIEncoderV2;

/**
 * @title iOVM_L1ERC20Bridge
 */
interface iOVM_L1ERC20Bridge {

    /**********
     * Events *
     **********/

    event ERC20DepositInitiated(
        address indexed _l1Token,
        address indexed _l2Token,
        address indexed _from,
        address _to,
        uint256 _amount,
        bytes _data
    );

    event ERC20WithdrawalFinalized(
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

    function depositERC20(
        address _l1Token,
		address _l2Token,
        uint _amount,
        uint32 _l2Gas,
        bytes calldata _data
    )
        external;

    function depositERC20To(
        address _l1Token,
		address _l2Token,
        address _to,
        uint _amount,
        uint32 _l2Gas,
        bytes calldata _data
    )
        external;


    /*************************
     * Cross-chain Functions *
     *************************/

    function finalizeERC20Withdrawal(
        address _l1Token,
		address _l2Token,
        address _from,
        address _to,
        uint _amount,
        bytes calldata _data
    )
        external;

}
